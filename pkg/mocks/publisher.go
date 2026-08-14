// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

// Publisher covers the methods this mock's callers (http/mqtt/modbus/downlinks,
// pkg/sdk tests) use.
type Publisher interface {
	messaging.CommandPublisher
	messaging.MessageDispatcher
}

type mockPublisher struct{}

// NewPublisher returns mock message publisher.
func NewPublisher() Publisher {
	return mockPublisher{}
}

func (pub mockPublisher) PublishCommand(_ string, cmd protomfx.Command) error {
	return nil
}

func (pub mockPublisher) PublishByFlags(_ protomfx.Message, _ *domain.ProfileConfig) error {
	return nil
}
