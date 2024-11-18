// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"strconv"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
)

var _ things.Service = (*mainfluxThings)(nil)

type mainfluxThings struct {
	mu          sync.Mutex
	counter     uint64
	things      map[string]things.Thing
	profiles    map[string]things.Profile
	auth        protomfx.AuthServiceClient
	connections map[string][]string
}

// NewThingsService returns Mainflux Things service mock.
// Only methods used by SDK are mocked.
func NewThingsService(things map[string]things.Thing, profiles map[string]things.Profile, auth protomfx.AuthServiceClient) things.Service {
	return &mainfluxThings{
		things:      things,
		profiles:    profiles,
		auth:        auth,
		connections: make(map[string][]string),
	}
}

func (svc *mainfluxThings) CreateThings(_ context.Context, token string, ths ...things.Thing) ([]things.Thing, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	for i := range ths {
		svc.counter++
		ths[i].ID = strconv.FormatUint(svc.counter, 10)
		ths[i].Key = ths[i].ID
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

	return things.Thing{}, errors.ErrNotFound
}

func (svc *mainfluxThings) Connect(_ context.Context, token string, chID string, thIDs []string) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	svc.connections[chID] = append(svc.connections[chID], thIDs...)

	return nil
}

func (svc *mainfluxThings) Disconnect(_ context.Context, token string, chID string, thIDs []string) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	ids := svc.connections[chID]
	var count int
	var newConns []string
	for _, thID := range thIDs {
		for _, id := range ids {
			if id == thID {
				count++
				continue
			}
			newConns = append(newConns, id)
		}

		if len(newConns)-len(ids) != count {
			return errors.ErrNotFound
		}
		svc.connections[chID] = newConns
	}

	return nil
}

func (svc *mainfluxThings) RemoveThings(_ context.Context, token string, ids ...string) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	for _, id := range ids {
		if _, ok := svc.things[id]; !ok {
			return errors.ErrNotFound
		}

		delete(svc.things, id)
	}

	return nil
}

func (svc *mainfluxThings) ViewProfile(_ context.Context, token, id string) (things.Profile, error) {
	if c, ok := svc.profiles[id]; ok {
		return c, nil
	}
	return things.Profile{}, errors.ErrNotFound
}

func (svc *mainfluxThings) UpdateThing(context.Context, string, things.Thing) error {
	panic("not implemented")
}

func (svc *mainfluxThings) UpdateKey(context.Context, string, string, string) error {
	panic("not implemented")
}

func (svc *mainfluxThings) ListThings(context.Context, string, things.PageMetadata) (things.ThingsPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ViewProfileByThing(context.Context, string, string) (things.Profile, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ListThingsByProfile(context.Context, string, string, things.PageMetadata) (things.ThingsPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) Backup(context.Context, string) (things.Backup, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) Restore(context.Context, string, things.Backup) error {
	panic("not implemented")
}

func (svc *mainfluxThings) CreateProfiles(_ context.Context, token string, chs ...things.Profile) ([]things.Profile, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	for i := range chs {
		svc.counter++
		chs[i].ID = strconv.FormatUint(svc.counter, 10)
		svc.profiles[chs[i].ID] = chs[i]
	}

	return chs, nil
}

func (svc *mainfluxThings) UpdateProfile(context.Context, string, things.Profile) error {
	panic("not implemented")
}

func (svc *mainfluxThings) ListProfiles(context.Context, string, things.PageMetadata) (things.ProfilesPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) RemoveProfiles(context.Context, string, ...string) error {
	panic("not implemented")
}

func (svc *mainfluxThings) ViewProfileConfig(_ context.Context, chID string) (things.Config, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) GetConnByKey(context.Context, string) (things.Connection, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) Authorize(context.Context, things.AuthorizeReq) error {
	panic("not implemented")
}

func (svc *mainfluxThings) Identify(context.Context, string) (string, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) GetConfigByThingID(_ context.Context, thingID string) (things.Config, error) {
	panic("implement me")
}

func (svc *mainfluxThings) GetGroupIDByThingID(_ context.Context, thingID string) (string, error) {
	panic("implement me")
}

func (svc *mainfluxThings) ListThingsByGroup(_ context.Context, token, groupID string, pm things.PageMetadata) (things.ThingsPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) CreateGroups(_ context.Context, token string, groups ...things.Group) ([]things.Group, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ListGroups(_ context.Context, token, orgID string, pm things.PageMetadata) (things.GroupPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ListGroupsByIDs(_ context.Context, groupIDs []string) ([]things.Group, error) {
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

func (svc *mainfluxThings) ViewGroupByProfile(_ context.Context, token string, profileID string) (things.Group, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ListProfilesByGroup(_ context.Context, token, groupID string, pm things.PageMetadata) (things.ProfilesPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) CreateRolesByGroup(_ context.Context, token string, gms ...things.GroupMember) error {
	panic("not implemented")
}

func (svc *mainfluxThings) ListRolesByGroup(_ context.Context, token, groupID string, pm things.PageMetadata) (things.GroupMembersPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) UpdateRolesByGroup(_ context.Context, token string, gms ...things.GroupMember) error {
	panic("not implemented")
}

func (svc *mainfluxThings) RemoveRolesByGroup(_ context.Context, token, groupID string, memberIDs ...string) error {
	panic("not implemented")
}
