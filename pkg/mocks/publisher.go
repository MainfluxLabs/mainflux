// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

type mockPublisher struct{}

// NewPublisher returns mock message publisher.
func NewPublisher() nats.Publisher {
	return mockPublisher{}
}

func (pub mockPublisher) Publish(_ string, msg protomfx.Message) error {
	return nil
}

func (pub mockPublisher) PublishAlarm(_ string, alarm protomfx.Alarm) error {
	return nil
}

func (pub mockPublisher) PublishCommand(_ string, cmd protomfx.Command) error {
	return nil
}

func (pub mockPublisher) Close() error {
	return nil
}
