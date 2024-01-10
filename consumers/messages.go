// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package consumers

import (
	"fmt"
	"os"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	"github.com/MainfluxLabs/mainflux/pkg/transformers"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
)

const (
	senmlContentType = "application/senml+json"
	cborContentType  = "application/senml+cbor"
	jsonContentType  = "application/json"
)

var timeFields = []json.TimeField{
	{
		FieldName:   "seconds_key",
		FieldFormat: "unix",
		Location:    "UTC",
	},
	{
		FieldName:   "millis_key",
		FieldFormat: "unix_ms",
		Location:    "UTC",
	},
	{
		FieldName:   "micros_key",
		FieldFormat: "unix_us",
		Location:    "UTC",
	},
	{
		FieldName:   "nanos_key",
		FieldFormat: "unix_ns",
		Location:    "UTC",
	},
}

// Start method starts consuming messages received from Message broker.
// This method transforms messages to SenML format before
// using MessageRepository to store them.
func Start(id string, sub messaging.Subscriber, consumer Consumer, logger logger.Logger) error {
	subjects := map[string]transformerConfig{
		brokers.SubjectSenMLMessages: {
			ContentType: senmlContentType,
		},
		brokers.SubjectJSONMessages: {
			ContentType: jsonContentType,
		},
	}

	for subject, cfg := range subjects {
		if err := sub.Subscribe(id, subject, handle(cfg, consumer, logger)); err != nil {
			return err
		}
	}
	return nil
}

func handle(cfg transformerConfig, c Consumer, logger logger.Logger) handleFunc {
	return func(msg messaging.Message) error {
		if msg.Profile != nil {
			timeField := json.TimeField{
				FieldName:   msg.Profile.TimeField.Name,
				FieldFormat: msg.Profile.TimeField.Format,
				Location:    msg.Profile.TimeField.Location,
			}
			cfg.TimeFields = append(cfg.TimeFields, timeField)
		}
		t := makeTransformer(cfg, logger)
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
	ContentType string
	TimeFields  []json.TimeField
}

func makeTransformer(cfg transformerConfig, logger logger.Logger) transformers.Transformer {
	switch cfg.ContentType {
	case senmlContentType, cborContentType:
		logger.Info("Using SenML transformer")
		return senml.New(cfg.ContentType)
	case jsonContentType:
		logger.Info("Using JSON transformer")
		return json.New(cfg.TimeFields)
	default:
		logger.Error(fmt.Sprintf("Can't create transformer: unknown transformer type %s", cfg.ContentType))
		os.Exit(1)
		return nil
	}
}
