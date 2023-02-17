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
	assignOrg       = "assign"
	saveOrg         = "save_org"
	deleteOrg       = "delete_org"
	updateOrg       = "update_org"
	retrieveByIDOrg = "retrieve_by_id"
	retrieveAllOrg  = "retrieve_all_orgs"
	membershipsOrg  = "memberships"
	membersOrg      = "members"
	unassignOrg     = "unassign"
)

var _ auth.OrgRepository = (*orgRepositoryMiddleware)(nil)

type orgRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   auth.OrgRepository
}

// OrgRepositoryMiddleware tracks request and their latency, and adds spans to context.
func OrgRepositoryMiddleware(tracer opentracing.Tracer, gr auth.OrgRepository) auth.OrgRepository {
	return orgRepositoryMiddleware{
		tracer: tracer,
		repo:   gr,
	}
}

func (orm orgRepositoryMiddleware) Save(ctx context.Context, g auth.Org) error {
	span := createSpan(ctx, orm.tracer, saveOrg)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.Save(ctx, g)
}

func (orm orgRepositoryMiddleware) Update(ctx context.Context, g auth.Org) error {
	span := createSpan(ctx, orm.tracer, updateOrg)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.Update(ctx, g)
}

func (orm orgRepositoryMiddleware) Delete(ctx context.Context, owner, orgID string) error {
	span := createSpan(ctx, orm.tracer, deleteOrg)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.Delete(ctx, owner, orgID)
}

func (orm orgRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (auth.Org, error) {
	span := createSpan(ctx, orm.tracer, retrieveByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveByID(ctx, id)
}

func (orm orgRepositoryMiddleware) RetrieveAll(ctx context.Context, ownerID string, pm auth.OrgPageMetadata) (auth.OrgPage, error) {
	span := createSpan(ctx, orm.tracer, retrieveAll)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveAll(ctx, ownerID, pm)
}

func (orm orgRepositoryMiddleware) Memberships(ctx context.Context, memberID string, pm auth.OrgPageMetadata) (auth.OrgPage, error) {
	span := createSpan(ctx, orm.tracer, memberships)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.Memberships(ctx, memberID, pm)
}

func (orm orgRepositoryMiddleware) Members(ctx context.Context, orgID string, pm auth.OrgPageMetadata) (auth.OrgMembersPage, error) {
	span := createSpan(ctx, orm.tracer, members)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.Members(ctx, orgID, pm)
}

func (orm orgRepositoryMiddleware) Assign(ctx context.Context, orgID string, memberIDs ...string) error {
	span := createSpan(ctx, orm.tracer, assign)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.Assign(ctx, orgID, memberIDs...)
}

func (orm orgRepositoryMiddleware) Unassign(ctx context.Context, orgID string, memberIDs ...string) error {
	span := createSpan(ctx, orm.tracer, unassign)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.Unassign(ctx, orgID, memberIDs...)
}
