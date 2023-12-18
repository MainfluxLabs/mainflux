// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package consumers

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pelletier/go-toml"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/brokers"
	"github.com/MainfluxLabs/mainflux/pkg/transformers"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
)

const (
	defContentType  = "application/senml+json"
	jsonContentType = "application/json"
	defFormat       = "senml"
	jsonFormat      = "json"
	MessagesProfile = "messages"
	JSONProfile     = "json"
)

var (
	errOpenConfFile  = errors.New("unable to open configuration file")
	errParseConfFile = errors.New("unable to parse configuration file")
)

// Start method starts consuming messages received from Message broker.
// This method transforms messages to SenML format before
// using MessageRepository to store them.
func Start(id string, sub messaging.Subscriber, consumer Consumer, configPath string, logger logger.Logger) error {
	var profile string
	cfg, err := loadConfig(profile, configPath)
	if err != nil {
		logger.Warn(fmt.Sprintf("Failed to load consumer config: %s", err))
	}

	transformer := makeTransformer(cfg.TransformerCfg, logger)

	var subject string
	switch profile {
	case MessagesProfile:
		subject = brokers.SubjectAllMessages
	case JSONProfile:
		subject = brokers.SubjectAllJSON
	default:
		subject = brokers.SubjectAllChannels
	}

	if err := sub.Subscribe(id, subject, handle(transformer, consumer)); err != nil {
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

type subscriberConfig struct {
	Subjects []string `toml:"subjects"`
}

type transformerConfig struct {
	Format      string           `toml:"format"`
	ContentType string           `toml:"content_type"`
	TimeFields  []json.TimeField `toml:"time_fields"`
}

type config struct {
	SubscriberCfg  subscriberConfig  `toml:"subscriber"`
	TransformerCfg transformerConfig `toml:"transformer"`
}

func loadConfig(profile, configPath string) (config, error) {
	var cfg config
	switch profile {
	case MessagesProfile:
		cfg = config{
			SubscriberCfg: subscriberConfig{
				Subjects: []string{brokers.SubjectAllMessages},
			},
			TransformerCfg: transformerConfig{
				Format:      defFormat,
				ContentType: defContentType,
			},
		}
	case JSONProfile:
		cfg = config{
			SubscriberCfg: subscriberConfig{
				Subjects: []string{brokers.SubjectAllJSON},
			},
			TransformerCfg: transformerConfig{
				Format:      jsonFormat,
				ContentType: jsonContentType,
			},
		}
	default:
		cfg = config{
			SubscriberCfg: subscriberConfig{
				Subjects: []string{brokers.SubjectAllChannels},
			},
			TransformerCfg: transformerConfig{
				Format:      defFormat,
				ContentType: defContentType,
			},
		}
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return cfg, errors.Wrap(errOpenConfFile, err)
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, errors.Wrap(errParseConfFile, err)
	}

	return cfg, nil
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
