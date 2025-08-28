package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/users"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	savePlatformInvite         = "save_platform_invite"
	retrievePlatformInviteByID = "retrieve_platform_invite_by_id"
	retrievePlatformInvites    = "retrieve_platform_invites"
	updatePlatformInviteState  = "update_platform_invite_state"
)

var _ users.PlatformInvitesRepository = (*invitesRepositoryMiddleware)(nil)

type invitesRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   users.PlatformInvitesRepository
}

func PlatformInvitesRepositoryMiddleware(repo users.PlatformInvitesRepository, tracer opentracing.Tracer) users.PlatformInvitesRepository {
	return invitesRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (irm invitesRepositoryMiddleware) SavePlatformInvite(ctx context.Context, invites ...users.PlatformInvite) error {
	span := createSpan(ctx, irm.tracer, savePlatformInvite)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.SavePlatformInvite(ctx, invites...)
}

func (irm invitesRepositoryMiddleware) RetrievePlatformInviteByID(ctx context.Context, inviteID string) (users.PlatformInvite, error) {
	span := createSpan(ctx, irm.tracer, retrievePlatformInviteByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RetrievePlatformInviteByID(ctx, inviteID)
}

func (irm invitesRepositoryMiddleware) RetrievePlatformInvites(ctx context.Context, pm users.PageMetadataInvites) (users.PlatformInvitesPage, error) {
	span := createSpan(ctx, irm.tracer, retrievePlatformInvites)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RetrievePlatformInvites(ctx, pm)
}

func (irm invitesRepositoryMiddleware) UpdatePlatformInviteState(ctx context.Context, inviteID string, state string) error {
	span := createSpan(ctx, irm.tracer, updatePlatformInviteState)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.UpdatePlatformInviteState(ctx, inviteID, state)
}
