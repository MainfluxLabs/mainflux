// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"

	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-redis/redis/v8"
)

const (
	streamID  = "mainflux.things"
	streamLen = 1000
)

var _ things.Service = (*eventStore)(nil)

type eventStore struct {
	svc    things.Service
	client *redis.Client
}

// NewEventStoreMiddleware returns wrapper around things service that sends
// events to event store.
func NewEventStoreMiddleware(svc things.Service, client *redis.Client) things.Service {
	return eventStore{
		svc:    svc,
		client: client,
	}
}

func (es eventStore) CreateThings(ctx context.Context, token string, ths ...things.Thing) ([]things.Thing, error) {
	sths, err := es.svc.CreateThings(ctx, token, ths...)
	if err != nil {
		return sths, err
	}

	for _, thing := range sths {
		event := createThingEvent{
			id:       thing.ID,
			ownerID:  thing.OwnerID,
			name:     thing.Name,
			metadata: thing.Metadata,
		}
		record := &redis.XAddArgs{
			Stream:       streamID,
			MaxLenApprox: streamLen,
			Values:       event.Encode(),
		}
		es.client.XAdd(ctx, record).Err()
	}

	return sths, nil
}

func (es eventStore) UpdateThing(ctx context.Context, token string, thing things.Thing) error {
	if err := es.svc.UpdateThing(ctx, token, thing); err != nil {
		return err
	}

	event := updateThingEvent{
		id:       thing.ID,
		name:     thing.Name,
		metadata: thing.Metadata,
	}
	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       event.Encode(),
	}
	es.client.XAdd(ctx, record).Err()

	return nil
}

// UpdateKey doesn't send event because key shouldn't be sent over stream.
// Maybe we can start publishing this event at some point, without key value
// in order to notify adapters to disconnect connected things after key update.
func (es eventStore) UpdateKey(ctx context.Context, token, id, key string) error {
	return es.svc.UpdateKey(ctx, token, id, key)
}

func (es eventStore) ViewThing(ctx context.Context, token, id string) (things.Thing, error) {
	return es.svc.ViewThing(ctx, token, id)
}

func (es eventStore) ListThings(ctx context.Context, token string, pm things.PageMetadata) (things.ThingsPage, error) {
	return es.svc.ListThings(ctx, token, pm)
}

func (es eventStore) ListThingsByIDs(ctx context.Context, ids []string) (things.ThingsPage, error) {
	return es.svc.ListThingsByIDs(ctx, ids)
}

func (es eventStore) ListThingsByChannel(ctx context.Context, token, chID string, pm things.PageMetadata) (things.ThingsPage, error) {
	return es.svc.ListThingsByChannel(ctx, token, chID, pm)
}

func (es eventStore) Backup(ctx context.Context, token string) (things.Backup, error) {
	return es.svc.Backup(ctx, token)
}

func (es eventStore) Restore(ctx context.Context, token string, backup things.Backup) error {
	return es.svc.Restore(ctx, token, backup)
}

func (es eventStore) RemoveThings(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		if err := es.svc.RemoveThings(ctx, token, id); err != nil {
			return err
		}

		event := removeThingEvent{
			id: id,
		}
		record := &redis.XAddArgs{
			Stream:       streamID,
			MaxLenApprox: streamLen,
			Values:       event.Encode(),
		}
		es.client.XAdd(ctx, record).Err()
	}

	return nil
}

func (es eventStore) CreateChannels(ctx context.Context, token string, channels ...things.Channel) ([]things.Channel, error) {
	schs, err := es.svc.CreateChannels(ctx, token, channels...)
	if err != nil {
		return schs, err
	}

	for _, channel := range schs {
		event := createChannelEvent{
			id:       channel.ID,
			ownerID:  channel.OwnerID,
			name:     channel.Name,
			metadata: channel.Metadata,
		}
		record := &redis.XAddArgs{
			Stream:       streamID,
			MaxLenApprox: streamLen,
			Values:       event.Encode(),
		}
		es.client.XAdd(ctx, record).Err()
	}

	return schs, nil
}

func (es eventStore) UpdateChannel(ctx context.Context, token string, channel things.Channel) error {
	if err := es.svc.UpdateChannel(ctx, token, channel); err != nil {
		return err
	}

	event := updateChannelEvent{
		id:       channel.ID,
		name:     channel.Name,
		metadata: channel.Metadata,
	}
	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       event.Encode(),
	}
	es.client.XAdd(ctx, record).Err()

	return nil
}

func (es eventStore) ViewChannel(ctx context.Context, token, id string) (things.Channel, error) {
	return es.svc.ViewChannel(ctx, token, id)
}

func (es eventStore) ListChannels(ctx context.Context, token string, pm things.PageMetadata) (things.ChannelsPage, error) {
	return es.svc.ListChannels(ctx, token, pm)
}

func (es eventStore) ViewChannelByThing(ctx context.Context, token, thID string) (things.Channel, error) {
	return es.svc.ViewChannelByThing(ctx, token, thID)
}

