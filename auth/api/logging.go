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

func (lm *loggingMiddleware) RemoveOrg(ctx context.Context, token string, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_org for id %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveOrg(ctx, token, id)
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

func (lm *loggingMiddleware) ListOrgs(ctx context.Context, token string, pm auth.PageMetadata) (gp auth.OrgsPage, err error) {
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

func (lm *loggingMiddleware) ViewMember(ctx context.Context, token, orgID, memberID string) (om auth.OrgMember, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_member for org id %s and member id %s took %s to complete", orgID, memberID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewMember(ctx, token, orgID, memberID)
}

func (lm *loggingMiddleware) ListMembersByOrg(ctx context.Context, token, orgID string, pm auth.PageMetadata) (op auth.OrgMembersPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_members_by_org for org id %s took %s to complete", orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListMembersByOrg(ctx, token, orgID, pm)
}

func (lm *loggingMiddleware) ListOrgsByMember(ctx context.Context, token, memberID string, pm auth.PageMetadata) (op auth.OrgsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_orgs_by_member for member id %s took %s to complete", memberID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListOrgsByMember(ctx, token, memberID, pm)
}

func (lm *loggingMiddleware) AssignMembers(ctx context.Context, token, orgID string, oms ...auth.OrgMember) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method assign_members for members %s and org id %s took %s to complete", oms, orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.AssignMembers(ctx, token, orgID, oms...)
}

func (lm *loggingMiddleware) UnassignMembers(ctx context.Context, token string, orgID string, memberIDs ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method unassign_members for member ids %s and org id %s took %s to complete", memberIDs, orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UnassignMembers(ctx, token, orgID, memberIDs...)
}

func (lm *loggingMiddleware) UpdateMembers(ctx context.Context, token, orgID string, oms ...auth.OrgMember) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_members for members %s and org id %s took %s to complete", oms, orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateMembers(ctx, token, orgID, oms...)
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
