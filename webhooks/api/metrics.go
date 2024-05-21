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

func (ms *metricsMiddleware) CreateWebhooks(ctx context.Context, token string, webhooks ...webhooks.Webhook) (response []webhooks.Webhook, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_webhooks").Add(1)
		ms.latency.With("method", "create_webhooks").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateWebhooks(ctx, token, webhooks...)
}

func (ms *metricsMiddleware) ListWebhooksByGroup(ctx context.Context, token string, thingID string) ([]webhooks.Webhook, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_webhooks_by_group").Add(1)
		ms.latency.With("method", "list_webhooks_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListWebhooksByGroup(ctx, token, thingID)
}

func (ms *metricsMiddleware) ViewWebhook(ctx context.Context, token, id string) (webhooks.Webhook, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_webhook").Add(1)
		ms.latency.With("method", "view_webhook").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewWebhook(ctx, token, id)
}

func (ms *metricsMiddleware) UpdateWebhook(ctx context.Context, token string, webhook webhooks.Webhook) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_webhook").Add(1)
		ms.latency.With("method", "update_webhook").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateWebhook(ctx, token, webhook)
}

func (ms *metricsMiddleware) RemoveWebhooks(ctx context.Context, token, groupID string, id ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_webhooks").Add(1)
		ms.latency.With("method", "remove_webhooks").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveWebhooks(ctx, token, groupID, id...)
}

func (ms *metricsMiddleware) Consume(message interface{}) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "consume").Add(1)
		ms.latency.With("method", "consume").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Consume(message)
}
