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
	saveOrg                 = "save_org"
	deleteOrg               = "delete_org"
	updateOrg               = "update_org"
	retrieveByID            = "retrieve_by_id"
	retrieveByOwner         = "retrieve_by_owner"
	retrieveOrgsByMember    = "retrieve_orgs_by_member"
	retrieveAll             = "retrieve_all_orgs"
	retrieveAllMembersByOrg = "retrieve_all_members_by_org"
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

func (orm orgRepositoryMiddleware) Save(ctx context.Context, orgs ...auth.Org) error {
	span := createSpan(ctx, orm.tracer, saveOrg)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.Save(ctx, orgs...)
}

func (orm orgRepositoryMiddleware) Update(ctx context.Context, org auth.Org) error {
	span := createSpan(ctx, orm.tracer, updateOrg)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.Update(ctx, org)
}

func (orm orgRepositoryMiddleware) Remove(ctx context.Context, owner, orgID string) error {
	span := createSpan(ctx, orm.tracer, deleteOrg)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.Remove(ctx, owner, orgID)
}

func (orm orgRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (auth.Org, error) {
	span := createSpan(ctx, orm.tracer, retrieveByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveByID(ctx, id)
}

func (orm orgRepositoryMiddleware) RetrieveAll(ctx context.Context) ([]auth.Org, error) {
	span := createSpan(ctx, orm.tracer, retrieveAll)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveAll(ctx)
}

func (orm orgRepositoryMiddleware) RetrieveByAdmin(ctx context.Context, pm apiutil.PageMetadata) (auth.OrgsPage, error) {
	span := createSpan(ctx, orm.tracer, retrieveAll)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveByAdmin(ctx, pm)
}

func (orm orgRepositoryMiddleware) RetrieveByMemberID(ctx context.Context, memberID string, pm apiutil.PageMetadata) (auth.OrgsPage, error) {
	span := createSpan(ctx, orm.tracer, retrieveOrgsByMember)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveByMemberID(ctx, memberID, pm)
}
