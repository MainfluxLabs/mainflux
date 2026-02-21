// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var _ messaging.PubSub = (*mockPubSub)(nil)

type mockPubSub struct {
	fail bool
}

// NewPubSub returns a mock PubSub that succeeds by default.
func NewPubSub() messaging.PubSub {
	return &mockPubSub{}
}

// NewFailingPubSub returns a mock PubSub whose Publish always fails.
func NewFailingPubSub() messaging.PubSub {
	return &mockPubSub{fail: true}
}

func (ps *mockPubSub) Publish(protomfx.Message) error {
	if ps.fail {
		return messaging.ErrPublishMessage
	}
	return nil
}

func (ps *mockPubSub) Subscribe(string, string, messaging.MessageHandler) error {
	return nil
}

func (ps *mockPubSub) Unsubscribe(string, string) error {
	return nil
}

func (ps *mockPubSub) Close() error {
	return nil
}
