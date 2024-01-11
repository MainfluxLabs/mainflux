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
	maxReconnects    = -1
	senmlContentType = "application/senml+json"
	cborContentType  = "application/senml+cbor"
	jsonContentType  = "application/json"
	senmlFormat      = "senml"
	jsonFormat       = "json"
	messagesSuffix   = "messages"
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

func (pub *publisher) Publish(conn *mainflux.ConnByKeyRes, msg messaging.Message) error {
	topic := conn.ChannelID
	if topic == "" {
		return ErrEmptyTopic
	}

	profile := conn.Profile
	if profile == nil {
		profile = &mainflux.Profile{}
	}

	switch profile.ContentType {
	case "":
		msg.Profile = &messaging.Profile{
			ContentType: senmlContentType,
			TimeField:   &messaging.TimeField{},
		}
	default:
		msg.Profile = &messaging.Profile{
			ContentType: profile.ContentType,
			TimeField: &messaging.TimeField{
				Name:     profile.TimeField.Name,
				Format:   profile.TimeField.Format,
				Location: profile.TimeField.Location,
			},
		}
	}

	data, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}

	var format string
	switch msg.Profile.ContentType {
	case senmlContentType, cborContentType:
		format = senmlFormat
	case jsonContentType:
		format = jsonFormat
	default:
		return ErrUnknownContent
	}

	topic = fmt.Sprintf("%s.%s.%s", topic, format, messagesSuffix)
	subject := fmt.Sprintf("%s.%s", chansPrefix, topic)
	if msg.Subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, msg.Subtopic)
	}

	if err := pub.conn.Publish(subject, data); err != nil {
		return err
	}

	return nil
}

func (pub *publisher) Close() error {
	pub.conn.Close()
	return nil
}
