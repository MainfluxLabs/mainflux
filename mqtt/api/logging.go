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
	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

var _ mqtt.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    mqtt.Service
	auth   domain.AuthClient
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc mqtt.Service, logger log.Logger, auth domain.AuthClient) mqtt.Service {
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

func (lm *loggingMiddleware) ListSubscriptions(ctx context.Context, groupID, token string, pm mqtt.PageMetadata) (_ mqtt.Page, err error) {
	defer func(begin time.Time) {
		email := lm.identify(token)
		message := fmt.Sprintf("Method list_subscriptions by user %s, group id %s took %s to complete", email, groupID, time.Since(begin))
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
		message := fmt.Sprintf("Method create_subscription for thing id %s, group id %s, and topic %s took %s to complete",
			sub.ThingID, sub.GroupID, sub.Topic, time.Since(begin))
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
		message := fmt.Sprintf("Method remove_subscription for thing id %s, group id %s, and topic %s took %s to complete",
			sub.ThingID, sub.GroupID, sub.Topic, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveSubscription(ctx, sub)
}

func (lm *loggingMiddleware) RemoveSubscriptionsByThing(ctx context.Context, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_subscriptions_by_thing for thing id %s took %s to complete",
			thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveSubscriptionsByThing(ctx, thingID)
}

func (lm *loggingMiddleware) RemoveSubscriptionsByGroup(ctx context.Context, groupID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_subscriptions_by_group for group id %s took %s to complete",
			groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveSubscriptionsByGroup(ctx, groupID)
}
