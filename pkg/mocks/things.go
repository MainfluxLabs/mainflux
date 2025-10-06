// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"strconv"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
)

var _ things.Service = (*mainfluxThings)(nil)

type mainfluxThings struct {
	mu       sync.Mutex
	counter  uint64
	things   map[string]things.Thing
	profiles map[string]things.Profile
	auth     protomfx.AuthServiceClient
}

// NewThingsService returns Mainflux Things service mock.
// Only methods used by SDK are mocked.
func NewThingsService(things map[string]things.Thing, profiles map[string]things.Profile, auth protomfx.AuthServiceClient) things.Service {
	return &mainfluxThings{
		things:   things,
		profiles: profiles,
		auth:     auth,
	}
}

func (svc *mainfluxThings) CreateThings(_ context.Context, token, profileID string, ths ...things.Thing) ([]things.Thing, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	profile, ok := svc.profiles[profileID]
	if !ok {
		return []things.Thing{}, dbutil.ErrNotFound
	}

	for i := range ths {
		svc.counter++
		ths[i].ID = strconv.FormatUint(svc.counter, 10)
		ths[i].Key = ths[i].ID
		ths[i].ProfileID = profileID
		ths[i].GroupID = profile.GroupID
		svc.things[ths[i].ID] = ths[i]
	}

	return ths, nil
}

func (svc *mainfluxThings) ViewThing(_ context.Context, token, id string) (things.Thing, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	if t, ok := svc.things[id]; ok {
		return t, nil

	}

	return things.Thing{}, dbutil.ErrNotFound
}

func (svc *mainfluxThings) RemoveThings(_ context.Context, token string, ids ...string) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	for _, id := range ids {
		if _, ok := svc.things[id]; !ok {
			return dbutil.ErrNotFound
		}

		delete(svc.things, id)
	}

	return nil
}

func (svc *mainfluxThings) ViewProfile(_ context.Context, token, id string) (things.Profile, error) {
	if c, ok := svc.profiles[id]; ok {
		return c, nil
	}
	return things.Profile{}, dbutil.ErrNotFound
}

func (svc *mainfluxThings) UpdateThing(context.Context, string, things.Thing) error {
	panic("not implemented")
}

func (svc *mainfluxThings) UpdateThingsMetadata(context.Context, string, ...things.Thing) error {
	panic("not implemented")
}

func (svc *mainfluxThings) UpdateKey(context.Context, string, string, string) error {
	panic("not implemented")
}

