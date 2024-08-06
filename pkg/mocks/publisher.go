// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

type mockPublisher struct{}

// NewPublisher returns mock message publisher.
func NewPublisher() messaging.Publisher {
	return mockPublisher{}
}

func (pub mockPublisher) Publish(msg protomfx.Message) error {
	return nil
}

func (pub mockPublisher) Close() error {
	return nil
}
