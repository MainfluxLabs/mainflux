// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package http contains the domain concept definitions needed to support
// Mainflux http adapter service functionality.
package http

import (
	"context"
	"fmt"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

// Service specifies coap service API.
type Service interface {
	// Publish Message
	Publish(ctx context.Context, token string, msg protomfx.Message) error
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	publisher messaging.Publisher
	things    protomfx.ThingsServiceClient
	rules     protomfx.RulesServiceClient
	logger    logger.Logger
}

// New instantiates the HTTP adapter implementation.
func New(publisher messaging.Publisher, things protomfx.ThingsServiceClient, rules protomfx.RulesServiceClient, logger logger.Logger) Service {
	return &adapterService{
		publisher: publisher,
		things:    things,
		rules:     rules,
		logger:    logger,
	}
}

func (as *adapterService) Publish(ctx context.Context, key string, message protomfx.Message) error {
	cr := &protomfx.PubConfByKeyReq{
		Key: key,
	}
	pc, err := as.things.GetPubConfByKey(ctx, cr)
	if err != nil {
		return err
	}

	msgs, err := messaging.SplitMessage(pc, message)
	if err != nil {
		return err
	}

	for _, m := range msgs {
		mr := m
		go func(m protomfx.Message) {
			_, err := as.rules.Publish(context.Background(), &protomfx.PublishReq{Message: &m})
			if err != nil {
				as.logger.Error(fmt.Sprintf("%s: %s", messaging.ErrPublishMessage, err))
			}
		}(mr)

		subjects := nats.GetSubjects(pc.GetProfileConfig(), m.Subtopic)
		for _, sub := range subjects {
			mp := m
			mp.Subject = sub

			go func(m protomfx.Message) {
				if err := as.publisher.Publish(m); err != nil {
					as.logger.Error(fmt.Sprintf("%s: %s", messaging.ErrPublishMessage, err))
				}
			}(mp)
		}
	}

	return nil
}
