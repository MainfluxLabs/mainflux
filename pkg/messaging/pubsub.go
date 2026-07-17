// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messaging

import (
	"errors"
	"net/url"
	"strings"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

const (
	SenMLContentType = "application/senml+json"
	JSONContentType  = "application/json"
)

var (
	// ErrPublishMessage indicates that message publishing failed.
	ErrPublishMessage = errors.New("failed to publish message")

	// ErrFailedSubscribe indicates that subscribing to a topic failed.
	ErrFailedSubscribe = errors.New("failed to subscribe")

	// ErrFailedUnsubscribe indicates that unsubscribing from a topic failed.
	ErrFailedUnsubscribe = errors.New("failed to unsubscribe")

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

	// ErrMalformedSubtopic indicates that the subtopic is malformed.
	ErrMalformedSubtopic = errors.New("malformed subtopic")

	// ErrEmptyID indicates the absence of ID.
	ErrEmptyID = errors.New("empty ID")

	// ErrInvalidContentType indicates an invalid Content-Type
	ErrInvalidContentType = errors.New("invalid content type")
)

// Publisher specifies message publishing API.
type Publisher interface {
	// Publish publishes message to the message broker.
	Publish(subject string, msg protomfx.Message) error

	// Close gracefully closes message publisher's connection.
	Close() error
}

// MessageHandler represents protomfx.Message handler for Subscriber.
type MessageHandler interface {
	// Handle handles messages passed by underlying implementation.
	Handle(subject string, msg protomfx.Message) error

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

// AlarmPublisher specifies the alarm publishing API.
type AlarmPublisher interface {
	// PublishAlarm publishes an alarm to the message broker.
	PublishAlarm(subject string, alarm protomfx.Alarm) error
}

// CommandPublisher specifies the command publishing API.
type CommandPublisher interface {
	// PublishCommand publishes a command to the message broker.
	PublishCommand(subject string, cmd protomfx.Command) error
}

// AlarmHandler represents protomfx.Alarm handler for AlarmSubscriber.
type AlarmHandler interface {
	// Handle handles alarms passed by underlying implementation.
	Handle(subject string, alarm protomfx.Alarm) error

	// Cancel is used for cleanup during unsubscribing and it's optional.
	Cancel() error
}

// AlarmSubscriber specifies the alarm subscription API.
type AlarmSubscriber interface {
	// SubscribeAlarms subscribes to the alarm stream.
	SubscribeAlarms(id string, handler AlarmHandler) error

	// UnsubscribeAlarms unsubscribes from the alarm stream.
	UnsubscribeAlarms(id string) error
}

// CommandHandler represents protomfx.Command handler for CommandSubscriber.
type CommandHandler interface {
	// Handle handles commands passed by underlying implementation.
	Handle(subject string, cmd protomfx.Command) error

	// Cancel is used for cleanup during unsubscribing and it's optional.
	Cancel() error
}

// CommandSubscriber specifies the command subscription API.
type CommandSubscriber interface {
	// SubscribeCommands subscribes to the command stream for the given topic.
	SubscribeCommands(id, topic string, handler CommandHandler) error

	// UnsubscribeCommands unsubscribes from the command stream for the given topic.
	UnsubscribeCommands(id, topic string) error
}

func NormalizeSubtopic(topic string) (string, error) {
	if topic == "" {
		return topic, nil
	}

	// URL decode if needed
	decoded, err := url.QueryUnescape(topic)
	if err != nil {
		return "", ErrMalformedSubtopic
	}

	// Replace slashes with dots
	normalized := strings.Replace(decoded, "/", ".", -1)

	// Split and filter empty elements
	elems := strings.Split(normalized, ".")
	filteredElems := []string{}

	for _, elem := range elems {
		if elem == "" {
			continue
		}

		if len(elem) > 1 && (strings.Contains(elem, "*") || strings.Contains(elem, ">")) {
			return "", ErrMalformedSubtopic
		}

		filteredElems = append(filteredElems, elem)
	}

	return strings.Join(filteredElems, "."), nil
}
