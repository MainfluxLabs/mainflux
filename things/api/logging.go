// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/invites"
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

func (lm *loggingMiddleware) UpdateThingGroupAndProfile(ctx context.Context, token string, thing things.Thing) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_thing_group_and_profile for id %s took %s to complete", thing.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateThingGroupAndProfile(ctx, token, thing)
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

func (lm *loggingMiddleware) ViewMetadataByKey(ctx context.Context, key things.ThingKey) (metadata things.Metadata, err error) {
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

func (lm *loggingMiddleware) BackupThingsByGroup(ctx context.Context, token string, groupID string) (tb things.ThingsBackup, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method backup_things_by_group took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.BackupThingsByGroup(ctx, token, groupID)
}

func (lm *loggingMiddleware) RestoreThingsByGroup(ctx context.Context, token string, groupID string, backup things.ThingsBackup) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method restore_things_by_group took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RestoreThingsByGroup(ctx, token, groupID, backup)
}

func (lm *loggingMiddleware) BackupThingsByOrg(ctx context.Context, token string, orgID string) (tb things.ThingsBackup, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method backup_things_by_org took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.BackupThingsByOrg(ctx, token, orgID)
}

func (lm *loggingMiddleware) RestoreThingsByOrg(ctx context.Context, token string, orgID string, backup things.ThingsBackup) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method restore_things_by_org took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RestoreThingsByOrg(ctx, token, orgID, backup)
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

func (lm *loggingMiddleware) GetPubConfByKey(ctx context.Context, key things.ThingKey) (pc things.PubConfInfo, err error) {
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

func (lm *loggingMiddleware) Identify(ctx context.Context, key things.ThingKey) (id string, err error) {
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

func (lm *loggingMiddleware) GetGroupIDByProfileID(ctx context.Context, profileID string) (_ string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method get_group_id_by_profile_id for profile %s took %s to complete", profileID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.GetGroupIDByProfileID(ctx, profileID)
}

func (lm *loggingMiddleware) GetGroupIDsByOrg(ctx context.Context, orgID string, token string) (_ []string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method get_group_ids_by_org for org %s took %s to complete", orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.GetGroupIDsByOrg(ctx, orgID, token)
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

func (lm *loggingMiddleware) BackupGroupsByOrg(ctx context.Context, token string, orgID string) (bk things.GroupsBackup, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method backup_groups_by_org took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.BackupGroupsByOrg(ctx, token, orgID)
}

func (lm *loggingMiddleware) RestoreGroupsByOrg(ctx context.Context, token string, orgID string, backup things.GroupsBackup) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method restore_groups_by_org took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RestoreGroupsByOrg(ctx, token, orgID, backup)
}

func (lm *loggingMiddleware) BackupGroupMemberships(ctx context.Context, token string, groupID string) (bk things.GroupMembershipsBackup, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method backup_group_memberships took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.BackupGroupMemberships(ctx, token, groupID)
}

func (lm *loggingMiddleware) RestoreGroupMemberships(ctx context.Context, token string, groupID string, backup things.GroupMembershipsBackup) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method restore_group_memberships took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RestoreGroupMemberships(ctx, token, groupID, backup)
}

func (lm *loggingMiddleware) BackupProfilesByOrg(ctx context.Context, token string, orgID string) (pb things.ProfilesBackup, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method backup_profiles_by_org took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.BackupProfilesByOrg(ctx, token, orgID)
}

func (lm *loggingMiddleware) RestoreProfilesByOrg(ctx context.Context, token string, orgID string, backup things.ProfilesBackup) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method restore_profiles_by_org took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RestoreProfilesByOrg(ctx, token, orgID, backup)
}

func (lm *loggingMiddleware) BackupProfilesByGroup(ctx context.Context, token string, groupID string) (pb things.ProfilesBackup, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method backup_profiles_by_group took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.BackupProfilesByGroup(ctx, token, groupID)
}

func (lm *loggingMiddleware) RestoreProfilesByGroup(ctx context.Context, token string, groupID string, backup things.ProfilesBackup) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method restore_profiles_by_group took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RestoreProfilesByGroup(ctx, token, groupID, backup)
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

