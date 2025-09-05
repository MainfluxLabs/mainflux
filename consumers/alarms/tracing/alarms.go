package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/opentracing/opentracing-go"
)

const (
	saveAlarms             = "save_alarms"
	retrieveAlarmsByGroup  = "retrieve_alarms_by_group"
	retrieveAlarmsByThing  = "retrieve_alarms_by_thing"
	retrieveAlarmsByGroups = "retrieve_alarms_by_groups"
	retrieveAlarmByID      = "retrieve_alarm_by_id"
	removeAlarms           = "remove_alarms"
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
	span := createSpan(ctx, arm.tracer, saveAlarms)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return arm.repo.Save(ctx, ams...)
}

func (arm alarmRepositoryMiddleware) RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (alarms.AlarmsPage, error) {
	span := createSpan(ctx, arm.tracer, retrieveAlarmsByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return arm.repo.RetrieveByGroup(ctx, groupID, pm)
}

func (arm alarmRepositoryMiddleware) RetrieveByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (alarms.AlarmsPage, error) {
	span := createSpan(ctx, arm.tracer, retrieveAlarmsByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return arm.repo.RetrieveByThing(ctx, thingID, pm)
}

func (arm alarmRepositoryMiddleware) RetrieveByGroups(ctx context.Context, ids []string, pm apiutil.PageMetadata) (alarms.AlarmsPage, error) {
	span := createSpan(ctx, arm.tracer, retrieveAlarmsByGroups)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return arm.repo.RetrieveByGroups(ctx, ids, pm)
}

func (arm alarmRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (alarms.Alarm, error) {
	span := createSpan(ctx, arm.tracer, retrieveAlarmByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return arm.repo.RetrieveByID(ctx, id)
}

func (arm alarmRepositoryMiddleware) Remove(ctx context.Context, ids ...string) error {
	span := createSpan(ctx, arm.tracer, removeAlarms)
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
