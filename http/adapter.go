// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package http contains the domain concept definitions needed to support
// Mainflux http adapter service functionality.
package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
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
	cr := &protomfx.PubConfByKeyReq{Key: key}
	pc, err := as.things.GetPubConfByKey(ctx, cr)
	if err != nil {
		return err
	}

	if err := messaging.FormatMessage(pc, &message); err != nil {
		return err
	}

	if _, err = as.rules.Publish(context.Background(), &protomfx.PublishReq{Message: &message}); err != nil {
		return err
	}

	return nil
}
