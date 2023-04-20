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
		message := fmt.Sprintf("Method create_org for token %s and name %s took %s to complete", token, org.Name, time.Since(begin))
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
		message := fmt.Sprintf("Method update_org for token %s and name %s took %s to complete", token, org.Name, time.Since(begin))
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
		message := fmt.Sprintf("Method remove_org for token %s and id %s took %s to complete", token, id, time.Since(begin))
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
		message := fmt.Sprintf("Method view_org for token %s and id %s took %s to complete", token, id, time.Since(begin))
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
		message := fmt.Sprintf("Method list_orgs for token %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListOrgs(ctx, token, pm)
}

func (lm *loggingMiddleware) ListOrgMembers(ctx context.Context, token, orgID string, pm auth.PageMetadata) (op auth.MembersPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_org_members for token %s and org id %s took %s to complete", token, orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListOrgMembers(ctx, token, orgID, pm)
}

func (lm *loggingMiddleware) ListOrgMemberships(ctx context.Context, token, memberID string, pm auth.PageMetadata) (op auth.OrgsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_org_memberships for token %s and member id %s took %s to complete", token, memberID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListOrgMemberships(ctx, token, memberID, pm)
}

func (lm *loggingMiddleware) AssignMembersByIDs(ctx context.Context, token, orgID string, memberIDs ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method assign_members_by_ids for token %s , member ids %s and org id %s took %s to complete", token, memberIDs, orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.AssignMembersByIDs(ctx, token, orgID, memberIDs...)
}

func (lm *loggingMiddleware) AssignMembers(ctx context.Context, token, orgID string, members ...auth.Member) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method assign_members for token %s , members %s and org id %s took %s to complete", token, members, orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.AssignMembers(ctx, token, orgID, members...)
}

func (lm *loggingMiddleware) UnassignMembersByIDs(ctx context.Context, token string, orgID string, memberIDs ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method unassign_members_by_ids for token %s , member ids %s and org id %s took %s to complete", token, memberIDs, orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UnassignMembersByIDs(ctx, token, orgID, memberIDs...)
}

func (lm *loggingMiddleware) UnassignMembers(ctx context.Context, token string, orgID string, memberEmails ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method unassign_members for token %s , member ids %s and org id %s took %s to complete", token, memberEmails, orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UnassignMembers(ctx, token, orgID, memberEmails...)
}

func (lm *loggingMiddleware) AssignGroups(ctx context.Context, token, orgID string, groupIDs ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method assign_groups for token %s , group ids %s and org id %s took %s to complete", token, groupIDs, orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.AssignGroups(ctx, token, orgID, groupIDs...)
}

func (lm *loggingMiddleware) UnassignGroups(ctx context.Context, token string, orgID string, groupIDs ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method unassign_groups for token %s, group ids %s and org id %s took %s to complete", token, groupIDs, orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UnassignGroups(ctx, token, orgID, groupIDs...)
}

func (lm *loggingMiddleware) ListOrgGroups(ctx context.Context, token, orgID string, pm auth.PageMetadata) (op auth.GroupsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_org_groups for token %s and org id %s took %s to complete", token, orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListOrgGroups(ctx, token, orgID, pm)
}

func (lm *loggingMiddleware) CanAccessGroup(ctx context.Context, token, orgID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method can_access_group for token %s and org id %s took %s to complete", token, orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CanAccessGroup(ctx, token, orgID)
}
