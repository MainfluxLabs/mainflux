// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package consumers

import (
	"fmt"
	"os"
	"strings"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	"github.com/MainfluxLabs/mainflux/pkg/transformers"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
)

const (
	senmlContentType = "application/senml+json"
	jsonContentType  = "application/json"
	senmlFormat      = "senml"
	jsonFormat       = "json"
)

// Start method starts consuming messages received from Message broker.
// This method transforms messages to SenML format before
// using MessageRepository to store them.
func Start(id string, sub messaging.Subscriber, consumer Consumer, logger logger.Logger) error {
	subjects := map[string]transformerConfig{
		brokers.SubjectAllMessages: {
			Format:      senmlFormat,
			ContentType: senmlContentType,
		},
		brokers.SubjectAllJSON: {
			Format:      jsonFormat,
			ContentType: jsonContentType,
		},
	}

	for subject, cfg := range subjects {
		transformer := makeTransformer(cfg, logger)
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

type transformerConfig struct {
	Format      string
	ContentType string
	TimeFields  []json.TimeField
}

func makeTransformer(cfg transformerConfig, logger logger.Logger) transformers.Transformer {
	switch strings.ToUpper(cfg.Format) {
	case "SENML":
		logger.Info("Using SenML transformer")
		return senml.New(cfg.ContentType)
	case "JSON":
		logger.Info("Using JSON transformer")
		return json.New(cfg.TimeFields)
	default:
		logger.Error(fmt.Sprintf("Can't create transformer: unknown transformer type %s", cfg.Format))
		os.Exit(1)
		return nil
	}
}
