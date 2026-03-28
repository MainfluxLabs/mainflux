// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package http contains the domain concept definitions needed to support
// Mainflux http adapter service functionality.
package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
)

// Service specifies coap service API.
type Service interface {
	// Publish Message
	Publish(ctx context.Context, key domain.ThingKey, msg protomfx.Message) error
	// SendCommandToThing publishes a command message to the specified thing.
	SendCommandToThing(ctx context.Context, token, thingID string, msg protomfx.Message) error
	// SendCommandToThingByKey publishes a command message to the specified thing, authorized by publisher thing key (M2M).
	SendCommandToThingByKey(ctx context.Context, key domain.ThingKey, thingID string, msg protomfx.Message) error
	// SendCommandToGroup publishes a command message to things that belong to a specified group, authorized by user token.
	SendCommandToGroup(ctx context.Context, token, groupID string, msg protomfx.Message) error
	// SendCommandToGroupByKey publishes a command message to a group, authorized by publisher thing key (M2M).
	SendCommandToGroupByKey(ctx context.Context, key domain.ThingKey, groupID string, msg protomfx.Message) error
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	publisher messaging.Publisher
	things    protomfx.ThingsServiceClient
}

// New instantiates the HTTP adapter implementation.
func New(publisher messaging.Publisher, things protomfx.ThingsServiceClient) Service {
	return &adapterService{
		publisher: publisher,
		things:    things,
	}
}

func (as *adapterService) Publish(ctx context.Context, key domain.ThingKey, msg protomfx.Message) error {
	tk := &protomfx.ThingKey{
		Value: key.Value,
		Type:  key.Type,
	}
	pc, err := as.things.GetPubConfigByKey(ctx, tk)
	if err != nil {
		return err
	}

	if err := messaging.FormatMessage(pc, &msg); err != nil {
		return err
	}

	if err := as.publisher.Publish(nats.GetMessagesSubject(msg.Publisher, msg.Subtopic), msg); err != nil {
		return err
	}

	return nil
}

func (as *adapterService) SendCommandToThing(ctx context.Context, token, thingID string, message protomfx.Message) error {
	if _, err := as.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: thingID, Action: domain.GroupEditor}); err != nil {
		return err
	}

	if err := as.publisher.Publish(nats.GetThingCommandsSubject(thingID, message.Subtopic), message); err != nil {
		return err
	}

	return nil
}

func (as *adapterService) SendCommandToGroup(ctx context.Context, token, groupID string, message protomfx.Message) error {
	if _, err := as.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: groupID, Action: domain.GroupEditor}); err != nil {
		return err
	}

	if err := as.publisher.Publish(nats.GetGroupCommandsSubject(groupID, message.Subtopic), message); err != nil {
		return err
	}

	return nil
}

func (as *adapterService) SendCommandToThingByKey(ctx context.Context, key things.ThingKey, thingID string, message protomfx.Message) error {
	res, err := as.things.Identify(ctx, &protomfx.ThingKey{Value: key.Value, Type: key.Type})
	if err != nil {
		return err
	}

	if _, err := as.things.CanThingCommand(ctx, &protomfx.ThingCommandReq{PublisherID: res.GetValue(), RecipientID: thingID}); err != nil {
		return err
	}

	return as.publisher.Publish(nats.GetThingCommandsSubject(thingID, message.Subtopic), message)
}

func (as *adapterService) SendCommandToGroupByKey(ctx context.Context, key things.ThingKey, groupID string, message protomfx.Message) error {
	res, err := as.things.Identify(ctx, &protomfx.ThingKey{Value: key.Value, Type: key.Type})
	if err != nil {
		return err
	}

	if _, err := as.things.CanThingGroupCommand(ctx, &protomfx.ThingGroupCommandReq{PublisherID: res.GetValue(), GroupID: groupID}); err != nil {
		return err
	}

	return as.publisher.Publish(nats.GetGroupCommandsSubject(groupID, message.Subtopic), message)
}
