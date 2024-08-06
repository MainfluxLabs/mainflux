// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"encoding/json"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/ws"
	"github.com/gorilla/websocket"
)

var _ messaging.PubSub = (*mockPubSub)(nil)

type MockPubSub interface {
	Publish(protomfx.Message) error
	Subscribe(string, string, messaging.MessageHandler) error
	Unsubscribe(string, string) error
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
func (pubsub *mockPubSub) Publish(msg protomfx.Message) error {
	if pubsub.conn != nil {
		data, err := json.Marshal(msg)
		if err != nil {
			fmt.Println("can't marshall")
			return ws.ErrFailedMessagePublish
		}
		return pubsub.conn.WriteMessage(websocket.BinaryMessage, data)
	}
	if pubsub.fail {
		return ws.ErrFailedMessagePublish
	}
	return nil
}

func (pubsub *mockPubSub) Subscribe(string, string, messaging.MessageHandler) error {
	if pubsub.fail {
		return ws.ErrFailedSubscription
	}
	return nil
}

func (pubsub *mockPubSub) Unsubscribe(string, string) error {
	if pubsub.fail {
		return ws.ErrFailedUnsubscribe
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
