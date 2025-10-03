// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package tracing contains middlewares that will add spans to existing traces.
package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/opentracing/opentracing-go"
)

const (
	saveOrgMemberships      = "save_org_memberships"
	removeOrgMemberships    = "remove_org_memberships"
	updateOrgMemberships    = "update_org_memberships"
	retrieveOrgMemberships  = "retrieve_org_memberships"
	backupAllOrgMemberships = "backup_all_org_memberships"
	backupOrgMemberships    = "backup_org_memberships"
)

var _ auth.OrgMembershipsRepository = (*orgMembershipsRepositoryMiddleware)(nil)

type orgMembershipsRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   auth.OrgMembershipsRepository
}

// OrgMembershipsRepositoryMiddleware tracks request and their latency, and adds spans to context.
func OrgMembershipsRepositoryMiddleware(tracer opentracing.Tracer, gr auth.OrgMembershipsRepository) auth.OrgMembershipsRepository {
	return orgMembershipsRepositoryMiddleware{
		tracer: tracer,
		repo:   gr,
	}
}

func (orm orgMembershipsRepositoryMiddleware) Save(ctx context.Context, oms ...auth.OrgMembership) error {
	span := dbutil.CreateSpan(ctx, orm.tracer, saveOrgMemberships)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.Save(ctx, oms...)
}

func (orm orgMembershipsRepositoryMiddleware) Remove(ctx context.Context, orgID string, memberIDs ...string) error {
	span := dbutil.CreateSpan(ctx, orm.tracer, removeOrgMemberships)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.Remove(ctx, orgID, memberIDs...)
}

func (orm orgMembershipsRepositoryMiddleware) Update(ctx context.Context, oms ...auth.OrgMembership) error {
	span := dbutil.CreateSpan(ctx, orm.tracer, updateOrgMemberships)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.Update(ctx, oms...)
}

func (orm orgMembershipsRepositoryMiddleware) RetrieveRole(ctx context.Context, orgID, memberID string) (string, error) {
	span := dbutil.CreateSpan(ctx, orm.tracer, retrieveRole)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveRole(ctx, orgID, memberID)
}

func (orm orgMembershipsRepositoryMiddleware) RetrieveByOrg(ctx context.Context, orgID string, pm apiutil.PageMetadata) (auth.OrgMembershipsPage, error) {
	span := dbutil.CreateSpan(ctx, orm.tracer, retrieveOrgMemberships)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveByOrg(ctx, orgID, pm)
}

func (orm orgMembershipsRepositoryMiddleware) BackupAll(ctx context.Context) ([]auth.OrgMembership, error) {
	span := dbutil.CreateSpan(ctx, orm.tracer, backupAllOrgMemberships)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.BackupAll(ctx)
}

func (orm orgMembershipsRepositoryMiddleware) BackupByOrg(ctx context.Context, orgID string) ([]auth.OrgMembership, error) {
	span := dbutil.CreateSpan(ctx, orm.tracer, backupOrgMemberships)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.BackupByOrg(ctx, orgID)
}
