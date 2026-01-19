// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package ws contains the domain concept definitions needed to support
// Mainflux ws adapter service functionality

package ws

import (
	"context"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
)

var (
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
	Publish(ctx context.Context, key things.ThingKey, msg protomfx.Message) error

	// Subscribe  subscribes to a profile with specified id.
	Subscribe(ctx context.Context, key things.ThingKey, subtopic string, client *Client) error

	// Unsubscribe method is used to stop observing resource.
	Unsubscribe(ctx context.Context, key things.ThingKey, subtopic string) error
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	things protomfx.ThingsServiceClient
	pubsub messaging.PubSub
	logger logger.Logger
}

// New instantiates the WS adapter implementation
func New(things protomfx.ThingsServiceClient, pubsub messaging.PubSub, logger logger.Logger) Service {
	return &adapterService{
		things: things,
		pubsub: pubsub,
		logger: logger,
	}
}

func (svc *adapterService) Publish(ctx context.Context, key things.ThingKey, message protomfx.Message) error {
	pc, err := svc.authorize(ctx, key)
	if err != nil {
		return ErrUnauthorizedAccess
	}

	if len(message.Payload) == 0 {
		return messaging.ErrPublishMessage
	}

	if err := messaging.FormatMessage(pc, &message); err != nil {
		return err
	}

	m := message
	m.Subject = nats.GetSubject(message.Publisher, message.Subtopic)

	if err := svc.pubsub.Publish(m); err != nil {
		svc.logger.Error(errors.Wrap(messaging.ErrPublishMessage, err).Error())
	}

	return nil
}

func (svc *adapterService) Subscribe(ctx context.Context, key things.ThingKey, subtopic string, c *Client) error {
	if key.Value == "" {
		return ErrUnauthorizedAccess
	}

	pc, err := svc.authorize(ctx, key)
	if err != nil {
		return ErrUnauthorizedAccess
	}

	c.id = pc.PublisherID

	return svc.pubsub.Subscribe(c.id, subtopic, c)
}

func (svc *adapterService) Unsubscribe(ctx context.Context, key things.ThingKey, subtopic string) error {
	if key.Value == "" {
		return ErrUnauthorizedAccess
	}

	pc, err := svc.authorize(ctx, key)
	if err != nil {
		return ErrUnauthorizedAccess
	}

	return svc.pubsub.Unsubscribe(pc.PublisherID, subtopic)
}

func (svc *adapterService) authorize(ctx context.Context, key things.ThingKey) (*protomfx.PubConfByKeyRes, error) {
	ar := &protomfx.ThingKey{
		Value: key.Value,
		Type:  key.Type,
	}
	pc, err := svc.things.GetPubConfByKey(ctx, ar)
	if err != nil {
		return nil, errors.Wrap(errors.ErrAuthorization, err)
	}

	return pc, nil
}
