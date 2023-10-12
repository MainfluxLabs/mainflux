// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/go-kit/kit/metrics"
)

var _ auth.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     auth.Service
}

// MetricsMiddleware instruments core service by tracking request count and latency.
func MetricsMiddleware(svc auth.Service, counter metrics.Counter, latency metrics.Histogram) auth.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) Issue(ctx context.Context, token string, key auth.Key) (auth.Key, string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "issue_key").Add(1)
		ms.latency.With("method", "issue_key").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Issue(ctx, token, key)
}

func (ms *metricsMiddleware) Revoke(ctx context.Context, token, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "revoke_key").Add(1)
		ms.latency.With("method", "revoke_key").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Revoke(ctx, token, id)
}

func (ms *metricsMiddleware) RetrieveKey(ctx context.Context, token, id string) (auth.Key, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "retrieve_key").Add(1)
		ms.latency.With("method", "retrieve_key").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RetrieveKey(ctx, token, id)
}

func (ms *metricsMiddleware) Identify(ctx context.Context, token string) (auth.Identity, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "identify").Add(1)
		ms.latency.With("method", "identify").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Identify(ctx, token)
}

func (ms *metricsMiddleware) Authorize(ctx context.Context, ar auth.AuthzReq) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "authorize").Add(1)
		ms.latency.With("method", "authorize").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Authorize(ctx, ar)
}

func (ms *metricsMiddleware) CreateOrg(ctx context.Context, token string, org auth.Org) (auth.Org, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_org").Add(1)
		ms.latency.With("method", "create_org").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.CreateOrg(ctx, token, org)
}

func (ms *metricsMiddleware) UpdateOrg(ctx context.Context, token string, org auth.Org) (auth.Org, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_org").Add(1)
		ms.latency.With("method", "update_org").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateOrg(ctx, token, org)
}

func (ms *metricsMiddleware) RemoveOrg(ctx context.Context, token string, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_org").Add(1)
		ms.latency.With("method", "remove_org").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.RemoveOrg(ctx, token, id)
}

func (ms *metricsMiddleware) ViewOrg(ctx context.Context, token, id string) (auth.Org, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_org").Add(1)
		ms.latency.With("method", "view_org").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewOrg(ctx, token, id)
}

func (ms *metricsMiddleware) ListOrgs(ctx context.Context, token string, admin bool, pm auth.PageMetadata) (auth.OrgsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_orgs").Add(1)
		ms.latency.With("method", "list_orgs").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListOrgs(ctx, token, admin, pm)
}

func (ms *metricsMiddleware) ListOrgMemberships(ctx context.Context, token, memberID string, pm auth.PageMetadata) (auth.OrgsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_org_memberships").Add(1)
		ms.latency.With("method", "list_org_memberships").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListOrgMemberships(ctx, token, memberID, pm)
}

func (ms *metricsMiddleware) AssignMembersByIDs(ctx context.Context, token, orgID string, memberIDs ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "assign_members_by_ids").Add(1)
		ms.latency.With("method", "assign_members_by_ids").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.AssignMembersByIDs(ctx, token, orgID, memberIDs...)
}

func (ms *metricsMiddleware) AssignMembers(ctx context.Context, token, orgID string, members ...auth.Member) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "assign_members").Add(1)
		ms.latency.With("method", "assign_members").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.AssignMembers(ctx, token, orgID, members...)
}

func (ms *metricsMiddleware) UnassignMembersByIDs(ctx context.Context, token, orgID string, memberIDs ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "unassign_members_by_ids").Add(1)
		ms.latency.With("method", "unassign_members_by_ids").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UnassignMembersByIDs(ctx, token, orgID, memberIDs...)
}

func (ms *metricsMiddleware) UnassignMembers(ctx context.Context, token, orgID string, memberIDs ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "unassign_members").Add(1)
		ms.latency.With("method", "unassign_members").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UnassignMembers(ctx, token, orgID, memberIDs...)
}

func (ms *metricsMiddleware) UpdateMembers(ctx context.Context, token, orgID string, members ...auth.Member) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_members").Add(1)
		ms.latency.With("method", "update_members").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateMembers(ctx, token, orgID, members...)
}

func (ms *metricsMiddleware) ViewMember(ctx context.Context, token, orgID, memberID string) (auth.Member, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_member").Add(1)
		ms.latency.With("method", "view_member").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewMember(ctx, token, orgID, memberID)
}

func (ms *metricsMiddleware) ListOrgMembers(ctx context.Context, token, orgID string, pm auth.PageMetadata) (auth.MembersPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_org_members").Add(1)
		ms.latency.With("method", "list_org_members").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListOrgMembers(ctx, token, orgID, pm)
}

func (ms *metricsMiddleware) AssignGroups(ctx context.Context, token, orgID string, groupIDs ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "assign_groups").Add(1)
		ms.latency.With("method", "assign_groups").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.AssignGroups(ctx, token, orgID, groupIDs...)
}

func (ms *metricsMiddleware) UnassignGroups(ctx context.Context, token, orgID string, groupIDs ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "unassign_groups").Add(1)
		ms.latency.With("method", "unassign_groups").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UnassignGroups(ctx, token, orgID, groupIDs...)
}

func (ms *metricsMiddleware) ListOrgGroups(ctx context.Context, token, groupID string, pm auth.PageMetadata) (auth.GroupsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_org_groups").Add(1)
		ms.latency.With("method", "list_org_groups").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListOrgGroups(ctx, token, groupID, pm)
}

func (ms *metricsMiddleware) AddPolicy(ctx context.Context, token, groupID, policy string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "add_policy").Add(1)
		ms.latency.With("method", "add_policy").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.AddPolicy(ctx, token, groupID, policy)
}

func (ms *metricsMiddleware) Backup(ctx context.Context, token string) (auth.Backup, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "backup").Add(1)
		ms.latency.With("method", "backup").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Backup(ctx, token)
}

func (ms *metricsMiddleware) Restore(ctx context.Context, token string, backup auth.Backup) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "restore").Add(1)
		ms.latency.With("method", "restore").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Restore(ctx, token, backup)
}

func (ms *metricsMiddleware) AssignRole(ctx context.Context, id, role string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "assign_role").Add(1)
		ms.latency.With("method", "assign_role").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.AssignRole(ctx, id, role)
}
