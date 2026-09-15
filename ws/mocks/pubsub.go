// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/ws"
)

var _ ws.PubSub = (*mockPubSub)(nil)

type MockPubSub interface {
	ws.PubSub
	SetFail(bool)
}

type mockPubSub struct {
	fail bool
}

// NewPubSub returns mock message publisher-subscriber
func NewPubSub() MockPubSub {
	return &mockPubSub{false}
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

func (pubsub *mockPubSub) PublishCommand(string, protomfx.Command) error {
	if pubsub.fail {
		return messaging.ErrPublishCommand
	}
	return nil
}

func (pubsub *mockPubSub) Dispatch(protomfx.Message, *domain.ProfileConfig) error {
	if pubsub.fail {
		return messaging.ErrPublishMessage
	}
	return nil
}

func (pubsub *mockPubSub) SetFail(fail bool) {
	pubsub.fail = fail
}

func (pubsub *mockPubSub) Close() error {
	return nil
}
