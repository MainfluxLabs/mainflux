// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messaging

import (
	"errors"

	"github.com/MainfluxLabs/mainflux"
)

const (
	senmlContentType = "application/senml+json"
	cborContentType  = "application/senml+cbor"
	jsonContentType  = "application/json"
	senmlFormat      = "senml"
	jsonFormat       = "json"
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

func AddProfileToMessage(conn *mainflux.ConnByKeyRes, msg Message) (Message, string, error) {
	if conn.Profile == nil || conn.Profile.ContentType == "" {
		msg.Profile = &Profile{
			ContentType: senmlContentType,
			TimeField:   &TimeField{},
			Retain:      true,
		}
		return msg, senmlFormat, nil
	}

	switch conn.Profile.ContentType {
	case jsonContentType:
		msg.Profile = &Profile{
			ContentType: conn.Profile.ContentType,
			TimeField: &TimeField{
				Name:     conn.Profile.TimeField.Name,
				Format:   conn.Profile.TimeField.Format,
				Location: conn.Profile.TimeField.Location,
			},
			Retain: conn.Profile.Retain,
			Notifier: &Notifier{
				Protocol:  conn.Profile.Notifier.Protocol,
				Contacts:  conn.Profile.Notifier.Contacts,
				Subtopics: conn.Profile.Notifier.Subtopics,
			},
		}
		return msg, jsonFormat, nil
	case senmlContentType, cborContentType:
		msg.Profile = &Profile{
			ContentType: conn.Profile.ContentType,
			TimeField:   &TimeField{},
			Retain:      conn.Profile.Retain,
			Notifier: &Notifier{
				Protocol:  conn.Profile.Notifier.Protocol,
				Contacts:  conn.Profile.Notifier.Contacts,
				Subtopics: conn.Profile.Notifier.Subtopics,
			},
		}
		return msg, senmlFormat, nil

	default:
		return Message{}, "", ErrUnknownContent
	}

}
