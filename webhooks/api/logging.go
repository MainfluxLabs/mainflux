// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test
// +build !test

package api

import (
	"context"
	"fmt"
	"time"

	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/webhooks"
)

var _ webhooks.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    webhooks.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc webhooks.Service, logger log.Logger) webhooks.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) CreateWebhooks(ctx context.Context, token string, webhooks ...webhooks.Webhook) (response []webhooks.Webhook, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_webhooks for token %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateWebhooks(ctx, token, webhooks...)
}

func (lm *loggingMiddleware) ListWebhooksByThing(ctx context.Context, token string, thingID string) (response []webhooks.Webhook, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_webhooks_by_thing for token %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListWebhooksByThing(ctx, token, thingID)
}

func (lm *loggingMiddleware) Forward(ctx context.Context, message messaging.Message) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method forward took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Forward(ctx, message)
}

func (lm *loggingMiddleware) Consume(message interface{}) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method consume took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Consume(message)
}
