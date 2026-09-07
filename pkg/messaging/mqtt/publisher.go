// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Publisher extends the base messaging.Publisher with command publishing capabilities.
type Publisher interface {
	messaging.Publisher
	messaging.CommandPublisher
}

var _ Publisher = (*publisher)(nil)

type publisher struct {
	client  mqtt.Client
	timeout time.Duration
}

// NewPublisher returns a new MQTT message publisher.
func NewPublisher(address string, timeout time.Duration) (Publisher, error) {
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

func (pub publisher) Publish(subject string, msg protomfx.Message) error {
	payload, err := json.Marshal(messaging.ToJSONMessage(msg))
	if err != nil {
		return errors.Wrap(messaging.ErrPublishMessage, err)
	}

	if err := pub.publish(subject, payload); err != nil {
		return errors.Wrap(messaging.ErrPublishMessage, err)
	}

	return nil
}

func (pub publisher) PublishCommand(subject string, cmd protomfx.Command) error {
	payload, err := json.Marshal(messaging.ToJSONCommand(cmd))
	if err != nil {
		return errors.Wrap(messaging.ErrPublishCommand, err)
	}

	if err := pub.publish(subject, payload); err != nil {
		return errors.Wrap(messaging.ErrPublishCommand, err)
	}

	return nil
}

func (pub publisher) publish(subject string, payload []byte) error {
	topic := strings.ReplaceAll(subject, ".", "/")

	token := pub.client.Publish(topic, qos, false, payload)
	if !token.WaitTimeout(pub.timeout) {
		return ErrTimeoutReached
	}

	return token.Error()
}

func (pub publisher) Close() error {
	pub.client.Disconnect(uint(pub.timeout))
	return nil
}