func (svc *mainfluxThings) ViewMetadataByKey(context.Context, apiutil.ThingKey) (things.Metadata, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ListThings(context.Context, string, apiutil.PageMetadata) (things.ThingsPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ViewProfileByThing(context.Context, string, string) (things.Profile, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ListThingsByProfile(context.Context, string, string, apiutil.PageMetadata) (things.ThingsPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ListThingsByOrg(context.Context, string, string, apiutil.PageMetadata) (things.ThingsPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) Backup(context.Context, string) (things.Backup, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) BackupGroupsByOrg(context.Context, string, string) (things.GroupsBackup, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) RestoreGroupsByOrg(context.Context, string, string, things.GroupsBackup) error {
	panic("not implemented")
}

func (svc *mainfluxThings) BackupGroupMemberships(context.Context, string, string) (things.GroupMembershipsBackup, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) RestoreGroupMemberships(context.Context, string, string, things.GroupMembershipsBackup) error {
	panic("not implemented")
}

func (svc *mainfluxThings) BackupProfilesByOrg(context.Context, string, string) (things.ProfilesBackup, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) RestoreProfilesByOrg(context.Context, string, string, things.ProfilesBackup) error {
	panic("not implemented")
}

func (svc *mainfluxThings) BackupProfilesByGroup(context.Context, string, string) (things.ProfilesBackup, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) RestoreProfilesByGroup(context.Context, string, string, things.ProfilesBackup) error {
	panic("not implemented")
}

func (svc *mainfluxThings) BackupThingsByGroup(context.Context, string, string) (things.ThingsBackup, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) RestoreThingsByGroup(context.Context, string, string, things.ThingsBackup) error {
	panic("not implemented")
}

func (svc *mainfluxThings) BackupThingsByOrg(context.Context, string, string) (things.ThingsBackup, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) RestoreThingsByOrg(context.Context, string, string, things.ThingsBackup) error {
	panic("not implemented")
}

func (svc *mainfluxThings) Restore(context.Context, string, things.Backup) error {
	panic("not implemented")
}

func (svc *mainfluxThings) CreateProfiles(_ context.Context, token, grID string, prs ...things.Profile) ([]things.Profile, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	for i := range prs {
		svc.counter++
		prs[i].ID = strconv.FormatUint(svc.counter, 10)
		prs[i].GroupID = grID
		svc.profiles[prs[i].ID] = prs[i]
	}

	return prs, nil
}

func (svc *mainfluxThings) UpdateProfile(context.Context, string, things.Profile) error {
	panic("not implemented")
}

func (svc *mainfluxThings) ListProfiles(context.Context, string, apiutil.PageMetadata) (things.ProfilesPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ListProfilesByOrg(context.Context, string, string, apiutil.PageMetadata) (things.ProfilesPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) RemoveProfiles(context.Context, string, ...string) error {
	panic("not implemented")
}

func (svc *mainfluxThings) GetPubConfByKey(context.Context, apiutil.ThingKey) (things.PubConfInfo, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) GetConfigByThingID(_ context.Context, thingID string) (map[string]interface{}, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) CanUserAccessThing(ctx context.Context, req things.UserAccessReq) error {
	panic("not implemented")
}

func (svc *mainfluxThings) CanUserAccessProfile(ctx context.Context, req things.UserAccessReq) error {
	panic("not implemented")
}

func (svc *mainfluxThings) CanUserAccessGroup(ctx context.Context, req things.UserAccessReq) error {
	panic("not implemented")
}

func (svc *mainfluxThings) CanThingAccessGroup(context.Context, things.ThingAccessReq) error {
	panic("not implemented")
}

func (svc *mainfluxThings) Identify(context.Context, apiutil.ThingKey) (string, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) GetGroupIDByThingID(_ context.Context, thingID string) (string, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) GetGroupIDByProfileID(_ context.Context, profileID string) (string, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) GetProfileIDByThingID(_ context.Context, thingID string) (string, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) GetGroupIDsByOrg(_ context.Context, orgID string, token string) ([]string, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ListThingsByGroup(_ context.Context, token, groupID string, pm apiutil.PageMetadata) (things.ThingsPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) CreateGroups(_ context.Context, token, orgID string, groups ...things.Group) ([]things.Group, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ListGroups(_ context.Context, token string, pm apiutil.PageMetadata) (things.GroupPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ListGroupsByOrg(_ context.Context, token, orgID string, pm apiutil.PageMetadata) (things.GroupPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) RemoveGroups(_ context.Context, token string, ids ...string) error {
	panic("not implemented")
}

func (svc *mainfluxThings) UpdateGroup(_ context.Context, token string, group things.Group) (things.Group, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ViewGroup(_ context.Context, token, id string) (things.Group, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ViewGroupByThing(_ context.Context, token string, thingID string) (things.Group, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ViewGroupByProfile(_ context.Context, token, profileID string) (things.Group, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ListProfilesByGroup(_ context.Context, token, groupID string, pm apiutil.PageMetadata) (things.ProfilesPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) CreateGroupMemberships(_ context.Context, token string, gms ...things.GroupMembership) error {
	panic("not implemented")
}

func (svc *mainfluxThings) ListGroupMemberships(_ context.Context, token, groupID string, pm apiutil.PageMetadata) (things.GroupMembershipsPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) UpdateGroupMemberships(_ context.Context, token string, gms ...things.GroupMembership) error {
	panic("not implemented")
}

func (svc *mainfluxThings) RemoveGroupMemberships(_ context.Context, token, groupID string, memberIDs ...string) error {
	panic("not implemented")
}

func (svc *mainfluxThings) UpdateExternalKey(ctx context.Context, token, key, thingID string) error {
	panic("not implemented")
}

func (svc *mainfluxThings) RemoveExternalKey(ctx context.Context, token, thingID string) error {
	panic("not implemented")

}
