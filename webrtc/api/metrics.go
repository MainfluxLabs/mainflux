// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test
// +build !test

package api

import (
	"context"
	"time"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/webrtc"
	"github.com/go-kit/kit/metrics"
)

var _ webrtc.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     webrtc.Service
}

// MetricsMiddleware instruments core service by tracking request count and
// latency.
func MetricsMiddleware(svc webrtc.Service, counter metrics.Counter, latency metrics.Histogram) webrtc.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) ViewICEServers(ctx context.Context, token string) ([]webrtc.ICEServer, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_ice_servers").Add(1)
		ms.latency.With("method", "view_ice_servers").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewICEServers(ctx, token)
}

func (ms *metricsMiddleware) View(ctx context.Context, token, thingID, offer string) (webrtc.Session, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view").Add(1)
		ms.latency.With("method", "view").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.View(ctx, token, thingID, offer)
}

func (ms *metricsMiddleware) AddViewerCandidate(ctx context.Context, token, sessionID string, candidate webrtc.ICECandidate) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "add_viewer_candidate").Add(1)
		ms.latency.With("method", "add_viewer_candidate").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.AddViewerCandidate(ctx, token, sessionID, candidate)
}

func (ms *metricsMiddleware) ConsumeMessage(subject string, msg protomfx.Message) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "consume_message").Add(1)
		ms.latency.With("method", "consume_message").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ConsumeMessage(subject, msg)
}