func (es eventStore) RemoveChannels(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		if err := es.svc.RemoveChannels(ctx, token, id); err != nil {
			return err
		}

		event := removeChannelEvent{
			id: id,
		}
		record := &redis.XAddArgs{
			Stream:       streamID,
			MaxLenApprox: streamLen,
			Values:       event.Encode(),
		}
		es.client.XAdd(ctx, record).Err()

	}

	return nil
}

func (es eventStore) ViewChannelProfile(ctx context.Context, chID string) (things.Profile, error) {
	return es.svc.ViewChannelProfile(ctx, chID)
}

func (es eventStore) Connect(ctx context.Context, token, chID string, thIDs []string) error {
	if err := es.svc.Connect(ctx, token, chID, thIDs); err != nil {
		return err
	}

	for _, thID := range thIDs {
		event := connectThingEvent{
			chanID:  chID,
			thingID: thID,
		}
		record := &redis.XAddArgs{
			Stream:       streamID,
			MaxLenApprox: streamLen,
			Values:       event.Encode(),
		}
		es.client.XAdd(ctx, record).Err()
	}

	return nil
}

func (es eventStore) Disconnect(ctx context.Context, token, chID string, thIDs []string) error {
	if err := es.svc.Disconnect(ctx, token, chID, thIDs); err != nil {
		return err
	}

	for _, thID := range thIDs {
		event := disconnectThingEvent{
			chanID:  chID,
			thingID: thID,
		}
		record := &redis.XAddArgs{
			Stream:       streamID,
			MaxLenApprox: streamLen,
			Values:       event.Encode(),
		}
		es.client.XAdd(ctx, record).Err()
	}

	return nil
}

func (es eventStore) GetConnByKey(ctx context.Context, key string) (things.Connection, error) {
	return es.svc.GetConnByKey(ctx, key)
}

func (es eventStore) IsChannelOwner(ctx context.Context, owner, chanID string) error {
	return es.svc.IsChannelOwner(ctx, owner, chanID)
}

func (es eventStore) CanAccessGroup(ctx context.Context, token, groupID, action string) error {
	return es.svc.CanAccessGroup(ctx, token, groupID, action)
}

func (es eventStore) Identify(ctx context.Context, key string) (string, error) {
	return es.svc.Identify(ctx, key)
}

func (es eventStore) ListThingsByGroup(ctx context.Context, token, groupID string, pm things.PageMetadata) (things.ThingsPage, error) {
	return es.svc.ListThingsByGroup(ctx, token, groupID, pm)
}

func (es eventStore) ListChannelsByGroup(ctx context.Context, token, groupID string, pm things.PageMetadata) (things.ChannelsPage, error) {
	return es.svc.ListChannelsByGroup(ctx, token, groupID, pm)
}

func (es eventStore) CreateGroups(ctx context.Context, token string, grs ...things.Group) ([]things.Group, error) {
	return es.svc.CreateGroups(ctx, token, grs...)
}

func (es eventStore) ListGroups(ctx context.Context, token, orgID string, pm things.PageMetadata) (things.GroupPage, error) {
	return es.svc.ListGroups(ctx, token, orgID, pm)
}

func (es eventStore) ListGroupsByIDs(ctx context.Context, groupIDs []string) ([]things.Group, error) {
	return es.svc.ListGroupsByIDs(ctx, groupIDs)
}

func (es eventStore) RemoveGroups(ctx context.Context, token string, ids ...string) error {
	return es.svc.RemoveGroups(ctx, token, ids...)
}

func (es eventStore) UpdateGroup(ctx context.Context, token string, group things.Group) (things.Group, error) {
	return es.svc.UpdateGroup(ctx, token, group)
}

func (es eventStore) ViewGroup(ctx context.Context, token, id string) (things.Group, error) {
	return es.svc.ViewGroup(ctx, token, id)
}

func (es eventStore) ViewGroupByThing(ctx context.Context, token string, thingID string) (things.Group, error) {
	return es.svc.ViewGroupByThing(ctx, token, thingID)
}

func (es eventStore) ViewGroupByChannel(ctx context.Context, token string, channelID string) (things.Group, error) {
	return es.svc.ViewGroupByChannel(ctx, token, channelID)
}

func (es eventStore) CreateRolesByGroup(ctx context.Context, token, groupID string, gps ...things.GroupRoles) error {
	return es.svc.CreateRolesByGroup(ctx, token, groupID, gps...)
}

func (es eventStore) ListRolesByGroup(ctx context.Context, token, groupID string, pm things.PageMetadata) (things.GroupRolesPage, error) {
	return es.svc.ListRolesByGroup(ctx, token, groupID, pm)
}

func (es eventStore) UpdateRolesByGroup(ctx context.Context, token, groupID string, gps ...things.GroupRoles) error {
	return es.svc.UpdateRolesByGroup(ctx, token, groupID, gps...)
}

func (es eventStore) RemoveRolesByGroup(ctx context.Context, token, groupID string, memberIDs ...string) error {
	return es.svc.RemoveRolesByGroup(ctx, token, groupID, memberIDs...)
}
