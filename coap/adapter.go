// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package coap contains the domain concept definitions needed to support
// Mainflux CoAP adapter service functionality. All constant values are taken
// from RFC, and could be adjusted based on specific use case.
package coap

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
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
	SendCommandToThing(ctx context.Context, key domain.ThingKey, thingID string, cmd protomfx.Command) error

	// SendCommandToGroup publishes a command to a group, authorized by publisher thing key (M2M).
	SendCommandToGroup(ctx context.Context, key domain.ThingKey, groupID string, cmd protomfx.Command) error
}

// PubSub specifies the minimal publish/subscribe capability the CoAP adapter needs.
type PubSub interface {
	messaging.CommandPublisher
	messaging.MessageDispatcher
	messaging.Subscriber
	messaging.CommandSubscriber
}

const (
	subjectPrefixThings = "things"
	subjectPrefixGroups = "groups"
)

var _ Service = (*adapterService)(nil)

type adapterService struct {
	things  domain.ThingsClient
	pubsub  PubSub
	obsLock sync.Mutex
}

// New instantiates the CoAP adapter implementation.
func New(things domain.ThingsClient, pubsub PubSub) Service {
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

	if err := messaging.FormatMessage(pc, &msg); err != nil {
		return err
	}

	return svc.pubsub.Dispatch(msg, pc.ProfileConfig)
}

func (svc *adapterService) Subscribe(ctx context.Context, key domain.ThingKey, subtopic string, c Client) error {
	thingID, err := svc.things.Identify(ctx, key)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	if err := svc.authorizeSubscribe(ctx, thingID, subtopic); err != nil {
		return err
	}

	if nats.IsCommandsSubject(subtopic) {
		return svc.pubsub.SubscribeCommands(c.Token(), subtopic, c)
	}

	return svc.pubsub.Subscribe(c.Token(), subtopic, c)
}

func isWildcard(elem string) bool {
	return elem == "*" || elem == ">"
}

// authorizeSubscribe restricts observation of thing and group subjects to the
// observing thing's own ID and its group, like the MQTT adapter does for its
// custom topics. Outside of those subjects, wildcards are rejected so that a
// thing cannot observe every subject on the broker.
func (svc *adapterService) authorizeSubscribe(ctx context.Context, thingID, subject string) error {
	elems := strings.Split(subject, ".")

	if len(elems) >= 2 {
		switch elems[0] {
		case subjectPrefixThings:
			if elems[1] != thingID {
				return errors.ErrAuthorization
			}
			return authorizeSubjectType(elems)
		case subjectPrefixGroups:
			groupID, err := svc.things.GetGroupIDByThing(ctx, thingID)
			if err != nil {
				return errors.Wrap(errors.ErrAuthorization, err)
			}
			if elems[1] != groupID {
				return errors.ErrAuthorization
			}
			return authorizeSubjectType(elems)
		}
	}

	if slices.ContainsFunc(elems, isWildcard) {
		return errors.ErrAuthorization
	}

	return nil
}

// authorizeSubjectType rejects a wildcard in place of messages or commands,
// since the observed type must be known to decode it.
func authorizeSubjectType(elems []string) error {
	if len(elems) >= 3 && isWildcard(elems[2]) {
		return errors.ErrAuthorization
	}

	return nil
}

func (svc *adapterService) SendCommandToThing(ctx context.Context, key domain.ThingKey, thingID string, cmd protomfx.Command) error {
	res, err := svc.things.Identify(ctx, key)
	if err != nil {
		return err
	}

	if err := svc.things.CanThingCommand(ctx, domain.ThingCommandReq{PublisherID: res, RecipientID: thingID}); err != nil {
		return err
	}

	cmd.Publisher = res
	cmd.RecipientId = thingID
	return svc.pubsub.PublishCommand(nats.GetThingCommandsSubject(thingID, cmd.Subtopic), cmd)
}

func (svc *adapterService) SendCommandToGroup(ctx context.Context, key domain.ThingKey, groupID string, cmd protomfx.Command) error {
	thingID, err := svc.things.Identify(ctx, key)
	if err != nil {
		return err
	}

	if err := svc.things.CanThingGroupCommand(ctx, domain.ThingGroupCommandReq{PublisherID: thingID, GroupID: groupID}); err != nil {
		return err
	}

	cmd.Publisher = thingID
	return svc.pubsub.PublishCommand(nats.GetGroupCommandsSubject(groupID, cmd.Subtopic), cmd)
}

func (svc *adapterService) Unsubscribe(ctx context.Context, key domain.ThingKey, subtopic, token string) error {
	if _, err := svc.things.GetPubConfigByKey(ctx, key); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return svc.pubsub.Unsubscribe(token, subtopic)
}
