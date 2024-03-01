// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
)

type mockPublisher struct{}

// NewPublisher returns mock message publisher.
func NewPublisher() messaging.Publisher {
	return mockPublisher{}
}

func (pub mockPublisher) Publish(profile mainflux.Profile, msg messaging.Message) error {
	return nil
}

func (pub mockPublisher) Close() error {
	return nil
}
