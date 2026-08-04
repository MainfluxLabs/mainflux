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
		if err := sub.Subscribe(id, subject, messageHandler{c}); err != nil {
			return err
		}
	}
	return nil
}

// Alarms subscribes the given AlarmConsumer to alarms.
func Alarms(id string, sub messaging.AlarmSubscriber, c AlarmConsumer) error {
	return sub.SubscribeAlarms(id, alarmHandler{c})
}

// Notifications subscribes the given NotificationConsumer to the given subject.
func Notifications(id string, sub messaging.NotificationSubscriber, c NotificationConsumer, subject string) error {
	return sub.SubscribeNotifications(id, subject, notificationHandler{c})
}

// Webhooks subscribes the given WebhookConsumer to webhook forwarding.
func Webhooks(id string, sub messaging.WebhookSubscriber, c WebhookConsumer) error {
	return sub.SubscribeWebhooks(id, webhookHandler{c})
}

type messageHandler struct{ c MessageConsumer }

func (h messageHandler) Handle(subject string, msg protomfx.Message) error {
	return h.c.ConsumeMessage(subject, msg)
}

func (h messageHandler) Cancel() error { return nil }

type alarmHandler struct{ c AlarmConsumer }

func (h alarmHandler) Handle(subject string, alarm protomfx.Alarm) error {
	return h.c.ConsumeAlarm(subject, alarm)
}

func (h alarmHandler) Cancel() error { return nil }

type notificationHandler struct{ c NotificationConsumer }

func (h notificationHandler) Handle(subject string, notification protomfx.Notification) error {
	return h.c.ConsumeNotification(subject, notification)
}

func (h notificationHandler) Cancel() error { return nil }

type webhookHandler struct{ c WebhookConsumer }

func (h webhookHandler) Handle(subject string, webhook protomfx.Webhook) error {
	return h.c.ConsumeWebhook(subject, webhook)
}

func (h webhookHandler) Cancel() error { return nil }
