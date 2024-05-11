// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
)

var _ things.GroupRepository = (*groupRepositoryMock)(nil)

type groupRepositoryMock struct {
	mu sync.Mutex
	// Map of groups, group id as a key.
	// groups      map[GroupID]auth.Group
	groups map[string]things.Group
	// Map of group thing membership where thing id is a key and group id is a value.
	thingMembership map[string]string
	// Map of group thing where group id is a key and thing ids are values.
	things map[string][]string
	// Map of group channel membership where channel id is a key and group id is a value.
	channelMembership map[string]string
	// Map of group channel where group id is a key and channel ids are values.
	channels map[string][]string
}

// NewGroupRepository creates in-memory user repository
func NewGroupRepository() things.GroupRepository {
	return &groupRepositoryMock{
		groups:            make(map[string]things.Group),
		thingMembership:   make(map[string]string),
		things:            make(map[string][]string),
		channelMembership: make(map[string]string),
		channels:          make(map[string][]string),
	}
}

func (grm *groupRepositoryMock) Save(ctx context.Context, group things.Group) (things.Group, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	if _, ok := grm.groups[group.ID]; ok {
		return things.Group{}, errors.ErrConflict
	}

	grm.groups[group.ID] = group
	return group, nil
}

func (grm *groupRepositoryMock) Update(ctx context.Context, group things.Group) (things.Group, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	up, ok := grm.groups[group.ID]
	if !ok {
		return things.Group{}, errors.ErrNotFound
	}
	up.Name = group.Name
	up.Description = group.Description
	up.Metadata = group.Metadata
	up.UpdatedAt = time.Now()

	grm.groups[group.ID] = up
	return up, nil
}

func (grm *groupRepositoryMock) Remove(ctx context.Context, ids ...string) error {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	for _, id := range ids {
		if _, ok := grm.groups[id]; !ok {
			return errors.ErrNotFound
		}

		for _, thingID := range grm.things[id] {
			delete(grm.thingMembership, thingID)
		}

		for _, channelID := range grm.channels[id] {
			delete(grm.channelMembership, channelID)
		}

		// This is not quite exact, it should go in depth
		delete(grm.groups, id)
	}
	return nil

}

func (grm *groupRepositoryMock) RetrieveAll(ctx context.Context) ([]things.Group, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	var items []things.Group
	for _, g := range grm.groups {
		items = append(items, g)
	}

	return items, nil
}

func (grm *groupRepositoryMock) RetrieveByID(ctx context.Context, id string) (things.Group, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	val, ok := grm.groups[id]
	if !ok {
		return things.Group{}, errors.ErrNotFound
	}
	return val, nil
}

func (grm *groupRepositoryMock) RetrieveByIDs(ctx context.Context, groupIDs []string) (things.GroupPage, error) {
	panic("not implemented")
}

func (grm *groupRepositoryMock) RetrieveByOwner(ctx context.Context, ownerID, orgID string, pm things.PageMetadata) (things.GroupPage, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	var items []things.Group
	for _, g := range grm.groups {
		items = append(items, g)
	}
	return things.GroupPage{
		Groups: items,
		PageMetadata: things.PageMetadata{
			Total: uint64(len(items)),
		},
	}, nil
}

func (grm *groupRepositoryMock) RetrieveThingMembership(ctx context.Context, thingID string) (string, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	groupID, ok := grm.thingMembership[thingID]
	if !ok {
		return "", errors.ErrNotFound
	}
	return groupID, nil
}

func (grm *groupRepositoryMock) RetrieveThingsByGroup(ctx context.Context, groupID string, pm things.PageMetadata) (things.ThingsPage, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	var items []things.Thing
	ths, ok := grm.things[groupID]
	if !ok {
		return things.ThingsPage{}, errors.ErrNotFound
	}

	first := uint64(pm.Offset)
	last := first + uint64(pm.Limit)

	if last > uint64(len(ths)) {
		last = uint64(len(ths))
	}

	for i := first; i < last; i++ {
		items = append(items, things.Thing{ID: ths[i]})
	}

	return things.ThingsPage{
		Things: items,
		PageMetadata: things.PageMetadata{
			Total: uint64(len(items)),
		},
	}, nil
}

func (grm *groupRepositoryMock) RetrieveChannelMembership(ctx context.Context, channelID string) (string, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	groupID, ok := grm.channelMembership[channelID]
	if !ok {
		return "", errors.ErrNotFound
	}

	return groupID, nil
}

func (grm *groupRepositoryMock) RetrieveChannelsByGroup(ctx context.Context, groupID string, pm things.PageMetadata) (things.ChannelsPage, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	var items []things.Channel
	chs, ok := grm.channels[groupID]
	if !ok {
		return things.ChannelsPage{}, nil
	}

	first := uint64(pm.Offset)
	last := first + uint64(pm.Limit)

	if last > uint64(len(chs)) {
		last = uint64(len(chs))
	}

	for i := first; i < last; i++ {
		items = append(items, things.Channel{ID: chs[i]})
	}

	return things.ChannelsPage{
		Channels: items,
		PageMetadata: things.PageMetadata{
			Total: uint64(len(items)),
		},
	}, nil
}

func (grm *groupRepositoryMock) RetrieveGroupThingsByChannel(ctx context.Context, groupID, channelID string, pm things.PageMetadata) (things.ThingsPage, error) {
	panic("not implemented")
}

func (grm *groupRepositoryMock) RetrieveByAdmin(ctx context.Context, orgID string, pm things.PageMetadata) (things.GroupPage, error) {
	panic("not implemented")
}
