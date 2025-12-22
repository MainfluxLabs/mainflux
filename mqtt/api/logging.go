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

func (lm *loggingMiddleware) ListSubscriptions(ctx context.Context, groupID, token string, pm mqtt.PageMetadata) (page mqtt.Page, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_subscriptions for group id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListSubscriptions(ctx, groupID, token, pm)
}

func (lm *loggingMiddleware) CreateSubscription(ctx context.Context, sub mqtt.Subscription) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_subscription for thing id %s, group id %s, and subtopic %s took %s to complete",
			sub.ThingID, sub.GroupID, sub.Subtopic, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateSubscription(ctx, sub)
}

func (lm *loggingMiddleware) RemoveSubscription(ctx context.Context, sub mqtt.Subscription) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_subscription for thing id %s, group id %s, and subtopic %s took %s to complete",
			sub.ThingID, sub.GroupID, sub.Subtopic, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveSubscription(ctx, sub)
}

func (lm *loggingMiddleware) HasClientID(ctx context.Context, clientID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method has_client_id for client id %s took %s to complete", clientID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.HasClientID(ctx, clientID)
}

func (lm *loggingMiddleware) UpdateStatus(ctx context.Context, sub mqtt.Subscription) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_status for thing id %s, group id %s and subtopic %s took %s to complete",
			sub.ThingID, sub.GroupID, sub.Subtopic, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateStatus(ctx, sub)
}
