// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package tracing contains middlewares that will add spans to existing traces.
package tracing

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveOrgInvite                       = "save_org_invite"
	saveDormantInviteRelation           = "save_dormant_invite_relation"
	retrieveOrgInviteByID               = "retrieve_org_invite_by_id"
	retrieveOrgInviteByPlatformInviteID = "retrieve_org_invite_by_platform_invite_id"
	removeOrgInvite                     = "remove_org_invite"
	retrieveOrgInvitesByUser            = "retrieve_org_invites_by_user"
	updateOrgInviteState                = "update_org_invite_state"
	activateOrgInvite                   = "activate_org_invite"
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
	span := dbutil.CreateSpan(ctx, irm.tracer, saveOrgInvite)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.SaveOrgInvite(ctx, invites...)
}

func (irm invitesRepositoryMiddleware) RetrieveOrgInviteByID(ctx context.Context, inviteID string) (auth.OrgInvite, error) {
	span := dbutil.CreateSpan(ctx, irm.tracer, retrieveOrgInviteByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RetrieveOrgInviteByID(ctx, inviteID)
}

func (irm invitesRepositoryMiddleware) RemoveOrgInvite(ctx context.Context, inviteID string) error {
	span := dbutil.CreateSpan(ctx, irm.tracer, removeOrgInvite)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RemoveOrgInvite(ctx, inviteID)
}

func (irm invitesRepositoryMiddleware) RetrieveOrgInvitesByUser(ctx context.Context, userType string, userID string, pm auth.PageMetadataInvites) (auth.OrgInvitesPage, error) {
	span := dbutil.CreateSpan(ctx, irm.tracer, retrieveOrgInvitesByUser)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RetrieveOrgInvitesByUser(ctx, userType, userID, pm)
}

func (irm invitesRepositoryMiddleware) RetrieveOrgInvitesByOrg(ctx context.Context, orgID string, pm auth.PageMetadataInvites) (auth.OrgInvitesPage, error) {
	span := dbutil.CreateSpan(ctx, irm.tracer, retrieveOrgInvitesByUser)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RetrieveOrgInvitesByOrg(ctx, orgID, pm)
}

func (irm invitesRepositoryMiddleware) UpdateOrgInviteState(ctx context.Context, inviteID string, state string) error {
	span := dbutil.CreateSpan(ctx, irm.tracer, updateOrgInviteState)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.UpdateOrgInviteState(ctx, inviteID, state)
}

func (irm invitesRepositoryMiddleware) SaveDormantInviteRelation(ctx context.Context, orgInviteID, platformInviteID string) error {
	span := dbutil.CreateSpan(ctx, irm.tracer, saveDormantInviteRelation)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.SaveDormantInviteRelation(ctx, orgInviteID, platformInviteID)
}

func (irm invitesRepositoryMiddleware) RetrieveOrgInviteByPlatformInvite(ctx context.Context, platformInviteID string) (auth.OrgInvite, error) {
	span := dbutil.CreateSpan(ctx, irm.tracer, retrieveOrgInviteByPlatformInviteID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RetrieveOrgInviteByPlatformInvite(ctx, platformInviteID)
}

func (irm invitesRepositoryMiddleware) ActivateOrgInvite(ctx context.Context, platformInviteID, userID string, expiresAt time.Time) ([]auth.OrgInvite, error) {
	span := dbutil.CreateSpan(ctx, irm.tracer, activateOrgInvite)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.ActivateOrgInvite(ctx, platformInviteID, userID, expiresAt)
}
