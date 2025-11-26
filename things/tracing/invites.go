package tracing

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/invites"
	"github.com/MainfluxLabs/mainflux/things"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveGroupInvite             = "save_group_invite"
	retrieveGroupInviteByID     = "retrieve_group_invite_by_id"
	removeGroupInvite           = "remove_group_invite"
	retrieveGroupInvitesByUser  = "retrieve_group_invites_by_user"
	retrieveGroupInvitesByGroup = "retrieve_group_invites_by_group"
	updateGroupInviteState      = "update_group_invite_state"
	saveDormantInviteRelations  = "save_dormant_invite_relations"
	activateGroupInvites        = "activate_group_invites"
)

var _ things.GroupInviteRepository = (*invitesRepositoryMiddleware)(nil)

type invitesRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   things.GroupInviteRepository
}

func InvitesRepositoryMiddleware(tracer opentracing.Tracer, repo things.GroupInviteRepository) things.GroupInviteRepository {
	return invitesRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (irm invitesRepositoryMiddleware) SaveInvites(ctx context.Context, invites ...things.GroupInvite) error {
	span := dbutil.CreateSpan(ctx, irm.tracer, saveGroupInvite)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.SaveInvites(ctx, invites...)
}

func (irm invitesRepositoryMiddleware) RetrieveInviteByID(ctx context.Context, inviteID string) (things.GroupInvite, error) {
	span := dbutil.CreateSpan(ctx, irm.tracer, retrieveGroupInviteByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RetrieveInviteByID(ctx, inviteID)
}

func (irm invitesRepositoryMiddleware) RemoveInvite(ctx context.Context, inviteID string) error {
	span := dbutil.CreateSpan(ctx, irm.tracer, removeGroupInvite)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RemoveInvite(ctx, inviteID)
}

func (irm invitesRepositoryMiddleware) RetrieveInvitesByUser(ctx context.Context, userType string, userID string, pm invites.PageMetadataInvites) (invites.InvitesPage[things.GroupInvite], error) {
	span := dbutil.CreateSpan(ctx, irm.tracer, retrieveGroupInvitesByUser)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RetrieveInvitesByUser(ctx, userType, userID, pm)
}

func (irm invitesRepositoryMiddleware) RetrieveInvitesByDestination(ctx context.Context, orgID string, pm invites.PageMetadataInvites) (invites.InvitesPage[things.GroupInvite], error) {
	span := dbutil.CreateSpan(ctx, irm.tracer, retrieveGroupInvitesByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RetrieveInvitesByDestination(ctx, orgID, pm)
}

func (irm invitesRepositoryMiddleware) UpdateInviteState(ctx context.Context, inviteID string, state string) error {
	span := dbutil.CreateSpan(ctx, irm.tracer, updateGroupInviteState)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.UpdateInviteState(ctx, inviteID, state)
}

func (irm invitesRepositoryMiddleware) SaveDormantInviteRelations(ctx context.Context, orgInviteID string, groupInviteIDs ...string) error {
	span := dbutil.CreateSpan(ctx, irm.tracer, saveDormantInviteRelations)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.SaveDormantInviteRelations(ctx, orgInviteID, groupInviteIDs...)
}

func (irm invitesRepositoryMiddleware) ActivateGroupInvites(ctx context.Context, orgInviteID, userID string, expirationTime time.Time) ([]things.GroupInvite, error) {
	span := dbutil.CreateSpan(ctx, irm.tracer, activateGroupInvites)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.ActivateGroupInvites(ctx, orgInviteID, userID, expirationTime)
}
