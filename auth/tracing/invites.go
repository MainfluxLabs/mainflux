// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package tracing contains middlewares that will add spans to existing traces.
package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveOrgInvite              = "save_org_invite"
	retrieveOrgInviteByID      = "retrieve_org_invite_by_id"
	removeOrgInvite            = "remove_org_invite"
	retrieveOrgInvitesByUserID = "retrieve_org_invites_by_user_id"
	updateOrgInviteState       = "update_org_invite_state"
	savePlatformInvite         = "save_platform_invite"
	retrievePlatformInviteByID = "retrieve_platform_invite_by_id"
	retrievePlatformInvites    = "retrieve_platform_invites"
	updatePlatformInviteState  = "update_platform_invite_state"
)

var _ auth.InvitesRepository = (*invitesRepositoryMiddleware)(nil)

type invitesRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   auth.InvitesRepository
}

func InvitesRepositoryMiddleware(tracer opentracing.Tracer, repo auth.InvitesRepository) auth.InvitesRepository {
	return invitesRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (irm invitesRepositoryMiddleware) SaveOrgInvite(ctx context.Context, invites ...auth.OrgInvite) error {
	span := createSpan(ctx, irm.tracer, saveOrgInvite)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.SaveOrgInvite(ctx, invites...)
}

func (irm invitesRepositoryMiddleware) RetrieveOrgInviteByID(ctx context.Context, inviteID string) (auth.OrgInvite, error) {
	span := createSpan(ctx, irm.tracer, retrieveOrgInviteByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RetrieveOrgInviteByID(ctx, inviteID)
}

func (irm invitesRepositoryMiddleware) RemoveOrgInvite(ctx context.Context, inviteID string) error {
	span := createSpan(ctx, irm.tracer, removeOrgInvite)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RemoveOrgInvite(ctx, inviteID)
}

func (irm invitesRepositoryMiddleware) RetrieveOrgInvitesByUserID(ctx context.Context, userType string, userID string, pm apiutil.PageMetadata) (auth.OrgInvitesPage, error) {
	span := createSpan(ctx, irm.tracer, retrieveOrgInvitesByUserID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RetrieveOrgInvitesByUserID(ctx, userType, userID, pm)
}

func (irm invitesRepositoryMiddleware) UpdateOrgInviteState(ctx context.Context, inviteID string, state string) error {
	span := createSpan(ctx, irm.tracer, updateOrgInviteState)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.UpdateOrgInviteState(ctx, inviteID, state)
}

func (irm invitesRepositoryMiddleware) SavePlatformInvite(ctx context.Context, invites ...auth.PlatformInvite) error {
	span := createSpan(ctx, irm.tracer, savePlatformInvite)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.SavePlatformInvite(ctx, invites...)
}

func (irm invitesRepositoryMiddleware) RetrievePlatformInviteByID(ctx context.Context, inviteID string) (auth.PlatformInvite, error) {
	span := createSpan(ctx, irm.tracer, retrievePlatformInviteByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RetrievePlatformInviteByID(ctx, inviteID)
}

func (irm invitesRepositoryMiddleware) RetrievePlatformInvites(ctx context.Context, pm apiutil.PageMetadata) (auth.PlatformInvitesPage, error) {
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
