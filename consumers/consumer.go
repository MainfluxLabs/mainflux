// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package consumers

import (
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

// MessageConsumer specifies an API for consuming protomfx.Message.
type MessageConsumer interface {
	ConsumeMessage(subject string, msg protomfx.Message) error
}

// AlarmConsumer specifies an API for consuming protomfx.Alarm.
type AlarmConsumer interface {
	ConsumeAlarm(subject string, alarm protomfx.Alarm) error
}

// NotificationConsumer specifies an API for consuming protomfx.Notification.
type NotificationConsumer interface {
	ConsumeNotification(subject string, notification protomfx.Notification) error
}

// WebhookConsumer specifies an API for consuming protomfx.Webhook.
type WebhookConsumer interface {
	ConsumeWebhook(subject string, webhook protomfx.Webhook) error
}

// Messages subscribes the given MessageConsumer to the given subjects.
func Messages(id string, sub messaging.Subscriber, c MessageConsumer, subjects ...string) error {
	for _, subject := range subjects {
		if err := sub.Subscribe(id, subject, messageHandlerFunc(c.ConsumeMessage)); err != nil {
			return err
		}
	}
	return nil
}

// Alarms subscribes the given AlarmConsumer to alarms.
func Alarms(id string, sub messaging.AlarmSubscriber, c AlarmConsumer) error {
	return sub.SubscribeAlarms(id, c.ConsumeAlarm)
}

// Notifications subscribes the given NotificationConsumer to the given subject.
func Notifications(id string, sub messaging.NotificationSubscriber, c NotificationConsumer, subject string) error {
	return sub.SubscribeNotifications(id, subject, c.ConsumeNotification)
}

// Webhooks subscribes the given WebhookConsumer to webhook forwarding.
func Webhooks(id string, sub messaging.WebhookSubscriber, c WebhookConsumer) error {
	return sub.SubscribeWebhooks(id, c.ConsumeWebhook)
}

// messageHandlerFunc adapts a ConsumeMessage-shaped function to messaging.MessageHandler.
type messageHandlerFunc func(subject string, msg protomfx.Message) error

func (f messageHandlerFunc) Handle(subject string, msg protomfx.Message) error {
	return f(subject, msg)
}
