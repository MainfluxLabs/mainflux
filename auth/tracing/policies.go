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
	saveGroupMembers      = "save_group_policies"
	removeGroupMembers    = "remove_group_policies"
	retrieveGroupMember   = "retrieve_group_policy"
	RetrieveGroupPolicies = "retrieve_group_policies"
	updateGroupMembers    = "update_group_policies"
)

var _ auth.PoliciesRepository = (*policiesRepositoryMiddleware)(nil)

type policiesRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   auth.PoliciesRepository
}

// PoliciesRepositoryMiddleware tracks request and their latency, and adds spans to context.
func PoliciesRepositoryMiddleware(tracer opentracing.Tracer, pr auth.PoliciesRepository) auth.PoliciesRepository {
	return policiesRepositoryMiddleware{
		tracer: tracer,
		repo:   pr,
	}
}

func (mrm policiesRepositoryMiddleware) SaveGroupPolicies(ctx context.Context, groupID string, gps ...auth.GroupPolicyByID) error {
	span := createSpan(ctx, mrm.tracer, saveGroupMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return mrm.repo.SaveGroupPolicies(ctx, groupID, gps...)
}

func (mrm policiesRepositoryMiddleware) RetrieveGroupPolicy(ctx context.Context, gp auth.GroupsPolicy) (string, error) {
	span := createSpan(ctx, mrm.tracer, retrieveGroupMember)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return mrm.repo.RetrieveGroupPolicy(ctx, gp)
}

func (mrm policiesRepositoryMiddleware) RetrieveGroupPolicies(ctx context.Context, groupID string, pm auth.PageMetadata) (auth.GroupPoliciesPage, error) {
	span := createSpan(ctx, mrm.tracer, RetrieveGroupPolicies)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return mrm.repo.RetrieveGroupPolicies(ctx, groupID, pm)
}

func (mrm policiesRepositoryMiddleware) UpdateGroupPolicies(ctx context.Context, groupID string, gps ...auth.GroupPolicyByID) error {
	span := createSpan(ctx, mrm.tracer, updateGroupMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return mrm.repo.UpdateGroupPolicies(ctx, groupID, gps...)
}

func (mrm policiesRepositoryMiddleware) RemoveGroupPolicies(ctx context.Context, groupID string, memberIDs ...string) error {
	span := createSpan(ctx, mrm.tracer, removeGroupMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return mrm.repo.RemoveGroupPolicies(ctx, groupID, memberIDs...)
}
