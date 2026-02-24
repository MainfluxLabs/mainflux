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
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/uiconfigs"
)

var _ uiconfigs.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    uiconfigs.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc uiconfigs.Service, logger log.Logger) uiconfigs.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) ViewOrgConfig(ctx context.Context, token, orgID string) (response uiconfigs.OrgConfig, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_org_config for org %v took %s to complete", orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewOrgConfig(ctx, token, orgID)
}

func (lm *loggingMiddleware) ListOrgsConfigs(ctx context.Context, token string, pm apiutil.PageMetadata) (response uiconfigs.OrgConfigPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_orgs_configs took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListOrgsConfigs(ctx, token, pm)
}

func (lm *loggingMiddleware) UpdateOrgConfig(ctx context.Context, token string, orgConfig uiconfigs.OrgConfig) (response uiconfigs.OrgConfig, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_org_config for org %v  took %s to complete", orgConfig.OrgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateOrgConfig(ctx, token, orgConfig)
}

func (lm *loggingMiddleware) RemoveOrgConfig(ctx context.Context, orgID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_org_config for org %v took %s to complete", orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveOrgConfig(ctx, orgID)
}

func (lm *loggingMiddleware) BackupOrgsConfigs(ctx context.Context, token string) (response uiconfigs.OrgConfigBackup, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method backup_orgs_configs took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.BackupOrgsConfigs(ctx, token)
}

func (lm *loggingMiddleware) ViewThingConfig(ctx context.Context, token, thingID string) (response uiconfigs.ThingConfig, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_thing_config for thing %v took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewThingConfig(ctx, token, thingID)
}

func (lm *loggingMiddleware) ListThingsConfigs(ctx context.Context, token string, pm apiutil.PageMetadata) (response uiconfigs.ThingConfigPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_things_configs took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListThingsConfigs(ctx, token, pm)
}

func (lm *loggingMiddleware) UpdateThingConfig(ctx context.Context, token string, thingConfig uiconfigs.ThingConfig) (response uiconfigs.ThingConfig, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_thing_config for thing %v took %s to complete", thingConfig.ThingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateThingConfig(ctx, token, thingConfig)
}

func (lm *loggingMiddleware) RemoveThingConfig(ctx context.Context, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_thing_config for thing %v took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveThingConfig(ctx, thingID)
}

func (lm *loggingMiddleware) RemoveThingConfigByGroup(ctx context.Context, groupID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_thing_config_by_group for group %v took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveThingConfigByGroup(ctx, groupID)
}

func (lm *loggingMiddleware) BackupThingsConfigs(ctx context.Context, token string) (response uiconfigs.ThingConfigBackup, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method backup_things_configs took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.BackupThingsConfigs(ctx, token)
}

func (lm *loggingMiddleware) Backup(ctx context.Context, token string) (response uiconfigs.Backup, err error) {
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

func (lm *loggingMiddleware) Restore(ctx context.Context, token string, backup uiconfigs.Backup) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method restore took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Restore(ctx, token, backup)
}
