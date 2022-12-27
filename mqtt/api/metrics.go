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

func (ms *metricsMiddleware) ListAllSubscriptions(ctx context.Context, token string, pm mqtt.PageMetadata) (mqtt.Page, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_all_subscriptions").Add(1)
		ms.latency.With("method", "list_all_subscriptions").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListAllSubscriptions(ctx, token, pm)
}
