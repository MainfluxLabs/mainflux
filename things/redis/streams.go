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
			groupID:  thing.GroupID,
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

func (es eventStore) ListThingsByProfile(ctx context.Context, token, prID string, pm things.PageMetadata) (things.ThingsPage, error) {
	return es.svc.ListThingsByProfile(ctx, token, prID, pm)
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

func (es eventStore) CreateProfiles(ctx context.Context, token string, profiles ...things.Profile) ([]things.Profile, error) {
	sprs, err := es.svc.CreateProfiles(ctx, token, profiles...)
	if err != nil {
		return sprs, err
	}

	for _, profile := range sprs {
		event := createProfileEvent{
			id:       profile.ID,
			groupID:  profile.GroupID,
			name:     profile.Name,
			metadata: profile.Metadata,
		}
		record := &redis.XAddArgs{
			Stream:       streamID,
			MaxLenApprox: streamLen,
			Values:       event.Encode(),
		}
		es.client.XAdd(ctx, record).Err()
	}

	return sprs, nil
}

func (es eventStore) UpdateProfile(ctx context.Context, token string, profile things.Profile) error {
	if err := es.svc.UpdateProfile(ctx, token, profile); err != nil {
		return err
	}

	event := updateProfileEvent{
		id:       profile.ID,
		name:     profile.Name,
		metadata: profile.Metadata,
	}
	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       event.Encode(),
	}
	es.client.XAdd(ctx, record).Err()

	return nil
}

func (es eventStore) ViewProfile(ctx context.Context, token, id string) (things.Profile, error) {
	return es.svc.ViewProfile(ctx, token, id)
}

func (es eventStore) ListProfiles(ctx context.Context, token string, pm things.PageMetadata) (things.ProfilesPage, error) {
	return es.svc.ListProfiles(ctx, token, pm)
}

func (es eventStore) ViewProfileByThing(ctx context.Context, token, thID string) (things.Profile, error) {
	return es.svc.ViewProfileByThing(ctx, token, thID)
}

func (es eventStore) RemoveProfiles(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		if err := es.svc.RemoveProfiles(ctx, token, id); err != nil {
			return err
		}

		event := removeProfileEvent{
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

func (es eventStore) ViewProfileConfig(ctx context.Context, prID string) (things.Config, error) {
	return es.svc.ViewProfileConfig(ctx, prID)
}

func (es eventStore) GetConnByKey(ctx context.Context, key string) (things.Connection, error) {
	return es.svc.GetConnByKey(ctx, key)
}

func (es eventStore) Authorize(ctx context.Context, req things.AuthorizeReq) error {
	return es.svc.Authorize(ctx, req)
}

func (es eventStore) Identify(ctx context.Context, key string) (string, error) {
	return es.svc.Identify(ctx, key)
}

func (es eventStore) GetGroupIDByThingID(ctx context.Context, thingID string) (string, error) {
	return es.svc.GetGroupIDByThingID(ctx, thingID)
}

func (es eventStore) ListThingsByGroup(ctx context.Context, token, groupID string, pm things.PageMetadata) (things.ThingsPage, error) {
	return es.svc.ListThingsByGroup(ctx, token, groupID, pm)
}

func (es eventStore) ListProfilesByGroup(ctx context.Context, token, groupID string, pm things.PageMetadata) (things.ProfilesPage, error) {
	return es.svc.ListProfilesByGroup(ctx, token, groupID, pm)
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

func (es eventStore) ViewGroupByProfile(ctx context.Context, token string, profileID string) (things.Group, error) {
	return es.svc.ViewGroupByProfile(ctx, token, profileID)
}

func (es eventStore) CreateRolesByGroup(ctx context.Context, token string, gms ...things.GroupMember) error {
	return es.svc.CreateRolesByGroup(ctx, token, gms...)
}

func (es eventStore) ListRolesByGroup(ctx context.Context, token, groupID string, pm things.PageMetadata) (things.GroupMembersPage, error) {
	return es.svc.ListRolesByGroup(ctx, token, groupID, pm)
}

func (es eventStore) UpdateRolesByGroup(ctx context.Context, token string, gms ...things.GroupMember) error {
	return es.svc.UpdateRolesByGroup(ctx, token, gms...)
}

func (es eventStore) RemoveRolesByGroup(ctx context.Context, token, groupID string, memberIDs ...string) error {
	return es.svc.RemoveRolesByGroup(ctx, token, groupID, memberIDs...)
}
