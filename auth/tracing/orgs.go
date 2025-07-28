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
	saveOrg              = "save_org"
	deleteOrg            = "delete_org"
	updateOrg            = "update_org"
	retrieveOrgByID      = "retrieve_org_by_id"
	retrieveOrgsByMember = "retrieve_orgs_by_member"
	retrieveAllOrgs      = "retrieve_all_orgs"
	backupAllOrgs        = "backup_all_orgs"
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

func (orm orgRepositoryMiddleware) Remove(ctx context.Context, owner string, orgIDs ...string) error {
	span := createSpan(ctx, orm.tracer, deleteOrg)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.Remove(ctx, owner, orgIDs...)
}

func (orm orgRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (auth.Org, error) {
	span := createSpan(ctx, orm.tracer, retrieveOrgByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveByID(ctx, id)
}

func (orm orgRepositoryMiddleware) BackupAll(ctx context.Context) ([]auth.Org, error) {
	span := createSpan(ctx, orm.tracer, backupAllOrgs)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.BackupAll(ctx)
}

func (orm orgRepositoryMiddleware) RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (auth.OrgsPage, error) {
	span := createSpan(ctx, orm.tracer, retrieveAllOrgs)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveAll(ctx, pm)
}

func (orm orgRepositoryMiddleware) RetrieveByMember(ctx context.Context, memberID string, pm apiutil.PageMetadata) (auth.OrgsPage, error) {
	span := createSpan(ctx, orm.tracer, retrieveOrgsByMember)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveByMember(ctx, memberID, pm)
}
