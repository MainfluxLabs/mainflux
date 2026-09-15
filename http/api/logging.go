// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/http"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var _ http.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    http.Service
}

// LoggingMiddleware adds logging facilities to the adapter.
func LoggingMiddleware(svc http.Service, logger log.Logger) http.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) Publish(ctx context.Context, key domain.ThingKey, msg protomfx.Message) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method publish took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Publish(ctx, key, msg)
}

func (lm *loggingMiddleware) SendCommandToThing(ctx context.Context, token, thingID string, cmd protomfx.Command) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method send_command_to_thing for thing id %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.SendCommandToThing(ctx, token, thingID, cmd)
}

func (lm *loggingMiddleware) SendCommandToGroup(ctx context.Context, token, groupID string, cmd protomfx.Command) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method send_command_to_group for group id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.SendCommandToGroup(ctx, token, groupID, cmd)
}

func (lm *loggingMiddleware) SendCommandToThingByKey(ctx context.Context, key domain.ThingKey, thingID string, cmd protomfx.Command) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method send_command_to_thing_by_key for thing id %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.SendCommandToThingByKey(ctx, key, thingID, cmd)
}

func (lm *loggingMiddleware) SendCommandToGroupByKey(ctx context.Context, key domain.ThingKey, groupID string, cmd protomfx.Command) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method send_command_to_group_by_key for group id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.SendCommandToGroupByKey(ctx, key, groupID, cmd)
}
