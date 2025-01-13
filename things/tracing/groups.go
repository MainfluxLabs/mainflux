package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/things"
	"github.com/opentracing/opentracing-go"
)

const (
	saveGroupOp                = "save_group"
	saveOrgIDByGroupIDOp       = "save_org_id_by_group_id"
	updateGroupOp              = "update_group"
	removeGroupOp              = "remove_group"
	retrieveAllOp              = "retrieve_all"
	retrieveGroupByIDOp        = "retrieve_group_by_id"
	retrieveGroupByIDsOp       = "retrieve_group_by_ids"
	retrieveOrgIDByGroupIDOp   = "retrieve_org_id_by_group_id"
	saveRoleOp                 = "save_role"
	retrieveRoleOp             = "retrieve_role"
	removeRoleOp               = "remove_role"
	retrieveGroupIDsByMemberOp = "retrieve_group_ids_by_member"
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

func (grm groupRepositoryMiddleware) Save(ctx context.Context, g things.Group) (things.Group, error) {
	span := createSpan(ctx, grm.tracer, saveGroupOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Save(ctx, g)
}

func (grm groupRepositoryMiddleware) Update(ctx context.Context, g things.Group) (things.Group, error) {
	span := createSpan(ctx, grm.tracer, updateGroupOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Update(ctx, g)
}

func (grm groupRepositoryMiddleware) Remove(ctx context.Context, groupIDs ...string) error {
	span := createSpan(ctx, grm.tracer, removeGroupOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Remove(ctx, groupIDs...)
}

func (grm groupRepositoryMiddleware) RetrieveAll(ctx context.Context) ([]things.Group, error) {
	span := createSpan(ctx, grm.tracer, retrieveAllOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveAll(ctx)
}

func (grm groupRepositoryMiddleware) RetrieveByAdmin(ctx context.Context, orgID string, pm things.PageMetadata) (things.GroupPage, error) {
	span := createSpan(ctx, grm.tracer, retrieveAllOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveByAdmin(ctx, orgID, pm)
}

func (grm groupRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (things.Group, error) {
	span := createSpan(ctx, grm.tracer, retrieveGroupByIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveByID(ctx, id)
}
func (grm groupRepositoryMiddleware) RetrieveByIDs(ctx context.Context, groupIDs []string, pm things.PageMetadata) (things.GroupPage, error) {
	span := createSpan(ctx, grm.tracer, retrieveGroupByIDsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveByIDs(ctx, groupIDs, pm)
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

func (gcm groupCacheMiddleware) SaveOrgID(ctx context.Context, groupID, orgID string) error {
	span := createSpan(ctx, gcm.tracer, saveOrgIDByGroupIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return gcm.cache.SaveOrgID(ctx, groupID, orgID)
}

func (gcm groupCacheMiddleware) OrgID(ctx context.Context, groupID string) (string, error) {
	span := createSpan(ctx, gcm.tracer, retrieveOrgIDByGroupIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return gcm.cache.OrgID(ctx, groupID)
}

func (gcm groupCacheMiddleware) Remove(ctx context.Context, groupID string) error {
	span := createSpan(ctx, gcm.tracer, removeGroupOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return gcm.cache.Remove(ctx, groupID)
}

func (gcm groupCacheMiddleware) SaveRole(ctx context.Context, groupID, memberID, role string) error {
	span := createSpan(ctx, gcm.tracer, saveRoleOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return gcm.cache.SaveRole(ctx, groupID, memberID, role)
}

func (gcm groupCacheMiddleware) Role(ctx context.Context, groupID, memberID string) (string, error) {
	span := createSpan(ctx, gcm.tracer, retrieveRoleOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return gcm.cache.Role(ctx, groupID, memberID)
}

func (gcm groupCacheMiddleware) RemoveRole(ctx context.Context, groupID, memberID string) error {
	span := createSpan(ctx, gcm.tracer, removeRoleOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return gcm.cache.RemoveRole(ctx, groupID, memberID)
}

func (gcm groupCacheMiddleware) GroupIDsByMember(ctx context.Context, memberID string) ([]string, error) {
	span := createSpan(ctx, gcm.tracer, retrieveGroupIDsByMemberOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return gcm.cache.GroupIDsByMember(ctx, memberID)
}
