// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package rabbitmq

import (
	"context"
	"fmt"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/gogo/protobuf/proto"
	amqp "github.com/rabbitmq/amqp091-go"
)

var _ messaging.Publisher = (*publisher)(nil)

type publisher struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

// NewPublisher returns RabbitMQ message Publisher.
func NewPublisher(url string) (messaging.Publisher, error) {
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

func (pub *publisher) Publish(msg protomfx.Message) error {
	data, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}
	subject := fmt.Sprintf("%s.%s", chansPrefix, msg.Channel)
	if msg.Subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, msg.Subtopic)
	}
	subject = formatTopic(subject)

	err = pub.ch.PublishWithContext(
		context.Background(),
		exchangeName,
		subject,
		false,
		false,
		amqp.Publishing{
			Headers:     amqp.Table{},
			ContentType: "application/octet-stream",
			AppId:       "mainflux-publisher",
			Body:        data,
		})

	if err != nil {
		return err
	}

	return nil
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
