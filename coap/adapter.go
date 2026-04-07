// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package coap contains the domain concept definitions needed to support
// Mainflux CoAP adapter service functionality. All constant values are taken
// from RFC, and could be adjusted based on specific use case.
package coap

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/protoutil"
)

// Service specifies CoAP service API.
type Service interface {
	// Publish Message
	Publish(ctx context.Context, key domain.ThingKey, msg protomfx.Message) error

	// Subscribe subscribes to profile with specified id, subtopic and adds subscription to
	// service map of subscriptions under given ID.
	Subscribe(ctx context.Context, key domain.ThingKey, subtopic string, c Client) error

	// Unsubscribe method is used to stop observing resource.
	Unsubscribe(ctx context.Context, key domain.ThingKey, subtopic, token string) error

	// SendCommandToThing publishes a command to the specified thing, authorized by publisher thing key (M2M).
	SendCommandToThing(ctx context.Context, key domain.ThingKey, thingID string, msg protomfx.Message) error

	// SendCommandToGroup publishes a command to a group, authorized by publisher thing key (M2M).
	SendCommandToGroup(ctx context.Context, key domain.ThingKey, groupID string, msg protomfx.Message) error
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	things  domain.ThingsClient
	pubsub  messaging.PubSub
	obsLock sync.Mutex
}

// New instantiates the CoAP adapter implementation.
func New(things domain.ThingsClient, pubsub messaging.PubSub) Service {
	as := &adapterService{
		things:  things,
		pubsub:  pubsub,
		obsLock: sync.Mutex{},
	}

	return as
}

func (svc *adapterService) Publish(ctx context.Context, key domain.ThingKey, msg protomfx.Message) error {
	pc, err := svc.things.GetPubConfigByKey(ctx, key)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	if err := messaging.FormatMessage(protoutil.PubConfigInfoToProto(pc), &msg); err != nil {
		return err
	}

	if err := svc.pubsub.Publish(nats.GetMessagesSubject(msg.Publisher, msg.Subtopic), msg); err != nil {
		return err
	}

	return nil
}

func (svc *adapterService) Subscribe(ctx context.Context, key domain.ThingKey, subtopic string, c Client) error {
	if _, err := svc.things.GetPubConfigByKey(ctx, key); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return svc.pubsub.Subscribe(c.Token(), subtopic, c)
}

func (svc *adapterService) SendCommandToThing(ctx context.Context, key domain.ThingKey, thingID string, msg protomfx.Message) error {
	res, err := svc.things.Identify(ctx, key)
	if err != nil {
		return err
	}

	if err := svc.things.CanThingCommand(ctx, domain.ThingCommandReq{PublisherID: res, RecipientID: thingID}); err != nil {
		return err
	}

	return svc.pubsub.Publish(nats.GetThingCommandsSubject(thingID, msg.Subtopic), msg)
}

func (svc *adapterService) SendCommandToGroup(ctx context.Context, key domain.ThingKey, groupID string, msg protomfx.Message) error {
	thingID, err := svc.things.Identify(ctx, key)
	if err != nil {
		return err
	}

	if err := svc.things.CanThingGroupCommand(ctx, domain.ThingGroupCommandReq{PublisherID: thingID, GroupID: groupID}); err != nil {
		return err
	}

	return svc.pubsub.Publish(nats.GetGroupCommandsSubject(groupID, msg.Subtopic), msg)
}

func (svc *adapterService) Unsubscribe(ctx context.Context, key domain.ThingKey, subtopic, token string) error {
	if _, err := svc.things.GetPubConfigByKey(ctx, key); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return svc.pubsub.Unsubscribe(token, subtopic)
}
