// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/go-redis/redis/v8"
)

const (
	tailCursor = "$"

	readCount     = 100
	blockDuration = time.Second

	handlerRetries = 1
	handlerBackoff = 500 * time.Millisecond

	readBackoffInitial = time.Second
	readBackoffMax     = 30 * time.Second
)

var _ Subscriber = (*subEventStore)(nil)

var (
	// ErrEmptyStream is returned when stream name is empty.
	ErrEmptyStream = errors.New("stream name cannot be empty")

	// ErrEmptyName is returned when the subscriber name is empty.
	ErrEmptyName = errors.New("subscriber name cannot be empty")
)

// Subscriber specifies event subscription API.
type Subscriber interface {
	// Subscribe subscribes to the event stream and consumes events.
	Subscribe(ctx context.Context, handler EventHandler) error

	// Close gracefully closes event subscriber's connection.
	Close() error
}

// SubscriberConfig holds the parameters for creating a Subscriber.
type SubscriberConfig struct {
	// URL is the Redis connection URL (e.g. redis://host:6379/0).
	URL string
	// Stream is the name of the Redis stream to read from.
	Stream string
	// Name identifies this subscriber. Used in the cursor key so
	// independent subscribers maintain independent cursors on the same
	// stream.
	Name string
}

type subEventStore struct {
	client *redis.Client
	stream string
	name   string
	cursor string
	logger logger.Logger

	cancel context.CancelFunc
}

// NewSubscriber returns a Subscriber that reads from a Redis stream using
// XRead and tracks its position via a per-subscriber Redis key.
func NewSubscriber(cfg SubscriberConfig, log logger.Logger) (Subscriber, error) {
	if cfg.Stream == "" {
		return nil, ErrEmptyStream
	}

	if cfg.Name == "" {
		return nil, ErrEmptyName
	}

	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, err
	}

	return &subEventStore{
		client: redis.NewClient(opts),
		stream: cfg.Stream,
		name:   cfg.Name,
		cursor: cursorKey(cfg.Name, cfg.Stream),
		logger: log,
	}, nil
}

func (es *subEventStore) Subscribe(ctx context.Context, handler EventHandler) error {
	ctx, cancel := context.WithCancel(ctx)
	es.cancel = cancel
	defer cancel()

	startID, err := es.loadCursor(ctx)
	if err != nil {
		return err
	}

	backoff := readBackoffInitial
	lastID := startID

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		msgs, err := es.client.XRead(ctx, &redis.XReadArgs{
			Streams: []string{es.stream, lastID},
			Count:   readCount,
			Block:   blockDuration,
		}).Result()

		if err != nil {
			if err == redis.Nil {
				continue
			}

			if errors.Contains(err, context.Canceled) || errors.Contains(err, context.DeadlineExceeded) {
				return err
			}

			es.logger.Warn(fmt.Sprintf("failed to read from redis stream %q: %s", es.stream, err))
			if sleepErr := sleepCtx(ctx, backoff); sleepErr != nil {
				return sleepErr
			}

			backoff = nextBackoff(backoff)
			continue
		}

		backoff = readBackoffInitial

		if len(msgs) == 0 || len(msgs[0].Messages) == 0 {
			continue
		}

		processedID := es.handleBatch(ctx, msgs[0].Messages, handler)
		if processedID != "" {
			lastID = processedID
			if err := es.saveCursor(processedID); err != nil {
				es.logger.Warn(fmt.Sprintf("failed to persist cursor for %q: %s", es.name, err))
			}
		}
	}
}

func (es *subEventStore) Close() error {
	if es.cancel != nil {
		es.cancel()
	}

	return es.client.Close()
}

// handleBatch processes a batch of messages (events) using an EventHandler and returns the ID of the last
// one received.
func (es *subEventStore) handleBatch(ctx context.Context, msgs []redis.XMessage, h EventHandler) string {
	var lastID string

	for _, msg := range msgs {
		if ctx.Err() != nil {
			return lastID
		}

		re := RedisEvent(msg.Values)
		event := decodeEvent(re)
		if event == nil {
			es.logger.Warn(fmt.Sprintf(
				"unknown operation %q in stream %q (id=%s); skipping",
				re.Operation(), es.stream, msg.ID,
			))
			lastID = msg.ID
			continue
		}

		if err := es.dispatch(ctx, h, event); err != nil {
			es.logger.Warn(fmt.Sprintf(
				"giving up on event %s (operation=%s) after retries: %s",
				msg.ID, re.Operation(), err,
			))
		}
		lastID = msg.ID
	}
	return lastID
}

// dispatch processes a single event using an EventHandler, retrying at most once if
// .Handle() errors.
func (es *subEventStore) dispatch(ctx context.Context, h EventHandler, event Event) error {
	var err error

	for attempt := 0; attempt <= handlerRetries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err = h.Handle(ctx, event); err == nil {
			return nil
		}

		if attempt < handlerRetries {
			if sleepErr := sleepCtx(ctx, handlerBackoff); sleepErr != nil {
				return sleepErr
			}
		}
	}

	return err
}

// loadCursor returns the starting stream ID for this subscriber. If no
// persisted cursor exists, it starts reading from `$`.
func (es *subEventStore) loadCursor(ctx context.Context) (string, error) {
	val, err := es.client.Get(ctx, es.cursor).Result()
	if err == redis.Nil {
		return tailCursor, nil
	}

	if err != nil {
		return "", err
	}

	if val == "" {
		return tailCursor, nil
	}

	return val, nil
}

// saveCursor persists the last received message (event) ID. It uses a seaparate
// detached context to avoid getting interrupted by outside cancellations and potentially
// failing to persist the ID of a message that was already received.
func (es *subEventStore) saveCursor(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return es.client.Set(ctx, es.cursor, id, 0).Err()
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

func nextBackoff(current time.Duration) time.Duration {
	next := current * 2
	if next > readBackoffMax {
		return readBackoffMax
	}

	return next
}
