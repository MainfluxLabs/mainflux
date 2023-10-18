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
	retrieveAll                = "retrieve_all_orgs"
	retrieveAllMemberRelations = "retrieve_all_member_elations"
	retrieveAllGroupRelations  = "retrieve_all_group_relations"
	savePolicies               = "save_policies"
	removePolicy               = "remove_policy"
	retrievePolicy             = "retrieve_policy"
	updatePolicies             = "update_policies"
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

func (orm orgRepositoryMiddleware) RetrieveAll(ctx context.Context) ([]auth.Org, error) {
	span := createSpan(ctx, orm.tracer, retrieveAll)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveAll(ctx)
}

func (orm orgRepositoryMiddleware) RetrieveByAdmin(ctx context.Context, pm auth.PageMetadata) (auth.OrgsPage, error) {
	span := createSpan(ctx, orm.tracer, retrieveAll)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveByAdmin(ctx, pm)
}

func (orm orgRepositoryMiddleware) RetrieveMemberships(ctx context.Context, memberID string, pm auth.PageMetadata) (auth.OrgsPage, error) {
	span := createSpan(ctx, orm.tracer, orgMemberships)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveMemberships(ctx, memberID, pm)
}

func (orm orgRepositoryMiddleware) AssignMembers(ctx context.Context, mrs ...auth.MemberRelation) error {
	span := createSpan(ctx, orm.tracer, assignOrgMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.AssignMembers(ctx, mrs...)
}

func (orm orgRepositoryMiddleware) UnassignMembers(ctx context.Context, orgID string, memberIDs ...string) error {
	span := createSpan(ctx, orm.tracer, unassignOrgMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.UnassignMembers(ctx, orgID, memberIDs...)
}

func (orm orgRepositoryMiddleware) UpdateMembers(ctx context.Context, mrs ...auth.MemberRelation) error {
	span := createSpan(ctx, orm.tracer, updateOrgMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.UpdateMembers(ctx, mrs...)
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

func (orm orgRepositoryMiddleware) RetrieveAllMemberRelations(ctx context.Context) ([]auth.MemberRelation, error) {
	span := createSpan(ctx, orm.tracer, retrieveAllMemberRelations)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveAllMemberRelations(ctx)
}

func (orm orgRepositoryMiddleware) AssignGroups(ctx context.Context, grs ...auth.GroupRelation) error {
	span := createSpan(ctx, orm.tracer, assignOrgGroups)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.AssignGroups(ctx, grs...)
}

func (orm orgRepositoryMiddleware) UnassignGroups(ctx context.Context, orgID string, groupIDs ...string) error {
	span := createSpan(ctx, orm.tracer, unassignOrgGroups)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.UnassignGroups(ctx, orgID, groupIDs...)
}

func (orm orgRepositoryMiddleware) RetrieveGroups(ctx context.Context, orgID string, pm auth.PageMetadata) (auth.GroupRelationsPage, error) {
	span := createSpan(ctx, orm.tracer, retrieveGroups)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveGroups(ctx, orgID, pm)
}

func (orm orgRepositoryMiddleware) RetrieveByGroupID(ctx context.Context, groupID string) (auth.Org, error) {
	span := createSpan(ctx, orm.tracer, retrieveByGroupID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveByGroupID(ctx, groupID)
}

func (orm orgRepositoryMiddleware) RetrieveAllGroupRelations(ctx context.Context) ([]auth.GroupRelation, error) {
	span := createSpan(ctx, orm.tracer, retrieveAllGroupRelations)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrieveAllGroupRelations(ctx)
}

func (orm orgRepositoryMiddleware) SavePolicies(ctx context.Context, groupID string, mp ...auth.MemberPolicy) error {
	span := createSpan(ctx, orm.tracer, savePolicies)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.SavePolicies(ctx, groupID, mp...)
}

func (orm orgRepositoryMiddleware) RetrievePolicy(ctx context.Context, gp auth.GroupsPolicy) (string, error) {
	span := createSpan(ctx, orm.tracer, retrievePolicy)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RetrievePolicy(ctx, gp)
}

func (orm orgRepositoryMiddleware) UpdatePolicies(ctx context.Context, groupID string, mp ...auth.MemberPolicy) error {
	span := createSpan(ctx, orm.tracer, updatePolicies)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.UpdatePolicies(ctx, groupID, mp...)
}

func (orm orgRepositoryMiddleware) RemovePolicies(ctx context.Context, groupID string, memberIDs ...string) error {
	span := createSpan(ctx, orm.tracer, removePolicy)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return orm.repo.RemovePolicies(ctx, groupID, memberIDs...)
}
