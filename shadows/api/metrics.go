// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test
// +build !test

package api

import (
	"context"
	"time"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/shadows"
	"github.com/go-kit/kit/metrics"
)

var _ shadows.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     shadows.Service
}

// MetricsMiddleware instruments core service by tracking request count and latency.
func MetricsMiddleware(svc shadows.Service, counter metrics.Counter, latency metrics.Histogram) shadows.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) UpdateDesiredState(ctx context.Context, token, thingID string, desired shadows.State) (shadows.Shadow, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_desired_state").Add(1)
		ms.latency.With("method", "update_desired_state").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateDesiredState(ctx, token, thingID, desired)
}

func (ms *metricsMiddleware) ViewShadow(ctx context.Context, token, thingID string) (shadows.Shadow, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_shadow").Add(1)
		ms.latency.With("method", "view_shadow").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewShadow(ctx, token, thingID)
}

func (ms *metricsMiddleware) RemoveShadow(ctx context.Context, token, thingID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_shadow").Add(1)
		ms.latency.With("method", "remove_shadow").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveShadow(ctx, token, thingID)
}

func (ms *metricsMiddleware) RemoveByThing(ctx context.Context, thingID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_by_thing").Add(1)
		ms.latency.With("method", "remove_by_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveByThing(ctx, thingID)
}

func (ms *metricsMiddleware) ConsumeMessage(subject string, msg protomfx.Message) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "consume_message").Add(1)
		ms.latency.With("method", "consume_message").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ConsumeMessage(subject, msg)
}
