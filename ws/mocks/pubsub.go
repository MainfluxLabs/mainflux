// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/ws"
)

var _ ws.PubSub = (*mockPubSub)(nil)

type MockPubSub interface {
	ws.PubSub
	SetFail(bool)
	Published() []pkgmocks.PublishedMessage
	Dispatched() []protomfx.Message
}

type mockPubSub struct {
	mu         sync.Mutex
	fail       bool
	published  []pkgmocks.PublishedMessage
	dispatched []protomfx.Message
}

// NewPubSub returns mock message publisher-subscriber
func NewPubSub() MockPubSub {
	return &mockPubSub{}
}

func (pubsub *mockPubSub) Subscribe(string, string, messaging.MessageHandler) error {
	if pubsub.fail {
		return messaging.ErrFailedSubscribe
	}
	return nil
}

func (pubsub *mockPubSub) Unsubscribe(string, string) error {
	if pubsub.fail {
		return messaging.ErrFailedUnsubscribe
	}
	return nil
}

func (pubsub *mockPubSub) Publish(subject string, msg protomfx.Message) error {
	if pubsub.fail {
		return messaging.ErrPublishMessage
	}

	pubsub.mu.Lock()
	defer pubsub.mu.Unlock()

	pubsub.published = append(pubsub.published, pkgmocks.PublishedMessage{Subject: subject, Message: msg})
	return nil
}

func (pubsub *mockPubSub) PublishCommand(string, protomfx.Command) error {
	if pubsub.fail {
		return messaging.ErrPublishCommand
	}
	return nil
}

func (pubsub *mockPubSub) Dispatch(msg protomfx.Message, _ *domain.ProfileConfig) error {
	if pubsub.fail {
		return messaging.ErrPublishMessage
	}

	pubsub.mu.Lock()
	defer pubsub.mu.Unlock()

	pubsub.dispatched = append(pubsub.dispatched, msg)
	return nil
}

func (pubsub *mockPubSub) Published() []pkgmocks.PublishedMessage {
	pubsub.mu.Lock()
	defer pubsub.mu.Unlock()

	return append([]pkgmocks.PublishedMessage(nil), pubsub.published...)
}

func (pubsub *mockPubSub) Dispatched() []protomfx.Message {
	pubsub.mu.Lock()
	defer pubsub.mu.Unlock()

	return append([]protomfx.Message(nil), pubsub.dispatched...)
}

func (pubsub *mockPubSub) SetFail(fail bool) {
	pubsub.fail = fail
}

func (pubsub *mockPubSub) Close() error {
	return nil
}