func (lm *loggingMiddleware) CreateGroupMemberships(ctx context.Context, token string, gms ...things.GroupMembership) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_group_memberships took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
	}(time.Now())

	return lm.svc.CreateGroupMemberships(ctx, token, gms...)
}

func (lm *loggingMiddleware) ListGroupMemberships(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (gpp things.GroupMembershipsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_group_memberships for group id %s and email %s took %s to complete", groupID, pm.Email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
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
	}(time.Now())

	return lm.svc.UpdateGroupMemberships(ctx, token, gms...)
}

func (lm *loggingMiddleware) RemoveGroupMemberships(ctx context.Context, token, groupID string, memberIDs ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_group_memberships for group id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
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
	}(time.Now())

	return lm.svc.UpdateExternalKey(ctx, token, key, thingID)
}

func (lm *loggingMiddleware) RemoveExternalKey(ctx context.Context, token, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_external_key for thing %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
	}(time.Now())

	return lm.svc.RemoveExternalKey(ctx, token, thingID)
}

func (lm *loggingMiddleware) GetThingIDsByProfile(ctx context.Context, profileID string) (_ []string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method get_thing_ids_by_profile for profile %s took %s to complete", profileID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.GetThingIDsByProfile(ctx, profileID)
}

func (lm *loggingMiddleware) CreateGroupInvite(ctx context.Context, token, email, role, groupID, grRedirectPath string) (invite things.GroupInvite, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_group_invite for group id %s and email %s took %s to complete", groupID, email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateGroupInvite(ctx, token, email, role, groupID, grRedirectPath)
}

func (lm *loggingMiddleware) RevokeGroupInvite(ctx context.Context, token, inviteID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method revoke_group_invite for id %s took %s to complete", inviteID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RevokeGroupInvite(ctx, token, inviteID)
}

func (lm *loggingMiddleware) RespondGroupInvite(ctx context.Context, token, inviteID string, accept bool) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method respond_group_invite for id %s (accept=%t) took %s to complete", inviteID, accept, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RespondGroupInvite(ctx, token, inviteID, accept)
}

func (lm *loggingMiddleware) ViewGroupInvite(ctx context.Context, token, inviteID string) (invite things.GroupInvite, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_group_invite for id %s took %s to complete", inviteID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewGroupInvite(ctx, token, inviteID)
}

func (lm *loggingMiddleware) ListGroupInvitesByUser(ctx context.Context, token, userType, userID string, pm invites.PageMetadataInvites) (_ things.GroupInvitesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_group_invites_by_user for user id %s and user-type %s took %s to complete", userID, userType, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListGroupInvitesByUser(ctx, token, userType, userID, pm)
}

func (lm *loggingMiddleware) ListGroupInvitesByGroup(ctx context.Context, token, groupID string, pm invites.PageMetadataInvites) (_ things.GroupInvitesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_group_invites_by_group for group %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListGroupInvitesByGroup(ctx, token, groupID, pm)
}

func (lm *loggingMiddleware) CreateDormantGroupInvites(ctx context.Context, token, orgInviteID string, groupMemberships ...things.GroupMembership) (err error) {
	defer func(begin time.Time) {
		var groupLogs []string
		for _, gm := range groupMemberships {
			msg := fmt.Sprintf("[%s; %s]", gm.GroupID, gm.Role)
			groupLogs = append(groupLogs, msg)
		}

		groupsMsg := strings.Join(groupLogs, ",")

		message := fmt.Sprintf("Method create_dormant_group_invites for org invite id %s and groups (%s) took %s to complete", orgInviteID, groupsMsg, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateDormantGroupInvites(ctx, token, orgInviteID, groupMemberships...)
}

func (lm *loggingMiddleware) ActivateGroupInvites(ctx context.Context, token, orgInviteID, invRedirectPath string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method activate_group_invites for org invite id %s took %s to complete", orgInviteID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ActivateGroupInvites(ctx, token, orgInviteID, invRedirectPath)
}

func (lm *loggingMiddleware) SendGroupInviteEmail(ctx context.Context, invite things.GroupInvite, email, orgName, invRedirectPath string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method send_group_invite_email for invite id %s and email %s took %s to complete", invite.ID, email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.SendGroupInviteEmail(ctx, invite, email, orgName, invRedirectPath)
}
