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
		message := fmt.Sprintf("Method create_things for token %s and things %s took %s to complete", token, saved, time.Since(begin))
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
		message := fmt.Sprintf("Method update_thing for token %s and thing %s took %s to complete", token, thing.ID, time.Since(begin))
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
		message := fmt.Sprintf("Method view_thing for token %s and thing %s took %s to complete", token, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewThing(ctx, token, id)
}

func (lm *loggingMiddleware) ListThings(ctx context.Context, token string, pm things.PageMetadata) (_ things.Page, err error) {
	defer func(begin time.Time) {
		nlog := ""
		if pm.Name != "" {
			nlog = fmt.Sprintf("with name %s", pm.Name)
		}
		message := fmt.Sprintf("Method list_things %s for token %s took %s to complete", nlog, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListThings(ctx, token, pm)
}

func (lm *loggingMiddleware) ListThingsByIDs(ctx context.Context, ids []string) (page things.Page, err error) {
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

func (lm *loggingMiddleware) ListThingsByChannel(ctx context.Context, token, chID string, pm things.PageMetadata) (_ things.Page, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_things_by_channel for channel %s took %s to complete", chID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s", message, err))
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListThingsByChannel(ctx, token, chID, pm)
}

func (lm *loggingMiddleware) RemoveThing(ctx context.Context, token, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_thing for token %s and thing %s took %s to complete", token, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveThing(ctx, token, id)
}

func (lm *loggingMiddleware) CreateChannels(ctx context.Context, token string, channels ...things.Channel) (saved []things.Channel, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_channels for token %s and channels %s took %s to complete", token, saved, time.Since(begin))
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
		message := fmt.Sprintf("Method update_channel for token %s and channel %s took %s to complete", token, channel.ID, time.Since(begin))
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
		message := fmt.Sprintf("Method view_channel for token %s and channel %s took %s to complete", token, id, time.Since(begin))
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
		nlog := ""
		if pm.Name != "" {
			nlog = fmt.Sprintf("with name %s", pm.Name)
		}
		message := fmt.Sprintf("Method list_channels %s for token %s took %s to complete", nlog, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListChannels(ctx, token, pm)
}

func (lm *loggingMiddleware) ListChannelsByThing(ctx context.Context, token, thID string, pm things.PageMetadata) (_ things.ChannelsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_channels_by_thing for thing %s took %s to complete", thID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s", message, err))
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListChannelsByThing(ctx, token, thID, pm)
}

func (lm *loggingMiddleware) RemoveChannel(ctx context.Context, token, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_channel for token %s and channel %s took %s to complete", token, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveChannel(ctx, token, id)
}

func (lm *loggingMiddleware) Connect(ctx context.Context, token string, chIDs, thIDs []string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method connect for token %s, channels %s and things %s took %s to complete", token, chIDs, thIDs, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Connect(ctx, token, chIDs, thIDs)
}

func (lm *loggingMiddleware) Disconnect(ctx context.Context, token string, chIDs, thIDs []string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method disconnect for token %s, channels %v and things %v took %s to complete", token, chIDs, thIDs, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Disconnect(ctx, token, chIDs, thIDs)
}

func (lm *loggingMiddleware) CanAccessByKey(ctx context.Context, id, key string) (thing string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method can_access for channel %s and thing %s took %s to complete", id, thing, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CanAccessByKey(ctx, id, key)
}

func (lm *loggingMiddleware) CanAccessByID(ctx context.Context, chanID, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method can_access_by_id for channel %s and thing %s took %s to complete", chanID, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CanAccessByID(ctx, chanID, thingID)
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

func (lm *loggingMiddleware) Identify(ctx context.Context, key string) (id string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method identify for token %s and thing %s took %s to complete", key, id, time.Since(begin))
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
		message := fmt.Sprintf("Method backup for token %s took %s to complete", token, time.Since(begin))
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
		message := fmt.Sprintf("Method restore for token %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Restore(ctx, token, backup)
}

func (lm *loggingMiddleware) CreateGroup(ctx context.Context, token string, gr things.Group) (g things.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_group for token %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateGroup(ctx, token, gr)
}

func (lm *loggingMiddleware) UpdateGroup(ctx context.Context, token string, gr things.Group) (g things.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_group for token %s took %s to complete", token, time.Since(begin))
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
		message := fmt.Sprintf("Method view_group for token %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewGroup(ctx, token, id)
}

func (lm *loggingMiddleware) ListGroups(ctx context.Context, token string, pm things.PageMetadata) (g things.GroupPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_groups for token %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListGroups(ctx, token, pm)
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

func (lm *loggingMiddleware) ListMembers(ctx context.Context, token, groupID string, pm things.PageMetadata) (mp things.MemberPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_members for token %s and group id %s took %s to complete", token, groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListMembers(ctx, token, groupID, pm)
}

func (lm *loggingMiddleware) ListMemberships(ctx context.Context, token, memberID string, pm things.PageMetadata) (g things.GroupPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_memberships for token %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListMemberships(ctx, token, memberID, pm)
}

func (lm *loggingMiddleware) RemoveGroup(ctx context.Context, token, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_group for token %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveGroup(ctx, token, id)
}

func (lm *loggingMiddleware) Assign(ctx context.Context, token, groupID string, memberIDs ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method assign for token %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Assign(ctx, token, groupID, memberIDs...)
}

func (lm *loggingMiddleware) Unassign(ctx context.Context, token, groupID string, memberIDs ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method unassign for token %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Unassign(ctx, token, groupID, memberIDs...)
}
