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
		message := fmt.Sprintf("Method create_webhooks for webhooks %s took %s to complete", response, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateWebhooks(ctx, token, webhooks...)
}

func (lm *loggingMiddleware) ListWebhooksByGroup(ctx context.Context, token string, groupID string) (response []webhooks.Webhook, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_webhooks_by_group for id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListWebhooksByGroup(ctx, token, groupID)
}

func (lm *loggingMiddleware) ViewWebhook(ctx context.Context, token, id string) (response webhooks.Webhook, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_webhook for id %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewWebhook(ctx, token, id)
}

func (lm *loggingMiddleware) UpdateWebhook(ctx context.Context, token string, webhook webhooks.Webhook) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_webhook for id %s took %s to complete", webhook.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateWebhook(ctx, token, webhook)
}

func (lm *loggingMiddleware) RemoveWebhooks(ctx context.Context, token, groupID string, id ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_webhooks took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveWebhooks(ctx, token, groupID, id...)
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
