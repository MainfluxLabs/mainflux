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
	"github.com/MainfluxLabs/mainflux/mqtt"
)

var _ mqtt.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    mqtt.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc mqtt.Service, logger log.Logger) mqtt.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) ListSubscriptions(ctx context.Context, token string, pm mqtt.PageMetadata) (page mqtt.Page, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_all_subscriptions took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListSubscriptions(ctx, token, pm)
}

func (lm *loggingMiddleware) CreateSubscription(ctx context.Context, token string, sub mqtt.Subscription) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_subscription took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateSubscription(ctx, token, sub)
}

func (lm *loggingMiddleware) RemoveSubscription(ctx context.Context, token string, sub mqtt.Subscription) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_subscription took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveSubscription(ctx, token, sub)
}
