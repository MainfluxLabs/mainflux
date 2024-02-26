// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messaging

import (
	"errors"

	"github.com/MainfluxLabs/mainflux"
)

const (
	SenmlContentType = "application/senml+json"
	CborContentType  = "application/senml+cbor"
	JsonContentType  = "application/json"
	SenmlFormat      = "senml"
	JsonFormat       = "json"
	CborFormat       = "cbor"
)

var (
	// ErrConnect indicates that connection to MQTT broker failed
	ErrConnect = errors.New("failed to connect to MQTT broker")

	// ErrPublishTimeout indicates that the publishing failed due to timeout.
	ErrPublishTimeout = errors.New("failed to publish due to timeout reached")

	// ErrSubscribeTimeout indicates that the subscription failed due to timeout.
	ErrSubscribeTimeout = errors.New("failed to subscribe due to timeout reached")

	// ErrUnsubscribeTimeout indicates that unsubscribe failed due to timeout.
	ErrUnsubscribeTimeout = errors.New("failed to unsubscribe due to timeout reached")

	// ErrUnsubscribeDeleteTopic indicates that unsubscribe failed because the topic was deleted.
	ErrUnsubscribeDeleteTopic = errors.New("failed to unsubscribe due to deletion of topic")

	// ErrNotSubscribed indicates that the topic is not subscribed to.
	ErrNotSubscribed = errors.New("not subscribed")

	// ErrEmptyTopic indicates the absence of topic.
	ErrEmptyTopic = errors.New("empty topic")

	// ErrEmptyID indicates the absence of ID.
	ErrEmptyID = errors.New("empty ID")

	// ErrUnknownContent indicates that the content type is unknown.
	ErrUnknownContent = errors.New("unknown content type")
)

// Publisher specifies message publishing API.
type Publisher interface {
	// Publish publishes message to the message broker.
	Publish(conn *mainflux.ConnByKeyRes, msg Message) error

	// Close gracefully closes message publisher's connection.
	Close() error
}

// MessageHandler represents Message handler for Subscriber.
type MessageHandler interface {
	// Handle handles messages passed by underlying implementation.
	Handle(msg Message) error

	// Cancel is used for cleanup during unsubscribing and it's optional.
	Cancel() error
}

// Subscriber specifies message subscription API.
type Subscriber interface {
	// Subscribe subscribes to the message stream and consumes messages.
	Subscribe(id, topic string, handler MessageHandler) error

	// Unsubscribe unsubscribes from the message stream and
	// stops consuming messages.
	Unsubscribe(id, topic string) error

	// Close gracefully closes message subscriber's connection.
	Close() error
}

// PubSub  represents aggregation interface for publisher and subscriber.
type PubSub interface {
	Publisher
	Subscriber
}

func AddProfileToMessage(conn *mainflux.ConnByKeyRes, msg Message) (Message, error) {
	if conn.Profile == nil || conn.Profile.ContentType == "" {
		msg.Profile = &Profile{
			ContentType: SenmlContentType,
			TimeField:   &TimeField{},
			Writer:      &Writer{Retain: true},
			Notifier:    &Notifier{},
		}
		return msg, nil
	}

	if conn.Profile.Writer != nil || conn.Profile.Notifier != nil {
		msg.Profile = &Profile{
			ContentType: conn.Profile.ContentType,
			Writer: &Writer{
				Retain:    conn.Profile.Writer.Retain,
				Subtopics: conn.Profile.Writer.Subtopics,
			},
			Notifier: &Notifier{
				Protocol:  conn.Profile.Notifier.Protocol,
				Contacts:  conn.Profile.Notifier.Contacts,
				Subtopics: conn.Profile.Notifier.Subtopics,
			},
		}
	}

	switch conn.Profile.ContentType {
	case JsonContentType:
		if conn.Profile.TimeField != nil {
			msg.Profile.TimeField = &TimeField{
				Name:     conn.Profile.TimeField.Name,
				Format:   conn.Profile.TimeField.Format,
				Location: conn.Profile.TimeField.Location,
			}
		}
		return msg, nil
	case SenmlContentType, CborContentType:
		if conn.Profile.TimeField != nil {
			msg.Profile.TimeField = &TimeField{}
		}
		return msg, nil

	default:
		return Message{}, ErrUnknownContent
	}
}
