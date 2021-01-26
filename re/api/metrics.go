//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

// +build !test

package api

import (
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

func (ms *metricsMiddleware) Info() (info re.Info, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "info").Add(1)
		ms.latency.With("method", "info").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Info()
}

func (ms *metricsMiddleware) CreateStream(sql string) (result string, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_stream").Add(1)
		ms.latency.With("method", "create_stream").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateStream(sql)
}

func (ms *metricsMiddleware) List() (streams []string, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "info").Add(1)
		ms.latency.With("method", "info").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.List()
}

func (ms *metricsMiddleware) View(id string) (stream re.Stream, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "info").Add(1)
		ms.latency.With("method", "info").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.View(id)
}
