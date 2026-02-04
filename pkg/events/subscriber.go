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
	eventCount = 100
	exists     = "BUSYGROUP Consumer Group name already exists"
)

var _ Subscriber = (*subEventStore)(nil)

var (
	// ErrEmptyStream is returned when stream name is empty.
	ErrEmptyStream = errors.New("stream name cannot be empty")

	// ErrEmptyConsumer is returned when consumer name is empty.
	ErrEmptyConsumer = errors.New("consumer name cannot be empty")

	// ErrEmptyGroup is returned when consumer group name is empty.
	ErrEmptyGroup = errors.New("consumer group cannot be empty")
)

type subEventStore struct {
	client   *redis.Client
	stream   string
	consumer string
	group    string
	logger   logger.Logger
}

func NewSubscriber(url, stream, group, consumer string, logger logger.Logger) (Subscriber, error) {
	if stream == "" {
		return nil, ErrEmptyStream
	}

	if consumer == "" {
		return nil, ErrEmptyConsumer
	}

	if group == "" {
		return nil, ErrEmptyGroup
	}

	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	return &subEventStore{
		client:   redis.NewClient(opts),
		stream:   stream,
		consumer: consumer,
		group:    group,
		logger:   logger,
	}, nil
}

func (es *subEventStore) Subscribe(ctx context.Context, handler EventHandler) error {
	err := es.client.XGroupCreateMkStream(ctx, es.stream, es.group, "$").Err()
	if err != nil && err.Error() != exists {
		return err
	}

	for {
		msgs, err := es.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    es.group,
			Consumer: es.consumer,
			Streams:  []string{es.stream, ">"},
			Count:    eventCount,
			Block:    5 * time.Second,
		}).Result()

		if err != nil && err != redis.Nil {
			if errors.Contains(err, context.Canceled) || errors.Contains(err, context.DeadlineExceeded) {
				return err
			}

			es.logger.Warn(fmt.Sprintf("failed to read from Redis stream: %s", err))

			continue
		}

		if len(msgs) == 0 {
			continue
		}

		es.handle(ctx, msgs[0].Messages, handler)
	}
}

func (es *subEventStore) Close() error {
	return es.client.Close()
}

type redisEvent struct {
	Data map[string]any
}

func (re redisEvent) Encode() (map[string]any, error) {
	return re.Data, nil
}

func (es *subEventStore) handle(ctx context.Context, msgs []redis.XMessage, h EventHandler) {
	for _, msg := range msgs {
		event := redisEvent{
			Data: msg.Values,
		}

		if err := h.Handle(ctx, event); err != nil {
			es.logger.Warn(fmt.Sprintf("failed to handle redis event: %s", err))
			return
		}

		if err := es.client.XAck(ctx, es.stream, es.group, msg.ID).Err(); err != nil {
			es.logger.Warn(fmt.Sprintf("failed to ack redis event: %s", err))
			return
		}
	}
}
