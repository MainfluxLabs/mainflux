// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messaging

import (
	"errors"
	"net/url"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var (
	// ErrPublishMessage indicates that message publishing failed.
	ErrPublishMessage = errors.New("failed to publish message")

	// ErrPublishTimeout indicates that the publishing failed due to timeout.
	ErrPublishTimeout = errors.New("failed to publish due to timeout reached")

	// ErrMalformedSubtopic indicates that the subtopic is malformed.
	ErrMalformedSubtopic = errors.New("malformed subtopic")
)

// Publisher specifies message publishing API.
type Publisher interface {
	// Publish publishes message to the message broker.
	Publish(subject string, msg protomfx.Message) error

	// Close gracefully closes message publisher's connection.
	Close() error
}

// PubSub represents aggregation interface for publisher and subscriber.
type PubSub interface {
	Publisher
	Subscriber
}

// MessageDispatcher routes a message to every subject enabled by a profile's dispatcher flags.
type MessageDispatcher interface {
	// Dispatch publishes msg to every subject enabled by the dispatcher flags in pc.
	Dispatch(msg protomfx.Message, pc *domain.ProfileConfig) error
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

// NotificationPublisher specifies the notification publishing API.
type NotificationPublisher interface {
	// PublishNotification publishes a notification to the message broker.
	PublishNotification(subject string, notification protomfx.Notification) error
}

// WebhookPublisher specifies the webhook publishing API.
type WebhookPublisher interface {
	// PublishWebhook publishes a webhook message to the message broker.
	PublishWebhook(subject string, webhook protomfx.Webhook) error
}

// NormalizeSubtopic decodes a client-supplied subtopic and converts it into
// the dot-separated form used in subjects, rejecting any element containing
// a NATS wildcard token (* or >) so a subtopic can't smuggle one in.
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
