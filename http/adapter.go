// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package http contains the domain concept definitions needed to support
// Mainflux http adapter service functionality.
package http

import (
	"context"

	domainthings "github.com/MainfluxLabs/mainflux/pkg/domain/things"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/protoutil"
)

// Service specifies coap service API.
type Service interface {
	// Publish Message
	Publish(ctx context.Context, key domainthings.ThingKey, msg protomfx.Message) error
	// SendCommandToThing publishes a command message to the specified thing.
	SendCommandToThing(ctx context.Context, token, thingID string, msg protomfx.Message) error
	// SendCommandToGroup publishes a command message to things that belong to a specified group.
	SendCommandToGroup(ctx context.Context, token, groupID string, msg protomfx.Message) error
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	publisher messaging.Publisher
	things    domainthings.Client
}

// New instantiates the HTTP adapter implementation.
func New(publisher messaging.Publisher, things domainthings.Client) Service {
	return &adapterService{
		publisher: publisher,
		things:    things,
	}
}

func (as *adapterService) Publish(ctx context.Context, key domainthings.ThingKey, msg protomfx.Message) error {
	pc, err := as.things.GetPubConfigByKey(ctx, key)
	if err != nil {
		return err
	}

	if err := messaging.FormatMessage(protoutil.PubConfigInfoToProto(pc), &msg); err != nil {
		return err
	}

	if err := as.publisher.Publish(nats.GetMessagesSubject(msg.Publisher, msg.Subtopic), msg); err != nil {
		return err
	}

	return nil
}

func (as *adapterService) SendCommandToThing(ctx context.Context, token, thingID string, message protomfx.Message) error {
	if err := as.things.CanUserAccessThing(ctx, domainthings.UserAccessReq{Token: token, ID: thingID, Action: domainthings.Editor}); err != nil {
		return err
	}

	if err := as.publisher.Publish(nats.GetThingCommandsSubject(thingID, message.Subtopic), message); err != nil {
		return err
	}

	return nil
}

func (as *adapterService) SendCommandToGroup(ctx context.Context, token, groupID string, message protomfx.Message) error {
	if err := as.things.CanUserAccessGroup(ctx, domainthings.UserAccessReq{Token: token, ID: groupID, Action: domainthings.Editor}); err != nil {
		return err
	}

	if err := as.publisher.Publish(nats.GetGroupCommandsSubject(groupID, message.Subtopic), message); err != nil {
		return err
	}

	return nil
}
