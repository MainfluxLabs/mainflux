// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package tracing contains middlewares that will add spans to existing traces.
package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/opentracing/opentracing-go"
)

const (
	createMemberships        = "create_memberships"
	removeMemberships        = "remove_memberships"
	updateMemberships        = "update_memberships"
	retrieveMembershipsByOrg = "retrieve_memberships_by_org"
	retrieveAllMemberships   = "retrieve_all_memberships"
)

var _ auth.MembershipsRepository = (*membershipsRepositoryMiddleware)(nil)

type membershipsRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   auth.MembershipsRepository
}

// MembershipsRepositoryMiddleware tracks request and their latency, and adds spans to context.
func MembershipsRepositoryMiddleware(tracer opentracing.Tracer, gr auth.MembershipsRepository) auth.MembershipsRepository {
	return membershipsRepositoryMiddleware{
		tracer: tracer,
		repo:   gr,
	}
}

func (orm membershipsRepositoryMiddleware) Save(ctx context.Context, oms ...auth.OrgMembership) error {
	span := createSpan(ctx, orm.tracer, createMemberships)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.Save(ctx, oms...)
}

func (orm membershipsRepositoryMiddleware) Remove(ctx context.Context, orgID string, memberIDs ...string) error {
	span := createSpan(ctx, orm.tracer, removeMemberships)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.Remove(ctx, orgID, memberIDs...)
}

func (orm membershipsRepositoryMiddleware) Update(ctx context.Context, oms ...auth.OrgMembership) error {
	span := createSpan(ctx, orm.tracer, updateMemberships)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.Update(ctx, oms...)
}

func (orm membershipsRepositoryMiddleware) RetrieveRole(ctx context.Context, orgID, memberID string) (string, error) {
	span := createSpan(ctx, orm.tracer, retrieveRole)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveRole(ctx, orgID, memberID)
}

func (orm membershipsRepositoryMiddleware) RetrieveByOrgID(ctx context.Context, orgID string, pm apiutil.PageMetadata) (auth.OrgMembershipsPage, error) {
	span := createSpan(ctx, orm.tracer, retrieveMembershipsByOrg)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveByOrgID(ctx, orgID, pm)
}

func (orm membershipsRepositoryMiddleware) RetrieveAll(ctx context.Context) ([]auth.OrgMembership, error) {
	span := createSpan(ctx, orm.tracer, retrieveAllMemberships)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveAll(ctx)
}
