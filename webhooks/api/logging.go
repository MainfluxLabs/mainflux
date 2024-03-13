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

func (lm *loggingMiddleware) CreateWebhook(ctx context.Context, token string, webhook webhooks.Webhook) (response webhooks.Webhook, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_webhook for token %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateWebhook(ctx, token, webhook)
}

func (lm *loggingMiddleware) ListWebhooks(ctx context.Context, token string) (response []webhooks.Webhook, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_webhooks for token %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListWebhooks(ctx, token)
}
