// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/things"
)

var _ things.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    things.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc things.Service, logger log.Logger) things.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) CreateThings(ctx context.Context, token, profileID string, ths ...things.Thing) (saved []things.Thing, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_things for things %s took %s to complete", saved, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateThings(ctx, token, profileID, ths...)
}

func (lm *loggingMiddleware) UpdateThing(ctx context.Context, token string, thing things.Thing) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_thing for thing id %s took %s to complete", thing.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateThing(ctx, token, thing)
}

func (lm *loggingMiddleware) UpdateThingsMetadata(ctx context.Context, token string, things ...things.Thing) (err error) {
	defer func(begin time.Time) {
		var ids []string
		for _, thing := range things {
			ids = append(ids, thing.ID)
		}
		message := fmt.Sprintf("Method update_things_metadata for thing ids %s took %s to complete", ids, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateThingsMetadata(ctx, token, things...)
}

func (lm *loggingMiddleware) UpdateThingGroupAndProfile(ctx context.Context, token string, thing things.Thing) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_thing_group_and_profile for thing id %s took %s to complete", thing.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateThingGroupAndProfile(ctx, token, thing)
}

func (lm *loggingMiddleware) ViewThing(ctx context.Context, token, id string) (_ things.Thing, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_thing for thing id %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewThing(ctx, token, id)
}

func (lm *loggingMiddleware) ViewMetadataByKey(ctx context.Context, key things.ThingKey) (_ things.Metadata, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_metadata_by_key took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewMetadataByKey(ctx, key)
}

func (lm *loggingMiddleware) ListThings(ctx context.Context, token string, pm apiutil.PageMetadata) (_ things.ThingsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_things took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListThings(ctx, token, pm)
}

func (lm *loggingMiddleware) ListThingsByProfile(ctx context.Context, token, prID string, pm apiutil.PageMetadata) (_ things.ThingsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_things_by_profile for profile id %s took %s to complete", prID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListThingsByProfile(ctx, token, prID, pm)
}

func (lm *loggingMiddleware) ListThingsByOrg(ctx context.Context, token string, orgID string, pm apiutil.PageMetadata) (_ things.ThingsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_things_by_org for org id %s took %s to complete", orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListThingsByOrg(ctx, token, orgID, pm)
}

func (lm *loggingMiddleware) RemoveThings(ctx context.Context, token string, ids ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_things for thing ids %v took %s to complete", ids, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveThings(ctx, token, ids...)
}

func (lm *loggingMiddleware) CreateProfiles(ctx context.Context, token, grID string, profiles ...things.Profile) (saved []things.Profile, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_profiles for profiles %s took %s to complete", saved, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateProfiles(ctx, token, grID, profiles...)
}

func (lm *loggingMiddleware) UpdateProfile(ctx context.Context, token string, profile things.Profile) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_profile for profile id %s took %s to complete", profile.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateProfile(ctx, token, profile)
}

func (lm *loggingMiddleware) ViewProfile(ctx context.Context, token, id string) (_ things.Profile, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_profile for profile id %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewProfile(ctx, token, id)
}

func (lm *loggingMiddleware) ListProfiles(ctx context.Context, token string, pm apiutil.PageMetadata) (_ things.ProfilesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_profiles took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListProfiles(ctx, token, pm)
}

func (lm *loggingMiddleware) ListProfilesByOrg(ctx context.Context, token string, orgID string, pm apiutil.PageMetadata) (_ things.ProfilesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_profiles_by_org for org id %s took %s to complete", orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListProfilesByOrg(ctx, token, orgID, pm)
}

func (lm *loggingMiddleware) ViewProfileByThing(ctx context.Context, token, thID string) (_ things.Profile, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_profile_by_thing for thing id %s took %s to complete", thID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewProfileByThing(ctx, token, thID)
}

func (lm *loggingMiddleware) RemoveProfiles(ctx context.Context, token string, ids ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_profiles for profile ids %v took %s to complete", ids, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveProfiles(ctx, token, ids...)
}

func (lm *loggingMiddleware) GetPubConfigByKey(ctx context.Context, key things.ThingKey) (pc things.PubConfigInfo, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method get_pub_config_by_key for thing id %s took %s to complete", pc.PublisherID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.GetPubConfigByKey(ctx, key)
}

func (lm *loggingMiddleware) GetConfigByThing(ctx context.Context, thingID string) (_ map[string]any, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method get_config_by_thing for thing id %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.GetConfigByThing(ctx, thingID)
}

func (lm *loggingMiddleware) CanUserAccessThing(ctx context.Context, req things.UserAccessReq) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method can_user_access_thing for thing id %s took %s to complete", req.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CanUserAccessThing(ctx, req)
}

func (lm *loggingMiddleware) CanUserAccessProfile(ctx context.Context, req things.UserAccessReq) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method can_user_access_profile for profile id %s took %s to complete", req.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CanUserAccessProfile(ctx, req)
}

func (lm *loggingMiddleware) CanUserAccessGroup(ctx context.Context, req things.UserAccessReq) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method can_user_access_group for group id %s took %s to complete", req.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CanUserAccessGroup(ctx, req)
}

func (lm *loggingMiddleware) CanThingAccessGroup(ctx context.Context, req things.ThingAccessReq) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method can_thing_access_group for group id %s took %s to complete", req.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CanThingAccessGroup(ctx, req)
}

