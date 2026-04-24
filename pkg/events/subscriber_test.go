// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/events"
	r "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingHandler captures every event it receives and optionally returns a
// configured error for the first N invocations.
type recordingHandler struct {
	mu        sync.Mutex
	received  []events.Event
	failTimes int
	failErr   error
}

func (h *recordingHandler) Handle(_ context.Context, e events.Event) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.received = append(h.received, e)
	if h.failTimes > 0 {
		h.failTimes--
		return h.failErr
	}
	return nil
}

func (h *recordingHandler) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.received)
}

// runSubscriber starts Subscribe in a goroutine and returns a cancel fn and a
// done channel that closes when Subscribe returns.
func runSubscriber(t *testing.T, sub events.Subscriber, h events.EventHandler) (context.CancelFunc, <-chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := sub.Subscribe(ctx, h); err != nil && !errors.Is(err, context.Canceled) {
			t.Logf("Subscribe returned: %v", err)
		}
	}()
	return cancel, done
}

func waitForCount(t *testing.T, h *recordingHandler, want, timeoutMS int) bool {
	t.Helper()
	deadline := time.Now().Add(time.Duration(timeoutMS) * time.Millisecond)
	for time.Now().Before(deadline) {
		if h.count() >= want {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return h.count() >= want
}

func publishThingRemoved(t *testing.T, stream, id string) {
	t.Helper()
	err := redisClient.XAdd(context.Background(), &r.XAddArgs{
		Stream: stream,
		Values: map[string]any(events.ThingRemoved{ID: id}.Encode()),
	}).Err()
	require.NoError(t, err, "publishing event")
}

func flushRedis(t *testing.T) {
	t.Helper()
	require.NoError(t, redisClient.FlushAll(context.Background()).Err())
}

func newSub(t *testing.T, name, stream string) events.Subscriber {
	t.Helper()
	sub, err := events.NewSubscriber(events.SubscriberConfig{
		URL:    redisURL,
		Stream: stream,
		Name:   name,
	}, logger.NewMock())
	require.NoError(t, err)
	return sub
}

func cursorValue(t *testing.T, name, stream string) string {
	t.Helper()
	v, err := redisClient.Get(context.Background(), fmt.Sprintf("mainflux:cursor:%s:%s", name, stream)).Result()
	if errors.Is(err, r.Nil) {
		return ""
	}
	require.NoError(t, err)
	return v
}

func TestNewSubscriberValidatesInput(t *testing.T) {
	_, err := events.NewSubscriber(events.SubscriberConfig{URL: redisURL, Name: "x"}, logger.NewMock())
	assert.Equal(t, events.ErrEmptyStream, err)

	_, err = events.NewSubscriber(events.SubscriberConfig{URL: redisURL, Stream: events.ThingsStream}, logger.NewMock())
	assert.Equal(t, events.ErrEmptyName, err)

	_, err = events.NewSubscriber(events.SubscriberConfig{URL: "::bogus::", Stream: events.ThingsStream, Name: "x"}, logger.NewMock())
	assert.Error(t, err)
}

func TestFirstBootSkipsHistory(t *testing.T) {
	flushRedis(t)

	// Publish events before any subscriber exists.
	publishThingRemoved(t, events.ThingsStream, "pre-boot-1")
	publishThingRemoved(t, events.ThingsStream, "pre-boot-2")

	h := &recordingHandler{}
	sub := newSub(t, "first-boot", events.ThingsStream)
	cancel, done := runSubscriber(t, sub, h)
	defer func() {
		cancel()
		<-done
		_ = sub.Close()
	}()

	// Give the subscriber time to enter its blocking read.
	time.Sleep(300 * time.Millisecond)

	// New events after boot should be delivered.
	publishThingRemoved(t, events.ThingsStream, "post-boot-1")
	require.True(t, waitForCount(t, h, 1, 2000), "expected post-boot event to be delivered")

	// Pre-boot events must not have been replayed.
	assert.Equal(t, 1, h.count())
	got := h.received[0].(events.ThingRemoved)
	assert.Equal(t, "post-boot-1", got.ID)
}

func TestCursorPersistsAcrossRestart(t *testing.T) {
	flushRedis(t)

	h1 := &recordingHandler{}
	sub1 := newSub(t, "persist", events.ThingsStream)
	cancel1, done1 := runSubscriber(t, sub1, h1)

	time.Sleep(300 * time.Millisecond)
	publishThingRemoved(t, events.ThingsStream, "a")
	require.True(t, waitForCount(t, h1, 1, 2000))

	cancel1()
	<-done1
	_ = sub1.Close()

	firstCursor := cursorValue(t, "persist", events.ThingsStream)
	require.NotEmpty(t, firstCursor, "cursor should be persisted after successful delivery")

	// Publish while no subscriber is running.
	publishThingRemoved(t, events.ThingsStream, "b")

	// Restart — should pick up "b" from the stored cursor, not "$".
	h2 := &recordingHandler{}
	sub2 := newSub(t, "persist", events.ThingsStream)
	cancel2, done2 := runSubscriber(t, sub2, h2)
	defer func() {
		cancel2()
		<-done2
		_ = sub2.Close()
	}()

	require.True(t, waitForCount(t, h2, 1, 2000), "expected event published while subscriber was down to be delivered on restart")
	assert.Equal(t, "b", h2.received[0].(events.ThingRemoved).ID)
}

func TestUnknownOperationIsSkipped(t *testing.T) {
	flushRedis(t)

	h := &recordingHandler{}
	sub := newSub(t, "unknown-op", events.ThingsStream)
	cancel, done := runSubscriber(t, sub, h)
	defer func() {
		cancel()
		<-done
		_ = sub.Close()
	}()

	time.Sleep(300 * time.Millisecond)

	// Raw XAdd with an operation the decoder does not recognize.
	err := redisClient.XAdd(context.Background(), &r.XAddArgs{
		Stream: events.ThingsStream,
		Values: map[string]any{"operation": "thing.bogus", "id": "x"},
	}).Err()
	require.NoError(t, err)

	// Follow with a known event; successful delivery implies the cursor
	// advanced past the bogus entry.
	publishThingRemoved(t, events.ThingsStream, "after-bogus")
	require.True(t, waitForCount(t, h, 1, 2000))
	assert.Equal(t, "after-bogus", h.received[0].(events.ThingRemoved).ID)
}

func TestHandlerErrorRetriesThenSkips(t *testing.T) {
	flushRedis(t)

	// Fail twice — one attempt plus one retry — which triggers the skip path
	// per the subscriber's single-retry policy.
	h := &recordingHandler{failTimes: 2, failErr: errors.New("boom")}

	sub := newSub(t, "handler-err", events.ThingsStream)
	cancel, done := runSubscriber(t, sub, h)
	defer func() {
		cancel()
		<-done
		_ = sub.Close()
	}()

	time.Sleep(300 * time.Millisecond)

	publishThingRemoved(t, events.ThingsStream, "will-fail")
	// The subscriber should retry once (so Handle is called twice) before giving up.
	require.True(t, waitForCount(t, h, 2, 2000), "expected Handle to be invoked twice (initial + 1 retry)")

	// Cursor must have advanced so the next event is delivered immediately.
	publishThingRemoved(t, events.ThingsStream, "after-fail")
	require.True(t, waitForCount(t, h, 3, 2000), "expected subsequent event to be delivered after skip")
	assert.Equal(t, "after-fail", h.received[2].(events.ThingRemoved).ID)
}

func TestCursorMatchesLastDeliveredID(t *testing.T) {
	flushRedis(t)

	h := &recordingHandler{}
	sub := newSub(t, "cursor-id", events.ThingsStream)
	cancel, done := runSubscriber(t, sub, h)
	defer func() {
		cancel()
		<-done
		_ = sub.Close()
	}()

	time.Sleep(300 * time.Millisecond)

	// Publish three events and capture the last message ID.
	publishThingRemoved(t, events.ThingsStream, "a")
	publishThingRemoved(t, events.ThingsStream, "b")
	publishThingRemoved(t, events.ThingsStream, "c")

	require.True(t, waitForCount(t, h, 3, 2000))
	time.Sleep(200 * time.Millisecond) // let the cursor write settle

	// XRANGE + to get the newest entry's ID.
	entries, err := redisClient.XRevRangeN(context.Background(), events.ThingsStream, "+", "-", 1).Result()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, entries[0].ID, cursorValue(t, "cursor-id", events.ThingsStream))
}

func TestCloseInterruptsBlockedRead(t *testing.T) {
	flushRedis(t)

	h := &recordingHandler{}
	sub := newSub(t, "close", events.ThingsStream)
	cancel, done := runSubscriber(t, sub, h)

	// Let the subscriber enter the blocking XRead.
	time.Sleep(300 * time.Millisecond)

	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Subscribe did not return within 3s after cancel")
	}
	_ = sub.Close()
}
