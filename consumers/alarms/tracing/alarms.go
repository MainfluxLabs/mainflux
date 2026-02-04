package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/opentracing/opentracing-go"
)

const (
	saveAlarms             = "save_alarms"
	retrieveAlarmsByGroup  = "retrieve_alarms_by_group"
	retrieveAlarmsByThing  = "retrieve_alarms_by_thing"
	retrieveAlarmsByGroups = "retrieve_alarms_by_groups"
	retrieveAlarmByID      = "retrieve_alarm_by_id"
	removeAlarms           = "remove_alarms"
	removeAlarmsByThing    = "remove_alarms_by_thing"
	removeAlarmsByGroup    = "remove_alarms_by_group"
	exportAlarmsByThing    = "export_alarms_by_thing"
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
	span := dbutil.CreateSpan(ctx, arm.tracer, saveAlarms)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return arm.repo.Save(ctx, ams...)
}

func (arm alarmRepositoryMiddleware) RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (alarms.AlarmsPage, error) {
	span := dbutil.CreateSpan(ctx, arm.tracer, retrieveAlarmsByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return arm.repo.RetrieveByGroup(ctx, groupID, pm)
}

func (arm alarmRepositoryMiddleware) RetrieveByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (alarms.AlarmsPage, error) {
	span := dbutil.CreateSpan(ctx, arm.tracer, retrieveAlarmsByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return arm.repo.RetrieveByThing(ctx, thingID, pm)
}

func (arm alarmRepositoryMiddleware) RetrieveByGroups(ctx context.Context, ids []string, pm apiutil.PageMetadata) (alarms.AlarmsPage, error) {
	span := dbutil.CreateSpan(ctx, arm.tracer, retrieveAlarmsByGroups)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return arm.repo.RetrieveByGroups(ctx, ids, pm)
}

func (arm alarmRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (alarms.Alarm, error) {
	span := dbutil.CreateSpan(ctx, arm.tracer, retrieveAlarmByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return arm.repo.RetrieveByID(ctx, id)
}

func (arm alarmRepositoryMiddleware) Remove(ctx context.Context, ids ...string) error {
	span := dbutil.CreateSpan(ctx, arm.tracer, removeAlarms)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return arm.repo.Remove(ctx, ids...)
}

func (arm alarmRepositoryMiddleware) RemoveByThing(ctx context.Context, thingID string) error {
	span := dbutil.CreateSpan(ctx, arm.tracer, removeAlarmsByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return arm.repo.RemoveByThing(ctx, thingID)
}

func (arm alarmRepositoryMiddleware) RemoveByGroup(ctx context.Context, groupID string) error {
	span := dbutil.CreateSpan(ctx, arm.tracer, removeAlarmsByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return arm.repo.RemoveByGroup(ctx, groupID)
}

func (arm alarmRepositoryMiddleware) ExportByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (alarms.AlarmsPage, error) {
	span := dbutil.CreateSpan(ctx, arm.tracer, exportAlarmsByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return arm.repo.ExportByThing(ctx, thingID, pm)
}
