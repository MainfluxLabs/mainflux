// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package consumers

import (
	"errors"

	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/transformers"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
)

var errUnkownSubject = errors.New("unknown subject")

// Consumer specifies message consuming API.
type Consumer interface {
	// Consume method is used to consumed received messages.
	// A non-nil error is returned to indicate operation failure.
	Consume(messages interface{}) error
}

// Start method starts consuming messages received from Message broker.
func Start(id string, sub messaging.Subscriber, consumer Consumer, subjects ...string) error {
	for _, subject := range subjects {
		var transformer transformers.Transformer
		switch subject {
		case brokers.SubjectSenML:
			transformer = senml.New()
		case brokers.SubjectJSON, brokers.SubjectWebhook:
			transformer = json.New()
		case brokers.SubjectSmtp, brokers.SubjectSmpp:
			transformer = nil
		default:
			return errUnkownSubject
		}

		if err := sub.Subscribe(id, subject, handle(transformer, consumer)); err != nil {
			return err
		}
	}

	return nil
}

func handle(t transformers.Transformer, c Consumer) handleFunc {
	return func(msg protomfx.Message) error {
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

type handleFunc func(msg protomfx.Message) error

func (h handleFunc) Handle(msg protomfx.Message) error {
	return h(msg)

}

func (h handleFunc) Cancel() error {
	return nil
}
