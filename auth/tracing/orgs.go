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
	saveOrg                    = "save_org"
	deleteOrg                  = "delete_org"
	updateOrg                  = "update_org"
	retrieveByID               = "retrieve_by_id"
	retrieveByOwner            = "retrieve_by_owner"
	orgMemberships             = "org_memberships"
	orgMembers                 = "org_members"
	retrieveGroups             = "retrieve_groups"
	assignOrgMembers           = "assign_org_members"
	assignOrgGroups            = "assign_org_groups"
	unassignOrgMembers         = "unassign_org_members"
	unassignOrgGroups          = "unassign_org_groups"
	retrieveByGroupID          = "retrieve_by_group_id"
	updateOrgMembers           = "update_org_members"
	retrieveAllMemberRelations = "retrieve_all_member_relations"
	retrieveAllGroupRelations  = "retrieve_all_group_relations"
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

func (orm orgRepositoryMiddleware) RetrieveByOwner(ctx context.Context, ownerID string, pm auth.PageMetadata) (auth.OrgsPage, error) {
	span := createSpan(ctx, orm.tracer, retrieveByOwner)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveByOwner(ctx, ownerID, pm)
}

func (orm orgRepositoryMiddleware) RetrieveMemberships(ctx context.Context, memberID string, pm auth.PageMetadata) (auth.OrgsPage, error) {
	span := createSpan(ctx, orm.tracer, orgMemberships)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveMemberships(ctx, memberID, pm)
}

func (orm orgRepositoryMiddleware) AssignMembers(ctx context.Context, orgID string, members ...auth.Member) error {
	span := createSpan(ctx, orm.tracer, assignOrgMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.AssignMembers(ctx, orgID, members...)
}

func (orm orgRepositoryMiddleware) UnassignMembers(ctx context.Context, orgID string, memberIDs ...string) error {
	span := createSpan(ctx, orm.tracer, unassignOrgMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.UnassignMembers(ctx, orgID, memberIDs...)
}

func (orm orgRepositoryMiddleware) UpdateMembers(ctx context.Context, orgID string, members ...auth.Member) error {
	span := createSpan(ctx, orm.tracer, updateOrgMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.UpdateMembers(ctx, orgID, members...)
}

func (orm orgRepositoryMiddleware) RetrieveRole(ctx context.Context, orgID, memberID string) (string, error) {
	span := createSpan(ctx, orm.tracer, retrieveByGroupID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveRole(ctx, orgID, memberID)
}

func (orm orgRepositoryMiddleware) RetrieveMembers(ctx context.Context, orgID string, pm auth.PageMetadata) (auth.OrgMembersPage, error) {
	span := createSpan(ctx, orm.tracer, orgMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveMembers(ctx, orgID, pm)
}

func (orm orgRepositoryMiddleware) AssignGroups(ctx context.Context, orgID string, groupIDs ...string) error {
	span := createSpan(ctx, orm.tracer, assignOrgGroups)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.AssignGroups(ctx, orgID, groupIDs...)
}

func (orm orgRepositoryMiddleware) UnassignGroups(ctx context.Context, orgID string, groupIDs ...string) error {
	span := createSpan(ctx, orm.tracer, unassignOrgGroups)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.UnassignGroups(ctx, orgID, groupIDs...)
}

func (orm orgRepositoryMiddleware) RetrieveGroups(ctx context.Context, orgID string, pm auth.PageMetadata) (auth.OrgGroupsPage, error) {
	span := createSpan(ctx, orm.tracer, retrieveGroups)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveGroups(ctx, orgID, pm)
}

func (orm orgRepositoryMiddleware) RetrieveByGroupID(ctx context.Context, groupID string) (auth.OrgsPage, error) {
	span := createSpan(ctx, orm.tracer, retrieveByGroupID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveByGroupID(ctx, groupID)
}

func (orm orgRepositoryMiddleware) RetrieveAllMemberRelations(ctx context.Context) ([]auth.MemberRelation, error) {
	span := createSpan(ctx, orm.tracer, retrieveAllMemberRelations)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveAllMemberRelations(ctx)
}

func (orm orgRepositoryMiddleware) RetrieveAllGroupRelations(ctx context.Context) ([]auth.GroupRelation, error) {
	span := createSpan(ctx, orm.tracer, retrieveAllGroupRelations)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveAllGroupRelations(ctx)
}
