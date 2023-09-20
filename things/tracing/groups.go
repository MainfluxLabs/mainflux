package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/things"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveGroupOp                    = "save_group"
	updateGroupOp                  = "update_group"
	removeGroupOp                  = "remove_group"
	retrieveAllOp                  = "retrieve_all"
	retrieveGroupByIDOp            = "retrieve_group_by_id"
	retrieveGroupByIDsOp           = "retrieve_group_by_ids"
	retrieveByOwnerOp              = "retrieve_by_owner"
	retrieveThingMembershipOp      = "retrieve_thing_membership"
	retrieveChannelMembershipOp    = "retrieve_channel_membership"
	retrieveGroupThingsOp          = "retrieve_group_things"
	retrieveGroupThingsByChannelOp = "retrieve_group_things_by_channel"
	retrieveGroupChannelsOp        = "retrieve_group_channels"
	assignThingOp                  = "assign_thing"
	unassignThingOp                = "unassign_thing"
	assignChannelOp                = "assign_channel"
	unassignChannelOp              = "unassign_channel"
	retrieveAllThingRelationsOp    = "retrieve_all_thing_relations"
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

func (grm groupRepositoryMiddleware) RetrieveThingMembership(ctx context.Context, thingID string) (string, error) {
	span := createSpan(ctx, grm.tracer, retrieveThingMembershipOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveThingMembership(ctx, thingID)
}

func (grm groupRepositoryMiddleware) RetrieveGroupThings(ctx context.Context, groupID string, pm things.PageMetadata) (things.GroupThingsPage, error) {
	span := createSpan(ctx, grm.tracer, retrieveGroupThingsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveGroupThings(ctx, groupID, pm)
}

func (grm groupRepositoryMiddleware) RetrieveGroupThingsByChannel(ctx context.Context, groupID, channelID string, pm things.PageMetadata) (things.GroupThingsPage, error) {
	span := createSpan(ctx, grm.tracer, retrieveGroupThingsByChannelOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveGroupThingsByChannel(ctx, groupID, channelID, pm)
}

func (grm groupRepositoryMiddleware) RetrieveAllThingRelations(ctx context.Context) ([]things.GroupThingRelation, error) {
	span := createSpan(ctx, grm.tracer, retrieveAllThingRelationsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveAllThingRelations(ctx)
}

func (grm groupRepositoryMiddleware) AssignThing(ctx context.Context, groupID string, thingIDs ...string) error {
	span := createSpan(ctx, grm.tracer, assignThingOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.AssignThing(ctx, groupID, thingIDs...)
}

func (grm groupRepositoryMiddleware) UnassignThing(ctx context.Context, groupID string, thingIDs ...string) error {
	span := createSpan(ctx, grm.tracer, unassignThingOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.UnassignThing(ctx, groupID, thingIDs...)
}

func (grm groupRepositoryMiddleware) RetrieveChannelMembership(ctx context.Context, channelID string) (string, error) {
	span := createSpan(ctx, grm.tracer, retrieveChannelMembershipOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveChannelMembership(ctx, channelID)
}

func (grm groupRepositoryMiddleware) RetrieveGroupChannels(ctx context.Context, groupID string, pm things.PageMetadata) (things.GroupChannelsPage, error) {
	span := createSpan(ctx, grm.tracer, retrieveGroupChannelsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveGroupChannels(ctx, groupID, pm)
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
