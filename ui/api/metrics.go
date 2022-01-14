// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test
// +build !test

package api

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux/pkg/messaging"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/ui"
)

var _ ui.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     ui.Service
}

// MetricsMiddleware instruments adapter by tracking request count and latency.
func MetricsMiddleware(svc ui.Service, counter metrics.Counter, latency metrics.Histogram) ui.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (mm *metricsMiddleware) Index(ctx context.Context, token string) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "index").Add(1)
		mm.latency.With("method", "index").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Index(ctx, token)
}

func (mm *metricsMiddleware) CreateThings(ctx context.Context, token string, things ...sdk.Thing) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "create_things").Add(1)
		mm.latency.With("method", "create_things").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.CreateThings(ctx, token, things...)
}

func (mm *metricsMiddleware) ViewThing(ctx context.Context, token, id string) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "view_thing").Add(1)
		mm.latency.With("method", "view_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ViewThing(ctx, token, id)
}

func (mm *metricsMiddleware) UpdateThing(ctx context.Context, token, id string, thing sdk.Thing) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "update_thing").Add(1)
		mm.latency.With("method", "update_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.UpdateThing(ctx, token, id, thing)
}

func (mm *metricsMiddleware) ListThings(ctx context.Context, token string) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "list_things").Add(1)
		mm.latency.With("method", "list_things").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ListThings(ctx, token)
}

func (mm *metricsMiddleware) RemoveThing(ctx context.Context, token, id string) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "remove_thing").Add(1)
		mm.latency.With("method", "remove_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.RemoveThing(ctx, token, id)
}

func (mm *metricsMiddleware) CreateChannels(ctx context.Context, token string, channels ...sdk.Channel) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "create_channels").Add(1)
		mm.latency.With("method", "create_channels").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.CreateChannels(ctx, token, channels...)
}

func (mm *metricsMiddleware) ViewChannel(ctx context.Context, token, id string) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "view_channel").Add(1)
		mm.latency.With("method", "view_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ViewChannel(ctx, token, id)
}

func (mm *metricsMiddleware) UpdateChannel(ctx context.Context, token, id string, channel sdk.Channel) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "update_channel").Add(1)
		mm.latency.With("method", "update_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.UpdateChannel(ctx, token, id, channel)
}

func (mm *metricsMiddleware) ListChannels(ctx context.Context, token string) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "list_channels").Add(1)
		mm.latency.With("method", "list_channels").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ListChannels(ctx, token)
}

func (mm *metricsMiddleware) RemoveChannel(ctx context.Context, token, id string) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "remove_channel").Add(1)
		mm.latency.With("method", "remove_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.RemoveChannel(ctx, token, id)
}

func (mm *metricsMiddleware) Connect(ctx context.Context, token string, chIDs, thIDs []string) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "connect").Add(1)
		mm.latency.With("method", "connect").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Connect(ctx, token, chIDs, thIDs)
}

func (mm *metricsMiddleware) ListThingConnections(ctx context.Context, token, id string) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "view_connections").Add(1)
		mm.latency.With("method", "view_connections").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ListThingConnections(ctx, token, id)
}

func (mm *metricsMiddleware) ListChannelConnections(ctx context.Context, token, id string) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "view_connections").Add(1)
		mm.latency.With("method", "view_connections").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ListChannelConnections(ctx, token, id)
}

func (mm *metricsMiddleware) DisconnectThing(ctx context.Context, token string, chIDs, thIDs []string) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "disconnect").Add(1)
		mm.latency.With("method", "disconnect").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.DisconnectThing(ctx, token, chIDs, thIDs)
}

func (mm *metricsMiddleware) DisconnectChannel(ctx context.Context, token string, chIDs, thIDs []string) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "disconnect_channel").Add(1)
		mm.latency.With("method", "disconnect_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.DisconnectChannel(ctx, token, chIDs, thIDs)
}

func (mm *metricsMiddleware) CreateGroups(ctx context.Context, token string, groups ...sdk.Group) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "create_groups").Add(1)
		mm.latency.With("method", "create_groups").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.CreateGroups(ctx, token, groups...)
}

func (mm *metricsMiddleware) ListGroups(ctx context.Context, token string) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "list_groups").Add(1)
		mm.latency.With("method", "list_groups").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ListGroups(ctx, token)
}

func (mm *metricsMiddleware) ViewGroup(ctx context.Context, token, id string) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "view_group").Add(1)
		mm.latency.With("method", "view_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ViewGroup(ctx, token, id)
}

func (mm *metricsMiddleware) Assign(ctx context.Context, token, groupID, groupType string, memberIDs ...string) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "assign").Add(1)
		mm.latency.With("method", "assign").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Assign(ctx, token, groupID, groupType, memberIDs...)
}

func (mm *metricsMiddleware) Unassign(ctx context.Context, token, groupID string, memberIDs ...string) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "unassign").Add(1)
		mm.latency.With("method", "unassign").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Unassign(ctx, token, groupID, memberIDs...)
}

func (mm *metricsMiddleware) UpdateGroup(ctx context.Context, token, id string, group sdk.Group) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "update_group").Add(1)
		mm.latency.With("method", "update_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.UpdateGroup(ctx, token, id, group)
}

func (mm *metricsMiddleware) RemoveGroup(ctx context.Context, token, id string) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "remove_group").Add(1)
		mm.latency.With("method", "remove_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.RemoveGroup(ctx, token, id)
}

func (mm *metricsMiddleware) Publish(ctx context.Context, thingKey string, msg messaging.Message) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "publish").Add(1)
		mm.latency.With("method", "publish").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Publish(ctx, thingKey, msg)
}

func (mm *metricsMiddleware) SendMessage(ctx context.Context, token string) (b []byte, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "send_message").Add(1)
		mm.latency.With("method", "send_message").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.SendMessage(ctx, token)
}
