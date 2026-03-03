// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test
// +build !test

package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/downlinks"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/go-kit/kit/metrics"
)

var _ downlinks.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     downlinks.Service
}

// MetricsMiddleware instruments core service by tracking request count and
// latency.
func MetricsMiddleware(svc downlinks.Service, counter metrics.Counter, latency metrics.Histogram) downlinks.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) CreateDownlinks(ctx context.Context, token, thingID string, downlinks ...downlinks.Downlink) (response []downlinks.Downlink, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_downlinks").Add(1)
		ms.latency.With("method", "create_downlinks").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateDownlinks(ctx, token, thingID, downlinks...)
}

func (ms *metricsMiddleware) ListDownlinksByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (downlinks.DownlinksPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_downlinks_by_thing").Add(1)
		ms.latency.With("method", "list_downlinks_by_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListDownlinksByThing(ctx, token, thingID, pm)
}

func (ms *metricsMiddleware) ListDownlinksByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (downlinks.DownlinksPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_downlinks_by_group").Add(1)
		ms.latency.With("method", "list_downlinks_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListDownlinksByGroup(ctx, token, groupID, pm)
}

func (ms *metricsMiddleware) ViewDownlink(ctx context.Context, token, id string) (downlinks.Downlink, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_downlink").Add(1)
		ms.latency.With("method", "view_downlink").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewDownlink(ctx, token, id)
}

func (ms *metricsMiddleware) UpdateDownlink(ctx context.Context, token string, downlink downlinks.Downlink) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_downlink").Add(1)
		ms.latency.With("method", "update_downlink").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateDownlink(ctx, token, downlink)
}

func (ms *metricsMiddleware) RemoveDownlinks(ctx context.Context, token string, id ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_downlinks").Add(1)
		ms.latency.With("method", "remove_downlinks").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveDownlinks(ctx, token, id...)
}

func (ms *metricsMiddleware) RemoveDownlinksByThing(ctx context.Context, thingID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_downlinks_by_thing").Add(1)
		ms.latency.With("method", "remove_downlinks_by_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveDownlinksByThing(ctx, thingID)
}

func (ms *metricsMiddleware) RemoveDownlinksByGroup(ctx context.Context, groupID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_downlinks_by_group").Add(1)
		ms.latency.With("method", "remove_downlinks_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveDownlinksByGroup(ctx, groupID)
}

func (ms *metricsMiddleware) RescheduleTasks(ctx context.Context, profileID string, config map[string]any) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "reschedule_tasks").Add(1)
		ms.latency.With("method", "reschedule_tasks").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RescheduleTasks(ctx, profileID, config)
}

func (ms *metricsMiddleware) LoadAndScheduleTasks(ctx context.Context) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "load_and_schedule_tasks").Add(1)
		ms.latency.With("method", "load_and_schedule_tasks").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.LoadAndScheduleTasks(ctx)
}

func (ms *metricsMiddleware) Backup(ctx context.Context, token string) ([]downlinks.Downlink, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "backup").Add(1)
		ms.latency.With("method", "backup").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Backup(ctx, token)
}

func (ms *metricsMiddleware) Restore(ctx context.Context, token string, dls []downlinks.Downlink) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "restore").Add(1)
		ms.latency.With("method", "restore").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Restore(ctx, token, dls)
}
