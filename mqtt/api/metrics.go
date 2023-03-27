// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test
// +build !test

package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/mqtt"
	"github.com/go-kit/kit/metrics"
)

var _ mqtt.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     mqtt.Service
}

// MetricsMiddleware instruments core service by tracking request count and
// latency.
func MetricsMiddleware(svc mqtt.Service, counter metrics.Counter, latency metrics.Histogram) mqtt.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) ListSubscriptions(ctx context.Context, chanID, token, key string, pm mqtt.PageMetadata) (mqtt.Page, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_subscriptions").Add(1)
		ms.latency.With("method", "list_subscriptions").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListSubscriptions(ctx, chanID, token, key, pm)
}

func (ms *metricsMiddleware) CreateSubscription(ctx context.Context, sub mqtt.Subscription) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_subscription").Add(1)
		ms.latency.With("method", "create_subscription").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateSubscription(ctx, sub)
}

func (ms *metricsMiddleware) RemoveSubscription(ctx context.Context, sub mqtt.Subscription) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_subscription").Add(1)
		ms.latency.With("method", "remove_subscription").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveSubscription(ctx, sub)
}

func (ms *metricsMiddleware) HasClientID(ctx context.Context, clientID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "has_client_id").Add(1)
		ms.latency.With("method", "has_client_id").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.HasClientID(ctx, clientID)
}

func (ms *metricsMiddleware) UpdateStatus(ctx context.Context, sub mqtt.Subscription) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_status").Add(1)
		ms.latency.With("method", "update_status").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateStatus(ctx, sub)
}
