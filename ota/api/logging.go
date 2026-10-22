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
	"github.com/MainfluxLabs/mainflux/ota"
	"github.com/MainfluxLabs/mainflux/pkg/authn"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var _ ota.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    ota.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc ota.Service, logger log.Logger) ota.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) CreateUpdate(ctx context.Context, token, groupID string, u ota.Update, thingIDs ...string) (response ota.Update, err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method create_update by user %s, group id %s, id %s took %s to complete", email, groupID, response.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateUpdate(ctx, token, groupID, u, thingIDs...)
}

func (lm *loggingMiddleware) ListUpdatesByGroup(ctx context.Context, token, groupID string, pm ota.PageMetadata) (response ota.UpdatesPage, err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method list_updates_by_group by user %s, id %s took %s to complete", email, groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListUpdatesByGroup(ctx, token, groupID, pm)
}

func (lm *loggingMiddleware) ViewUpdate(ctx context.Context, token, id string) (response ota.Update, err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method view_update by user %s, id %s took %s to complete", email, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewUpdate(ctx, token, id)
}

func (lm *loggingMiddleware) ListThingUpdates(ctx context.Context, token, updateID string, pm ota.PageMetadata) (response ota.ThingUpdatesPage, err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method list_thing_updates by user %s, id %s took %s to complete", email, updateID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListThingUpdates(ctx, token, updateID, pm)
}

func (lm *loggingMiddleware) RetryFailed(ctx context.Context, token, updateID string) (err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method retry_failed by user %s, id %s took %s to complete", email, updateID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RetryFailed(ctx, token, updateID)
}

func (lm *loggingMiddleware) RemoveUpdate(ctx context.Context, token, id string) (err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method remove_update by user %s, id %s took %s to complete", email, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveUpdate(ctx, token, id)
}

func (lm *loggingMiddleware) ViewPending(ctx context.Context, key domain.ThingKey) (response ota.Command, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_pending took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewPending(ctx, key)
}

func (lm *loggingMiddleware) ReportStatus(ctx context.Context, key domain.ThingKey, report ota.StatusReport) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method report_status for update id %s, status %s took %s to complete", report.UpdateID, report.Status, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ReportStatus(ctx, key, report)
}

func (lm *loggingMiddleware) FailTimedOut(ctx context.Context) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method fail_timed_out took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.FailTimedOut(ctx)
}

func (lm *loggingMiddleware) RemoveThingUpdatesByThing(ctx context.Context, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_thing_updates_by_thing for id %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveThingUpdatesByThing(ctx, thingID)
}

func (lm *loggingMiddleware) RemoveUpdatesByGroup(ctx context.Context, groupID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_updates_by_group for id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveUpdatesByGroup(ctx, groupID)
}

func (lm *loggingMiddleware) ConsumeMessage(subject string, msg protomfx.Message) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method consume_message for thing id %s took %s to complete", msg.ThingId, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ConsumeMessage(subject, msg)
}
