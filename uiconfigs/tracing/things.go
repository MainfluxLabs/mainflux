package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/uiconfigs"
	"github.com/opentracing/opentracing-go"
)

const (
	saveThingConfigs            = "save_thing_configs"
	retrieveThingConfigsByThing = "retrieve_thing_configs"
	retrieveAllThingConfigs     = "retrieve_all_thing_configs"
	updateThingConfig           = "update_thing_configs"
	removeThingConfig           = "remove_thing_config"
	backupAllThingConfigs       = "backup_all_thing_configs"
	removeThingConfigByGroup    = "remove_thing_config_by_group"
)

var (
	_ uiconfigs.ThingConfigRepository = (*thingConfigRepositoryMiddleware)(nil)
)

type thingConfigRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   uiconfigs.ThingConfigRepository
}

// ThingConfigRepositoryMiddleware tracks request and their latency, and adds spans to context.
func ThingConfigRepositoryMiddleware(tracer opentracing.Tracer, repo uiconfigs.ThingConfigRepository) uiconfigs.ThingConfigRepository {
	return thingConfigRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (uirm thingConfigRepositoryMiddleware) Save(ctx context.Context, d uiconfigs.ThingConfig) (uiconfigs.ThingConfig, error) {
	span := dbutil.CreateSpan(ctx, uirm.tracer, saveThingConfigs)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return uirm.repo.Save(ctx, d)
}

func (uirm thingConfigRepositoryMiddleware) RetrieveByThing(ctx context.Context, thingID string) (uiconfigs.ThingConfig, error) {
	span := dbutil.CreateSpan(ctx, uirm.tracer, retrieveThingConfigsByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return uirm.repo.RetrieveByThing(ctx, thingID)
}

func (uirm thingConfigRepositoryMiddleware) RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (uiconfigs.ThingConfigPage, error) {
	span := dbutil.CreateSpan(ctx, uirm.tracer, retrieveAllThingConfigs)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return uirm.repo.RetrieveAll(ctx, pm)
}

func (uirm thingConfigRepositoryMiddleware) Update(ctx context.Context, d uiconfigs.ThingConfig) (uiconfigs.ThingConfig, error) {
	span := dbutil.CreateSpan(ctx, uirm.tracer, updateThingConfig)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return uirm.repo.Update(ctx, d)
}

func (uirm thingConfigRepositoryMiddleware) Remove(ctx context.Context, thingID string) error {
	span := dbutil.CreateSpan(ctx, uirm.tracer, removeThingConfig)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return uirm.repo.Remove(ctx, thingID)
}

func (uirm thingConfigRepositoryMiddleware) RemoveByGroup(ctx context.Context, groupID string) error {
	span := dbutil.CreateSpan(ctx, uirm.tracer, removeThingConfigByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return uirm.repo.RemoveByGroup(ctx, groupID)
}

func (uirm thingConfigRepositoryMiddleware) BackupAll(ctx context.Context) (uiconfigs.ThingConfigBackup, error) {
	span := dbutil.CreateSpan(ctx, uirm.tracer, backupAllThingConfigs)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return uirm.repo.BackupAll(ctx)
}
