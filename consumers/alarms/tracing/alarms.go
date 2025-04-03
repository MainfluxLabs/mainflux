package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/opentracing/opentracing-go"
)

var (
	_ alarms.AlarmRepository = (*alarmRepositoryMiddleware)(nil)
)

type alarmRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   alarms.AlarmRepository
}

// AlarmRepositoryMiddleware tracks request and their latency, and adds spans
// to context.
func AlarmRepositoryMiddleware(tracer opentracing.Tracer, repo alarms.AlarmRepository) alarms.AlarmRepository {
	return alarmRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (arm alarmRepositoryMiddleware) Save(ctx context.Context, ams ...alarms.Alarm) error {
	span := createSpan(ctx, arm.tracer, "save_alarms")
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return arm.repo.Save(ctx, ams...)
}

func (arm alarmRepositoryMiddleware) RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (alarms.AlarmsPage, error) {
	span := createSpan(ctx, arm.tracer, "retrieve_by_group")
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return arm.repo.RetrieveByGroup(ctx, groupID, pm)
}

func (arm alarmRepositoryMiddleware) RetrieveByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (alarms.AlarmsPage, error) {
	span := createSpan(ctx, arm.tracer, "retrieve_by_thing")
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return arm.repo.RetrieveByThing(ctx, thingID, pm)
}

func (arm alarmRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (alarms.Alarm, error) {
	span := createSpan(ctx, arm.tracer, "retrieve_alarm_by_id")
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return arm.repo.RetrieveByID(ctx, id)
}

func (arm alarmRepositoryMiddleware) Remove(ctx context.Context, ids ...string) error {
	span := createSpan(ctx, arm.tracer, "remove_alarms")
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return arm.repo.Remove(ctx, ids...)
}

func createSpan(ctx context.Context, tracer opentracing.Tracer, opName string) opentracing.Span {
	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		return tracer.StartSpan(
			opName,
			opentracing.ChildOf(parentSpan.Context()),
		)
	}
	return tracer.StartSpan(opName)
}
