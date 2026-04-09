// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package ws contains the domain concept definitions needed to support
// Mainflux ws adapter service functionality

package ws

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

// Service specifies web socket service API.
type Service interface {
	// Publish Message
	Publish(ctx context.Context, key domain.ThingKey, msg protomfx.Message) error

	// Subscribe  subscribes to a profile with specified id.
	Subscribe(ctx context.Context, key domain.ThingKey, subtopic string, client *Client) error

	// Unsubscribe method is used to stop observing resource.
	Unsubscribe(ctx context.Context, key domain.ThingKey, subtopic string) error

	// SendCommandToThing publishes a command to the specified thing, authorized by user token.
	SendCommandToThing(ctx context.Context, token, thingID string, msg protomfx.Message) error

	// SendCommandToThingByKey publishes a command to the specified thing, authorized by publisher thing key (M2M).
	SendCommandToThingByKey(ctx context.Context, key domain.ThingKey, thingID string, msg protomfx.Message) error

	// SendCommandToGroup publishes a command to things that belong to a specified group, authorized by user token.
	SendCommandToGroup(ctx context.Context, token, groupID string, msg protomfx.Message) error

	// SendCommandToGroupByKey publishes a command to a group, authorized by publisher thing key (M2M).
	SendCommandToGroupByKey(ctx context.Context, key domain.ThingKey, groupID string, msg protomfx.Message) error
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

func (svc *adapterService) Publish(ctx context.Context, key domain.ThingKey, msg protomfx.Message) error {
	pc, err := svc.things.GetPubConfigByKey(ctx, &protomfx.ThingKey{Value: key.Value, Type: key.Type})
	if err != nil {
		return err
	}

	if len(msg.Payload) == 0 {
		return messaging.ErrPublishMessage
	}

	if err := messaging.FormatMessage(pc, &msg); err != nil {
		return err
	}

	return svc.pubsub.Publish(nats.GetMessagesSubject(msg.Publisher, msg.Subtopic), msg)
}

func (svc *adapterService) Subscribe(ctx context.Context, key domain.ThingKey, subtopic string, c *Client) error {
	res, err := svc.things.Identify(ctx, &protomfx.ThingKey{Value: key.Value, Type: key.Type})
	if err != nil {
		return err
	}
	c.id = res.GetValue()

	return svc.pubsub.Subscribe(c.id, subtopic, c)
}

func (svc *adapterService) Unsubscribe(ctx context.Context, key domain.ThingKey, subtopic string) error {
	res, err := svc.things.Identify(ctx, &protomfx.ThingKey{Value: key.Value, Type: key.Type})
	if err != nil {
		return err
	}

	return svc.pubsub.Unsubscribe(res.GetValue(), subtopic)
}

func (svc *adapterService) SendCommandToThing(ctx context.Context, token, thingID string, msg protomfx.Message) error {
	if _, err := svc.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: thingID, Action: domain.GroupEditor}); err != nil {
		return err
	}

	return svc.pubsub.Publish(nats.GetThingCommandsSubject(thingID, msg.Subtopic), msg)
}

func (svc *adapterService) SendCommandToThingByKey(ctx context.Context, key domain.ThingKey, thingID string, msg protomfx.Message) error {
	res, err := svc.things.Identify(ctx, &protomfx.ThingKey{Value: key.Value, Type: key.Type})
	if err != nil {
		return err
	}

	if _, err := svc.things.CanThingCommand(ctx, &protomfx.ThingCommandReq{PublisherID: res.GetValue(), RecipientID: thingID}); err != nil {
		return err
	}

	return svc.pubsub.Publish(nats.GetThingCommandsSubject(thingID, msg.Subtopic), msg)
}

func (svc *adapterService) SendCommandToGroup(ctx context.Context, token, groupID string, msg protomfx.Message) error {
	if _, err := svc.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: groupID, Action: domain.GroupEditor}); err != nil {
		return err
	}

	return svc.pubsub.Publish(nats.GetGroupCommandsSubject(groupID, msg.Subtopic), msg)
}

func (svc *adapterService) SendCommandToGroupByKey(ctx context.Context, key domain.ThingKey, groupID string, msg protomfx.Message) error {
	res, err := svc.things.Identify(ctx, &protomfx.ThingKey{Value: key.Value, Type: key.Type})
	if err != nil {
		return err
	}

	if _, err := svc.things.CanThingGroupCommand(ctx, &protomfx.ThingGroupCommandReq{PublisherID: res.GetValue(), GroupID: groupID}); err != nil {
		return err
	}

	return svc.pubsub.Publish(nats.GetGroupCommandsSubject(groupID, msg.Subtopic), msg)
}
