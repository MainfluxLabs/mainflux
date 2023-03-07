package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/things"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveGroupOp           = "save_group"
	updateGroupOp         = "update_group"
	removeGroupOp         = "remove_group"
	retrieveAllGroupsOp   = "retrieve_all_groups"
	retrieveGroupByIDOp   = "retrieve_group_by_id"
	retrieveByOwnerOp     = "retrieve_by_owner"
	retrieveMembershipsOp = "retrieve_memberships"
	retrieveMembersOp     = "retrieve_members"
	assignMemberOp        = "assign_member"
	unassignMemberOp      = "unassign_member"
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

func (grm groupRepositoryMiddleware) Remove(ctx context.Context, id string) error {
	span := createSpan(ctx, grm.tracer, removeGroupOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Remove(ctx, id)
}

func (grm groupRepositoryMiddleware) RetrieveAll(ctx context.Context) ([]things.Group, error) {
	span := createSpan(ctx, grm.tracer, retrieveAllGroupsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveAll(ctx)
}
func (grm groupRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (things.Group, error) {
	span := createSpan(ctx, grm.tracer, retrieveGroupByIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveByID(ctx, id)
}

func (grm groupRepositoryMiddleware) RetrieveByOwner(ctx context.Context, ownerID string, pm things.PageMetadata) (things.GroupPage, error) {
	span := createSpan(ctx, grm.tracer, retrieveByOwnerOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveByOwner(ctx, ownerID, pm)
}

func (grm groupRepositoryMiddleware) RetrieveMemberships(ctx context.Context, memberID string, pm things.PageMetadata) (things.GroupPage, error) {
	span := createSpan(ctx, grm.tracer, retrieveMembershipsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveMemberships(ctx, memberID, pm)
}

func (grm groupRepositoryMiddleware) RetrieveMembers(ctx context.Context, groupID string, pm things.PageMetadata) (things.MemberPage, error) {
	span := createSpan(ctx, grm.tracer, retrieveMembersOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveMembers(ctx, groupID, pm)
}

func (grm groupRepositoryMiddleware) AssignMember(ctx context.Context, groupID string, memberIDs ...string) error {
	span := createSpan(ctx, grm.tracer, assignMemberOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.AssignMember(ctx, groupID, memberIDs...)
}

func (grm groupRepositoryMiddleware) UnassignMember(ctx context.Context, groupID string, memberIDs ...string) error {
	span := createSpan(ctx, grm.tracer, unassignMemberOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.UnassignMember(ctx, groupID, memberIDs...)
}
