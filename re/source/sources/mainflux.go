package main

import (
	"encoding/json"
	"fmt"

	"github.com/emqx/kuiper/common"
	"github.com/emqx/kuiper/xstream/api"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/pkg/messaging/nats"
	"github.com/mainflux/senml"
)

const (
	queue = "kuiper"
)

type mainfluxConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Channel  string `json:"channel"`
	Subtopic string `json:"subtopic"`
}

type mainfluxSource struct {
	consumer chan<- api.SourceTuple
	errCh    chan<- error
	logger   api.Logger
	cfg      *mainfluxConfig
	pubSub   nats.PubSub
}

var _ api.Source = (*mainfluxSource)(nil)

func (ms *mainfluxSource) Configure(topic string, props map[string]interface{}) error {
	cfg := &mainfluxConfig{}
	if err := common.MapToStruct(props, cfg); err != nil {
		return fmt.Errorf("Read properties %v fail with error: %v", props, err)
	}
	if cfg.Host == "" {
		return fmt.Errorf("Property Host is required.")
	}
	if cfg.Port == "" {
		return fmt.Errorf("Property Port is required.")
	}
	ms.cfg = cfg

	return nil
}

func (ms *mainfluxSource) Open(ctx api.StreamContext, consumer chan<- api.SourceTuple, errCh chan<- error) {
	logger := ctx.GetLogger()
	logger.Debug("Opening mainflux source.")

	addr := fmt.Sprintf("tcp://%s:%s/", ms.cfg.Host, ms.cfg.Port)
	pubSub, err := nats.NewPubSub(addr, queue, nil)
	if err != nil {
		errCh <- fmt.Errorf("Failed to connect to nats at address %s with error: %v", addr, err)
	}
	ms.pubSub = pubSub

	topic := nats.SubjectAllChannels
	if len(ms.cfg.Channel) > 0 {
		topic = "channels." + ms.cfg.Channel
		if len(ms.cfg.Subtopic) > 0 {
			topic += "." + ms.cfg.Subtopic
		}
	}
	if err := ms.pubSub.Subscribe(topic, ms.handle); err != nil {
		errCh <- fmt.Errorf("Failed to subscribe to nats topic %s with error: %v", topic, err)
		return
	}
	logger.Debugf("Subscribed to nats topic %s.", topic)

	ms.logger = logger
	ms.consumer = consumer
	ms.errCh = errCh

	<-ctx.Done()
}

func (ms *mainfluxSource) handle(message messaging.Message) error {
	ms.logger.Debugf("Received SenML message %v.", message)

	meta := make(map[string]interface{})
	meta["channel"] = message.Channel
	meta["subtopic"] = message.Subtopic
	meta["publisher"] = message.Publisher
	meta["created"] = message.Created

	pack, err := senml.Decode(message.Payload, senml.JSON)
	if err != nil {
		ms.errCh <- err
	}
	pack, err = senml.Normalize(pack)
	if err != nil {
		ms.errCh <- err
	}

	for _, rec := range pack.Records {
		// convert struct to map
		recJson, err := json.Marshal(rec)
		if err != nil {
			ms.errCh <- err
		}
		recMap := make(map[string]interface{})
		err = json.Unmarshal(recJson, &recMap)
		if err != nil {
			ms.errCh <- err
		}

		ms.consumer <- api.NewDefaultSourceTuple(recMap, meta)
	}

	return nil
}

func (ms *mainfluxSource) Close(ctx api.StreamContext) error {
	if ms.pubSub != nil {
		ms.pubSub.Close()
	}

	return nil
}

// Mainflux exports the constructor used to instantiates the plugin
func Mainflux() api.Source {
	return &mainfluxSource{}
}
