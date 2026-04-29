// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/events"
	r "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testStream = "mainflux.publisher.test"

// failingXAdd wraps a real *redis.Client and lets a test inject XAdd
// failures and/or block (gate) the drainer at will.
type failingXAdd struct {
	inner *r.Client

	mu       sync.Mutex
	failLeft int
	failErr  error
	gate     chan struct{} // when non-nil, every XAdd waits for a send/close before returning

	calls int64
}

func newFailingXAdd(client *r.Client) *failingXAdd {
	return &failingXAdd{inner: client}
}

func (f *failingXAdd) setFail(n int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failLeft = n
	f.failErr = err
}

func (f *failingXAdd) setGate(g chan struct{}) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gate = g
}

func (f *failingXAdd) callCount() int64 {
	return atomic.LoadInt64(&f.calls)
}

func (f *failingXAdd) XAdd(ctx context.Context, args *r.XAddArgs) *r.StringCmd {
	atomic.AddInt64(&f.calls, 1)

	f.mu.Lock()
	gate := f.gate
	failLeft := f.failLeft
	failErr := f.failErr
	if failLeft > 0 {
		f.failLeft--
	}
	f.mu.Unlock()

	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			cmd := r.NewStringCmd(ctx)
			cmd.SetErr(ctx.Err())
			return cmd
		}
	}

	if failLeft > 0 {
		cmd := r.NewStringCmd(ctx)
		cmd.SetErr(failErr)
		return cmd
	}

	return f.inner.XAdd(ctx, args)
}

// newPublisher builds a Publisher around a *failingXAdd via the unexported
// seam. Fields left zero on cfg are filled with sane test defaults so each
// test can specify only what it wants to vary.
func newPublisher(t *testing.T, cfg events.PublisherConfig, fx *failingXAdd) events.Publisher {
	t.Helper()
	if cfg.URL == "" {
		cfg.URL = redisURL
	}
	if cfg.BufferSize == 0 {
		cfg.BufferSize = events.DefBufferSize
	}
	if cfg.DrainIntervalInitial == 0 {
		cfg.DrainIntervalInitial = 50 * time.Millisecond
	}
	if cfg.DrainBackoffMax == 0 {
		cfg.DrainBackoffMax = 200 * time.Millisecond
	}
	if cfg.ShutdownDrainTimeout == 0 {
		cfg.ShutdownDrainTimeout = 2 * time.Second
	}
	pub, err := events.NewPublisher(cfg, logger.NewMock())
	require.NoError(t, err)
	if fx != nil {
		swapXAdder(t, pub, fx)
	}
	return pub
}

// swapXAdder replaces the publisher's internal xadder with fx. The seam is
// intentionally unexported, so we reach for the field via reflect+unsafe;
// this is test-only and contained in this file.
func swapXAdder(t *testing.T, pub events.Publisher, fx *failingXAdd) {
	t.Helper()
	v := reflect.ValueOf(pub).Elem()
	field := v.FieldByName("xa")
	require.True(t, field.IsValid(), "publisher must expose internal xa seam")
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(fx))
}

// publishOne builds a deterministic event with a given id.
func publishOne(t *testing.T, pub events.Publisher, id string) {
	t.Helper()
	pub.Publish(context.Background(), events.ThingRemoved{ID: id})
}

func streamLen(t *testing.T, stream string) int64 {
	t.Helper()
	n, err := redisClient.XLen(context.Background(), stream).Result()
	require.NoError(t, err)
	return n
}

