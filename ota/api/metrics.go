// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test
// +build !test

package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/ota"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/go-kit/kit/metrics"
)

var _ ota.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     ota.Service
}

// MetricsMiddleware instruments core service by tracking request count and
// latency.
func MetricsMiddleware(svc ota.Service, counter metrics.Counter, latency metrics.Histogram) ota.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) CreateUpdate(ctx context.Context, token, groupID string, u ota.Update, thingIDs ...string) (ota.Update, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_update").Add(1)
		ms.latency.With("method", "create_update").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateUpdate(ctx, token, groupID, u, thingIDs...)
}

func (ms *metricsMiddleware) ListUpdatesByGroup(ctx context.Context, token, groupID string, pm ota.PageMetadata) (ota.UpdatesPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_updates_by_group").Add(1)
		ms.latency.With("method", "list_updates_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListUpdatesByGroup(ctx, token, groupID, pm)
}

func (ms *metricsMiddleware) ViewUpdate(ctx context.Context, token, id string) (ota.Update, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_update").Add(1)
		ms.latency.With("method", "view_update").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewUpdate(ctx, token, id)
}

func (ms *metricsMiddleware) ListThingUpdates(ctx context.Context, token, updateID string, pm ota.PageMetadata) (ota.ThingUpdatesPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_thing_updates").Add(1)
		ms.latency.With("method", "list_thing_updates").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListThingUpdates(ctx, token, updateID, pm)
}

func (ms *metricsMiddleware) RetryFailed(ctx context.Context, token, updateID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "retry_failed").Add(1)
		ms.latency.With("method", "retry_failed").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RetryFailed(ctx, token, updateID)
}

func (ms *metricsMiddleware) RemoveUpdate(ctx context.Context, token, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_update").Add(1)
		ms.latency.With("method", "remove_update").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveUpdate(ctx, token, id)
}

func (ms *metricsMiddleware) ViewPending(ctx context.Context, key domain.ThingKey) (ota.Command, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_pending").Add(1)
		ms.latency.With("method", "view_pending").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewPending(ctx, key)
}

func (ms *metricsMiddleware) ReportStatus(ctx context.Context, key domain.ThingKey, report ota.StatusReport) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "report_status").Add(1)
		ms.latency.With("method", "report_status").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ReportStatus(ctx, key, report)
}

func (ms *metricsMiddleware) FailTimedOut(ctx context.Context) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "fail_timed_out").Add(1)
		ms.latency.With("method", "fail_timed_out").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.FailTimedOut(ctx)
}

func (ms *metricsMiddleware) RemoveThingUpdatesByThing(ctx context.Context, thingID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_thing_updates_by_thing").Add(1)
		ms.latency.With("method", "remove_thing_updates_by_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveThingUpdatesByThing(ctx, thingID)
}

func (ms *metricsMiddleware) RemoveUpdatesByGroup(ctx context.Context, groupID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_updates_by_group").Add(1)
		ms.latency.With("method", "remove_updates_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveUpdatesByGroup(ctx, groupID)
}

func (ms *metricsMiddleware) ConsumeMessage(subject string, msg protomfx.Message) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "consume_message").Add(1)
		ms.latency.With("method", "consume_message").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ConsumeMessage(subject, msg)
}
