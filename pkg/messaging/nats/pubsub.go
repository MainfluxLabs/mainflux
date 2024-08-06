// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"fmt"
	"sync"

	"github.com/gogo/protobuf/proto"

	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	broker "github.com/nats-io/nats.go"
)

const chansPrefix = "channels"

var _ messaging.PubSub = (*pubsub)(nil)

type subscription struct {
	*broker.Subscription
	cancel func() error
}

type pubsub struct {
	publisher
	logger        log.Logger
	mu            sync.Mutex
	queue         string
	subscriptions map[string]map[string]subscription
}

// NewPubSub returns NATS message publisher/subscriber.
// Parameter queue specifies the queue for the Subscribe method.
// If queue is specified (is not an empty string), Subscribe method
// will execute NATS QueueSubscribe which is conceptually different
// from ordinary subscribe. For more information, please take a look
// here: https://docs.nats.io/developing-with-nats/receiving/queues.
// If the queue is empty, Subscribe will be used.
func NewPubSub(url, queue string, logger log.Logger) (messaging.PubSub, error) {
	conn, err := broker.Connect(url, broker.MaxReconnects(maxReconnects))
	if err != nil {
		return nil, err
	}

	ret := &pubsub{
		publisher: publisher{
			conn: conn,
		},
		queue:         queue,
		logger:        logger,
		subscriptions: make(map[string]map[string]subscription),
	}
	return ret, nil
}

func (ps *pubsub) Subscribe(id, topic string, handler messaging.MessageHandler) error {
	if id == "" {
		return messaging.ErrEmptyID
	}
	if topic == "" {
		return messaging.ErrEmptyTopic
	}

	ps.mu.Lock()
	// Check topic
	s, ok := ps.subscriptions[topic]
	if ok {
		// Check client ID
		if _, ok := s[id]; ok {
			// Unlocking, so that Unsubscribe() can access ps.subscriptions
			ps.mu.Unlock()
			if err := ps.Unsubscribe(id, topic); err != nil {
				return err
			}

			ps.mu.Lock()
			// value of s can be changed while ps.mu is unlocked
			s = ps.subscriptions[topic]
		}
	}
	defer ps.mu.Unlock()
	if s == nil {
		s = make(map[string]subscription)
		ps.subscriptions[topic] = s
	}

	nh := ps.natsHandler(handler)

	if ps.queue != "" {
		sub, err := ps.conn.QueueSubscribe(topic, ps.queue, nh)
		if err != nil {
			return err
		}
		s[id] = subscription{
			Subscription: sub,
			cancel:       handler.Cancel,
		}
		return nil
	}
	sub, err := ps.conn.Subscribe(topic, nh)
	if err != nil {
		return err
	}
	s[id] = subscription{
		Subscription: sub,
		cancel:       handler.Cancel,
	}

	return nil
}

func (ps *pubsub) Unsubscribe(id, topic string) error {
	if id == "" {
		return messaging.ErrEmptyID
	}
	if topic == "" {
		return messaging.ErrEmptyTopic
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()
	// Check topic
	s, ok := ps.subscriptions[topic]
	if !ok {
		return messaging.ErrNotSubscribed
	}
	// Check topic ID
	current, ok := s[id]
	if !ok {
		return messaging.ErrNotSubscribed
	}
	if current.cancel != nil {
		if err := current.cancel(); err != nil {
			return err
		}
	}
	if err := current.Unsubscribe(); err != nil {
		return err
	}

	delete(s, id)
	if len(s) == 0 {
		delete(ps.subscriptions, topic)
	}
	return nil
}

func (ps *pubsub) natsHandler(h messaging.MessageHandler) broker.MsgHandler {
	return func(m *broker.Msg) {
		var msg protomfx.Message
		if err := proto.Unmarshal(m.Data, &msg); err != nil {
			ps.logger.Warn(fmt.Sprintf("Failed to unmarshal received message: %s", err))
			return
		}
		if err := h.Handle(msg); err != nil {
			ps.logger.Warn(fmt.Sprintf("Failed to handle Mainflux message: %s", err))
		}
	}
}
