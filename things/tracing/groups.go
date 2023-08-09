package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/things"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveGroupOp                 = "save_group"
	updateGroupOp               = "update_group"
	removeGroupOp               = "remove_group"
	retrieveAllOp               = "retrieve_all"
	retrieveGroupByIDOp         = "retrieve_group_by_id"
	retrieveGroupByIDsOp        = "retrieve_group_by_ids"
	retrieveByOwnerOp           = "retrieve_by_owner"
	retrieveMembershipOp        = "retrieve_membership"
	retrieveMembersOp           = "retrieve_members"
	assignMemberOp              = "assign_member"
	unassignMemberOp            = "unassign_member"
	assignChannelOp             = "assign_channel"
	unassignChannelOp           = "unassign_channel"
	retrieveAllGroupRelationsOp = "retrieve_all_group_relations"
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
	span := createSpan(ctx, grm.tracer, retrieveAllOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveAll(ctx)
}

func (grm groupRepositoryMiddleware) RetrieveByAdmin(ctx context.Context, pm things.PageMetadata) (things.GroupPage, error) {
	span := createSpan(ctx, grm.tracer, retrieveAllOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveByAdmin(ctx, pm)
}

func (grm groupRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (things.Group, error) {
	span := createSpan(ctx, grm.tracer, retrieveGroupByIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveByID(ctx, id)
}
func (grm groupRepositoryMiddleware) RetrieveByIDs(ctx context.Context, groupIDs []string) (things.GroupPage, error) {
	span := createSpan(ctx, grm.tracer, retrieveGroupByIDsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveByIDs(ctx, groupIDs)
}

func (grm groupRepositoryMiddleware) RetrieveByOwner(ctx context.Context, ownerID string, pm things.PageMetadata) (things.GroupPage, error) {
	span := createSpan(ctx, grm.tracer, retrieveByOwnerOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveByOwner(ctx, ownerID, pm)
}

func (grm groupRepositoryMiddleware) RetrieveMembership(ctx context.Context, memberID string) (string, error) {
	span := createSpan(ctx, grm.tracer, retrieveMembershipOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveMembership(ctx, memberID)
}

func (grm groupRepositoryMiddleware) RetrieveMembers(ctx context.Context, groupID string, pm things.PageMetadata) (things.MemberPage, error) {
	span := createSpan(ctx, grm.tracer, retrieveMembersOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveMembers(ctx, groupID, pm)
}

func (grm groupRepositoryMiddleware) RetrieveChannels(ctx context.Context, groupID string, pm things.PageMetadata) (things.GroupChannelsPage, error) {
	span := createSpan(ctx, grm.tracer, retrieveMembershipOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveChannels(ctx, groupID, pm)
}

func (grm groupRepositoryMiddleware) RetrieveAllGroupRelations(ctx context.Context) ([]things.GroupRelation, error) {
	span := createSpan(ctx, grm.tracer, retrieveAllGroupRelationsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveAllGroupRelations(ctx)
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

func (grm groupRepositoryMiddleware) AssignChannel(ctx context.Context, groupID string, channelIDs ...string) error {
	span := createSpan(ctx, grm.tracer, assignChannelOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.AssignChannel(ctx, groupID, channelIDs...)
}

func (grm groupRepositoryMiddleware) UnassignChannel(ctx context.Context, groupID string, channelIDs ...string) error {
	span := createSpan(ctx, grm.tracer, unassignChannelOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.UnassignChannel(ctx, groupID, channelIDs...)
}
