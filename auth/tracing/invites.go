// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package tracing contains middlewares that will add spans to existing traces.
package tracing

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/invites"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveOrgInvite             = "save_org_invite"
	saveDormantInviteRelation = "save_dormant_invite_relation"
	retrieveOrgInviteByID     = "retrieve_org_invite_by_id"
	removeOrgInvite           = "remove_org_invite"
	retrieveOrgInvitesByUser  = "retrieve_org_invites_by_user"
	updateOrgInviteState      = "update_org_invite_state"
	activateOrgInvite         = "activate_org_invite"
)

var _ auth.OrgInviteRepository = (*invitesRepositoryMiddleware)(nil)

type invitesRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   auth.OrgInviteRepository
}

func InvitesRepositoryMiddleware(tracer opentracing.Tracer, repo auth.OrgInviteRepository) auth.OrgInviteRepository {
	return invitesRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (irm invitesRepositoryMiddleware) SaveInvites(ctx context.Context, invites ...auth.OrgInvite) error {
	span := dbutil.CreateSpan(ctx, irm.tracer, saveOrgInvite)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.SaveInvites(ctx, invites...)
}

func (irm invitesRepositoryMiddleware) RetrieveInviteByID(ctx context.Context, inviteID string) (auth.OrgInvite, error) {
	span := dbutil.CreateSpan(ctx, irm.tracer, retrieveOrgInviteByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RetrieveInviteByID(ctx, inviteID)
}

func (irm invitesRepositoryMiddleware) RemoveInvite(ctx context.Context, inviteID string) error {
	span := dbutil.CreateSpan(ctx, irm.tracer, removeOrgInvite)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RemoveInvite(ctx, inviteID)
}

func (irm invitesRepositoryMiddleware) RetrieveInvitesByUser(ctx context.Context, userType string, userID string, pm invites.PageMetadataInvites) (invites.InvitesPage[auth.OrgInvite], error) {
	span := dbutil.CreateSpan(ctx, irm.tracer, retrieveOrgInvitesByUser)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RetrieveInvitesByUser(ctx, userType, userID, pm)
}

func (irm invitesRepositoryMiddleware) RetrieveInvitesByDestination(ctx context.Context, orgID string, pm invites.PageMetadataInvites) (invites.InvitesPage[auth.OrgInvite], error) {
	span := dbutil.CreateSpan(ctx, irm.tracer, retrieveOrgInvitesByUser)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RetrieveInvitesByDestination(ctx, orgID, pm)
}

func (irm invitesRepositoryMiddleware) UpdateInviteState(ctx context.Context, inviteID string, state string) error {
	span := dbutil.CreateSpan(ctx, irm.tracer, updateOrgInviteState)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.UpdateInviteState(ctx, inviteID, state)
}

func (irm invitesRepositoryMiddleware) SaveDormantInviteRelation(ctx context.Context, orgInviteID, platformInviteID string) error {
	span := dbutil.CreateSpan(ctx, irm.tracer, saveDormantInviteRelation)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.SaveDormantInviteRelation(ctx, orgInviteID, platformInviteID)
}

func (irm invitesRepositoryMiddleware) ActivateOrgInvite(ctx context.Context, platformInviteID, userID string, expiresAt time.Time) ([]auth.OrgInvite, error) {
	span := dbutil.CreateSpan(ctx, irm.tracer, activateOrgInvite)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.ActivateOrgInvite(ctx, platformInviteID, userID, expiresAt)
}
