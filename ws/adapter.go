// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package ws contains the domain concept definitions needed to support
// Mainflux ws adapter service functionality

package ws

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var (
	// ErrFailedMessagePublish indicates that message publishing failed.
	ErrFailedMessagePublish = errors.New("failed to publish message")

	// ErrFailedSubscription indicates that client couldn't subscribe.
	ErrFailedSubscription = errors.New("failed to subscribe")

	// ErrFailedUnsubscribe indicates that client couldn't unsubscribe.
	ErrFailedUnsubscribe = errors.New("failed to unsubscribe")

	// ErrUnauthorizedAccess indicates that client provided missing or invalid credentials.
	ErrUnauthorizedAccess = errors.New("missing or invalid credentials provided")

	// ErrEmptyTopic indicate absence of thingKey in the request.
	ErrEmptyTopic = errors.New("empty topic")
)

// Service specifies web socket service API.
type Service interface {
	// Publish Message
	Publish(ctx context.Context, thingKey string, msg protomfx.Message) error

	// Subscribe  subscribes to a profile with specified id.
	Subscribe(ctx context.Context, thingKey, subtopic string, client *Client) error

	// Unsubscribe method is used to stop observing resource.
	Unsubscribe(ctx context.Context, thingKey, subtopic string) error
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

func (svc *adapterService) Publish(ctx context.Context, thingKey string, msg protomfx.Message) error {
	pc, err := svc.authorize(ctx, thingKey)
	if err != nil {
		return ErrUnauthorizedAccess
	}

	if len(msg.Payload) == 0 {
		return ErrFailedMessagePublish
	}

	m := messaging.CreateMessage(pc, msg.Protocol, msg.Subtopic, &msg.Payload)

	if err := svc.pubsub.Publish(m); err != nil {
		return ErrFailedMessagePublish
	}

	return nil
}

func (svc *adapterService) Subscribe(ctx context.Context, thingKey, subtopic string, c *Client) error {
	if thingKey == "" {
		return ErrUnauthorizedAccess
	}

	pc, err := svc.authorize(ctx, thingKey)
	if err != nil {
		return ErrUnauthorizedAccess
	}

	c.id = pc.PublisherID

	return svc.pubsub.Subscribe(c.id, subtopic, c)
}

func (svc *adapterService) Unsubscribe(ctx context.Context, thingKey, subtopic string) error {
	if thingKey == "" {
		return ErrUnauthorizedAccess
	}

	pc, err := svc.authorize(ctx, thingKey)
	if err != nil {
		return ErrUnauthorizedAccess
	}

	return svc.pubsub.Unsubscribe(pc.PublisherID, subtopic)
}

func (svc *adapterService) authorize(ctx context.Context, thingKey string) (*protomfx.PubConfByKeyRes, error) {
	ar := &protomfx.PubConfByKeyReq{
		Key: thingKey,
	}
	pc, err := svc.things.GetPubConfByKey(ctx, ar)
	if err != nil {
		return nil, errors.Wrap(errors.ErrAuthorization, err)
	}

	return pc, nil
}
