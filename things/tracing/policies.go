// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package tracing contains middlewares that will add spans to existing traces.
package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/things"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveGroupMembers      = "save_group_policies"
	removeGroupMembers    = "remove_group_policies"
	retrieveGroupMember   = "retrieve_group_policy"
	RetrieveGroupPolicies = "retrieve_group_policies"
	updateGroupMembers    = "update_group_policies"
)

var _ things.PoliciesRepository = (*policiesRepositoryMiddleware)(nil)

type policiesRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   things.PoliciesRepository
}

// PoliciesRepositoryMiddleware tracks request and their latency, and adds spans to context.
func PoliciesRepositoryMiddleware(tracer opentracing.Tracer, pr things.PoliciesRepository) things.PoliciesRepository {
	return policiesRepositoryMiddleware{
		tracer: tracer,
		repo:   pr,
	}
}

func (prm policiesRepositoryMiddleware) SavePoliciesByGroup(ctx context.Context, groupID string, gps ...things.GroupPolicyByID) error {
	span := createSpan(ctx, prm.tracer, saveGroupMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.SavePoliciesByGroup(ctx, groupID, gps...)
}

func (prm policiesRepositoryMiddleware) RetrievePolicyByGroup(ctx context.Context, gp things.GroupPolicy) (string, error) {
	span := createSpan(ctx, prm.tracer, retrieveGroupMember)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.RetrievePolicyByGroup(ctx, gp)
}

func (prm policiesRepositoryMiddleware) RetrievePoliciesByGroup(ctx context.Context, groupID string, pm things.PageMetadata) (things.GroupPoliciesPage, error) {
	span := createSpan(ctx, prm.tracer, RetrieveGroupPolicies)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.RetrievePoliciesByGroup(ctx, groupID, pm)
}

func (prm policiesRepositoryMiddleware) RetrieveAllPoliciesByGroup(ctx context.Context) ([]things.GroupPolicy, error) {
	span := createSpan(ctx, prm.tracer, RetrieveGroupPolicies)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.RetrieveAllPoliciesByGroup(ctx)
}

func (prm policiesRepositoryMiddleware) UpdatePoliciesByGroup(ctx context.Context, groupID string, gps ...things.GroupPolicyByID) error {
	span := createSpan(ctx, prm.tracer, updateGroupMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.UpdatePoliciesByGroup(ctx, groupID, gps...)
}

func (prm policiesRepositoryMiddleware) RemovePoliciesByGroup(ctx context.Context, groupID string, memberIDs ...string) error {
	span := createSpan(ctx, prm.tracer, removeGroupMembers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.RemovePoliciesByGroup(ctx, groupID, memberIDs...)
}
