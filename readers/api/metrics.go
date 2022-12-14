// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"time"

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

func (mm *metricsMiddleware) ListChannelMessages(chanID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "list_channel_messages").Add(1)
		mm.latency.With("method", "list_channel_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ListChannelMessages(chanID, rpm)
}

func (mm *metricsMiddleware) ListAllMessages(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "list_all_messages").Add(1)
		mm.latency.With("method", "list_all_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ListAllMessages(rpm)
}
