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

func (mm *metricsMiddleware) PublishSenMLMessagesFromJSON(ctx context.Context, token string, records []map[string]any) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "publish_senml_messages_from_json").Add(1)
		mm.latency.With("method", "publish_senml_messages_from_json").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.PublishSenMLMessagesFromJSON(ctx, token, records)
}

func (mm *metricsMiddleware) PublishJSONMessagesFromJSON(ctx context.Context, token string, records []map[string]any) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "publish_json_messages_from_json").Add(1)
		mm.latency.With("method", "publish_json_messages_from_json").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.PublishJSONMessagesFromJSON(ctx, token, records)
}

func (mm *metricsMiddleware) PublishJSONMessagesFromCSV(ctx context.Context, token string, csvLines [][]string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "publish_json_messages_from_csv").Add(1)
		mm.latency.With("method", "publish_json_messages_from_csv").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.PublishJSONMessagesFromCSV(ctx, token, csvLines)
}

func (mm *metricsMiddleware) PublishSenMLMessagesFromCSV(ctx context.Context, token string, csvLines [][]string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "publish_senml_messages_from_csv").Add(1)
		mm.latency.With("method", "publish_senml_messages_from_csv").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.PublishSenMLMessagesFromCSV(ctx, token, csvLines)
}
