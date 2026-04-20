// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"time"

	"github.com/MainfluxLabs/mainflux/consumers"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/go-kit/kit/metrics"
)

var _ consumers.MessageConsumer = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter  metrics.Counter
	latency  metrics.Histogram
	consumer consumers.MessageConsumer
}

// MetricsMiddleware returns new message repository
// with Save method wrapped to expose metrics.
func MetricsMiddleware(consumer consumers.MessageConsumer, counter metrics.Counter, latency metrics.Histogram) consumers.MessageConsumer {
	return &metricsMiddleware{
		counter:  counter,
		latency:  latency,
		consumer: consumer,
	}
}

func (mm *metricsMiddleware) ConsumeMessage(subject string, msg protomfx.Message) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "consume_message").Add(1)
		mm.latency.With("method", "consume_message").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.consumer.ConsumeMessage(subject, msg)
}