func waitStreamLen(t *testing.T, stream string, want int64, timeoutMS int) bool {
	t.Helper()
	deadline := time.Now().Add(time.Duration(timeoutMS) * time.Millisecond)
	for time.Now().Before(deadline) {
		if streamLen(t, stream) >= want {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return streamLen(t, stream) >= want
}

func streamIDs(t *testing.T, stream string) []string {
	t.Helper()
	entries, err := redisClient.XRange(context.Background(), stream, "-", "+").Result()
	require.NoError(t, err)
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if id, _ := e.Values["id"].(string); id != "" {
			out = append(out, id)
		}
	}
	return out
}

func TestPublishesToStream(t *testing.T) {
	require.NoError(t, redisClient.Del(context.Background(), testStream).Err())

	pub := newPublisher(t, events.PublisherConfig{Stream: testStream}, nil)
	defer pub.Close()

	publishOne(t, pub, "alpha")

	require.True(t, waitStreamLen(t, testStream, 1, 2000), "event did not reach stream")

	got := streamIDs(t, testStream)
	require.Equal(t, []string{"alpha"}, got)
}

func TestBufferAbsorbsTransientFailure(t *testing.T) {
	require.NoError(t, redisClient.Del(context.Background(), testStream).Err())

	fx := newFailingXAdd(redisClient)
	fx.setFail(3, errors.New("simulated redis blip"))

	pub := newPublisher(t, events.PublisherConfig{
		Stream:               testStream,
		DrainIntervalInitial: 50 * time.Millisecond,
		DrainBackoffMax:      100 * time.Millisecond,
	}, fx)
	defer pub.Close()

	publishOne(t, pub, "one")
	publishOne(t, pub, "two")
	publishOne(t, pub, "three")

	require.True(t, waitStreamLen(t, testStream, 3, 4000), fmt.Sprintf(
		"events did not arrive after transient failure; got %d, calls=%d",
		streamLen(t, testStream), fx.callCount(),
	))

	got := streamIDs(t, testStream)
	assert.Equal(t, []string{"one", "two", "three"}, got, "order must be preserved across retries")
}

func TestOverflowDropsOldest(t *testing.T) {
	require.NoError(t, redisClient.Del(context.Background(), testStream).Err())

	gate := make(chan struct{})
	fx := newFailingXAdd(redisClient)
	fx.setGate(gate)

	pub := newPublisher(t, events.PublisherConfig{
		Stream:     testStream,
		BufferSize: 2,
	}, fx)
	defer pub.Close()

	// Push the first event and wait until the drainer has picked it up and
	// is parked inside XAdd (gated). Without this synchronisation the
	// drainer goroutine may not be scheduled yet, in which case all events
	// would compete for buffer slots.
	publishOne(t, pub, "a")
	require.Eventually(t, func() bool { return fx.callCount() >= 1 }, time.Second, 10*time.Millisecond,
		"drainer did not pick up the first event")

	// Now the buffer is empty and drainer is gated. Subsequent Publishes
	// will fill the buffer (cap 2) and then start dropping the oldest.
	for _, id := range []string{"b", "c", "d", "e"} {
		publishOne(t, pub, id)
	}

	// Release the gate so the drainer can finish.
	close(gate)

	// Three events must arrive: "a" (the in-flight one) plus the two
	// newest ("d", "e") — "b" and "c" were dropped by drop-oldest.
	require.True(t, waitStreamLen(t, testStream, 3, 3000))
	got := streamIDs(t, testStream)
	assert.Equal(t, []string{"a", "d", "e"}, got)
}

func TestCloseDrainsRemaining(t *testing.T) {
	require.NoError(t, redisClient.Del(context.Background(), testStream).Err())

	pub := newPublisher(t, events.PublisherConfig{
		Stream:               testStream,
		ShutdownDrainTimeout: 3 * time.Second,
	}, nil)

	for i := 0; i < 10; i++ {
		publishOne(t, pub, fmt.Sprintf("e-%02d", i))
	}

	start := time.Now()
	require.NoError(t, pub.Close(), "Close should succeed")
	require.Less(t, time.Since(start), 4*time.Second, "Close took too long")

	assert.Equal(t, int64(10), streamLen(t, testStream), "all events must be drained on Close")
}

func TestCloseGivesUpAfterShutdownDrain(t *testing.T) {
	require.NoError(t, redisClient.Del(context.Background(), testStream).Err())

	fx := newFailingXAdd(redisClient)
	fx.setFail(1<<30, errors.New("redis is down forever"))

	pub := newPublisher(t, events.PublisherConfig{
		Stream:               testStream,
		DrainIntervalInitial: 50 * time.Millisecond,
		DrainBackoffMax:      50 * time.Millisecond,
		ShutdownDrainTimeout: 200 * time.Millisecond,
	}, fx)

	for i := 0; i < 5; i++ {
		publishOne(t, pub, fmt.Sprintf("lost-%d", i))
	}

	start := time.Now()
	require.NoError(t, pub.Close())
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 1*time.Second, "Close must respect ShutdownDrain bound")
}

func TestPublishNonBlocking(t *testing.T) {
	require.NoError(t, redisClient.Del(context.Background(), testStream).Err())

	gate := make(chan struct{})
	fx := newFailingXAdd(redisClient)
	fx.setGate(gate)
	defer close(gate)

	pub := newPublisher(t, events.PublisherConfig{
		Stream:     testStream,
		BufferSize: 4,
	}, fx)
	defer pub.Close()

	// Saturate buffer (the drainer is gated).
	for i := 0; i < 4; i++ {
		publishOne(t, pub, fmt.Sprintf("warm-%d", i))
	}

	// A subsequent Publish should NOT block — even though the buffer is full,
	// the drop-oldest policy makes room synchronously.
	start := time.Now()
	publishOne(t, pub, "fast")
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 50*time.Millisecond, "Publish must be non-blocking even when buffer is saturated")
}

func TestOrderPreservedOnTransientFailure(t *testing.T) {
	require.NoError(t, redisClient.Del(context.Background(), testStream).Err())

	fx := newFailingXAdd(redisClient)
	fx.setFail(2, errors.New("transient"))

	pub := newPublisher(t, events.PublisherConfig{
		Stream:               testStream,
		DrainIntervalInitial: 30 * time.Millisecond,
		DrainBackoffMax:      30 * time.Millisecond,
	}, fx)
	defer pub.Close()

	ids := []string{"k1", "k2", "k3", "k4"}
	for _, id := range ids {
		publishOne(t, pub, id)
	}

	require.True(t, waitStreamLen(t, testStream, int64(len(ids)), 3000))
	assert.Equal(t, ids, streamIDs(t, testStream))
}
