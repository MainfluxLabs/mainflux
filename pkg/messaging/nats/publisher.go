// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"fmt"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
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

func (pub *publisher) Publish(conn *mainflux.ConnByKeyRes, msg messaging.Message) (err error) {
	msg, format, err := messaging.AddProfileToMessage(conn, msg)
	if err != nil {
		return err
	}

	data, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}

	var subjects []string
	if msg.Profile.Retain {
		subject := fmt.Sprintf("%s.%s.%s.%s", chansPrefix, conn.ChannelID, format, messagesSuffix)
		if msg.Subtopic != "" {
			subject = fmt.Sprintf("%s.%s", subject, msg.Subtopic)
		}
		subjects = append(subjects, subject)
	}

	if conn.Profile.Notifier.Type == subjectSMTP || conn.Profile.Notifier.Type == subjectSMPP {
		sub := conn.Profile.Notifier.Type
		subjects = append(subjects, sub)
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
