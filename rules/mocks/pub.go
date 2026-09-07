// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/rules"
)

var _ rules.Publisher = (*mockPublisher)(nil)

type mockPublisher struct {
	fail bool
}

// NewPublisher returns a mock Publisher that succeeds by default.
func NewPublisher() rules.Publisher {
	return &mockPublisher{}
}

// NewFailingPublisher returns a mock Publisher whose Publish always fails.
func NewFailingPublisher() rules.Publisher {
	return &mockPublisher{fail: true}
}

func (ps *mockPublisher) PublishAlarm(string, protomfx.Alarm) error {
	if ps.fail {
		return messaging.ErrPublishAlarm
	}
	return nil
}

func (ps *mockPublisher) PublishNotification(string, protomfx.Notification) error {
	if ps.fail {
		return messaging.ErrPublishNotification
	}
	return nil
}

func (ps *mockPublisher) PublishWebhook(string, protomfx.Webhook) error {
	if ps.fail {
		return messaging.ErrPublishWebhook
	}
	return nil
}
