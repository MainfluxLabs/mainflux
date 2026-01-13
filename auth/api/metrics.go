// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
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

func (ms *metricsMiddleware) RemoveOrgs(ctx context.Context, token string, ids ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_orgs").Add(1)
		ms.latency.With("method", "remove_orgs").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.RemoveOrgs(ctx, token, ids...)
}

func (ms *metricsMiddleware) ViewOrg(ctx context.Context, token, id string) (auth.Org, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_org").Add(1)
		ms.latency.With("method", "view_org").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewOrg(ctx, token, id)
}

func (ms *metricsMiddleware) ListOrgs(ctx context.Context, token string, pm apiutil.PageMetadata) (auth.OrgsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_orgs").Add(1)
		ms.latency.With("method", "list_orgs").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListOrgs(ctx, token, pm)
}

func (ms *metricsMiddleware) GetOwnerIDByOrgID(ctx context.Context, orgID string) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "get_owner_id_by_org_id").Add(1)
		ms.latency.With("method", "get_owner_id_by_org_id").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.GetOwnerIDByOrgID(ctx, orgID)
}

func (ms *metricsMiddleware) CreateOrgMemberships(ctx context.Context, token, orgID string, oms ...auth.OrgMembership) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_org_memberships").Add(1)
		ms.latency.With("method", "create_org_memberships").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateOrgMemberships(ctx, token, orgID, oms...)
}

func (ms *metricsMiddleware) ViewOrgMembership(ctx context.Context, token, orgID, memberID string) (auth.OrgMembership, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_org_membership").Add(1)
		ms.latency.With("method", "view_org_membership").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewOrgMembership(ctx, token, orgID, memberID)
}

func (ms *metricsMiddleware) ListOrgMemberships(ctx context.Context, token, orgID string, pm apiutil.PageMetadata) (auth.OrgMembershipsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_org_memberships").Add(1)
		ms.latency.With("method", "list_org_memberships").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListOrgMemberships(ctx, token, orgID, pm)
}

func (ms *metricsMiddleware) UpdateOrgMemberships(ctx context.Context, token, orgID string, oms ...auth.OrgMembership) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_org_memberships").Add(1)
		ms.latency.With("method", "update_org_memberships").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateOrgMemberships(ctx, token, orgID, oms...)
}

func (ms *metricsMiddleware) RemoveOrgMemberships(ctx context.Context, token, orgID string, memberIDs ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_org_memberships").Add(1)
		ms.latency.With("method", "remove_org_memberships").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveOrgMemberships(ctx, token, orgID, memberIDs...)
}

func (ms *metricsMiddleware) Backup(ctx context.Context, token string) (auth.Backup, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "backup").Add(1)
		ms.latency.With("method", "backup").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Backup(ctx, token)
}

func (ms *metricsMiddleware) BackupOrgMemberships(ctx context.Context, token string, orgID string) (auth.OrgMembershipsBackup, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "backup_org_memberships").Add(1)
		ms.latency.With("method", "backup_org_memberships").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.BackupOrgMemberships(ctx, token, orgID)
}

func (ms *metricsMiddleware) RestoreOrgMemberships(ctx context.Context, token string, orgID string, backup auth.OrgMembershipsBackup) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "restore_org_memberships").Add(1)
		ms.latency.With("method", "restore_org_memberships").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RestoreOrgMemberships(ctx, token, orgID, backup)
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

func (ms *metricsMiddleware) RetrieveRole(ctx context.Context, id string) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "retrieve_role").Add(1)
		ms.latency.With("method", "retrieve_role").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RetrieveRole(ctx, id)
}

func (ms *metricsMiddleware) CreateOrgInvite(ctx context.Context, token, email, role, orgID string, groups []auth.GroupInvite, invRedirectPath string) (auth.OrgInvite, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_org_invite").Add(1)
		ms.latency.With("method", "create_org_invite").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateOrgInvite(ctx, token, email, role, orgID, groups, invRedirectPath)
}

func (ms *metricsMiddleware) CreateDormantOrgInvite(ctx context.Context, token, orgID, role string, groups []auth.GroupInvite, platformInviteID string) (auth.OrgInvite, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_dormant_org_invite").Add(1)
		ms.latency.With("method", "create_dormant_org_invite").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateDormantOrgInvite(ctx, token, orgID, role, groups, platformInviteID)
}

func (ms *metricsMiddleware) ViewOrgInvite(ctx context.Context, token, inviteID string) (auth.OrgInvite, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_org_invite").Add(1)
		ms.latency.With("method", "view_org_invite").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewOrgInvite(ctx, token, inviteID)
}

func (ms *metricsMiddleware) RevokeOrgInvite(ctx context.Context, token, inviteID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "revoke_org_invite").Add(1)
		ms.latency.With("method", "revoke_org_invite").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RevokeOrgInvite(ctx, token, inviteID)
}

func (ms *metricsMiddleware) RespondOrgInvite(ctx context.Context, token, inviteID string, accept bool) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "respond_org_invite").Add(1)
		ms.latency.With("method", "respond_org_invite").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RespondOrgInvite(ctx, token, inviteID, accept)
}

func (ms *metricsMiddleware) ListOrgInvitesByUser(ctx context.Context, token, userType, userID string, pm auth.PageMetadataInvites) (auth.OrgInvitesPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_org_invites_by_user").Add(1)
		ms.latency.With("method", "list_org_invites_by_user").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListOrgInvitesByUser(ctx, token, userType, userID, pm)
}

func (ms *metricsMiddleware) ListOrgInvitesByOrg(ctx context.Context, token, orgID string, pm auth.PageMetadataInvites) (auth.OrgInvitesPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_org_invites_by_org").Add(1)
		ms.latency.With("method", "list_org_invites_by_org").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListOrgInvitesByOrg(ctx, token, orgID, pm)
}

func (ms *metricsMiddleware) SendOrgInviteEmail(ctx context.Context, invite auth.OrgInvite, email, orgName, invRedirectPath string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "send_org_invite_email").Add(1)
		ms.latency.With("method", "send_org_invite_email").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.SendOrgInviteEmail(ctx, invite, email, orgName, invRedirectPath)
}

func (ms *metricsMiddleware) ActivateOrgInvite(ctx context.Context, platformInviteID, userID, orgInviteRedirectPath string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "activate_org_invite").Add(1)
		ms.latency.With("method", "activate_org_invite").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ActivateOrgInvite(ctx, platformInviteID, userID, orgInviteRedirectPath)
}
