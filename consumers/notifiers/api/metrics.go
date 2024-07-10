// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	notifiers "github.com/MainfluxLabs/mainflux/consumers/notifiers"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/metrics"
)

var _ notifiers.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     notifiers.Service
}

// MetricsMiddleware instruments core service by tracking request count and latency.
func MetricsMiddleware(svc notifiers.Service, counter metrics.Counter, latency metrics.Histogram) notifiers.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) CreateNotifiers(ctx context.Context, token string, notifiers ...things.Notifier) ([]things.Notifier, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_notifiers").Add(1)
		ms.latency.With("method", "create_notifiers").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateNotifiers(ctx, token, notifiers...)
}

func (ms *metricsMiddleware) ListNotifiersByGroup(ctx context.Context, token string, groupID string) ([]things.Notifier, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_notifiers_by_group").Add(1)
		ms.latency.With("method", "list_notifiers_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListNotifiersByGroup(ctx, token, groupID)
}

func (ms *metricsMiddleware) ViewNotifier(ctx context.Context, token, id string) (things.Notifier, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_notifier").Add(1)
		ms.latency.With("method", "view_notifier").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewNotifier(ctx, token, id)
}

func (ms *metricsMiddleware) UpdateNotifier(ctx context.Context, token string, notifier things.Notifier) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_notifier").Add(1)
		ms.latency.With("method", "update_notifier").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateNotifier(ctx, token, notifier)
}

func (ms *metricsMiddleware) RemoveNotifiers(ctx context.Context, token, groupID string, id ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_notifiers").Add(1)
		ms.latency.With("method", "remove_notifiers").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveNotifiers(ctx, token, groupID, id...)
}

func (ms *metricsMiddleware) Consume(msg interface{}) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "consume").Add(1)
		ms.latency.With("method", "consume").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Consume(msg)
}
