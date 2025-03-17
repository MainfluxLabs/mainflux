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
	saveGroupMembers         = "save_group_members"
	updateGroupMembers       = "update_group_members"
	removeGroupMembers       = "remove_group_members"
	retrieveRole             = "retrieve_role"
	retrieveByGroup          = "retrieve_by_group"
	retrieveGroupIDsByMember = "retrieve_group_ids_by_member"
	retrieveAll              = "retrieve_all"
)

var _ things.GroupMembersRepository = (*groupMembersRepositoryMiddleware)(nil)

type groupMembersRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   things.GroupMembersRepository
}

// GroupMembersRepositoryMiddleware tracks request and their latency, and adds spans to context.
func GroupMembersRepositoryMiddleware(tracer opentracing.Tracer, pr things.GroupMembersRepository) things.GroupMembersRepository {
	return groupMembersRepositoryMiddleware{
		tracer: tracer,
		repo:   pr,
	}
}

func (prm groupMembersRepositoryMiddleware) Save(ctx context.Context, gms ...things.GroupMember) error {
	span := createSpan(ctx, prm.tracer, saveGroupMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.Save(ctx, gms...)
}

func (prm groupMembersRepositoryMiddleware) RetrieveRole(ctx context.Context, gp things.GroupMember) (string, error) {
	span := createSpan(ctx, prm.tracer, retrieveRole)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.RetrieveRole(ctx, gp)
}

func (prm groupMembersRepositoryMiddleware) RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (things.GroupMembersPage, error) {
	span := createSpan(ctx, prm.tracer, retrieveByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.RetrieveByGroup(ctx, groupID, pm)
}

func (prm groupMembersRepositoryMiddleware) RetrieveGroupIDsByMember(ctx context.Context, memberID string) ([]string, error) {
	span := createSpan(ctx, prm.tracer, retrieveGroupIDsByMember)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.RetrieveGroupIDsByMember(ctx, memberID)
}

func (prm groupMembersRepositoryMiddleware) RetrieveAll(ctx context.Context) ([]things.GroupMember, error) {
	span := createSpan(ctx, prm.tracer, retrieveAll)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.RetrieveAll(ctx)
}

func (prm groupMembersRepositoryMiddleware) Update(ctx context.Context, gms ...things.GroupMember) error {
	span := createSpan(ctx, prm.tracer, updateGroupMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.Update(ctx, gms...)
}

func (prm groupMembersRepositoryMiddleware) Remove(ctx context.Context, groupID string, memberIDs ...string) error {
	span := createSpan(ctx, prm.tracer, removeGroupMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.Remove(ctx, groupID, memberIDs...)
}
