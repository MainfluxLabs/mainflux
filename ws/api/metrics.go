// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/ws"
	"github.com/go-kit/kit/metrics"
)

var _ ws.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     ws.Service
}

// MetricsMiddleware instruments adapter by tracking request count and latency
func MetricsMiddleware(svc ws.Service, counter metrics.Counter, latency metrics.Histogram) ws.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (mm *metricsMiddleware) Publish(ctx context.Context, key domain.ThingKey, msg protomfx.Message) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "publish").Add(1)
		mm.latency.With("method", "publish").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Publish(ctx, key, msg)
}

func (mm *metricsMiddleware) Subscribe(ctx context.Context, key domain.ThingKey, subtopic string, c *ws.Client) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "subscribe").Add(1)
		mm.latency.With("method", "subscribe").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Subscribe(ctx, key, subtopic, c)
}

func (mm *metricsMiddleware) Unsubscribe(ctx context.Context, key domain.ThingKey, subtopic string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "unsubscribe").Add(1)
		mm.latency.With("method", "unsubscribe").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Unsubscribe(ctx, key, subtopic)
}

func (mm *metricsMiddleware) SendCommandToThing(ctx context.Context, token, thingID string, msg protomfx.Message) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "send_command_to_thing").Add(1)
		mm.latency.With("method", "send_command_to_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.SendCommandToThing(ctx, token, thingID, msg)
}

func (mm *metricsMiddleware) SendCommandToThingByKey(ctx context.Context, key domain.ThingKey, thingID string, msg protomfx.Message) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "send_command_to_thing_by_key").Add(1)
		mm.latency.With("method", "send_command_to_thing_by_key").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.SendCommandToThingByKey(ctx, key, thingID, msg)
}

func (mm *metricsMiddleware) SendCommandToGroup(ctx context.Context, token, groupID string, msg protomfx.Message) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "send_command_to_group").Add(1)
		mm.latency.With("method", "send_command_to_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.SendCommandToGroup(ctx, token, groupID, msg)
}

func (mm *metricsMiddleware) SendCommandToGroupByKey(ctx context.Context, key domain.ThingKey, groupID string, msg protomfx.Message) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "send_command_to_group_by_key").Add(1)
		mm.latency.With("method", "send_command_to_group_by_key").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.SendCommandToGroupByKey(ctx, key, groupID, msg)
}
