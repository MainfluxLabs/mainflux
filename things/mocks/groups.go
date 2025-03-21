// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
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
	// Map of group profile membership where profile id is a key and group id is a value.
	profileMembership map[string]string
	// Map of group profile where group id is a key and profile ids are values.
	profiles map[string][]string
	members  things.GroupMembersRepository
}

// NewGroupRepository creates in-memory user repository
func NewGroupRepository(members things.GroupMembersRepository) things.GroupRepository {
	return &groupRepositoryMock{
		groups:            make(map[string]things.Group),
		thingMembership:   make(map[string]string),
		things:            make(map[string][]string),
		profileMembership: make(map[string]string),
		profiles:          make(map[string][]string),
		members:           members,
	}
}

func (grm *groupRepositoryMock) Save(_ context.Context, group things.Group) (things.Group, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	if _, ok := grm.groups[group.ID]; ok {
		return things.Group{}, errors.ErrConflict
	}

	grm.groups[group.ID] = group
	return group, nil
}

func (grm *groupRepositoryMock) Update(_ context.Context, group things.Group) (things.Group, error) {
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

func (grm *groupRepositoryMock) Remove(_ context.Context, ids ...string) error {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	for _, id := range ids {
		if _, ok := grm.groups[id]; !ok {
			return errors.ErrNotFound
		}

		for _, thingID := range grm.things[id] {
			delete(grm.thingMembership, thingID)
		}

		for _, profileID := range grm.profiles[id] {
			delete(grm.profileMembership, profileID)
		}

		// This is not quite exact, it should go in depth
		delete(grm.groups, id)
	}
	return nil

}

func (grm *groupRepositoryMock) RetrieveAll(_ context.Context) ([]things.Group, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	var items []things.Group
	for _, g := range grm.groups {
		items = append(items, g)
	}

	return items, nil
}

func (grm *groupRepositoryMock) RetrieveIDsByOrgMember(ctx context.Context, orgID, memberID string) ([]string, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	var grIDs []string
	ids, _ := grm.members.RetrieveGroupIDsByMember(ctx, memberID)
	for _, gr := range grm.groups {
		for _, id := range ids {
			if gr.OrgID == orgID && gr.ID == id {
				grIDs = append(grIDs, gr.ID)
			}
		}
	}

	return grIDs, nil
}

func (grm *groupRepositoryMock) RetrieveIDsByOrg(_ context.Context, orgID string) ([]string, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	var grIDs []string
	for _, gr := range grm.groups {
		if gr.OrgID == orgID {
			grIDs = append(grIDs, gr.ID)
			continue
		}

	}

	return grIDs, nil
}

func (grm *groupRepositoryMock) RetrieveByID(_ context.Context, id string) (things.Group, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	val, ok := grm.groups[id]
	if !ok {
		return things.Group{}, errors.ErrNotFound
	}
	return val, nil
}

func (grm *groupRepositoryMock) RetrieveByIDs(_ context.Context, ids []string, pm apiutil.PageMetadata) (things.GroupPage, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	items := make([]things.Group, 0)
	filteredItems := make([]things.Group, 0)

	if pm.Limit == 0 {
		return things.GroupPage{}, nil
	}

	first := uint64(pm.Offset) + 1
	last := first + pm.Limit

	for _, grID := range ids {
		for _, v := range grm.groups {
			if v.ID == grID {
				id := uuid.ParseID(v.ID)
				if id >= first && id < last {
					items = append(items, v)
				}
			}
		}
	}

	if pm.Name != "" {
		for _, v := range items {
			if strings.Contains(v.Name, pm.Name) {
				filteredItems = append(filteredItems, v)
			}
		}
		items = filteredItems
	}

	items = mocks.SortItems(pm.Order, pm.Dir, items, func(i int) (string, string) {
		return items[i].Name, items[i].ID
	})

	page := things.GroupPage{
		Groups: items,
		PageMetadata: apiutil.PageMetadata{
			Total:  uint64(len(items)),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (grm *groupRepositoryMock) RetrieveByAdmin(_ context.Context, pm apiutil.PageMetadata) (things.GroupPage, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	items := make([]things.Group, 0)
	filteredItems := make([]things.Group, 0)

	if pm.Limit == 0 {
		return things.GroupPage{}, nil
	}

	first := pm.Offset + 1
	last := first + pm.Limit

	for _, v := range grm.groups {
		id := uuid.ParseID(v.ID)
		if id >= first && id < last {
			items = append(items, v)
		}
	}

	if pm.Name != "" {
		for _, v := range items {
			if strings.Contains(v.Name, pm.Name) {
				filteredItems = append(filteredItems, v)
			}
		}
		items = filteredItems
	}

	items = mocks.SortItems(pm.Order, pm.Dir, items, func(i int) (string, string) {
		return items[i].Name, items[i].ID
	})

	page := things.GroupPage{
		Groups: items,
		PageMetadata: apiutil.PageMetadata{
			Total:  uint64(len(items)),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}
