// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"encoding/json"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/gorilla/websocket"
)

var _ nats.PubSub = (*mockPubSub)(nil)

type MockPubSub interface {
	Publish(string, protomfx.Message) error
	Subscribe(string, string, messaging.MessageHandler) error
	Unsubscribe(string, string) error
	PublishAlarm(string, protomfx.Alarm) error
	SubscribeAlarms(string, messaging.AlarmHandler) error
	UnsubscribeAlarms(string) error
	PublishCommand(string, protomfx.Command) error
	SubscribeCommands(string, string, messaging.CommandHandler) error
	UnsubscribeCommands(string, string) error
	SetFail(bool)
	SetConn(*websocket.Conn)
	Close() error
}

type mockPubSub struct {
	fail bool
	conn *websocket.Conn
}

// NewPubSub returns mock message publisher-subscriber
func NewPubSub() MockPubSub {
	return &mockPubSub{false, nil}
}
func (pubsub *mockPubSub) Publish(_ string, msg protomfx.Message) error {
	if pubsub.conn != nil {
		data, err := json.Marshal(msg)
		if err != nil {
			fmt.Println("can't marshall")
			return messaging.ErrPublishMessage
		}
		return pubsub.conn.WriteMessage(websocket.BinaryMessage, data)
	}
	if pubsub.fail {
		return messaging.ErrPublishMessage
	}
	return nil
}

func (pubsub *mockPubSub) Subscribe(string, string, messaging.MessageHandler) error {
	if pubsub.fail {
		return messaging.ErrFailedSubscribe
	}
	return nil
}

func (pubsub *mockPubSub) Unsubscribe(string, string) error {
	if pubsub.fail {
		return messaging.ErrFailedUnsubscribe
	}
	return nil
}

func (pubsub *mockPubSub) PublishAlarm(string, protomfx.Alarm) error {
	if pubsub.fail {
		return messaging.ErrPublishMessage
	}
	return nil
}

func (pubsub *mockPubSub) SubscribeAlarms(string, messaging.AlarmHandler) error {
	if pubsub.fail {
		return messaging.ErrFailedSubscribe
	}
	return nil
}

func (pubsub *mockPubSub) UnsubscribeAlarms(string) error {
	if pubsub.fail {
		return messaging.ErrFailedUnsubscribe
	}
	return nil
}

func (pubsub *mockPubSub) PublishCommand(string, protomfx.Command) error {
	if pubsub.fail {
		return messaging.ErrPublishMessage
	}
	return nil
}

func (pubsub *mockPubSub) SubscribeCommands(string, string, messaging.CommandHandler) error {
	if pubsub.fail {
		return messaging.ErrFailedSubscribe
	}
	return nil
}

func (pubsub *mockPubSub) UnsubscribeCommands(string, string) error {
	if pubsub.fail {
		return messaging.ErrFailedUnsubscribe
	}
	return nil
}

func (pubsub *mockPubSub) SetFail(fail bool) {
	pubsub.fail = fail
}

func (pubsub *mockPubSub) SetConn(c *websocket.Conn) {
	pubsub.conn = c
}

func (pubsub *mockPubSub) Close() error {
	return nil
}
