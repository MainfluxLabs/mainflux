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
		message := fmt.Sprintf("Method revoke for key id %s took %s to complete", id, time.Since(begin))
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
		message := fmt.Sprintf("Method retrieve for key id %s took %s to complete", id, time.Since(begin))
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
		message := fmt.Sprintf("Method get_owner_id_by_org_id for org id %s took %s to complete", orgID, time.Since(begin))
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

func (lm *loggingMiddleware) CreateOrgInvite(ctx context.Context, token, email, role, orgID, invRedirectPath string) (invite auth.OrgInvite, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_org_invite for org id %s, role %s and user email %s took %s to complete", orgID, role, email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateOrgInvite(ctx, token, email, role, orgID, invRedirectPath)
}

func (lm *loggingMiddleware) CreateDormantOrgInvite(ctx context.Context, token, orgID, role, platformInviteID string) (invite auth.OrgInvite, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_dormant_org_invite for org id %s, role %s and platform invite id %s took %s to complete",
			orgID, role, platformInviteID, time.Since(begin))

		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateDormantOrgInvite(ctx, token, orgID, role, platformInviteID)
}

func (lm *loggingMiddleware) RevokeOrgInvite(ctx context.Context, token, inviteID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method revoke_org_invite for invite id %s took %s to complete", inviteID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RevokeOrgInvite(ctx, token, inviteID)
}

func (lm *loggingMiddleware) RespondOrgInvite(ctx context.Context, token, inviteID string, accept bool) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method respond_org_invite for invite id %s and accept %t took %s to complete", inviteID, accept, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RespondOrgInvite(ctx, token, inviteID, accept)
}

func (lm *loggingMiddleware) ListOrgInvitesByUser(ctx context.Context, token, userType, userID string, pm auth.PageMetadataInvites) (invPage auth.OrgInvitesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_org_invites_by_user for type %s and user id %s took %s to complete", userType, userID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListOrgInvitesByUser(ctx, token, userType, userID, pm)
}

func (lm *loggingMiddleware) ListOrgInvitesByOrg(ctx context.Context, token string, orgID string, pm auth.PageMetadataInvites) (invPage auth.OrgInvitesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_org_invites_by_org for org id %s took %s to complete", orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListOrgInvitesByOrg(ctx, token, orgID, pm)
}

func (lm *loggingMiddleware) SendOrgInviteEmail(ctx context.Context, invite auth.OrgInvite, email, orgName, invRedirectPath string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method send_org_invite_email for invite id %s and user email %s took %s to complete", invite.ID, email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.SendOrgInviteEmail(ctx, invite, email, orgName, invRedirectPath)
}

func (lm *loggingMiddleware) ViewOrgInvite(ctx context.Context, token, inviteID string) (invite auth.OrgInvite, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_org_invite %s for invite id took %s to complete", inviteID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewOrgInvite(ctx, token, inviteID)
}

func (lm *loggingMiddleware) ActivateOrgInvite(ctx context.Context, platformInviteID, userID, orgInviteRedirectPath string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method activate_org_invite for platform invite id %s and user id %s took %s to complete",
			platformInviteID, userID, time.Since(begin))

		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ActivateOrgInvite(ctx, platformInviteID, userID, orgInviteRedirectPath)
}
