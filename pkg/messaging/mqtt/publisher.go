// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	mqtt "github.com/eclipse/paho.mqtt.golang"
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
	topic := strings.ReplaceAll(msg.Subject, ".", "/")
	token := pub.client.Publish(topic, qos, false, msg.Payload)
	if !token.WaitTimeout(pub.timeout) {
		return messaging.ErrPublishTimeout
	}

	return token.Error()
}

func (pub publisher) Close() error {
	pub.client.Disconnect(uint(pub.timeout))
	return nil
}
