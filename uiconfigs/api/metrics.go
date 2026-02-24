// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test
// +build !test

package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/uiconfigs"
	"github.com/go-kit/kit/metrics"
)

var _ uiconfigs.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     uiconfigs.Service
}

// MetricsMiddleware instruments core service by tracking request count and
// latency.
func MetricsMiddleware(svc uiconfigs.Service, counter metrics.Counter, latency metrics.Histogram) uiconfigs.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) ViewOrgConfig(ctx context.Context, token, orgID string) (uiconfigs.OrgConfig, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_org_config").Add(1)
		ms.latency.With("method", "view_org_config").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewOrgConfig(ctx, token, orgID)
}

func (ms *metricsMiddleware) ListOrgsConfigs(ctx context.Context, token string, pm apiutil.PageMetadata) (uiconfigs.OrgConfigPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_orgs_configs").Add(1)
		ms.latency.With("method", "list_orgs_configs").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListOrgsConfigs(ctx, token, pm)
}

func (ms *metricsMiddleware) UpdateOrgConfig(ctx context.Context, token string, orgConfig uiconfigs.OrgConfig) (uiconfigs.OrgConfig, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_org_config").Add(1)
		ms.latency.With("method", "update_org_config").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateOrgConfig(ctx, token, orgConfig)
}

func (ms *metricsMiddleware) RemoveOrgConfig(ctx context.Context, orgID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_org_config").Add(1)
		ms.latency.With("method", "remove_org_config").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveOrgConfig(ctx, orgID)
}

func (ms *metricsMiddleware) BackupOrgsConfigs(ctx context.Context, token string) (uiconfigs.OrgConfigBackup, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "backup_orgs_configs").Add(1)
		ms.latency.With("method", "backup_orgs_configs").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.BackupOrgsConfigs(ctx, token)
}

func (ms *metricsMiddleware) ViewThingConfig(ctx context.Context, token, orgID string) (uiconfigs.ThingConfig, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_thing_config").Add(1)
		ms.latency.With("method", "view_thing_config").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewThingConfig(ctx, token, orgID)
}

func (ms *metricsMiddleware) ListThingsConfigs(ctx context.Context, token string, pm apiutil.PageMetadata) (uiconfigs.ThingConfigPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_things_configs").Add(1)
		ms.latency.With("method", "list_things_configs").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListThingsConfigs(ctx, token, pm)
}

func (ms *metricsMiddleware) UpdateThingConfig(ctx context.Context, token string, thingConfig uiconfigs.ThingConfig) (uiconfigs.ThingConfig, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_thing_config").Add(1)
		ms.latency.With("method", "update_thing_config").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateThingConfig(ctx, token, thingConfig)
}

func (ms *metricsMiddleware) RemoveThingConfig(ctx context.Context, thingID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_thing_config").Add(1)
		ms.latency.With("method", "remove_thing_config").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveThingConfig(ctx, thingID)
}

func (ms *metricsMiddleware) RemoveThingConfigByGroup(ctx context.Context, groupID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_thing_config_by_group").Add(1)
		ms.latency.With("method", "remove_thing_config_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveThingConfigByGroup(ctx, groupID)
}

func (ms *metricsMiddleware) BackupThingsConfigs(ctx context.Context, token string) (uiconfigs.ThingConfigBackup, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "backup_things_configs").Add(1)
		ms.latency.With("method", "backup_things_configs").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.BackupThingsConfigs(ctx, token)
}

func (ms *metricsMiddleware) Backup(ctx context.Context, token string) (uiconfigs.Backup, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "backup").Add(1)
		ms.latency.With("method", "backup").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Backup(ctx, token)
}

func (ms *metricsMiddleware) Restore(ctx context.Context, token string, backup uiconfigs.Backup) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "restore").Add(1)
		ms.latency.With("method", "restore").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Restore(ctx, token, backup)
}
