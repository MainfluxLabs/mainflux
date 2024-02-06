// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"fmt"
	"log"

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

func (pub *publisher) Publish(conn *mainflux.ConnByKeyRes, msg messaging.Message) (err error) {
	msg, format, err := setMessageProfile(conn, msg)
	if err != nil {
		return err
	}

	if msg.Profile.Retention {
		log.Printf("Message retention is enabled for channel with an ID:  %s", conn.ChannelID)
		return nil
	}

	data, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}

	topic := fmt.Sprintf("%s.%s.%s", conn.ChannelID, format, messagesSuffix)
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

func setMessageProfile(conn *mainflux.ConnByKeyRes, msg messaging.Message) (messaging.Message, string, error) {
	if conn.Profile == nil || conn.Profile.ContentType == "" {
		msg.Profile = &messaging.Profile{
			ContentType: senmlContentType,
			TimeField:   &messaging.TimeField{},
			Retention:   false,
		}
		return msg, senmlFormat, nil
	}

	switch conn.Profile.ContentType {
	case jsonContentType:
		msg.Profile = &messaging.Profile{
			ContentType: conn.Profile.ContentType,
			TimeField: &messaging.TimeField{
				Name:     conn.Profile.TimeField.Name,
				Format:   conn.Profile.TimeField.Format,
				Location: conn.Profile.TimeField.Location,
			},
			Retention: conn.Profile.Retention,
		}
		return msg, jsonFormat, nil
	case senmlContentType, cborContentType:
		msg.Profile = &messaging.Profile{
			ContentType: conn.Profile.ContentType,
			TimeField:   &messaging.TimeField{},
			Retention:   conn.Profile.Retention,
		}
		return msg, senmlFormat, nil

	default:
		return messaging.Message{}, "", ErrUnknownContent
	}

}
