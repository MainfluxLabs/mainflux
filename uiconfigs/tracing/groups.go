package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/uiconfigs"
	"github.com/opentracing/opentracing-go"
)

const (
	saveGroupConfigs            = "save_group_configs"
	retrieveGroupConfigsByGroup = "retrieve_group_configs_by_group"
	retrieveAllGroupConfigs     = "retrieve_all_group_configs"
	updateGroupConfig           = "update_group_configs"
	removeGroupConfig           = "remove_group_config"
	backupAllGroupConfigs       = "backup_all_group_configs"
)

var (
	_ uiconfigs.GroupConfigRepository = (*groupConfigRepositoryMiddleware)(nil)
)

type groupConfigRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   uiconfigs.GroupConfigRepository
}

// GroupConfigRepositoryMiddleware tracks request and their latency, and adds spans to context.
func GroupConfigRepositoryMiddleware(tracer opentracing.Tracer, repo uiconfigs.GroupConfigRepository) uiconfigs.GroupConfigRepository {
	return groupConfigRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (uirm groupConfigRepositoryMiddleware) Save(ctx context.Context, d uiconfigs.GroupConfig) (uiconfigs.GroupConfig, error) {
	span := dbutil.CreateSpan(ctx, uirm.tracer, saveGroupConfigs)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return uirm.repo.Save(ctx, d)
}

func (uirm groupConfigRepositoryMiddleware) RetrieveByGroup(ctx context.Context, groupID string) (uiconfigs.GroupConfig, error) {
	span := dbutil.CreateSpan(ctx, uirm.tracer, retrieveGroupConfigsByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return uirm.repo.RetrieveByGroup(ctx, groupID)
}

func (uirm groupConfigRepositoryMiddleware) RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (uiconfigs.GroupConfigPage, error) {
	span := dbutil.CreateSpan(ctx, uirm.tracer, retrieveAllGroupConfigs)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return uirm.repo.RetrieveAll(ctx, pm)
}

func (uirm groupConfigRepositoryMiddleware) Update(ctx context.Context, d uiconfigs.GroupConfig) (uiconfigs.GroupConfig, error) {
	span := dbutil.CreateSpan(ctx, uirm.tracer, updateGroupConfig)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return uirm.repo.Update(ctx, d)
}

func (uirm groupConfigRepositoryMiddleware) Remove(ctx context.Context, groupID string) error {
	span := dbutil.CreateSpan(ctx, uirm.tracer, removeGroupConfig)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return uirm.repo.Remove(ctx, groupID)
}

func (uirm groupConfigRepositoryMiddleware) BackupAll(ctx context.Context) (uiconfigs.GroupConfigBackup, error) {
	span := dbutil.CreateSpan(ctx, uirm.tracer, backupAllGroupConfigs)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return uirm.repo.BackupAll(ctx)
}
