// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test
// +build !test

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/downlinks"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var _ downlinks.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    downlinks.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc downlinks.Service, logger log.Logger) downlinks.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) CreateDownlinks(ctx context.Context, token, thingID string, downlinks ...downlinks.Downlink) (response []downlinks.Downlink, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_downlinks for downlinks %v took %s to complete", response, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateDownlinks(ctx, token, thingID, downlinks...)
}

func (lm *loggingMiddleware) ListDownlinksByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (response downlinks.DownlinksPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_downlinks_by_thing for id %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListDownlinksByThing(ctx, token, thingID, pm)
}

func (lm *loggingMiddleware) ListDownlinksByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (response downlinks.DownlinksPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_downlinks_by_group for id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListDownlinksByGroup(ctx, token, groupID, pm)
}

func (lm *loggingMiddleware) ViewDownlink(ctx context.Context, token, id string) (response downlinks.Downlink, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_downlink for id %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewDownlink(ctx, token, id)
}

func (lm *loggingMiddleware) UpdateDownlink(ctx context.Context, token string, downlink downlinks.Downlink) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_downlink for id %s took %s to complete", downlink.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateDownlink(ctx, token, downlink)
}

func (lm *loggingMiddleware) RemoveDownlinks(ctx context.Context, token string, id ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_downlinks took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveDownlinks(ctx, token, id...)
}

func (lm *loggingMiddleware) RemoveDownlinksByThing(ctx context.Context, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_downlinks_by_thing for id %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveDownlinksByThing(ctx, thingID)
}

func (lm *loggingMiddleware) RemoveDownlinksByGroup(ctx context.Context, groupID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_downlinks_by_group for id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveDownlinksByGroup(ctx, groupID)
}

func (lm *loggingMiddleware) RescheduleTasks(ctx context.Context, profileID string, config map[string]any) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method reschedule_tasks for profile %s and config %v took %s to complete", profileID, config, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RescheduleTasks(ctx, profileID, config)
}

func (lm *loggingMiddleware) LoadAndScheduleTasks(ctx context.Context) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method load_and_schedule_tasks took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.LoadAndScheduleTasks(ctx)
}

func (lm *loggingMiddleware) Backup(ctx context.Context, token string) (response []downlinks.Downlink, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method backup took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Backup(ctx, token)
}

func (lm *loggingMiddleware) Restore(ctx context.Context, token string, dls []downlinks.Downlink) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method restore took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Restore(ctx, token, dls)
}
