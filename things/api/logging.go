// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	log "github.com/MainfluxLabs/mainflux/logger"
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
		message := fmt.Sprintf("Method update_thing for thing %s took %s to complete", thing.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateThing(ctx, token, thing)
}

func (lm *loggingMiddleware) UpdateKey(ctx context.Context, token, id, key string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_key for thing %s and key %s took %s to complete", id, key, time.Since(begin))
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
		message := fmt.Sprintf("Method view_thing for thing %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewThing(ctx, token, id)
}

func (lm *loggingMiddleware) ListThings(ctx context.Context, token string, pm things.PageMetadata) (_ things.ThingsPage, err error) {
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

func (lm *loggingMiddleware) ListThingsByIDs(ctx context.Context, ids []string) (page things.ThingsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_things_by_ids for ids %s took %s to complete", ids, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListThingsByIDs(ctx, ids)
}

func (lm *loggingMiddleware) ListThingsByChannel(ctx context.Context, token, chID string, pm things.PageMetadata) (_ things.ThingsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_things_by_channel for channel %s took %s to complete", chID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s", message, err))
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListThingsByChannel(ctx, token, chID, pm)
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

func (lm *loggingMiddleware) CreateChannels(ctx context.Context, token string, channels ...things.Channel) (saved []things.Channel, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_channels for channels %s took %s to complete", saved, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateChannels(ctx, token, channels...)
}

func (lm *loggingMiddleware) UpdateChannel(ctx context.Context, token string, channel things.Channel) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_channel for channel %s took %s to complete", channel.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateChannel(ctx, token, channel)
}

func (lm *loggingMiddleware) ViewChannel(ctx context.Context, token, id string) (channel things.Channel, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_channel for channel %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewChannel(ctx, token, id)
}

func (lm *loggingMiddleware) ListChannels(ctx context.Context, token string, pm things.PageMetadata) (_ things.ChannelsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_channels took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListChannels(ctx, token, pm)
}

func (lm *loggingMiddleware) ViewChannelByThing(ctx context.Context, token, thID string) (_ things.Channel, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_channel_by_thing for thing %s took %s to complete", thID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s", message, err))
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewChannelByThing(ctx, token, thID)
}

func (lm *loggingMiddleware) RemoveChannels(ctx context.Context, token string, ids ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_channels took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveChannels(ctx, token, ids...)
}

func (lm *loggingMiddleware) ViewChannelProfile(ctx context.Context, chID string) (profile things.Profile, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_channel_profile for channel ID %s took %s to complete", chID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewChannelProfile(ctx, chID)
}

func (lm *loggingMiddleware) Connect(ctx context.Context, token, chID string, thIDs []string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method connect for channel %s and things %s took %s to complete", chID, thIDs, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Connect(ctx, token, chID, thIDs)
}

func (lm *loggingMiddleware) Disconnect(ctx context.Context, token, chID string, thIDs []string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method disconnect for channel %v and things %v took %s to complete", chID, thIDs, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Disconnect(ctx, token, chID, thIDs)
}

func (lm *loggingMiddleware) GetConnByKey(ctx context.Context, key string) (conn things.Connection, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method can_access for thing %s took %s to complete", conn.ThingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.GetConnByKey(ctx, key)
}

func (lm *loggingMiddleware) IsChannelOwner(ctx context.Context, owner, chanID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method is_channel_owner for channel %s and user %s took %s to complete", chanID, owner, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.IsChannelOwner(ctx, owner, chanID)
}

func (lm *loggingMiddleware) IsThingOwner(ctx context.Context, token, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method is_thing_owner for thing %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.IsThingOwner(ctx, token, thingID)
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
		message := fmt.Sprintf("Method create_groups took %s to complete", time.Since(begin))
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
		message := fmt.Sprintf("Method update_group took %s to complete", time.Since(begin))
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
		message := fmt.Sprintf("Method view_group took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewGroup(ctx, token, id)
}

func (lm *loggingMiddleware) ListGroups(ctx context.Context, token, orgID string, pm things.PageMetadata) (g things.GroupPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_groups took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListGroups(ctx, token, orgID, pm)
}

func (lm *loggingMiddleware) ListGroupsByIDs(ctx context.Context, groupIDs []string) (g []things.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_groups_by_ids for group ids %s took %s to complete", groupIDs, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListGroupsByIDs(ctx, groupIDs)
}

func (lm *loggingMiddleware) ListGroupThings(ctx context.Context, token, groupID string, pm things.PageMetadata) (mp things.ThingsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_group_things for group id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListGroupThings(ctx, token, groupID, pm)
}

func (lm *loggingMiddleware) ViewThingGroup(ctx context.Context, token, thingID string) (gr things.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_thing_group took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewThingGroup(ctx, token, thingID)
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

func (lm *loggingMiddleware) ViewChannelGroup(ctx context.Context, token, channelID string) (gr things.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_channel_group took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewChannelGroup(ctx, token, channelID)
}

func (lm *loggingMiddleware) ListGroupChannels(ctx context.Context, token, groupID string, pm things.PageMetadata) (gchp things.ChannelsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_group_channels took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListGroupChannels(ctx, token, groupID, pm)
}

func (lm *loggingMiddleware) CreateGroupPolicies(ctx context.Context, token, groupID string, gps ...things.GroupPolicyByID) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_group_policies for group id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
	}(time.Now())

	return lm.svc.CreateGroupPolicies(ctx, token, groupID, gps...)
}

func (lm *loggingMiddleware) ListGroupPolicies(ctx context.Context, token, groupID string, pm things.PageMetadata) (gpp things.GroupPoliciesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_group_policies for group id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListGroupPolicies(ctx, token, groupID, pm)
}

func (lm *loggingMiddleware) UpdateGroupPolicies(ctx context.Context, token, groupID string, gps ...things.GroupPolicyByID) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_group_policies for group id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
	}(time.Now())

	return lm.svc.UpdateGroupPolicies(ctx, token, groupID, gps...)
}

func (lm *loggingMiddleware) RemoveGroupPolicies(ctx context.Context, token, groupID string, memberIDs ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_group_policies for group id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
	}(time.Now())

	return lm.svc.RemoveGroupPolicies(ctx, token, groupID, memberIDs...)
}
