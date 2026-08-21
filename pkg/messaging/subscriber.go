// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messaging

import (
	"errors"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var (
	// ErrFailedSubscribe indicates that subscribing to a topic failed.
	ErrFailedSubscribe = errors.New("failed to subscribe")

	// ErrFailedUnsubscribe indicates that unsubscribing from a topic failed.
	ErrFailedUnsubscribe = errors.New("failed to unsubscribe")

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
)

// MessageHandler represents protomfx.Message handler for Subscriber.
type MessageHandler interface {
	// Handle handles messages passed by underlying implementation.
	Handle(subject string, msg protomfx.Message) error
}

// Canceler is an optional capability a MessageHandler may implement for
// cleanup when its subscription ends. Subscriber implementations detect it
// via a type assertion.
type Canceler interface {
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

// AlarmHandler handles a received protomfx.Alarm for AlarmSubscriber.
type AlarmHandler func(subject string, alarm protomfx.Alarm) error

// AlarmSubscriber specifies the alarm subscription API.
type AlarmSubscriber interface {
	// SubscribeAlarms subscribes to the alarm stream.
	SubscribeAlarms(id string, handler AlarmHandler) error
}

// CommandHandler handles a received protomfx.Command for CommandSubscriber.
type CommandHandler func(subject string, cmd protomfx.Command) error

// CommandSubscriber specifies the command subscription API.
type CommandSubscriber interface {
	// SubscribeCommands subscribes to the command stream for the given topic.
	SubscribeCommands(id, topic string, handler CommandHandler) error
}

// NotificationHandler handles a received protomfx.Notification for NotificationSubscriber.
type NotificationHandler func(subject string, notification protomfx.Notification) error

// NotificationSubscriber specifies the notification subscription API.
type NotificationSubscriber interface {
	// SubscribeNotifications subscribes to the notification stream for the given topic.
	SubscribeNotifications(id, topic string, handler NotificationHandler) error
}

// WebhookHandler handles a received protomfx.Webhook for WebhookSubscriber.
type WebhookHandler func(subject string, webhook protomfx.Webhook) error

// WebhookSubscriber specifies the webhook subscription API.
type WebhookSubscriber interface {
	// SubscribeWebhooks subscribes to the webhook stream.
	SubscribeWebhooks(id string, handler WebhookHandler) error
}
