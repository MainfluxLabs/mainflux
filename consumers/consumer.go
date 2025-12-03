// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package consumers

import (
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

// Consumer specifies message consuming API.
type Consumer interface {
	// Consume method is used to consumed received messages.
	// A non-nil error is returned to indicate operation failure.
	Consume(messages any) error
}

// Start method starts consuming messages received from Message broker.
func Start(id string, sub messaging.Subscriber, consumer Consumer, subjects ...string) error {
	for _, subject := range subjects {
		if err := sub.Subscribe(id, subject, handle(consumer)); err != nil {
			return err
		}
	}

	return nil
}

func handle(c Consumer) handleFunc {
	return func(msg protomfx.Message) error {
		m := any(msg)

		return c.Consume(m)
	}
}

type handleFunc func(msg protomfx.Message) error

func (h handleFunc) Handle(msg protomfx.Message) error {
	return h(msg)

}

func (h handleFunc) Cancel() error {
	return nil
}
