// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package consumers

import (
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	"github.com/MainfluxLabs/mainflux/pkg/transformers"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
)

// Consumer specifies message consuming API.
type Consumer interface {
	// Consume method is used to consumed received messages.
	// A non-nil error is returned to indicate operation failure.
	Consume(messages interface{}) error
}

// Start method starts consuming messages received from Message broker.
func Start(id string, sub messaging.Subscriber, consumer Consumer) error {
	senmlTransformer := senml.New()
	jsonTransformer := json.New()

	if err := sub.Subscribe(id, brokers.SubjectSenMLMessages, handle(senmlTransformer, consumer)); err != nil {
		return err
	}
	if err := sub.Subscribe(id, brokers.SubjectJSONMessages, handle(jsonTransformer, consumer)); err != nil {
		return err
	}
	if err := sub.Subscribe(id, brokers.SubjectSmtp, handle(nil, consumer)); err != nil {
		return err
	}
	if err := sub.Subscribe(id, brokers.SubjectSmpp, handle(nil, consumer)); err != nil {
		return err
	}

	return nil
}

func handle(t transformers.Transformer, c Consumer) handleFunc {
	return func(msg messaging.Message) error {
		m := interface{}(msg)
		var err error
		if t != nil {
			m, err = t.Transform(msg)
			if err != nil {
				return err
			}
		}
		return c.Consume(m)
	}
}

type handleFunc func(msg messaging.Message) error

func (h handleFunc) Handle(msg messaging.Message) error {
	return h(msg)

}

func (h handleFunc) Cancel() error {
	return nil
}
