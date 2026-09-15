// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var _ messaging.CommandPublisher = (*commandPublisherMock)(nil)

type commandPublisherMock struct{}

// NewCommandPublisher returns a mock command publisher.
func NewCommandPublisher() messaging.CommandPublisher {
	return commandPublisherMock{}
}

func (pub commandPublisherMock) PublishCommand(_ string, _ protomfx.Command) error {
	return nil
}
