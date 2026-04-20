// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var _ nats.Publisher = (*mockPublisher)(nil)

type mockPublisher struct {
	fail bool
}

// NewPublisher returns a mock Publisher that succeeds by default.
func NewPublisher() nats.Publisher {
	return &mockPublisher{}
}

// NewFailingPublisher returns a mock Publisher whose Publish always fails.
func NewFailingPublisher() nats.Publisher {
	return &mockPublisher{fail: true}
}

func (ps *mockPublisher) Publish(string, protomfx.Message) error {
	if ps.fail {
		return messaging.ErrPublishMessage
	}
	return nil
}

func (ps *mockPublisher) PublishAlarm(string, protomfx.Alarm) error {
	if ps.fail {
		return messaging.ErrPublishMessage
	}
	return nil
}

func (ps *mockPublisher) Close() error {
	return nil
}
