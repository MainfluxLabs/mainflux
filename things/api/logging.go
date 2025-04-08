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

func (lm *loggingMiddleware) CreateThings(ctx context.Context, token string, ths ...things.Thing) (saved []things.Thing, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_things for things %s took %s to complete", saved, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateThings(ctx, token, ths...)
}

func (lm *loggingMiddleware) UpdateThing(ctx context.Context, token string, thing things.Thing) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_thing for id %s took %s to complete", thing.ID, time.Since(begin))
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
		message := fmt.Sprintf("Method update_things_metadata for ids %s took %s to complete", ids, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateThingsMetadata(ctx, token, things...)
}

func (lm *loggingMiddleware) UpdateKey(ctx context.Context, token, id, key string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_key for id %s and key %s took %s to complete", id, key, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateKey(ctx, token, id, key)
}

func (lm *loggingMiddleware) ViewThing(ctx context.Context, token, id string) (thing things.Thing, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_thing for id %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewThing(ctx, token, id)
}

func (lm *loggingMiddleware) ViewMetadataByKey(ctx context.Context, thingKey string) (metadata things.Metadata, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_metadata_by_key took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewMetadataByKey(ctx, thingKey)
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
		message := fmt.Sprintf("Method list_things_by_profile for id %s took %s to complete", prID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s", message, err))
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListThingsByProfile(ctx, token, prID, pm)
}

func (lm *loggingMiddleware) ListThingsByOrg(ctx context.Context, token string, orgID string, pm apiutil.PageMetadata) (tp things.ThingsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_things_by_org for id %s took %s to complete", orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s", message, err))
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListThingsByOrg(ctx, token, orgID, pm)
}

func (lm *loggingMiddleware) RemoveThings(ctx context.Context, token string, ids ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_things took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveThings(ctx, token, ids...)
}

func (lm *loggingMiddleware) CreateProfiles(ctx context.Context, token string, profiles ...things.Profile) (saved []things.Profile, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_profiles for profiles %s took %s to complete", saved, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateProfiles(ctx, token, profiles...)
}

func (lm *loggingMiddleware) UpdateProfile(ctx context.Context, token string, profile things.Profile) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_profile for id %s took %s to complete", profile.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateProfile(ctx, token, profile)
}

func (lm *loggingMiddleware) ViewProfile(ctx context.Context, token, id string) (profile things.Profile, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_profile for id %s took %s to complete", id, time.Since(begin))
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

func (lm *loggingMiddleware) ListProfilesByOrg(ctx context.Context, token string, orgID string, pm apiutil.PageMetadata) (prs things.ProfilesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_profiles_by_org for id %s took %s to complete", orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s", message, err))
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListProfilesByOrg(ctx, token, orgID, pm)
}

func (lm *loggingMiddleware) ViewProfileByThing(ctx context.Context, token, thID string) (_ things.Profile, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_profile_by_thing for id %s took %s to complete", thID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s", message, err))
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewProfileByThing(ctx, token, thID)
}

func (lm *loggingMiddleware) RemoveProfiles(ctx context.Context, token string, ids ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_profiles took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveProfiles(ctx, token, ids...)
}

func (lm *loggingMiddleware) GetPubConfByKey(ctx context.Context, key string) (pc things.PubConfInfo, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method get_pub_conf_by_key for thing %s took %s to complete", pc.PublisherID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.GetPubConfByKey(ctx, key)
}

func (lm *loggingMiddleware) GetConfigByThingID(ctx context.Context, thingID string) (config map[string]interface{}, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method get_config_by_thing_id for thing %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.GetConfigByThingID(ctx, thingID)
}

func (lm *loggingMiddleware) CanUserAccessThing(ctx context.Context, req things.UserAccessReq) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method can_user_access_thing took %s to complete", time.Since(begin))
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
		message := fmt.Sprintf("Method can_user_access_profile took %s to complete", time.Since(begin))
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
		message := fmt.Sprintf("Method can_user_access_group took %s to complete", time.Since(begin))
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
		message := fmt.Sprintf("Method can_thing_access_group took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CanThingAccessGroup(ctx, req)
}

func (lm *loggingMiddleware) Identify(ctx context.Context, key string) (id string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method identify for thing %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Identify(ctx, key)
}

func (lm *loggingMiddleware) GetGroupIDByThingID(ctx context.Context, thingID string) (_ string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method get_group_id_by_thing_id for thing %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.GetGroupIDByThingID(ctx, thingID)
}

func (lm *loggingMiddleware) Backup(ctx context.Context, token string) (bk things.Backup, err error) {
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

func (lm *loggingMiddleware) CreateGroups(ctx context.Context, token string, grs ...things.Group) (saved []things.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_groups for groups %s took %s to complete", saved, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateGroups(ctx, token, grs...)
}

func (lm *loggingMiddleware) UpdateGroup(ctx context.Context, token string, gr things.Group) (g things.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_group for id %s took %s to complete", gr.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateGroup(ctx, token, gr)
}

func (lm *loggingMiddleware) ViewGroup(ctx context.Context, token, id string) (g things.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_group for id %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewGroup(ctx, token, id)
}

func (lm *loggingMiddleware) ListGroups(ctx context.Context, token string, pm apiutil.PageMetadata) (g things.GroupPage, err error) {
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

func (lm *loggingMiddleware) ListGroupsByOrg(ctx context.Context, token, orgID string, pm apiutil.PageMetadata) (g things.GroupPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_groups_by_org took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListGroupsByOrg(ctx, token, orgID, pm)
}

func (lm *loggingMiddleware) ListThingsByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (mp things.ThingsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_things_by_group for id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListThingsByGroup(ctx, token, groupID, pm)
}

func (lm *loggingMiddleware) ViewGroupByThing(ctx context.Context, token, thingID string) (gr things.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_group_by_thing for id %s took %s to complete", thingID, time.Since(begin))
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
		message := fmt.Sprintf("Method remove_groups took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveGroups(ctx, token, ids...)
}

func (lm *loggingMiddleware) ViewGroupByProfile(ctx context.Context, token, profileID string) (gr things.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_group_by_profile for id %s took %s to complete", profileID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewGroupByProfile(ctx, token, profileID)
}

func (lm *loggingMiddleware) ListProfilesByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (gprp things.ProfilesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_profiles_by_group for id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListProfilesByGroup(ctx, token, groupID, pm)
}

func (lm *loggingMiddleware) CreateGroupMembers(ctx context.Context, token string, gms ...things.GroupMember) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_group_members took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
	}(time.Now())

	return lm.svc.CreateGroupMembers(ctx, token, gms...)
}

func (lm *loggingMiddleware) ListGroupMembers(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (gpp things.GroupMembersPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_group_members for id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListGroupMembers(ctx, token, groupID, pm)
}

func (lm *loggingMiddleware) UpdateGroupMembers(ctx context.Context, token string, gms ...things.GroupMember) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_group_members took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
	}(time.Now())

	return lm.svc.UpdateGroupMembers(ctx, token, gms...)
}

func (lm *loggingMiddleware) RemoveGroupMembers(ctx context.Context, token, groupID string, memberIDs ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_group_members for id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
	}(time.Now())

	return lm.svc.RemoveGroupMembers(ctx, token, groupID, memberIDs...)
}
