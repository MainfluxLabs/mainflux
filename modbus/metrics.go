// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package modbus

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics"
)

var _ Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     Service
}

// MetricsMiddleware instruments Modbus service with Prometheus metrics.
func MetricsMiddleware(svc Service, counter metrics.Counter, latency metrics.Histogram) Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (mm *metricsMiddleware) StartPolling(ctx context.Context) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "start_polling").Add(1)
		mm.latency.With("method", "start_polling").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.StartPolling(ctx)
}
