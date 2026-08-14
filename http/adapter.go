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
)

// Service specifies coap service API.
type Service interface {
	// Publish Message
	Publish(ctx context.Context, key domain.ThingKey, msg protomfx.Message) error
	// SendCommandToThing publishes a command to the specified thing.
	SendCommandToThing(ctx context.Context, token, thingID string, cmd protomfx.Command) error
	// SendCommandToThingByKey publishes a command to the specified thing, authorized by publisher thing key (M2M).
	SendCommandToThingByKey(ctx context.Context, key domain.ThingKey, thingID string, cmd protomfx.Command) error
	// SendCommandToGroup publishes a command to things that belong to a specified group, authorized by user token.
	SendCommandToGroup(ctx context.Context, token, groupID string, cmd protomfx.Command) error
	// SendCommandToGroupByKey publishes a command to a group, authorized by publisher thing key (M2M).
	SendCommandToGroupByKey(ctx context.Context, key domain.ThingKey, groupID string, cmd protomfx.Command) error
}

// Publisher specifies the minimal publishing capability the HTTP adapter needs.
type Publisher interface {
	messaging.CommandPublisher
	messaging.MessageDispatcher
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	publisher Publisher
	things    domain.ThingsClient
}

// New instantiates the HTTP adapter implementation.
func New(publisher Publisher, things domain.ThingsClient) Service {
	return &adapterService{
		publisher: publisher,
		things:    things,
	}
}

func (as *adapterService) Publish(ctx context.Context, key domain.ThingKey, msg protomfx.Message) error {
	pc, err := as.things.GetPubConfigByKey(ctx, key)
	if err != nil {
		return err
	}

	if err := messaging.FormatMessage(pc, &msg); err != nil {
		return err
	}

	return as.publisher.PublishByFlags(msg, pc.ProfileConfig)
}

func (as *adapterService) SendCommandToThing(ctx context.Context, token, thingID string, cmd protomfx.Command) error {
	if err := as.things.CanUserAccessThing(ctx, domain.UserAccessReq{Token: token, ID: thingID, Action: domain.GroupEditor}); err != nil {
		return err
	}

	cmd.RecipientID = thingID
	if err := as.publisher.PublishCommand(nats.GetThingCommandsSubject(thingID, cmd.Subtopic), cmd); err != nil {
		return err
	}

	return nil
}

func (as *adapterService) SendCommandToGroup(ctx context.Context, token, groupID string, cmd protomfx.Command) error {
	if err := as.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: groupID, Action: domain.GroupEditor}); err != nil {
		return err
	}

	if err := as.publisher.PublishCommand(nats.GetGroupCommandsSubject(groupID, cmd.Subtopic), cmd); err != nil {
		return err
	}

	return nil
}

func (as *adapterService) SendCommandToThingByKey(ctx context.Context, key domain.ThingKey, thingID string, cmd protomfx.Command) error {
	publisherID, err := as.things.Identify(ctx, key)
	if err != nil {
		return err
	}

	if err := as.things.CanThingCommand(ctx, domain.ThingCommandReq{PublisherID: publisherID, RecipientID: thingID}); err != nil {
		return err
	}

	cmd.Publisher = publisherID
	cmd.RecipientID = thingID
	return as.publisher.PublishCommand(nats.GetThingCommandsSubject(thingID, cmd.Subtopic), cmd)
}

func (as *adapterService) SendCommandToGroupByKey(ctx context.Context, key domain.ThingKey, groupID string, cmd protomfx.Command) error {
	publisherID, err := as.things.Identify(ctx, key)
	if err != nil {
		return err
	}

	if err := as.things.CanThingGroupCommand(ctx, domain.ThingGroupCommandReq{PublisherID: publisherID, GroupID: groupID}); err != nil {
		return err
	}

	cmd.Publisher = publisherID
	return as.publisher.PublishCommand(nats.GetGroupCommandsSubject(groupID, cmd.Subtopic), cmd)
}
