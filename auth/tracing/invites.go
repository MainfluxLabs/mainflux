// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package tracing contains middlewares that will add spans to existing traces.
package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveOrgInvite              = "save_org_invite"
	retrieveOrgInviteByID      = "retrieve_org_invite_by_id"
	removeOrgInvite            = "remove_org_invite"
	retrieveOrgInvitesByUserID = "retrieve_org_invites_by_user_id"
	updateOrgInviteState       = "update_org_invite_state"
)

var _ auth.OrgInvitesRepository = (*invitesRepositoryMiddleware)(nil)

type invitesRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   auth.OrgInvitesRepository
}

func InvitesRepositoryMiddleware(tracer opentracing.Tracer, repo auth.OrgInvitesRepository) auth.OrgInvitesRepository {
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

func (irm invitesRepositoryMiddleware) RetrieveOrgInvitesByUser(ctx context.Context, userType string, userID string, pm auth.PageMetadataInvites) (auth.OrgInvitesPage, error) {
	span := createSpan(ctx, irm.tracer, retrieveOrgInvitesByUserID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RetrieveOrgInvitesByUser(ctx, userType, userID, pm)
}

func (irm invitesRepositoryMiddleware) RetrieveOrgInvitesByOrg(ctx context.Context, orgID string, pm auth.PageMetadataInvites) (auth.OrgInvitesPage, error) {
	span := createSpan(ctx, irm.tracer, retrieveOrgInvitesByUserID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RetrieveOrgInvitesByOrg(ctx, orgID, pm)
}

func (irm invitesRepositoryMiddleware) UpdateOrgInviteState(ctx context.Context, inviteID string, state string) error {
	span := createSpan(ctx, irm.tracer, updateOrgInviteState)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.UpdateOrgInviteState(ctx, inviteID, state)
}
