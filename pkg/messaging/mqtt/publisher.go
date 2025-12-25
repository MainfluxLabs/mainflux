// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mproxy/pkg/errors"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gogo/protobuf/proto"
)

const (
	messages    = "messages"
	senmlFormat = "senml"
	jsonFormat  = "json"
)

var _ messaging.Publisher = (*publisher)(nil)

type publisher struct {
	client  mqtt.Client
	timeout time.Duration
}

// NewPublisher returns a new MQTT message publisher.
func NewPublisher(address string, timeout time.Duration) (messaging.Publisher, error) {
	client, err := newClient(address, "mqtt-publisher", timeout)
	if err != nil {
		return nil, err
	}

	ret := publisher{
		client:  client,
		timeout: timeout,
	}
	return ret, nil
}

func (pub publisher) Publish(msg protomfx.Message) error {
	var format string
	switch msg.ContentType {
	case messaging.SenMLContentType, messaging.CBORContentType:
		format = senmlFormat
	case messaging.JSONContentType:
		format = jsonFormat
	default:
		return errors.ErrUnsupportedContentType
	}

	topic := format + "/" + messages
	if msg.Subtopic != "" {
		topic += "/" + strings.ReplaceAll(msg.Subtopic, ".", "/")
	}

	data, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}
	token := pub.client.Publish(topic, qos, false, data)
	if token.Error() != nil {
		return token.Error()
	}
	ok := token.WaitTimeout(pub.timeout)
	if !ok {
		return messaging.ErrPublishTimeout
	}

	return token.Error()
}

func (pub publisher) Close() error {
	pub.client.Disconnect(uint(pub.timeout))
	return nil
}
