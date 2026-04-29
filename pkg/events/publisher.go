// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/go-redis/redis/v8"
)

const (
	// DefStreamMaxLen is the default approx number of messages in the Redis event stream.
	DefStreamMaxLen = 100000

	// DefBufferSize is the default capacity of the publish
	// buffer channel.
	DefBufferSize = 10000

	// DefDrainIntervalInitial is the initial sleep duration between XADD retries when
	// emitting an event fails.
	DefDrainIntervalInitial = time.Second

	// DefDrainBackoffMax is the maximum backoff duration between XADD retries.
	DefDrainBackoffMax = 30 * time.Second

	// DefShutdownDrainTimeout is the maximum duration that Close() can spends trying to drain
	// the remaining buffered events.
	DefShutdownDrainTimeout = 5 * time.Second

	// attemptTimeout is an upper limit of a single Redis XADD call so the drainer can't block
	// indefinitely on single emit attempt.
	attemptTimeout = 5 * time.Second
)

// Publisher delivers Events to the Redis event store asynchronously.
// It uses an in-memory buffer to queue and retry failed events.
type Publisher interface {
	// Publish queues the passed event for delivery by sending it to the buffer
	// channel. It's asynchronous, meaning that it doesn't block the caller
	// and as such returns no errors. If the underlying event buffer is full,
	// the oldest queued event is dropped to make space.
	Publish(ctx context.Context, e Event)

	// Close stops the event drainer goroutine, attempts a final drain,
	// and closes the underlying Redis client.
	Close() error
}

type PublisherConfig struct {
	URL    string
	Stream string

	// MaxLen is the approximate stream retention (XADD MAXLEN ~).
	MaxLen int64

	// BufferSize is the capacity of the buffer channel holding events.
	BufferSize int

	// DrainIntervalInitial is the initial XADD retry sleep on failure.
	DrainIntervalInitial time.Duration

	// DrainBackoffMax caps the exponential retry backoff.
	DrainBackoffMax time.Duration

	// ShutdownDrainTimeout bounds the final drain in Close().
	ShutdownDrainTimeout time.Duration
}

// xadder is the small slice of *redis.Client the drainer uses. It exists so
// tests can inject failures without standing up a full mock client.
type xadder interface {
	XAdd(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd
}

var _ Publisher = (*publisher)(nil)

type publisher struct {
	client *redis.Client
	xa     xadder

	stream        string
	maxLen        int64
	drainInterval time.Duration
	drainBackoff  time.Duration
	shutdownDrain time.Duration

	bufferCh chan RedisEvent
	logger   logger.Logger

	stopCh    chan struct{}
	doneCh    chan struct{}
	closeOnce sync.Once
}

// NewPublisher returns a Publisher that buffers events in process and drains
// them to Redis in the background.
func NewPublisher(cfg PublisherConfig, log logger.Logger) (Publisher, error) {
	if cfg.Stream == "" {
		return nil, ErrEmptyStream
	}

	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opts)

	p := &publisher{
		client:        client,
		xa:            client,
		stream:        cfg.Stream,
		maxLen:        cfg.MaxLen,
		drainInterval: cfg.DrainIntervalInitial,
		drainBackoff:  cfg.DrainBackoffMax,
		shutdownDrain: cfg.ShutdownDrainTimeout,
		bufferCh:      make(chan RedisEvent, cfg.BufferSize),
		logger:        log,
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
	}

	go p.runDrainer()
	return p, nil
}

func (p *publisher) Publish(_ context.Context, e Event) {
	re := e.Encode()

	// Attempt to enqueue the encoded event onto the buffer channel. If we succeed, simply return.
	select {
	case p.bufferCh <- re:
		return
	default:
	}

	// We couldn't enqueue the event, meaning the buffer is full.
	// Drop the oldest queued event (by receiving from the channel and voiding) to make room,
	// then attempt to enqueue the event again.
	select {
	case <-p.bufferCh:
		p.logger.Warn(fmt.Sprintf(
			"event publisher buffer full for stream %q; dropping oldest event",
			p.stream,
		))
	default:
		// Drainer goroutine managed to consume the event in the meantime.
	}

	// Buffer channel should have room at this point - try to enqueue the event again.
	select {
	case p.bufferCh <- re:
	default:
		// Buffer still full - another Publish() beat us to the slot that just got freed in the
		// buffer. Simply drop the incoming event and log.
		p.logger.Warn(fmt.Sprintf(
			"event publisher buffer full for stream %q; dropping incoming %s event",
			p.stream, re.Operation(),
		))
	}
}

func (p *publisher) Close() error {
	// Publisher closed from the outside - close stopCh to signal to the drainer goroutine that it should stop.
	p.closeOnce.Do(func() {
		close(p.stopCh)
	})

	// Block until the drainer notices the stop signal and finishes the finalDrain pass.
	<-p.doneCh
	return p.client.Close()
}

// runDrainer is ran in a background goroutine by the publisher. It receives events off
// of the buffer channel and emits them using the Redis client.
func (p *publisher) runDrainer() {
	defer close(p.doneCh)

	for {
		select {
		case ev := <-p.bufferCh:
			if !p.emit(ev, p.stopCh) {
				// Publisher was signalled to stop from the outside during a delivery attempt of `ev`.
				// Carry this event into the finalDrain pass so it gets retried at least once more.
				p.finalDrain(&ev)
				return
			}
		case <-p.stopCh:
			p.finalDrain(nil)
			return
		}
	}
}

// finalDrain is the last pass that attempts to emit all remaining queued events in the buffer (including the passed pendingEvent)
// after the Publisher has been signalled to stop from the outside, all the while respecting the maximum p.shutdownDrain time.
// Events that aren't emitted in time are dropped and lost.
func (p *publisher) finalDrain(pendingEvent *RedisEvent) {
	stop := make(chan struct{})
	timer := time.AfterFunc(p.shutdownDrain, func() { close(stop) })
	defer timer.Stop()

	if pendingEvent != nil {
		// Attempt to emit the pending event but respect the `p.shutdownDrain` duration.
		if !p.emit(*pendingEvent, stop) {
			return
		}
	}

	for {
		select {
		case ev := <-p.bufferCh:
			if !p.emit(ev, stop) {
				return
			}
		case <-stop:
			// p.shutdownDrain exceeded
			return
		default:
			// Buffer empty, no more events to process
			return
		}
	}
}

// emit attempts to XADD a specific event repeatedly until success or
// until signalled to stop via the passed stop channel.
// returns true on successful delivery, and false if signalled to stop in the midst of a retry attempt.
func (p *publisher) emit(ev RedisEvent, stop <-chan struct{}) bool {
	backoff := p.drainInterval

	for {
		ctx, cancel := context.WithTimeout(context.Background(), attemptTimeout)
		err := p.xa.XAdd(ctx, &redis.XAddArgs{
			Stream:       p.stream,
			MaxLenApprox: p.maxLen,
			Values:       map[string]any(ev),
		}).Err()
		cancel()

		if err == nil {
			return true
		}

		p.logger.Warn(fmt.Sprintf(
			"failed to publish %s event to stream %q: %s",
			ev.Operation(), p.stream, err,
		))

		// We failed to emit the event to Redis - retry after sleeping for the backoff time,
		// or return if signalled to stop from the outside.
		select {
		case <-time.After(backoff):
		case <-stop:
			return false
		}

		backoff = nextDrainBackoff(backoff, p.drainBackoff)
	}
}

func nextDrainBackoff(current, max time.Duration) time.Duration {
	next := current * 2
	if next > max {
		return max
	}
	return next
}
