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

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

// Service specifies CoAP service API.
type Service interface {
	// Publish Messssage
	Publish(ctx context.Context, key string, msg protomfx.Message) error

	// Subscribe subscribes to profile with specified id, subtopic and adds subscription to
	// service map of subscriptions under given ID.
	Subscribe(ctx context.Context, key, subtopic string, c Client) error

	// Unsubscribe method is used to stop observing resource.
	Unsubscribe(ctx context.Context, key, subptopic, token string) error
}

var _ Service = (*adapterService)(nil)

// Observers is a map of maps,
type adapterService struct {
	things  protomfx.ThingsServiceClient
	rules   protomfx.RulesServiceClient
	pubsub  messaging.PubSub
	logger  logger.Logger
	obsLock sync.Mutex
}

// New instantiates the CoAP adapter implementation.
func New(things protomfx.ThingsServiceClient, rules protomfx.RulesServiceClient, pubsub messaging.PubSub, logger logger.Logger) Service {
	as := &adapterService{
		things:  things,
		rules:   rules,
		pubsub:  pubsub,
		logger:  logger,
		obsLock: sync.Mutex{},
	}

	return as
}

func (svc *adapterService) Publish(ctx context.Context, key string, message protomfx.Message) error {
	cr := &protomfx.ThingKey{Key: key}
	pc, err := svc.things.GetPubConfByKey(ctx, cr)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	if err := messaging.FormatMessage(pc, &message); err != nil {
		return err
	}

	msg := message
	go func(m protomfx.Message) {
		_, err := svc.rules.Publish(context.Background(), &protomfx.PublishReq{Message: &m})
		if err != nil {
			svc.logger.Error(fmt.Sprintf("%s: %s", messaging.ErrPublishMessage, err))
		}
	}(msg)

	subjects := nats.GetSubjects(message.Subtopic)
	for _, sub := range subjects {
		msg := message
		msg.Subject = sub

		go func(m protomfx.Message) {
			if err := svc.pubsub.Publish(m); err != nil {
				svc.logger.Error(fmt.Sprintf("%s: %s", messaging.ErrPublishMessage, err))
			}
		}(msg)
	}

	return nil
}

func (svc *adapterService) Subscribe(ctx context.Context, key, subtopic string, c Client) error {
	cr := &protomfx.ThingKey{
		Key: key,
	}
	if _, err := svc.things.GetPubConfByKey(ctx, cr); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return svc.pubsub.Subscribe(c.Token(), subtopic, c)
}

func (svc *adapterService) Unsubscribe(ctx context.Context, key, subtopic, token string) error {
	cr := &protomfx.ThingKey{
		Key: key,
	}
	_, err := svc.things.GetPubConfByKey(ctx, cr)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return svc.pubsub.Unsubscribe(token, subtopic)
}
