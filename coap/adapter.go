// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package coap contains the domain concept definitions needed to support
// Mainflux CoAP adapter service functionality. All constant values are taken
// from RFC, and could be adjusted based on specific use case.
package coap

import (
	"context"
	"fmt"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

const chansPrefix = "channels"

// ErrUnsubscribe indicates an error to unsubscribe
var ErrUnsubscribe = errors.New("unable to unsubscribe")

// Service specifies CoAP service API.
type Service interface {
	// Publish Messssage
	Publish(ctx context.Context, key string, msg protomfx.Message) error

	// Subscribe subscribes to channel with specified id, subtopic and adds subscription to
	// service map of subscriptions under given ID.
	Subscribe(ctx context.Context, key, chanID, subtopic string, c Client) error

	// Unsubscribe method is used to stop observing resource.
	Unsubscribe(ctx context.Context, key, chanID, subptopic, token string) error
}

var _ Service = (*adapterService)(nil)

// Observers is a map of maps,
type adapterService struct {
	things  protomfx.ThingsServiceClient
	pubsub  messaging.PubSub
	obsLock sync.Mutex
}

// New instantiates the CoAP adapter implementation.
func New(things protomfx.ThingsServiceClient, pubsub messaging.PubSub) Service {
	as := &adapterService{
		things:  things,
		pubsub:  pubsub,
		obsLock: sync.Mutex{},
	}

	return as
}

func (svc *adapterService) Publish(ctx context.Context, key string, msg protomfx.Message) error {
	cr := &protomfx.ConnByKeyReq{
		Key: key,
	}
	conn, err := svc.things.GetConnByKey(ctx, cr)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	m := messaging.CreateMessage(conn, msg.Protocol, msg.Subtopic, &msg.Payload)

	return svc.pubsub.Publish(m)
}

func (svc *adapterService) Subscribe(ctx context.Context, key, chanID, subtopic string, c Client) error {
	cr := &protomfx.ConnByKeyReq{
		Key: key,
	}
	if _, err := svc.things.GetConnByKey(ctx, cr); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	subject := fmt.Sprintf("%s.%s", chansPrefix, chanID)
	if subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, subtopic)
	}
	return svc.pubsub.Subscribe(c.Token(), subject, c)
}

func (svc *adapterService) Unsubscribe(ctx context.Context, key, chanID, subtopic, token string) error {
	cr := &protomfx.ConnByKeyReq{
		Key: key,
	}
	conn, err := svc.things.GetConnByKey(ctx, cr)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	subject := fmt.Sprintf("%s.%s", chansPrefix, conn.ChannelID)
	if subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, subtopic)
	}
	return svc.pubsub.Unsubscribe(token, subject)
}
