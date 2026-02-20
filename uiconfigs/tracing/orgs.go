package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/uiconfigs"
	"github.com/opentracing/opentracing-go"
)

const (
	saveOrgConfigs          = "save_org_configs"
	retrieveOrgConfigsByOrg = "retrieve_org_configs_by_org"
	retrieveAllOrgConfigs   = "retrieve_all_org_configs"
	updateOrgConfig         = "update_org_configs"
	removeOrgConfig         = "remove_org_config"
	backupAllOrgConfigs     = "backup_all_org_configs"
)

var (
	_ uiconfigs.OrgConfigRepository = (*orgConfigRepositoryMiddleware)(nil)
)

type orgConfigRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   uiconfigs.OrgConfigRepository
}

// OrgConfigRepositoryMiddleware tracks request and their latency, and adds spans to context.
func OrgConfigRepositoryMiddleware(tracer opentracing.Tracer, repo uiconfigs.OrgConfigRepository) uiconfigs.OrgConfigRepository {
	return orgConfigRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (uirm orgConfigRepositoryMiddleware) Save(ctx context.Context, d uiconfigs.OrgConfig) (uiconfigs.OrgConfig, error) {
	span := dbutil.CreateSpan(ctx, uirm.tracer, saveOrgConfigs)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return uirm.repo.Save(ctx, d)
}

func (uirm orgConfigRepositoryMiddleware) RetrieveByOrg(ctx context.Context, orgID string) (uiconfigs.OrgConfig, error) {
	span := dbutil.CreateSpan(ctx, uirm.tracer, retrieveOrgConfigsByOrg)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return uirm.repo.RetrieveByOrg(ctx, orgID)
}

func (uirm orgConfigRepositoryMiddleware) RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (uiconfigs.OrgConfigPage, error) {
	span := dbutil.CreateSpan(ctx, uirm.tracer, retrieveAllOrgConfigs)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return uirm.repo.RetrieveAll(ctx, pm)
}

func (uirm orgConfigRepositoryMiddleware) Update(ctx context.Context, d uiconfigs.OrgConfig) (uiconfigs.OrgConfig, error) {
	span := dbutil.CreateSpan(ctx, uirm.tracer, updateOrgConfig)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return uirm.repo.Update(ctx, d)
}

func (uirm orgConfigRepositoryMiddleware) Remove(ctx context.Context, orgID string) error {
	span := dbutil.CreateSpan(ctx, uirm.tracer, removeOrgConfig)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return uirm.repo.Remove(ctx, orgID)
}

func (uirm orgConfigRepositoryMiddleware) BackupAll(ctx context.Context) (uiconfigs.OrgConfigBackup, error) {
	span := dbutil.CreateSpan(ctx, uirm.tracer, backupAllOrgConfigs)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return uirm.repo.BackupAll(ctx)
}
