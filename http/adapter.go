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
	"github.com/MainfluxLabs/mainflux/things"
)

// Service specifies coap service API.
type Service interface {
	// Publish Message
	Publish(ctx context.Context, key things.ThingKey, msg protomfx.Message) error
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	things protomfx.ThingsServiceClient
	rules  protomfx.RulesServiceClient
	logger logger.Logger
}

// New instantiates the HTTP adapter implementation.
func New(things protomfx.ThingsServiceClient, rules protomfx.RulesServiceClient, logger logger.Logger) Service {
	return &adapterService{
		things: things,
		rules:  rules,
		logger: logger,
	}
}

func (as *adapterService) Publish(ctx context.Context, key things.ThingKey, message protomfx.Message) error {
	cr := &protomfx.ThingKey{Value: key.Value, Type: key.Type}
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
