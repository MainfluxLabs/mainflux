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

// Messages subscribes c to the given subjects as a MessageConsumer.
func Messages(id string, sub messaging.Subscriber, c MessageConsumer, subjects ...string) error {
	for _, subject := range subjects {
		if err := sub.Subscribe(id, subject, messageHandler{c}); err != nil {
			return err
		}
	}
	return nil
}

// Alarms subscribes c as an AlarmConsumer.
func Alarms(id string, sub messaging.AlarmSubscriber, c AlarmConsumer) error {
	return sub.SubscribeAlarms(id, alarmHandler{c})
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
