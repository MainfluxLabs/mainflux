// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"github.com/MainfluxLabs/mainflux/audit"
	"github.com/go-kit/kit/metrics"
)

var _ audit.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     audit.Service
}

func MetricsMiddleware(svc audit.Service, counter metrics.Counter, latency metrics.Histogram) audit.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}
