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
	assignMembers         = "assign_members"
	unassignMembers       = "unassign_members"
	updateMembers         = "update_members"
	retrieveMembersByOrg  = "retrieve_members_by_org"
	retrieveAllMembers    = "retrieve_all_members"
)

var _ auth.MembersRepository = (*membersRepositoryMiddleware)(nil)

type membersRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   auth.MembersRepository
}

// MembersRepositoryMiddleware tracks request and their latency, and adds spans to context.
func MembersRepositoryMiddleware(tracer opentracing.Tracer, gr auth.MembersRepository) auth.MembersRepository {
	return membersRepositoryMiddleware{
		tracer: tracer,
		repo:   gr,
	}
}

func (orm membersRepositoryMiddleware) AssignMembers(ctx context.Context, oms ...auth.OrgMember) error {
	span := createSpan(ctx, orm.tracer, assignMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.AssignMembers(ctx, oms...)
}

func (orm membersRepositoryMiddleware) UnassignMembers(ctx context.Context, orgID string, memberIDs ...string) error {
	span := createSpan(ctx, orm.tracer, unassignMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.UnassignMembers(ctx, orgID, memberIDs...)
}

func (orm membersRepositoryMiddleware) UpdateMembers(ctx context.Context, oms ...auth.OrgMember) error {
	span := createSpan(ctx, orm.tracer, updateMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.UpdateMembers(ctx, oms...)
}

func (orm membersRepositoryMiddleware) RetrieveRole(ctx context.Context, orgID, memberID string) (string, error) {
	span := createSpan(ctx, orm.tracer, retrieveRole)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveRole(ctx, orgID, memberID)
}

func (orm membersRepositoryMiddleware) RetrieveMembersByOrg(ctx context.Context, orgID string, pm auth.PageMetadata) (auth.OrgMembersPage, error) {
	span := createSpan(ctx, orm.tracer, retrieveMembersByOrg)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveMembersByOrg(ctx, orgID, pm)
}

func (orm membersRepositoryMiddleware) RetrieveAllMembers(ctx context.Context) ([]auth.OrgMember, error) {
	span := createSpan(ctx, orm.tracer, retrieveAllMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveAllMembers(ctx)
}
