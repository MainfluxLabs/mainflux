// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/metrics"
)

var _ readers.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     readers.Service
}

// MetricsMiddleware instruments core service by tracking request count and latency.
func MetricsMiddleware(svc readers.Service, counter metrics.Counter, latency metrics.Histogram) readers.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (mm *metricsMiddleware) ListJSONMessages(ctx context.Context, token string, key things.ThingKey, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "list_json_messages").Add(1)
		mm.latency.With("method", "list_json_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ListJSONMessages(ctx, token, key, rpm)
}

func (mm *metricsMiddleware) ListSenMLMessages(ctx context.Context, token string, key things.ThingKey, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "list_senml_messages").Add(1)
		mm.latency.With("method", "list_senml_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ListSenMLMessages(ctx, token, key, rpm)
}

func (mm *metricsMiddleware) BackupJSONMessages(ctx context.Context, token string, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "backup_json_messages").Add(1)
		mm.latency.With("method", "backup_json_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.BackupJSONMessages(ctx, token, rpm)
}

func (mm *metricsMiddleware) BackupSenMLMessages(ctx context.Context, token string, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "backup_senml_messages").Add(1)
		mm.latency.With("method", "backup_senml_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.BackupSenMLMessages(ctx, token, rpm)
}

func (mm *metricsMiddleware) RestoreJSONMessages(ctx context.Context, token string, messages ...readers.Message) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "restore_json_messages").Add(1)
		mm.latency.With("method", "restore_json_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.RestoreJSONMessages(ctx, token, messages...)
}

func (mm *metricsMiddleware) RestoreSenMLMessages(ctx context.Context, token string, messages ...readers.Message) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "restore_senml_messages").Add(1)
		mm.latency.With("method", "restore_senml_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.RestoreSenMLMessages(ctx, token, messages...)
}

func (mm *metricsMiddleware) DeleteJSONMessages(ctx context.Context, token string, rpm readers.JSONPageMetadata) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "delete_json_messages").Add(1)
		mm.latency.With("method", "delete_json_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.DeleteJSONMessages(ctx, token, rpm)
}

func (mm *metricsMiddleware) DeleteSenMLMessages(ctx context.Context, token string, rpm readers.SenMLPageMetadata) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "delete_senml_messages").Add(1)
		mm.latency.With("method", "delete_senml_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.DeleteSenMLMessages(ctx, token, rpm)
}
