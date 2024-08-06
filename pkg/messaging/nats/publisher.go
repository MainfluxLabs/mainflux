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
	subjectSMTP    = "smtp"
	subjectSMPP    = "smpp"
	subjectWebhook = "webhook"
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
	format, err := getFormat(msg.Profile.ContentType)
	if err != nil {
		return err
	}

	data, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}

	var subjects []string
	subject := fmt.Sprintf("%s.%s.%s.%s", chansPrefix, msg.Channel, format, messagesSuffix)
	if msg.Profile.Write {
		if msg.Subtopic != "" {
			subject = fmt.Sprintf("%s.%s", subject, msg.Subtopic)
		}

		subjects = append(subjects, subject)
	}

	if msg.Profile.SmtpID != "" {
		subjects = append(subjects, subjectSMTP)
	}

	if msg.Profile.SmppID != "" {
		subjects = append(subjects, subjectSMPP)
	}

	if msg.Profile.WebhookID != "" {
		subjects = append(subjects, subjectWebhook)
	}

	for _, subject := range subjects {
		if err := pub.conn.Publish(subject, data); err != nil {
			return err
		}
	}

	return nil
}

func (pub *publisher) Close() error {
	pub.conn.Close()
	return nil
}

func getFormat(ct string) (format string, err error) {
	switch ct {
	case messaging.JSONContentType:
		return messaging.JSONFormat, nil
	case messaging.SenMLContentType:
		return messaging.SenMLFormat, nil
	case messaging.CBORContentType:
		return messaging.CBORFormat, nil
	default:
		return messaging.SenMLFormat, nil
	}
}
