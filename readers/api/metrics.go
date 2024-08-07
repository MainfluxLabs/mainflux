// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/go-kit/kit/metrics"
)

var _ readers.MessageRepository = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     readers.MessageRepository
}

// MetricsMiddleware instruments core service by tracking request count and latency.
func MetricsMiddleware(svc readers.MessageRepository, counter metrics.Counter, latency metrics.Histogram) readers.MessageRepository {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (mm *metricsMiddleware) ListAllMessages(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "list_all_messages").Add(1)
		mm.latency.With("method", "list_all_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ListAllMessages(rpm)
}

func (mm *metricsMiddleware) Backup(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "backup").Add(1)
		mm.latency.With("method", "backup").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Backup(rpm)
}

func (mm *metricsMiddleware) Restore(ctx context.Context, messages ...senml.Message) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "restore").Add(1)
		mm.latency.With("method", "restore").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Restore(ctx, messages...)
}
