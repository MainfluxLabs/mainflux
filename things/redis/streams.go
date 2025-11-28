// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
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

func (es eventStore) CreateThings(ctx context.Context, token, profileID string, ths ...things.Thing) ([]things.Thing, error) {
	sths, err := es.svc.CreateThings(ctx, token, profileID, ths...)
	if err != nil {
		return sths, err
	}

	for _, thing := range sths {
		event := createThingEvent{
			id:        thing.ID,
			groupID:   thing.GroupID,
			profileID: thing.ProfileID,
			name:      thing.Name,
			metadata:  thing.Metadata,
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
		id:        thing.ID,
		profileID: thing.ProfileID,
		name:      thing.Name,
		metadata:  thing.Metadata,
	}
	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       event.Encode(),
	}
	es.client.XAdd(ctx, record).Err()

	return nil
}

func (es eventStore) UpdateThingsMetadata(ctx context.Context, token string, things ...things.Thing) error {
	return es.svc.UpdateThingsMetadata(ctx, token, things...)
}

func (es eventStore) UpdateThingGroupAndProfile(ctx context.Context, token string, thing things.Thing) error {
	if err := es.svc.UpdateThing(ctx, token, thing); err != nil {
		return err
	}

	event := updateThingGroupAndProfileEvent{
		id:        thing.ID,
		profileID: thing.ProfileID,
		groupID:   thing.GroupID,
	}

	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       event.Encode(),
	}

	es.client.XAdd(ctx, record).Err()

	return nil
}

func (es eventStore) UpdateExternalKey(ctx context.Context, token, key, thingID string) error {
	return es.svc.UpdateExternalKey(ctx, token, key, thingID)
}

func (es eventStore) RemoveExternalKey(ctx context.Context, token, thingID string) error {
	return es.svc.RemoveExternalKey(ctx, token, thingID)
}

func (es eventStore) ViewThing(ctx context.Context, token, id string) (things.Thing, error) {
	return es.svc.ViewThing(ctx, token, id)
}

func (es eventStore) ViewMetadataByKey(ctx context.Context, key things.ThingKey) (things.Metadata, error) {
	return es.svc.ViewMetadataByKey(ctx, key)
}

func (es eventStore) ListThings(ctx context.Context, token string, pm apiutil.PageMetadata) (things.ThingsPage, error) {
	return es.svc.ListThings(ctx, token, pm)
}

func (es eventStore) ListThingsByProfile(ctx context.Context, token, prID string, pm apiutil.PageMetadata) (things.ThingsPage, error) {
	return es.svc.ListThingsByProfile(ctx, token, prID, pm)
}

func (es eventStore) ListThingsByOrg(ctx context.Context, token string, orgID string, pm apiutil.PageMetadata) (things.ThingsPage, error) {
	return es.svc.ListThingsByOrg(ctx, token, orgID, pm)
}

func (es eventStore) Backup(ctx context.Context, token string) (things.Backup, error) {
	return es.svc.Backup(ctx, token)
}

func (es eventStore) BackupGroupsByOrg(ctx context.Context, token string, orgID string) (things.GroupsBackup, error) {
	return es.svc.BackupGroupsByOrg(ctx, token, orgID)
}

func (es eventStore) RestoreGroupsByOrg(ctx context.Context, token string, orgID string, backup things.GroupsBackup) error {
	return es.svc.RestoreGroupsByOrg(ctx, token, orgID, backup)
}

func (es eventStore) BackupGroupMemberships(ctx context.Context, token string, groupID string) (things.GroupMembershipsBackup, error) {
	return es.svc.BackupGroupMemberships(ctx, token, groupID)
}

func (es eventStore) RestoreGroupMemberships(ctx context.Context, token string, groupID string, backup things.GroupMembershipsBackup) error {
	return es.svc.RestoreGroupMemberships(ctx, token, groupID, backup)
}

func (es eventStore) BackupProfilesByOrg(ctx context.Context, token string, orgID string) (things.ProfilesBackup, error) {
	return es.svc.BackupProfilesByOrg(ctx, token, orgID)
}

func (es eventStore) RestoreProfilesByOrg(ctx context.Context, token string, orgID string, backup things.ProfilesBackup) error {
	return es.svc.RestoreProfilesByOrg(ctx, token, orgID, backup)
}

func (es eventStore) BackupProfilesByGroup(ctx context.Context, token string, groupID string) (things.ProfilesBackup, error) {
	return es.svc.BackupProfilesByGroup(ctx, token, groupID)
}

