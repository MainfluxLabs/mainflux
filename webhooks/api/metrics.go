// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test
// +build !test

package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/webhooks"
	"github.com/go-kit/kit/metrics"
)

var _ webhooks.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     webhooks.Service
}

// MetricsMiddleware instruments core service by tracking request count and
// latency.
func MetricsMiddleware(svc webhooks.Service, counter metrics.Counter, latency metrics.Histogram) webhooks.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) CreateWebhook(ctx context.Context, token string, webhook webhooks.Webhook) (response webhooks.Webhook, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_webhook").Add(1)
		ms.latency.With("method", "create_webhook").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateWebhook(ctx, token, webhook)
}

func (ms *metricsMiddleware) ListWebhooks(ctx context.Context, token string) ([]webhooks.Webhook, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_webhooks").Add(1)
		ms.latency.With("method", "list_webhooks").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListWebhooks(ctx, token)
}
