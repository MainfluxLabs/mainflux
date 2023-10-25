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
	saveGroupMembers     = "save_group_members"
	removeGroupMembers   = "remove_group_members"
	retrieveGroupMember  = "retrieve_group_member"
	RetrieveGroupMembers = "retrieve_group_members"
	updateGroupMembers   = "update_group_members"
)

var _ auth.MembersRepository = (*membersRepositoryMiddleware)(nil)

type membersRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   auth.MembersRepository
}

// MembersRepositoryMiddleware tracks request and their latency, and adds spans to context.
func MembersRepositoryMiddleware(tracer opentracing.Tracer, pr auth.MembersRepository) auth.MembersRepository {
	return membersRepositoryMiddleware{
		tracer: tracer,
		repo:   pr,
	}
}

func (mrm membersRepositoryMiddleware) SaveGroupMembers(ctx context.Context, groupID string, giByIDs ...auth.GroupInvitationByID) error {
	span := createSpan(ctx, mrm.tracer, saveGroupMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return mrm.repo.SaveGroupMembers(ctx, groupID, giByIDs...)
}

func (mrm membersRepositoryMiddleware) RetrieveGroupMember(ctx context.Context, gp auth.GroupsPolicy) (string, error) {
	span := createSpan(ctx, mrm.tracer, retrieveGroupMember)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return mrm.repo.RetrieveGroupMember(ctx, gp)
}

func (mrm membersRepositoryMiddleware) RetrieveGroupMembers(ctx context.Context, groupID string, pm auth.PageMetadata) (auth.GroupMembersPage, error) {
	span := createSpan(ctx, mrm.tracer, RetrieveGroupMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return mrm.repo.RetrieveGroupMembers(ctx, groupID, pm)
}

func (mrm membersRepositoryMiddleware) UpdateGroupMembers(ctx context.Context, groupID string, giByIDs ...auth.GroupInvitationByID) error {
	span := createSpan(ctx, mrm.tracer, updateGroupMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return mrm.repo.UpdateGroupMembers(ctx, groupID, giByIDs...)
}

func (mrm membersRepositoryMiddleware) RemoveGroupMembers(ctx context.Context, groupID string, memberIDs ...string) error {
	span := createSpan(ctx, mrm.tracer, removeGroupMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return mrm.repo.RemoveGroupMembers(ctx, groupID, memberIDs...)
}
