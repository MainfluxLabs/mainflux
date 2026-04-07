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
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/webhooks"
)

var _ webhooks.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    webhooks.Service
	auth   domain.AuthClient
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc webhooks.Service, logger log.Logger, auth domain.AuthClient) webhooks.Service {
	return &loggingMiddleware{logger, svc, auth}
}

func (lm *loggingMiddleware) identify(token string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id, err := lm.auth.Identify(ctx, token)
	if err != nil {
		return ""
	}
	return id.Email
}

func (lm *loggingMiddleware) CreateWebhooks(ctx context.Context, token, thingID string, webhooks ...webhooks.Webhook) (response []webhooks.Webhook, err error) {
	defer func(begin time.Time) {
		email := lm.identify(token)
		message := fmt.Sprintf("Method create_webhooks by user %s, webhooks %s took %s to complete", email, response, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateWebhooks(ctx, token, thingID, webhooks...)
}

func (lm *loggingMiddleware) ListWebhooksByGroup(ctx context.Context, token, groupID string, pm webhooks.PageMetadata) (response webhooks.WebhooksPage, err error) {
	defer func(begin time.Time) {
		email := lm.identify(token)
		message := fmt.Sprintf("Method list_webhooks_by_group by user %s, id %s took %s to complete", email, groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListWebhooksByGroup(ctx, token, groupID, pm)
}

func (lm *loggingMiddleware) ListWebhooksByThing(ctx context.Context, token, thingID string, pm webhooks.PageMetadata) (response webhooks.WebhooksPage, err error) {
	defer func(begin time.Time) {
		email := lm.identify(token)
		message := fmt.Sprintf("Method list_webhooks_by_thing by user %s, id %s took %s to complete", email, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListWebhooksByThing(ctx, token, thingID, pm)
}

func (lm *loggingMiddleware) ViewWebhook(ctx context.Context, token, id string) (response webhooks.Webhook, err error) {
	defer func(begin time.Time) {
		email := lm.identify(token)
		message := fmt.Sprintf("Method view_webhook by user %s, id %s took %s to complete", email, id, time.Since(begin))
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
		email := lm.identify(token)
		message := fmt.Sprintf("Method update_webhook by user %s, id %s took %s to complete", email, webhook.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateWebhook(ctx, token, webhook)
}

func (lm *loggingMiddleware) RemoveWebhooks(ctx context.Context, token string, id ...string) (err error) {
	defer func(begin time.Time) {
		email := lm.identify(token)
		message := fmt.Sprintf("Method remove_webhooks by user %s took %s to complete", email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveWebhooks(ctx, token, id...)
}

func (lm *loggingMiddleware) RemoveWebhooksByThing(ctx context.Context, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_webhooks_by_thing for id %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveWebhooksByThing(ctx, thingID)
}

func (lm *loggingMiddleware) RemoveWebhooksByGroup(ctx context.Context, groupID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_webhooks_by_group for id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveWebhooksByGroup(ctx, groupID)
}

func (lm *loggingMiddleware) Consume(subject string, message any) (err error) {
	defer func(begin time.Time) {
		msg := fmt.Sprintf("Method consume took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", msg, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", msg))
	}(time.Now())

	return lm.svc.Consume(subject, message)
}
