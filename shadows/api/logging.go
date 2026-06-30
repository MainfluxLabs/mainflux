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
	"github.com/MainfluxLabs/mainflux/pkg/authn"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/shadows"
)

var _ shadows.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    shadows.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc shadows.Service, logger log.Logger) shadows.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) UpdateDesiredState(ctx context.Context, token, thingID string, desired shadows.State) (response shadows.Shadow, err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method update_desired_state by user %s, thing id %s took %s to complete", email, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateDesiredState(ctx, token, thingID, desired)
}

func (lm *loggingMiddleware) ViewShadow(ctx context.Context, token, thingID string) (response shadows.Shadow, err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method view_shadow by user %s, thing id %s took %s to complete", email, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewShadow(ctx, token, thingID)
}

func (lm *loggingMiddleware) RemoveShadow(ctx context.Context, token, thingID string) (err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method remove_shadow by user %s, thing id %s took %s to complete", email, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveShadow(ctx, token, thingID)
}

func (lm *loggingMiddleware) RemoveByThing(ctx context.Context, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_by_thing for thing %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveByThing(ctx, thingID)
}

func (lm *loggingMiddleware) ConsumeMessage(subject string, msg protomfx.Message) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method consume_message for thing %s took %s to complete", msg.Publisher, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ConsumeMessage(subject, msg)
}