func (lm *loggingMiddleware) Identify(ctx context.Context, key things.ThingKey) (id string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method identify for thing id %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Identify(ctx, key)
}

func (lm *loggingMiddleware) GetGroupIDByThing(ctx context.Context, thingID string) (_ string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method get_group_id_by_thing for thing id %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.GetGroupIDByThing(ctx, thingID)
}

func (lm *loggingMiddleware) GetGroupIDByProfile(ctx context.Context, profileID string) (_ string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method get_group_id_by_profile for profile id %s took %s to complete", profileID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.GetGroupIDByProfile(ctx, profileID)
}

func (lm *loggingMiddleware) GetGroupIDsByOrg(ctx context.Context, orgID string, token string) (_ []string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method get_group_ids_by_org for org id %s took %s to complete", orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.GetGroupIDsByOrg(ctx, orgID, token)
}

func (lm *loggingMiddleware) Backup(ctx context.Context, token string) (_ things.Backup, err error) {
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

func (lm *loggingMiddleware) Restore(ctx context.Context, token string, backup things.Backup) (err error) {
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

func (lm *loggingMiddleware) CreateGroups(ctx context.Context, token, orgID string, grs ...things.Group) (saved []things.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_groups for groups %s took %s to complete", saved, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateGroups(ctx, token, orgID, grs...)
}

func (lm *loggingMiddleware) UpdateGroup(ctx context.Context, token string, gr things.Group) (_ things.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_group for group id %s took %s to complete", gr.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateGroup(ctx, token, gr)
}

func (lm *loggingMiddleware) ViewGroup(ctx context.Context, token, id string) (_ things.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_group for group id %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewGroup(ctx, token, id)
}

func (lm *loggingMiddleware) ViewGroupInternal(ctx context.Context, id string) (_ things.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_group_internal for id %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewGroupInternal(ctx, id)
}

func (lm *loggingMiddleware) ListGroups(ctx context.Context, token string, pm apiutil.PageMetadata) (_ things.GroupPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_groups took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListGroups(ctx, token, pm)
}

func (lm *loggingMiddleware) ListGroupsByOrg(ctx context.Context, token, orgID string, pm apiutil.PageMetadata) (_ things.GroupPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_groups_by_org for org id %s took %s to complete", orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListGroupsByOrg(ctx, token, orgID, pm)
}

func (lm *loggingMiddleware) ListThingsByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (_ things.ThingsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_things_by_group for group id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListThingsByGroup(ctx, token, groupID, pm)
}

func (lm *loggingMiddleware) ViewGroupByThing(ctx context.Context, token, thingID string) (_ things.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_group_by_thing for thing id %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewGroupByThing(ctx, token, thingID)
}

func (lm *loggingMiddleware) RemoveGroups(ctx context.Context, token string, ids ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_groups for group ids %v took %s to complete", ids, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveGroups(ctx, token, ids...)
}

func (lm *loggingMiddleware) RemoveGroupsByOrg(ctx context.Context, orgID string) (_ []string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_groups_by_org for org id %s took %s to complete", orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveGroupsByOrg(ctx, orgID)
}

func (lm *loggingMiddleware) ViewGroupByProfile(ctx context.Context, token, profileID string) (_ things.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_group_by_profile for profile id %s took %s to complete", profileID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewGroupByProfile(ctx, token, profileID)
}

func (lm *loggingMiddleware) ListProfilesByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (_ things.ProfilesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_profiles_by_group for group id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListProfilesByGroup(ctx, token, groupID, pm)
}

func (lm *loggingMiddleware) CreateGroupMemberships(ctx context.Context, token string, gms ...things.GroupMembership) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_group_memberships for memberships %v took %s to complete", gms, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateGroupMemberships(ctx, token, gms...)
}

func (lm *loggingMiddleware) CreateGroupMembershipsInternal(ctx context.Context, gms ...things.GroupMembership) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_group_memberships_internal took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateGroupMembershipsInternal(ctx, gms...)
}

func (lm *loggingMiddleware) ListGroupMemberships(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (_ things.GroupMembershipsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_group_memberships for group id %s and email %s took %s to complete", groupID, pm.Email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListGroupMemberships(ctx, token, groupID, pm)
}

func (lm *loggingMiddleware) UpdateGroupMemberships(ctx context.Context, token string, gms ...things.GroupMembership) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_group_memberships took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateGroupMemberships(ctx, token, gms...)
}

func (lm *loggingMiddleware) RemoveGroupMemberships(ctx context.Context, token, groupID string, memberIDs ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_group_memberships for group id %s and member ids %v took %s to complete", groupID, memberIDs, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveGroupMemberships(ctx, token, groupID, memberIDs...)
}

func (lm *loggingMiddleware) UpdateExternalKey(ctx context.Context, token, key, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_external_key for thing id %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateExternalKey(ctx, token, key, thingID)
}

func (lm *loggingMiddleware) RemoveExternalKey(ctx context.Context, token, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_external_key for thing id %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveExternalKey(ctx, token, thingID)
}

func (lm *loggingMiddleware) GetThingIDsByProfile(ctx context.Context, profileID string) (_ []string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method get_thing_ids_by_profile for profile id %s took %s to complete", profileID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.GetThingIDsByProfile(ctx, profileID)
}
