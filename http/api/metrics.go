// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/http"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/metrics"
)

var _ http.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     http.Service
}

// MetricsMiddleware instruments adapter by tracking request count and latency.
func MetricsMiddleware(svc http.Service, counter metrics.Counter, latency metrics.Histogram) http.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (mm *metricsMiddleware) Publish(ctx context.Context, key things.ThingKey, msg protomfx.Message) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "publish").Add(1)
		mm.latency.With("method", "publish").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Publish(ctx, key, msg)
}

func (mm *metricsMiddleware) SendCommandByThing(ctx context.Context, token, thingID string, msg protomfx.Message) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "send_command_by_thing").Add(1)
		mm.latency.With("method", "send_command_by_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.SendCommandByThing(ctx, token, thingID, msg)
}
