// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package rabbitmq

import (
	"context"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/gogo/protobuf/proto"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher extends the base messaging.Publisher with command publishing capability.
type Publisher interface {
	messaging.Publisher
	messaging.CommandPublisher
}

var _ Publisher = (*publisher)(nil)

type publisher struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

// NewPublisher returns RabbitMQ message Publisher.
func NewPublisher(url string) (Publisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.ExchangeDeclare(exchangeName, amqp.ExchangeTopic, true, false, false, false, nil); err != nil {
		return nil, err
	}
	ret := &publisher{
		conn: conn,
		ch:   ch,
	}
	return ret, nil
}

func (pub *publisher) Publish(_ string, msg protomfx.Message) error {
	return pub.publish(msg.Subtopic, &msg)
}

func (pub *publisher) PublishCommand(subject string, cmd protomfx.Command) error {
	return pub.publish(subject, &cmd)
}

func (pub *publisher) publish(routingKey string, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	return pub.ch.PublishWithContext(
		context.Background(),
		exchangeName,
		formatTopic(routingKey),
		false,
		false,
		amqp.Publishing{
			Headers:     amqp.Table{},
			ContentType: "application/octet-stream",
			AppId:       "mainflux-publisher",
			Body:        data,
		})
}

func (pub *publisher) Close() error {
	if err := pub.ch.Close(); err != nil {
		return err
	}
	return pub.conn.Close()
}

func formatTopic(topic string) string {
	return strings.Replace(topic, ">", "#", -1)
}
