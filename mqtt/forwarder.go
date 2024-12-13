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
	// Forward subscribes to the Subscriber and
	// publishes messages using provided Publisher.
	Forward(id string, sub messaging.Subscriber, pub messaging.Publisher) error
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

func (f forwarder) Forward(id string, sub messaging.Subscriber, pub messaging.Publisher) error {
	for _, topic := range f.topics {
		if err := sub.Subscribe(id, topic, handle(pub, f.logger)); err != nil {
			return err
		}
	}

	return nil
}

func handle(pub messaging.Publisher, logger log.Logger) handleFunc {
	return func(msg protomfx.Message) error {
		if msg.Protocol == protocol {
			return nil
		}

		go func() {
			if err := pub.Publish(msg); err != nil {
				logger.Warn(fmt.Sprintf("Failed to forward message: %s", err))
			}
		}()

		return nil
	}
}

type handleFunc func(msg protomfx.Message) error

func (h handleFunc) Handle(msg protomfx.Message) error {
	return h(msg)

}

func (h handleFunc) Cancel() error {
	return nil
}
