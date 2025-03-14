// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package tracing contains middlewares that will add spans to existing traces.
package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/things"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveRolesByGroup         = "save_roles_by_group"
	updateRolesByGroup       = "update_roles_by_group"
	removeRolesByGroup       = "remove_roles_by_group"
	retrieveRole             = "retrieve_role"
	retrieveRolesByGroup     = "retrieve_roles_by_group"
	retrieveGroupIDsByMember = "retrieve_group_ids_by_member"
	retrieveAllRolesByGroup  = "retrieve_all_roles_by_group"
)

var _ things.RolesRepository = (*rolesRepositoryMiddleware)(nil)

type rolesRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   things.RolesRepository
}

// RolesRepositoryMiddleware tracks request and their latency, and adds spans to context.
func RolesRepositoryMiddleware(tracer opentracing.Tracer, pr things.RolesRepository) things.RolesRepository {
	return rolesRepositoryMiddleware{
		tracer: tracer,
		repo:   pr,
	}
}

func (prm rolesRepositoryMiddleware) SaveRolesByGroup(ctx context.Context, gms ...things.GroupMember) error {
	span := createSpan(ctx, prm.tracer, saveRolesByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.SaveRolesByGroup(ctx, gms...)
}

func (prm rolesRepositoryMiddleware) RetrieveRole(ctx context.Context, gp things.GroupMember) (string, error) {
	span := createSpan(ctx, prm.tracer, retrieveRole)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.RetrieveRole(ctx, gp)
}

func (prm rolesRepositoryMiddleware) RetrieveRolesByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (things.GroupMembersPage, error) {
	span := createSpan(ctx, prm.tracer, retrieveRolesByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.RetrieveRolesByGroup(ctx, groupID, pm)
}

func (prm rolesRepositoryMiddleware) RetrieveGroupIDsByMember(ctx context.Context, memberID string) ([]string, error) {
	span := createSpan(ctx, prm.tracer, retrieveGroupIDsByMember)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.RetrieveGroupIDsByMember(ctx, memberID)
}

func (prm rolesRepositoryMiddleware) RetrieveAllRolesByGroup(ctx context.Context) ([]things.GroupMember, error) {
	span := createSpan(ctx, prm.tracer, retrieveAllRolesByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.RetrieveAllRolesByGroup(ctx)
}

func (prm rolesRepositoryMiddleware) UpdateRolesByGroup(ctx context.Context, gms ...things.GroupMember) error {
	span := createSpan(ctx, prm.tracer, updateRolesByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.UpdateRolesByGroup(ctx, gms...)
}

func (prm rolesRepositoryMiddleware) RemoveRolesByGroup(ctx context.Context, groupID string, memberIDs ...string) error {
	span := createSpan(ctx, prm.tracer, removeRolesByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.RemoveRolesByGroup(ctx, groupID, memberIDs...)
}
