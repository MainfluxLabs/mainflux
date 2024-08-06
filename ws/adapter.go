// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package ws contains the domain concept definitions needed to support
// Mainflux ws adapter service functionality

package ws

import (
	"context"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

const (
	chansPrefix = "channels"
)

var (
	// ErrFailedMessagePublish indicates that message publishing failed.
	ErrFailedMessagePublish = errors.New("failed to publish message")

	// ErrFailedSubscription indicates that client couldn't subscribe to specified channel
	ErrFailedSubscription = errors.New("failed to subscribe to a channel")

	// ErrFailedUnsubscribe indicates that client couldn't unsubscribe from specified channel
	ErrFailedUnsubscribe = errors.New("failed to unsubscribe from a channel")

	// ErrFailedConnection indicates that service couldn't connect to message broker.
	ErrFailedConnection = errors.New("failed to connect to message broker")

	// ErrInvalidConnection indicates that client couldn't subscribe to message broker
	ErrInvalidConnection = errors.New("nats: invalid connection")

	// ErrUnauthorizedAccess indicates that client provided missing or invalid credentials
	ErrUnauthorizedAccess = errors.New("missing or invalid credentials provided")

	// ErrEmptyTopic indicate absence of thingKey in the request
	ErrEmptyTopic = errors.New("empty topic")

	// ErrEmptyID indicate absence of channelID in the request
	ErrEmptyID = errors.New("empty id")
)

// Service specifies web socket service API.
type Service interface {
	// Publish Message
	Publish(ctx context.Context, thingKey string, msg protomfx.Message) error

	// Subscribe  subscribes to a channel with specified id.
	Subscribe(ctx context.Context, thingKey, chanID, subtopic string, client *Client) error

	// Unsubscribe method is used to stop observing resource.
	Unsubscribe(ctx context.Context, thingKey, chanID, subtopic string) error
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	things protomfx.ThingsServiceClient
	pubsub messaging.PubSub
}

// New instantiates the WS adapter implementation
func New(things protomfx.ThingsServiceClient, pubsub messaging.PubSub) Service {
	return &adapterService{
		things: things,
		pubsub: pubsub,
	}
}

// Publish publishes the message using the broker
func (svc *adapterService) Publish(ctx context.Context, thingKey string, msg protomfx.Message) error {
	conn, err := svc.authorize(ctx, thingKey)
	if err != nil {
		return ErrUnauthorizedAccess
	}

	if len(msg.Payload) == 0 {
		return ErrFailedMessagePublish
	}

	m := messaging.CreateMessage(conn, msg.Protocol, msg.Subtopic, &msg.Payload)

	if err := svc.pubsub.Publish(m); err != nil {
		return ErrFailedMessagePublish
	}

	return nil
}

// Subscribe subscribes the thingKey and channelID to the topic
func (svc *adapterService) Subscribe(ctx context.Context, thingKey, chanID, subtopic string, c *Client) error {
	if chanID == "" || thingKey == "" {
		return ErrUnauthorizedAccess
	}

	conn, err := svc.authorize(ctx, thingKey)
	if err != nil {
		return ErrUnauthorizedAccess
	}

	c.id = conn.ThingID

	subject := fmt.Sprintf("%s.%s", chansPrefix, chanID)
	if subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, subtopic)
	}

	if err := svc.pubsub.Subscribe(conn.ChannelID, subject, c); err != nil {
		return ErrFailedSubscription
	}

	return nil
}

// Unsubscribe unsubscribes the thing and channel from the topic.
func (svc *adapterService) Unsubscribe(ctx context.Context, thingKey, chanID, subtopic string) error {
	if chanID == "" || thingKey == "" {
		return ErrUnauthorizedAccess
	}

	conn, err := svc.authorize(ctx, thingKey)
	if err != nil {
		return ErrUnauthorizedAccess
	}

	subject := fmt.Sprintf("%s.%s", chansPrefix, chanID)
	if subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, subtopic)
	}

	return svc.pubsub.Unsubscribe(conn.ChannelID, subject)
}

func (svc *adapterService) authorize(ctx context.Context, thingKey string) (*protomfx.ConnByKeyRes, error) {
	ar := &protomfx.ConnByKeyReq{
		Key: thingKey,
	}
	conn, err := svc.things.GetConnByKey(ctx, ar)
	if err != nil {
		return nil, errors.Wrap(errors.ErrAuthorization, err)
	}

	return conn, nil
}
