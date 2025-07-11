// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package tracing contains middlewares that will add spans to existing traces.
package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/opentracing/opentracing-go"
)

const (
	saveGroupMemberships       = "save_group_memberships"
	updateGroupMemberships     = "update_group_memberships"
	removeGroupMemberships     = "remove_group_memberships"
	retrieveMembershipsByGroup = "retrieve_memberships_by_group"
	backupAllGroupMemberships  = "backup_all_group_memberships"
	backupGroupMemberhips      = "backup_group_memberhips"
)

var _ things.GroupMembershipsRepository = (*groupMembershipsRepositoryMiddleware)(nil)

type groupMembershipsRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   things.GroupMembershipsRepository
}

// GroupMembershipsRepositoryMiddleware tracks requests and their latency, and adds spans to context.
func GroupMembershipsRepositoryMiddleware(tracer opentracing.Tracer, repo things.GroupMembershipsRepository) things.GroupMembershipsRepository {
	return groupMembershipsRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (gmr groupMembershipsRepositoryMiddleware) Save(ctx context.Context, gms ...things.GroupMembership) error {
	span := createSpan(ctx, gmr.tracer, saveGroupMemberships)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return gmr.repo.Save(ctx, gms...)
}

func (gmr groupMembershipsRepositoryMiddleware) RetrieveRole(ctx context.Context, gm things.GroupMembership) (string, error) {
	span := createSpan(ctx, gmr.tracer, retrieveRole)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return gmr.repo.RetrieveRole(ctx, gm)
}

func (gmr groupMembershipsRepositoryMiddleware) RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (things.GroupMembershipsPage, error) {
	span := createSpan(ctx, gmr.tracer, retrieveMembershipsByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return gmr.repo.RetrieveByGroup(ctx, groupID, pm)
}

func (gmr groupMembershipsRepositoryMiddleware) RetrieveGroupIDsByMember(ctx context.Context, memberID string) ([]string, error) {
	span := createSpan(ctx, gmr.tracer, retrieveGroupIDsByMember)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return gmr.repo.RetrieveGroupIDsByMember(ctx, memberID)
}

func (gmr groupMembershipsRepositoryMiddleware) BackupAll(ctx context.Context) ([]things.GroupMembership, error) {
	span := createSpan(ctx, gmr.tracer, backupAllGroupMemberships)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return gmr.repo.BackupAll(ctx)
}

func (gmr groupMembershipsRepositoryMiddleware) BackupByGroup(ctx context.Context, groupID string) ([]things.GroupMembership, error) {
	span := createSpan(ctx, gmr.tracer, backupGroupMemberhips)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return gmr.repo.BackupByGroup(ctx, groupID)
}

func (gmr groupMembershipsRepositoryMiddleware) Update(ctx context.Context, gms ...things.GroupMembership) error {
	span := createSpan(ctx, gmr.tracer, updateGroupMemberships)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return gmr.repo.Update(ctx, gms...)
}

func (gmr groupMembershipsRepositoryMiddleware) Remove(ctx context.Context, groupID string, memberIDs ...string) error {
	span := createSpan(ctx, gmr.tracer, removeGroupMemberships)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return gmr.repo.Remove(ctx, groupID, memberIDs...)
}
