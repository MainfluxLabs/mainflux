// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var _ auth.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    auth.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc auth.Service, logger log.Logger) auth.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) Issue(ctx context.Context, token string, newKey auth.Key) (key auth.Key, secret string, err error) {
	defer func(begin time.Time) {
		d := "infinite duration"
		if !key.ExpiresAt.IsZero() {
			d = fmt.Sprintf("the key with expiration date %v", key.ExpiresAt)
		}
		message := fmt.Sprintf("Method issue for %s took %s to complete", d, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Issue(ctx, token, newKey)
}

func (lm *loggingMiddleware) Revoke(ctx context.Context, token, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method revoke for key %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Revoke(ctx, token, id)
}

func (lm *loggingMiddleware) RetrieveKey(ctx context.Context, token, id string) (key auth.Key, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method retrieve for key %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RetrieveKey(ctx, token, id)
}

func (lm *loggingMiddleware) Identify(ctx context.Context, key string) (id auth.Identity, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method identify took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Identify(ctx, key)
}

func (lm *loggingMiddleware) Authorize(ctx context.Context, ar auth.AuthzReq) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method authorize took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.Authorize(ctx, ar)
}

func (lm *loggingMiddleware) CreateOrg(ctx context.Context, token string, org auth.Org) (o auth.Org, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_org for name %s took %s to complete", org.Name, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateOrg(ctx, token, org)
}

func (lm *loggingMiddleware) UpdateOrg(ctx context.Context, token string, org auth.Org) (o auth.Org, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_org for name %s took %s to complete", org.Name, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateOrg(ctx, token, org)
}

func (lm *loggingMiddleware) RemoveOrgs(ctx context.Context, token string, ids ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_orgs took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveOrgs(ctx, token, ids...)
}

func (lm *loggingMiddleware) ViewOrg(ctx context.Context, token, id string) (o auth.Org, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_org for id %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewOrg(ctx, token, id)
}

func (lm *loggingMiddleware) ListOrgs(ctx context.Context, token string, pm apiutil.PageMetadata) (gp auth.OrgsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_orgs took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListOrgs(ctx, token, pm)
}

func (lm *loggingMiddleware) GetOwnerIDByOrgID(ctx context.Context, orgID string) (ownerID string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method get_owner_id_by_org_id for id %s took %s to complete", orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.GetOwnerIDByOrgID(ctx, orgID)
}

func (lm *loggingMiddleware) CreateOrgMemberships(ctx context.Context, token, orgID string, oms ...auth.OrgMembership) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_org_memberships for org id %s took %s to complete", orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateOrgMemberships(ctx, token, orgID, oms...)
}

func (lm *loggingMiddleware) ViewOrgMembership(ctx context.Context, token, orgID, memberID string) (om auth.OrgMembership, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_org_membership for org id %s and member id %s took %s to complete", orgID, memberID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewOrgMembership(ctx, token, orgID, memberID)
}

func (lm *loggingMiddleware) ListOrgMemberships(ctx context.Context, token, orgID string, pm apiutil.PageMetadata) (op auth.OrgMembershipsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_org_memberships for org id %s took %s to complete", orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListOrgMemberships(ctx, token, orgID, pm)
}

func (lm *loggingMiddleware) UpdateOrgMemberships(ctx context.Context, token, orgID string, oms ...auth.OrgMembership) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_org_memberships for org id %s took %s to complete", orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateOrgMemberships(ctx, token, orgID, oms...)
}

func (lm *loggingMiddleware) RemoveOrgMemberships(ctx context.Context, token string, orgID string, memberIDs ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_org_memberships for member ids %s and org id %s took %s to complete", memberIDs, orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveOrgMemberships(ctx, token, orgID, memberIDs...)
}

func (lm *loggingMiddleware) Backup(ctx context.Context, token string) (backup auth.Backup, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method backup took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Backup(ctx, token)
}

func (lm *loggingMiddleware) BackupOrgMemberships(ctx context.Context, token string, orgID string) (backup auth.OrgMembershipsBackup, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method backup_org_memberships took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.BackupOrgMemberships(ctx, token, orgID)
}

