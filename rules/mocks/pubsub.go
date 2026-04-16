// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var _ nats.Publisher = (*mockPubSub)(nil)

type mockPubSub struct {
	fail bool
}

// NewPubSub returns a mock Publisher that succeeds by default.
func NewPubSub() nats.Publisher {
	return &mockPubSub{}
}

// NewFailingPubSub returns a mock Publisher whose Publish always fails.
func NewFailingPubSub() nats.Publisher {
	return &mockPubSub{fail: true}
}

func (ps *mockPubSub) Publish(_ string, _ protomfx.Message) error {
	if ps.fail {
		return messaging.ErrPublishMessage
	}
	return nil
}

func (ps *mockPubSub) PublishAlarm(_ string, _ *protomfx.Alarm) error {
	if ps.fail {
		return messaging.ErrPublishMessage
	}
	return nil
}

func (ps *mockPubSub) Close() error {
	return nil
}
