// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/coap"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/go-kit/kit/metrics"
)

var _ coap.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     coap.Service
}

// MetricsMiddleware instruments adapter by tracking request count and latency.
func MetricsMiddleware(svc coap.Service, counter metrics.Counter, latency metrics.Histogram) coap.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (mm *metricsMiddleware) Publish(ctx context.Context, keyType, key string, msg protomfx.Message) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "publish").Add(1)
		mm.latency.With("method", "publish").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Publish(ctx, keyType, key, msg)
}

func (mm *metricsMiddleware) Subscribe(ctx context.Context, keyType, key, subtopic string, c coap.Client) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "subscribe").Add(1)
		mm.latency.With("method", "subscribe").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Subscribe(ctx, keyType, key, subtopic, c)
}

func (mm *metricsMiddleware) Unsubscribe(ctx context.Context, keyType, key, subtopic, token string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "unsubscribe").Add(1)
		mm.latency.With("method", "unsubscribe").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Unsubscribe(ctx, keyType, key, subtopic, token)
}
