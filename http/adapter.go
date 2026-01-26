// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package http contains the domain concept definitions needed to support
// Mainflux http adapter service functionality.
package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
)

// Service specifies coap service API.
type Service interface {
	// Publish Message
	Publish(ctx context.Context, key things.ThingKey, msg protomfx.Message) error
	// SendCommandByThing publishes a command message to the specified thing.
	SendCommandByThing(ctx context.Context, token, thingID string, msg protomfx.Message) error
	// SendCommandByGroup publishes a command message to things that belong to a specified group.
	SendCommandByGroup(ctx context.Context, token, groupID string, msg protomfx.Message) error
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

func (as *adapterService) Publish(ctx context.Context, key things.ThingKey, msg protomfx.Message) error {
	cr := &protomfx.ThingKey{Value: key.Value, Type: key.Type}
	pc, err := as.things.GetPubConfByKey(ctx, cr)
	if err != nil {
		return err
	}

	if err := messaging.FormatMessage(pc, &msg); err != nil {
		return err
	}

	msg.Subject = nats.MessagesSubject(msg.Publisher, msg.Subtopic)
	if err := as.publisher.Publish(msg); err != nil {
		return err
	}

	return nil
}

func (as *adapterService) SendCommandByThing(ctx context.Context, token, thingID string, message protomfx.Message) error {
	if _, err := as.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: thingID, Action: things.Editor}); err != nil {
		return err
	}

	message.Subject = nats.ThingCommandsSubject(thingID, message.Subtopic)
	if err := as.publisher.Publish(message); err != nil {
		return err
	}

	return nil
}

func (as *adapterService) SendCommandByGroup(ctx context.Context, token, groupID string, message protomfx.Message) error {
	if _, err := as.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: groupID, Action: things.Editor}); err != nil {
		return err
	}

	message.Subject = nats.GroupCommandsSubject(groupID, message.Subtopic)
	if err := as.publisher.Publish(message); err != nil {
		return err
	}

	return nil
}
