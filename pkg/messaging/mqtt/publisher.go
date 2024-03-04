// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gogo/protobuf/proto"
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

func (pub publisher) Publish(msg messaging.Message) error {
	if !msg.Profile.Writer.Retain {
		return nil
	}

	data, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}
	token := pub.client.Publish(msg.Channel, qos, false, data)
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
