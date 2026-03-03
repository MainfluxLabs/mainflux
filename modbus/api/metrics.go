// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/modbus"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/go-kit/kit/metrics"
)

var _ modbus.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     modbus.Service
}

// MetricsMiddleware instruments core service by tracking request count and
// latency.
func MetricsMiddleware(svc modbus.Service, counter metrics.Counter, latency metrics.Histogram) modbus.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) CreateClients(ctx context.Context, token, thingID string, clients ...modbus.Client) (response []modbus.Client, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_clients").Add(1)
		ms.latency.With("method", "create_clients").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateClients(ctx, token, thingID, clients...)
}

func (ms *metricsMiddleware) ListClientsByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (modbus.ClientsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_clients_by_thing").Add(1)
		ms.latency.With("method", "list_clients_by_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListClientsByThing(ctx, token, thingID, pm)
}

func (ms *metricsMiddleware) ListClientsByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (modbus.ClientsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_clients_by_group").Add(1)
		ms.latency.With("method", "list_clients_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListClientsByGroup(ctx, token, groupID, pm)
}

func (ms *metricsMiddleware) ViewClient(ctx context.Context, token, id string) (modbus.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_client").Add(1)
		ms.latency.With("method", "view_client").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewClient(ctx, token, id)
}

func (ms *metricsMiddleware) UpdateClient(ctx context.Context, token string, client modbus.Client) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_client").Add(1)
		ms.latency.With("method", "update_client").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateClient(ctx, token, client)
}

func (ms *metricsMiddleware) RemoveClients(ctx context.Context, token string, id ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_clients").Add(1)
		ms.latency.With("method", "remove_clients").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveClients(ctx, token, id...)
}

func (ms *metricsMiddleware) RemoveClientsByThing(ctx context.Context, thingID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_clients_by_thing").Add(1)
		ms.latency.With("method", "remove_clients_by_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveClientsByThing(ctx, thingID)
}

func (ms *metricsMiddleware) RemoveClientsByGroup(ctx context.Context, groupID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_clients_by_group").Add(1)
		ms.latency.With("method", "remove_clients_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveClientsByGroup(ctx, groupID)
}

func (ms *metricsMiddleware) RescheduleTasks(ctx context.Context, profileID string, config map[string]any) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "reschedule_tasks").Add(1)
		ms.latency.With("method", "reschedule_tasks").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RescheduleTasks(ctx, profileID, config)
}

func (ms *metricsMiddleware) LoadAndScheduleTasks(ctx context.Context) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "load_and_schedule_tasks").Add(1)
		ms.latency.With("method", "load_and_schedule_tasks").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.LoadAndScheduleTasks(ctx)
}
