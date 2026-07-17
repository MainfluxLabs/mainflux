// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"fmt"

	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

// Forwarder specifies MQTT forwarder interface API.
type Forwarder interface {
	// Forward subscribes to the CommandSubscriber and
	// publishes commands using provided CommandPublisher.
	Forward(id string, sub messaging.CommandSubscriber, pub messaging.CommandPublisher) error
}

type forwarder struct {
	topics []string
	logger log.Logger
}

// NewForwarder returns new Forwarder implementation.
func NewForwarder(topics []string, logger log.Logger) Forwarder {
	return forwarder{
		topics: topics,
		logger: logger,
	}
}

func (f forwarder) Forward(id string, sub messaging.CommandSubscriber, pub messaging.CommandPublisher) error {
	for _, topic := range f.topics {
		if err := sub.SubscribeCommands(id, topic, handle(pub, f.logger)); err != nil {
			return err
		}
	}

	return nil
}

func handle(pub messaging.CommandPublisher, logger log.Logger) handleFunc {
	return func(subject string, cmd protomfx.Command) error {
		if cmd.Protocol == protocol {
			return nil
		}

		go func() {
			if err := pub.PublishCommand(subject, cmd); err != nil {
				logger.Warn(fmt.Sprintf("Failed to forward command: %s", err))
			}
		}()

		return nil
	}
}

type handleFunc func(subject string, cmd protomfx.Command) error

func (h handleFunc) Handle(subject string, cmd protomfx.Command) error {
	return h(subject, cmd)
}

func (h handleFunc) Cancel() error {
	return nil
}