func (lm *loggingMiddleware) RestoreOrgMemberships(ctx context.Context, token string, orgID string, backup auth.OrgMembershipsBackup) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method restore_org_memberships took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RestoreOrgMemberships(ctx, token, orgID, backup)
}

func (lm *loggingMiddleware) Restore(ctx context.Context, token string, backup auth.Backup) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method restore took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Restore(ctx, token, backup)
}

func (lm *loggingMiddleware) AssignRole(ctx context.Context, id, role string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method assign_role for id %s and role %s took %s to complete", id, role, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
	}(time.Now())

	return lm.svc.AssignRole(ctx, id, role)
}

func (lm *loggingMiddleware) RetrieveRole(ctx context.Context, id string) (role string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method retrieve_role for id %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
	}(time.Now())

	return lm.svc.RetrieveRole(ctx, id)
}

func (lm *loggingMiddleware) InviteOrgMember(ctx context.Context, token string, orgID string, invRedirectPath string, om auth.OrgMembership) (invite auth.OrgInvite, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method invite_org_member took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.InviteOrgMember(ctx, token, orgID, invRedirectPath, om)
}

func (lm *loggingMiddleware) RevokeOrgInvite(ctx context.Context, token string, inviteID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method revoke_org_invite took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RevokeOrgInvite(ctx, token, inviteID)
}

func (lm *loggingMiddleware) RespondOrgInvite(ctx context.Context, token string, inviteID string, accept bool) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method respond_org_invite took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RespondOrgInvite(ctx, token, inviteID, accept)
}

func (lm *loggingMiddleware) ListOrgInvitesByUser(ctx context.Context, token string, userType string, userID string, pm apiutil.PageMetadata) (invPage auth.OrgInvitesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_org_invites_by_user (%s) took %s to complete", userType, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListOrgInvitesByUser(ctx, token, userType, userID, pm)
}

func (lm *loggingMiddleware) ListOrgInvitesByOrgID(ctx context.Context, token string, orgID string, pm apiutil.PageMetadata) (invPage auth.OrgInvitesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_org_invites_by_org_id took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListOrgInvitesByOrgID(ctx, token, orgID, pm)
}

func (lm *loggingMiddleware) SendOrgInviteEmail(ctx context.Context, invite auth.OrgInvite, email string, orgName string, invRedirectPath string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method send_org_invite_email took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.SendOrgInviteEmail(ctx, invite, email, orgName, invRedirectPath)
}

func (lm *loggingMiddleware) ViewOrgInvite(ctx context.Context, token string, inviteID string) (invite auth.OrgInvite, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_org_invite took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewOrgInvite(ctx, token, inviteID)
}

func (lm *loggingMiddleware) InvitePlatformMember(ctx context.Context, token, redirectPath, email string) (invite auth.PlatformInvite, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method invite_platform_member took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.InvitePlatformMember(ctx, token, redirectPath, email)
}

func (lm *loggingMiddleware) RevokePlatformInvite(ctx context.Context, token, inviteID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method revoke_platform_invite took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RevokePlatformInvite(ctx, token, inviteID)
}

func (lm *loggingMiddleware) ViewPlatformInvite(ctx context.Context, token, inviteID string) (invite auth.PlatformInvite, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_platform_invite took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewPlatformInvite(ctx, token, inviteID)
}

func (lm *loggingMiddleware) ListPlatformInvites(ctx context.Context, token string, pm apiutil.PageMetadata) (invitesPage auth.PlatformInvitesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_platform_invites took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListPlatformInvites(ctx, token, pm)
}

func (lm *loggingMiddleware) ValidatePlatformInvite(ctx context.Context, inviteID string, email string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method validate_platform_invite took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ValidatePlatformInvite(ctx, inviteID, email)
}

func (lm *loggingMiddleware) SendPlatformInviteEmail(ctx context.Context, invite auth.PlatformInvite, redirectPath string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method send_platform_invite_email took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.SendPlatformInviteEmail(ctx, invite, redirectPath)
}
