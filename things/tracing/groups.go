package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/opentracing/opentracing-go"
)

const (
	saveGroup                       = "save_group"
	updateGroup                     = "update_group"
	removeGroup                     = "remove_group"
	removeGroupsByOrg               = "remove_groups_by_org"
	retrieveAllGroups               = "retrieve_all_groups"
	backupAllGroups                 = "backup_all_groups"
	retrieveGroupByID               = "retrieve_group_by_id"
	retrieveGroupByIDs              = "retrieve_group_by_ids"
	retrieveGroupIDsByOrg           = "retrieve_group_ids_by_org"
	saveGroupMembership             = "save_group_membership"
	retrieveRole                    = "retrieve_role"
	removeGroupMembership           = "remove_group_membership"
	retrieveGroupIDsByMember        = "retrieve_group_ids_by_member"
	retrieveGroupIDsByOrgMembership = "retrieve_group_ids_by_org_membership"
)

var _ things.GroupRepository = (*groupRepositoryMiddleware)(nil)

type groupRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   things.GroupRepository
}

// GroupRepositoryMiddleware tracks request and their latency, and adds spans to context.
func GroupRepositoryMiddleware(tracer opentracing.Tracer, repo things.GroupRepository) things.GroupRepository {
	return groupRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (grm groupRepositoryMiddleware) Save(ctx context.Context, grs ...things.Group) ([]things.Group, error) {
	span := dbutil.CreateSpan(ctx, grm.tracer, saveGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Save(ctx, grs...)
}

func (grm groupRepositoryMiddleware) Update(ctx context.Context, g things.Group) (things.Group, error) {
	span := dbutil.CreateSpan(ctx, grm.tracer, updateGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Update(ctx, g)
}

func (grm groupRepositoryMiddleware) Remove(ctx context.Context, groupIDs ...string) error {
	span := dbutil.CreateSpan(ctx, grm.tracer, removeGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Remove(ctx, groupIDs...)
}

func (grm groupRepositoryMiddleware) RemoveByOrg(ctx context.Context, orgID string) error {
	span := dbutil.CreateSpan(ctx, grm.tracer, removeGroupsByOrg)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RemoveByOrg(ctx, orgID)
}

func (grm groupRepositoryMiddleware) BackupAll(ctx context.Context) ([]things.Group, error) {
	span := dbutil.CreateSpan(ctx, grm.tracer, backupAllGroups)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.BackupAll(ctx)
}

func (grm groupRepositoryMiddleware) RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (things.GroupPage, error) {
	span := dbutil.CreateSpan(ctx, grm.tracer, retrieveAllGroups)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveAll(ctx, pm)
}

func (grm groupRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (things.Group, error) {
	span := dbutil.CreateSpan(ctx, grm.tracer, retrieveGroupByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveByID(ctx, id)
}
func (grm groupRepositoryMiddleware) RetrieveByIDs(ctx context.Context, groupIDs []string, pm apiutil.PageMetadata) (things.GroupPage, error) {
	span := dbutil.CreateSpan(ctx, grm.tracer, retrieveGroupByIDs)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveByIDs(ctx, groupIDs, pm)
}

func (grm groupRepositoryMiddleware) RetrieveIDsByOrg(ctx context.Context, orgID string) ([]string, error) {
	span := dbutil.CreateSpan(ctx, grm.tracer, retrieveGroupIDsByOrg)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveIDsByOrg(ctx, orgID)
}

func (grm groupRepositoryMiddleware) RetrieveIDsByOrgMembership(ctx context.Context, orgID, memberID string) ([]string, error) {
	span := dbutil.CreateSpan(ctx, grm.tracer, retrieveGroupIDsByOrgMembership)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveIDsByOrgMembership(ctx, orgID, memberID)
}

type groupCacheMiddleware struct {
	tracer opentracing.Tracer
	cache  things.GroupCache
}

// GroupCacheMiddleware tracks request and their latency, and adds spans
// to context.
func GroupCacheMiddleware(tracer opentracing.Tracer, cache things.GroupCache) things.GroupCache {
	return groupCacheMiddleware{
		tracer: tracer,
		cache:  cache,
	}
}

func (gcm groupCacheMiddleware) RemoveGroupEntities(ctx context.Context, groupID string) error {
	span := dbutil.CreateSpan(ctx, gcm.tracer, removeGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return gcm.cache.RemoveGroupEntities(ctx, groupID)
}

func (gcm groupCacheMiddleware) SaveGroupMembership(ctx context.Context, groupID, memberID, role string) error {
	span := dbutil.CreateSpan(ctx, gcm.tracer, saveGroupMembership)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return gcm.cache.SaveGroupMembership(ctx, groupID, memberID, role)
}

func (gcm groupCacheMiddleware) ViewRole(ctx context.Context, groupID, memberID string) (string, error) {
	span := dbutil.CreateSpan(ctx, gcm.tracer, retrieveRole)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return gcm.cache.ViewRole(ctx, groupID, memberID)
}

func (gcm groupCacheMiddleware) RemoveGroupMembership(ctx context.Context, groupID, memberID string) error {
	span := dbutil.CreateSpan(ctx, gcm.tracer, removeGroupMembership)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return gcm.cache.RemoveGroupMembership(ctx, groupID, memberID)
}

func (gcm groupCacheMiddleware) RetrieveGroupIDsByMember(ctx context.Context, memberID string) ([]string, error) {
	span := dbutil.CreateSpan(ctx, gcm.tracer, retrieveGroupIDsByMember)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return gcm.cache.RetrieveGroupIDsByMember(ctx, memberID)
}
