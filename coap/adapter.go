// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package coap contains the domain concept definitions needed to support
// Mainflux CoAP adapter service functionality. All constant values are taken
// from RFC, and could be adjusted based on specific use case.
package coap

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
)

// Service specifies CoAP service API.
type Service interface {
	// Publish Messssage
	Publish(ctx context.Context, key things.ThingKey, msg protomfx.Message) error

	// Subscribe subscribes to profile with specified id, subtopic and adds subscription to
	// service map of subscriptions under given ID.
	Subscribe(ctx context.Context, key things.ThingKey, subtopic string, c Client) error

	// Unsubscribe method is used to stop observing resource.
	Unsubscribe(ctx context.Context, key things.ThingKey, subptopic, token string) error
}

var _ Service = (*adapterService)(nil)

// Observers is a map of maps,
type adapterService struct {
	things  protomfx.ThingsServiceClient
	pubsub  messaging.PubSub
	logger  logger.Logger
	obsLock sync.Mutex
}

// New instantiates the CoAP adapter implementation.
func New(things protomfx.ThingsServiceClient, pubsub messaging.PubSub, logger logger.Logger) Service {
	as := &adapterService{
		things:  things,
		pubsub:  pubsub,
		logger:  logger,
		obsLock: sync.Mutex{},
	}

	return as
}

func (svc *adapterService) Publish(ctx context.Context, key things.ThingKey, message protomfx.Message) error {
	cr := &protomfx.ThingKey{Value: key.Value, Type: key.Type}
	pc, err := svc.things.GetPubConfByKey(ctx, cr)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	if err := messaging.FormatMessage(pc, &message); err != nil {
		return err
	}

	subs := nats.GetSubjects(message.Subtopic)
	for _, sub := range subs {
		m := message
		m.Subject = sub

		if err := svc.pubsub.Publish(m); err != nil {
			svc.logger.Error(errors.Wrap(messaging.ErrPublishMessage, err).Error())
		}
	}

	return nil
}

func (svc *adapterService) Subscribe(ctx context.Context, key things.ThingKey, subtopic string, c Client) error {
	cr := &protomfx.ThingKey{
		Value: key.Value,
		Type:  key.Type,
	}
	if _, err := svc.things.GetPubConfByKey(ctx, cr); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return svc.pubsub.Subscribe(c.Token(), subtopic, c)
}

func (svc *adapterService) Unsubscribe(ctx context.Context, key things.ThingKey, subtopic, token string) error {
	cr := &protomfx.ThingKey{
		Value: key.Value,
		Type:  key.Type,
	}
	_, err := svc.things.GetPubConfByKey(ctx, cr)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return svc.pubsub.Unsubscribe(token, subtopic)
}
