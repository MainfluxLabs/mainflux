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

	// RemoveObserver ends the observation of the given client, unless another client
	// has since observed the subtopic with the same token.
	RemoveObserver(subtopic string, c Client) error

	// SendCommandToThing publishes a command to the specified thing, authorized by thing key (M2M).
	SendCommandToThing(ctx context.Context, key domain.ThingKey, recipientID string, cmd protomfx.Command) error

	// SendCommandToGroup publishes a command to a group, authorized by thing key (M2M).
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

// observation identifies an observation the way the broker subscription does.
type observation struct {
	token    string
	subtopic string
}

type adapterService struct {
	things    domain.ThingsClient
	pubsub    PubSub
	obsLock   sync.Mutex
	observers map[observation]Client
}

// New instantiates the CoAP adapter implementation.
func New(things domain.ThingsClient, pubsub PubSub) Service {
	as := &adapterService{
		things:    things,
		pubsub:    pubsub,
		observers: make(map[observation]Client),
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

	svc.obsLock.Lock()
	defer svc.obsLock.Unlock()

	if nats.IsCommandsSubject(subtopic) {
		err = svc.pubsub.SubscribeCommands(c.Token(), subtopic, c)
	} else {
		err = svc.pubsub.Subscribe(c.Token(), subtopic, c)
	}
	if err != nil {
		return err
	}

	svc.observers[observation{token: c.Token(), subtopic: subtopic}] = c
	return nil
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

func (svc *adapterService) SendCommandToThing(ctx context.Context, key domain.ThingKey, recipientID string, cmd protomfx.Command) error {
	thingID, err := svc.things.Identify(ctx, key)
	if err != nil {
		return err
	}

	if err := svc.things.CanThingCommand(ctx, domain.ThingCommandReq{ThingID: thingID, RecipientID: recipientID}); err != nil {
		return err
	}

	cmd.ThingId = thingID
	cmd.RecipientId = recipientID
	return svc.pubsub.PublishCommand(nats.GetThingCommandsSubject(recipientID, cmd.Subtopic), cmd)
}

func (svc *adapterService) SendCommandToGroup(ctx context.Context, key domain.ThingKey, groupID string, cmd protomfx.Command) error {
	thingID, err := svc.things.Identify(ctx, key)
	if err != nil {
		return err
	}

	if err := svc.things.CanThingGroupCommand(ctx, domain.ThingGroupCommandReq{ThingID: thingID, GroupID: groupID}); err != nil {
		return err
	}

	cmd.ThingId = thingID
	return svc.pubsub.PublishCommand(nats.GetGroupCommandsSubject(groupID, cmd.Subtopic), cmd)
}

func (svc *adapterService) Unsubscribe(ctx context.Context, key domain.ThingKey, subtopic, token string) error {
	if _, err := svc.things.GetPubConfigByKey(ctx, key); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	svc.obsLock.Lock()
	defer svc.obsLock.Unlock()

	delete(svc.observers, observation{token: token, subtopic: subtopic})
	return svc.pubsub.Unsubscribe(token, subtopic)
}

func (svc *adapterService) RemoveObserver(subtopic string, c Client) error {
	svc.obsLock.Lock()
	defer svc.obsLock.Unlock()

	obs := observation{token: c.Token(), subtopic: subtopic}
	if svc.observers[obs] != c {
		return nil
	}

	delete(svc.observers, obs)
	return svc.pubsub.Unsubscribe(obs.token, subtopic)
}
