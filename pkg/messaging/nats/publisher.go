// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/gogo/protobuf/proto"
	broker "github.com/nats-io/nats.go"
)

// A maximum number of reconnect attempts before NATS connection closes permanently.
// Value -1 represents an unlimited number of reconnect retries, i.e. the client
// will never give up on retrying to re-establish connection to NATS server.
const (
	maxReconnects  = -1
	messagesSuffix = "messages"
	subjectWebhook = "webhooks"
)

var _ messaging.Publisher = (*publisher)(nil)

type publisher struct {
	conn *broker.Conn
}

// Publisher wraps messaging Publisher exposing
// Close() method for NATS connection.

// NewPublisher returns NATS message Publisher.
func NewPublisher(url string) (messaging.Publisher, error) {
	conn, err := broker.Connect(url, broker.MaxReconnects(maxReconnects))
	if err != nil {
		return nil, err
	}
	ret := &publisher{
		conn: conn,
	}
	return ret, nil
}

func (pub *publisher) Publish(msg protomfx.Message) (err error) {
	data, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}

	return pub.conn.Publish(msg.Subject, data)
}

func (pub *publisher) Close() error {
	pub.conn.Close()
	return nil
}

func GetSubjects(pc *protomfx.Config, subtopic string) []string {
	subjects := []string{subjectWebhook}

	if pc.GetWrite() {
		format := GetFormat(pc.ContentType)
		subject := fmt.Sprintf("%s.%s", format, messagesSuffix)
		if subtopic != "" {
			subject = fmt.Sprintf("%s.%s", subject, subtopic)
		}
		subjects = append(subjects, subject)
	}

	return subjects
}

func GetFormat(ct string) string {
	switch ct {
	case messaging.JSONContentType:
		return messaging.JSONFormat
	case messaging.SenMLContentType:
		return messaging.SenMLFormat
	case messaging.CBORContentType:
		return messaging.CBORFormat
	default:
		return messaging.SenMLFormat
	}
}
