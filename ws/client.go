// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package ws

import (
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/gorilla/websocket"
)

// Client handles messaging and websocket connection
type Client struct {
	conn *websocket.Conn
	id   string
}

// NewClient returns a new Client object
func NewClient(c *websocket.Conn) *Client {
	return &Client{
		conn: c,
		id:   "",
	}
}

// Cancel handles the websocket connection after unsubscribing
func (c *Client) Cancel() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// Handle handles the sending and receiving of messages via the broker
func (c *Client) Handle(_ string, msg protomfx.Message) error {
	// To prevent a thing from receiving its own published message
	if msg.GetThingId() == c.id {
		return nil
	}
	payload, err := messaging.ToPayload(msg)
	if err != nil {
		return err
	}

	return c.conn.WriteMessage(websocket.TextMessage, payload)
}
