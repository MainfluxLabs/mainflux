// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/audit"
	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/go-kit/kit/metrics"
)

var _ audit.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     audit.Service
}

func MetricsMiddleware(svc audit.Service, counter metrics.Counter, latency metrics.Histogram) audit.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (mm *metricsMiddleware) RecordEvent(ctx context.Context, e events.Event) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "record_event").Add(1)
		mm.latency.With("method", "record_event").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.RecordEvent(ctx, e)
}

func (mm *metricsMiddleware) ListEventsByOrg(ctx context.Context, token string, orgID string, pm audit.PageMetadata) (audit.EventsPage, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "list_events_by_org").Add(1)
		mm.latency.With("method", "list_events_by_org").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ListEventsByOrg(ctx, token, orgID, pm)
}
