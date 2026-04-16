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

// Subscription is a self-contained function that registers a consumer for a given id.
type Subscription func(id string) error

// Messages returns a Subscription that subscribes consumer to the given subjects as a MessageConsumer.
func Messages(sub messaging.Subscriber, c MessageConsumer, subjects ...string) Subscription {
	return func(id string) error {
		for _, subject := range subjects {
			if err := sub.Subscribe(id, subject, &messageAdapter{c}); err != nil {
				return err
			}
		}
		return nil
	}
}

// Alarms returns a Subscription that subscribes consumer as an AlarmConsumer.
func Alarms(sub messaging.AlarmSubscriber, c AlarmConsumer) Subscription {
	return func(id string) error {
		return sub.SubscribeAlarms(id, &alarmAdapter{c})
	}
}

// Start wires all provided subscriptions for the given id.
func Start(id string, subs ...Subscription) error {
	for _, s := range subs {
		if err := s(id); err != nil {
			return err
		}
	}
	return nil
}

type messageAdapter struct{ c MessageConsumer }

func (a *messageAdapter) Handle(subject string, msg protomfx.Message) error {
	return a.c.ConsumeMessage(subject, msg)
}

func (a *messageAdapter) Cancel() error { return nil }

type alarmAdapter struct{ c AlarmConsumer }

func (a *alarmAdapter) Handle(subject string, alarm protomfx.Alarm) error {
	return a.c.ConsumeAlarm(subject, alarm)
}

func (a *alarmAdapter) Cancel() error { return nil }
