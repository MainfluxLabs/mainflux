// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package consumers

import (
	"fmt"
	"github.com/MainfluxLabs/mainflux/logger"
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
func Start(id string, sub messaging.Subscriber, consumer Consumer, logger logger.Logger, subjects ...string) error {
	senmlTransformer := senml.New()
	jsonTransformer := json.New()

	for _, subject := range subjects {
		var transformer transformers.Transformer
		switch subject {
		case brokers.SubjectSenMLMessages:
			transformer = senmlTransformer
		case brokers.SubjectJSONMessages:
			transformer = jsonTransformer
		case brokers.SubjectSmtp, brokers.SubjectSmpp:
			transformer = nil
		default:
			logger.Error(fmt.Sprintf("Can't create transformer: unknown transformer for subject %s", subject))
		}

		if err := sub.Subscribe(id, subject, handle(transformer, consumer)); err != nil {
			return err
		}
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
