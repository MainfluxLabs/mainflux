// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package coap

import (
	"bytes"
	"context"
	"fmt"
	"sync/atomic"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	mux "github.com/plgd-dev/go-coap/v3/mux"
)

// Client wraps CoAP client.
type Client interface {
	// In CoAP terminology, Token similar to the Session ID.
	Token() string
	Handle(subject string, m protomfx.Message) error
	HandleCommand(subject string, cmd protomfx.Command) error
	Cancel() error
	Done() <-chan struct{}
}

type client struct {
	client  mux.Conn
	token   message.Token
	observe uint32
	logger  logger.Logger
}

// NewClient instantiates a new Observer.
func NewClient(c mux.Conn, tkn message.Token, l logger.Logger) Client {
	return &client{
		client:  c,
		token:   tkn,
		logger:  l,
		observe: 0,
	}
}

func (c *client) Done() <-chan struct{} {
	return c.client.Done()
}

func (c *client) Cancel() error {
	// A closed connection can't take the final notification.
	if c.client.Context().Err() != nil {
		return nil
	}

	m := c.client.AcquireMessage(context.Background())
	defer c.client.ReleaseMessage(m)
	m.SetCode(codes.Content)
	m.SetToken(c.token)
	if err := c.client.WriteMessage(m); err != nil {
		c.logger.Error(fmt.Sprintf("Error sending message: %s.", err))
	}
	return c.client.Close()
}

func (c *client) Token() string {
	return c.token.String()
}

func (c *client) Handle(_ string, msg protomfx.Message) error {
	payload, err := messaging.ToPayload(msg)
	if err != nil {
		return err
	}

	return c.notify(payload)
}

func (c *client) HandleCommand(_ string, cmd protomfx.Command) error {
	return c.notify(cmd.Payload)
}

// notify sends the payload to the observer as the next notification.
func (c *client) notify(payload []byte) error {
	m := c.client.AcquireMessage(c.client.Context())
	defer c.client.ReleaseMessage(m)
	m.SetCode(codes.Content)
	m.SetToken(c.token)
	m.SetContentFormat(message.TextPlain)
	m.SetObserve(atomic.AddUint32(&c.observe, 1))
	m.SetBody(bytes.NewReader(payload))

	return c.client.WriteMessage(m)
}
