package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/go-kit/kit/metrics"
)

var _ alarms.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     alarms.Service
}

// MetricsMiddleware instruments core service by tracking request count and latency.
func MetricsMiddleware(svc alarms.Service, counter metrics.Counter, latency metrics.Histogram) alarms.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) ListAlarmsByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (alarms.AlarmsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_alarms_by_group").Add(1)
		ms.latency.With("method", "list_alarms_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListAlarmsByGroup(ctx, token, groupID, pm)
}

func (ms *metricsMiddleware) ListAlarmsByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (alarms.AlarmsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_alarms_by_thing").Add(1)
		ms.latency.With("method", "list_alarms_by_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListAlarmsByThing(ctx, token, thingID, pm)
}

func (ms *metricsMiddleware) ViewAlarm(ctx context.Context, token, id string) (alarms.Alarm, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_alarm").Add(1)
		ms.latency.With("method", "view_alarm").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewAlarm(ctx, token, id)
}

func (ms *metricsMiddleware) RemoveAlarms(ctx context.Context, token string, id ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_alarms").Add(1)
		ms.latency.With("method", "remove_alarms").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveAlarms(ctx, token, id...)
}

func (ms *metricsMiddleware) Consume(message interface{}) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "consume").Add(1)
		ms.latency.With("method", "consume").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Consume(message)
}
