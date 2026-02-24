// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/converters"
	"github.com/go-kit/kit/metrics"
)

var _ converters.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     converters.Service
}

// MetricsMiddleware instruments adapter by tracking request count and latency.
func MetricsMiddleware(svc converters.Service, counter metrics.Counter, latency metrics.Histogram) converters.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (mm *metricsMiddleware) PublishJSONMessages(ctx context.Context, token string, csvLines [][]string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "publish_json_messages").Add(1)
		mm.latency.With("method", "publish_json_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.PublishJSONMessages(ctx, token, csvLines)
}

func (mm *metricsMiddleware) PublishSenMLMessages(ctx context.Context, token string, csvLines [][]string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "publish_senml_messages").Add(1)
		mm.latency.With("method", "publish_senml_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.PublishSenMLMessages(ctx, token, csvLines)
}