func (es eventStore) RestoreProfilesByGroup(ctx context.Context, token string, groupID string, backup things.ProfilesBackup) error {
	return es.svc.RestoreProfilesByGroup(ctx, token, groupID, backup)
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

func (es eventStore) CreateProfiles(ctx context.Context, token, grID string, profiles ...things.Profile) ([]things.Profile, error) {
	sprs, err := es.svc.CreateProfiles(ctx, token, grID, profiles...)
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
		config:   profile.Config,
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

func (es eventStore) ListProfiles(ctx context.Context, token string, pm apiutil.PageMetadata) (things.ProfilesPage, error) {
	return es.svc.ListProfiles(ctx, token, pm)
}

func (es eventStore) ListProfilesByOrg(ctx context.Context, token string, orgID string, pm apiutil.PageMetadata) (things.ProfilesPage, error) {
	return es.svc.ListProfilesByOrg(ctx, token, orgID, pm)
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

func (es eventStore) GetPubConfByKey(ctx context.Context, key things.ThingKey) (things.PubConfInfo, error) {
	return es.svc.GetPubConfByKey(ctx, key)
}

func (es eventStore) GetConfigByThingID(ctx context.Context, thingID string) (map[string]any, error) {
	return es.svc.GetConfigByThingID(ctx, thingID)
}

func (es eventStore) CanUserAccessThing(ctx context.Context, req things.UserAccessReq) error {
	return es.svc.CanUserAccessThing(ctx, req)
}

func (es eventStore) CanUserAccessProfile(ctx context.Context, req things.UserAccessReq) error {
	return es.svc.CanUserAccessProfile(ctx, req)
}

func (es eventStore) CanUserAccessGroup(ctx context.Context, req things.UserAccessReq) error {
	return es.svc.CanUserAccessGroup(ctx, req)
}

func (es eventStore) CanThingAccessGroup(ctx context.Context, req things.ThingAccessReq) error {
	return es.svc.CanThingAccessGroup(ctx, req)
}

func (es eventStore) Identify(ctx context.Context, key things.ThingKey) (string, error) {
	return es.svc.Identify(ctx, key)
}

func (es eventStore) GetGroupIDByThingID(ctx context.Context, thingID string) (string, error) {
	return es.svc.GetGroupIDByThingID(ctx, thingID)
}

func (es eventStore) GetGroupIDByProfileID(ctx context.Context, profileID string) (string, error) {
	return es.svc.GetGroupIDByProfileID(ctx, profileID)
}

func (es eventStore) GetGroupIDsByOrg(ctx context.Context, orgID string, token string) ([]string, error) {
	return es.svc.GetGroupIDsByOrg(ctx, orgID, token)
}

func (es eventStore) ListThingsByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (things.ThingsPage, error) {
	return es.svc.ListThingsByGroup(ctx, token, groupID, pm)
}

func (es eventStore) BackupThingsByGroup(ctx context.Context, token string, groupID string) (things.ThingsBackup, error) {
	return es.svc.BackupThingsByGroup(ctx, token, groupID)
}

func (es eventStore) RestoreThingsByGroup(ctx context.Context, token string, groupID string, backup things.ThingsBackup) error {
	return es.svc.RestoreThingsByGroup(ctx, token, groupID, backup)
}

func (es eventStore) BackupThingsByOrg(ctx context.Context, token string, orgID string) (things.ThingsBackup, error) {
	return es.svc.BackupThingsByOrg(ctx, token, orgID)
}

func (es eventStore) RestoreThingsByOrg(ctx context.Context, token string, orgID string, backup things.ThingsBackup) error {
	return es.svc.RestoreThingsByOrg(ctx, token, orgID, backup)
}

func (es eventStore) ListProfilesByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (things.ProfilesPage, error) {
	return es.svc.ListProfilesByGroup(ctx, token, groupID, pm)
}

func (es eventStore) CreateGroups(ctx context.Context, token, orgID string, grs ...things.Group) ([]things.Group, error) {
	return es.svc.CreateGroups(ctx, token, orgID, grs...)
}

func (es eventStore) ListGroups(ctx context.Context, token string, pm apiutil.PageMetadata) (things.GroupPage, error) {
	return es.svc.ListGroups(ctx, token, pm)
}

func (es eventStore) ListGroupsByOrg(ctx context.Context, token, orgID string, pm apiutil.PageMetadata) (things.GroupPage, error) {
	return es.svc.ListGroupsByOrg(ctx, token, orgID, pm)
}

func (es eventStore) RemoveGroups(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		if err := es.svc.RemoveGroups(ctx, token, id); err != nil {
			return err
		}

		event := removeGroupEvent{
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

func (es eventStore) UpdateGroup(ctx context.Context, token string, group things.Group) (things.Group, error) {
	return es.svc.UpdateGroup(ctx, token, group)
}

func (es eventStore) ViewGroup(ctx context.Context, token, id string) (things.Group, error) {
	return es.svc.ViewGroup(ctx, token, id)
}

func (es eventStore) ViewGroupByThing(ctx context.Context, token string, thingID string) (things.Group, error) {
	return es.svc.ViewGroupByThing(ctx, token, thingID)
}

func (es eventStore) ViewGroupByProfile(ctx context.Context, token, profileID string) (things.Group, error) {
	return es.svc.ViewGroupByProfile(ctx, token, profileID)
}

func (es eventStore) CreateGroupMemberships(ctx context.Context, token string, gms ...things.GroupMembership) error {
	return es.svc.CreateGroupMemberships(ctx, token, gms...)
}

func (es eventStore) ListGroupMemberships(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (things.GroupMembershipsPage, error) {
	return es.svc.ListGroupMemberships(ctx, token, groupID, pm)
}

func (es eventStore) UpdateGroupMemberships(ctx context.Context, token string, gms ...things.GroupMembership) error {
	return es.svc.UpdateGroupMemberships(ctx, token, gms...)
}

func (es eventStore) RemoveGroupMemberships(ctx context.Context, token, groupID string, memberIDs ...string) error {
	return es.svc.RemoveGroupMemberships(ctx, token, groupID, memberIDs...)
}

func (es eventStore) GetThingIDsByProfile(ctx context.Context, profileID string) ([]string, error) {
	return es.svc.GetThingIDsByProfile(ctx, profileID)
}
