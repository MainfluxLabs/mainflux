//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

// +build !test

package api

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux/re"
)

var _ re.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     re.Service
}

// MetricsMiddleware instruments core service by tracking request count and
// latency.
func MetricsMiddleware(svc re.Service, counter metrics.Counter, latency metrics.Histogram) re.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) Info(ctx context.Context) (info re.Info, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "info").Add(1)
		ms.latency.With("method", "info").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Info(ctx)
}

func (ms *metricsMiddleware) CreateStream(ctx context.Context, token, name, topic, row string) (result string, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_stream").Add(1)
		ms.latency.With("method", "create_stream").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateStream(ctx, token, name, topic, row)
}

func (ms *metricsMiddleware) UpdateStream(ctx context.Context, token, name, topic, row string) (result string, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_stream").Add(1)
		ms.latency.With("method", "create_stream").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateStream(ctx, token, name, topic, row)
}

func (ms *metricsMiddleware) ListStreams(ctx context.Context, token string) (streams []string, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_streams").Add(1)
		ms.latency.With("method", "list_streams").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListStreams(ctx, token)
}

func (ms *metricsMiddleware) ViewStream(ctx context.Context, token, id string) (stream re.Stream, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_stream").Add(1)
		ms.latency.With("method", "view_stream").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewStream(ctx, token, id)
}

func (ms *metricsMiddleware) DeleteStream(ctx context.Context, token string, id string) (result string, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "delete_stream").Add(1)
		ms.latency.With("method", "delete_stream").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.DeleteStream(ctx, token, id)
}
