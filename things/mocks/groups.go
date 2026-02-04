// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
)

var _ things.GroupRepository = (*groupRepositoryMock)(nil)

type groupRepositoryMock struct {
	mu sync.Mutex
	// Map of groups where group id is a key and group is a value.
	groups               map[string]things.Group
	groupMembershipsRepo things.GroupMembershipsRepository
}

// NewGroupRepository creates in-memory user repository
func NewGroupRepository(groupMembershipsRepo things.GroupMembershipsRepository) things.GroupRepository {
	return &groupRepositoryMock{
		groups:               make(map[string]things.Group),
		groupMembershipsRepo: groupMembershipsRepo,
	}
}

func (grm *groupRepositoryMock) Save(_ context.Context, groups ...things.Group) ([]things.Group, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	for _, gr := range groups {
		if _, ok := grm.groups[gr.ID]; ok {
			return []things.Group{}, dbutil.ErrConflict
		}

		grm.groups[gr.ID] = gr
	}

	return groups, nil
}

func (grm *groupRepositoryMock) Update(_ context.Context, group things.Group) (things.Group, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	up, ok := grm.groups[group.ID]
	if !ok {
		return things.Group{}, dbutil.ErrNotFound
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
			return dbutil.ErrNotFound
		}

		delete(grm.groups, id)
	}

	return nil
}

func (grm *groupRepositoryMock) RemoveByOrg(_ context.Context, orgID string) error {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	for id, group := range grm.groups {
		if group.OrgID == orgID {
			delete(grm.groups, id)
		}
	}

	return nil
}

func (grm *groupRepositoryMock) BackupAll(_ context.Context) ([]things.Group, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	var items []things.Group
	for _, g := range grm.groups {
		items = append(items, g)
	}

	return items, nil
}

func (grm *groupRepositoryMock) BackupByOrg(_ context.Context, orgID string) ([]things.Group, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	var items []things.Group
	for _, g := range grm.groups {
		if g.OrgID == orgID {
			items = append(items, g)
		}
	}

	return items, nil
}

func (grm *groupRepositoryMock) RetrieveIDsByOrgMembership(ctx context.Context, orgID, memberID string) ([]string, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	var grIDs []string
	ids, _ := grm.groupMembershipsRepo.RetrieveGroupIDsByMember(ctx, memberID)
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
		return things.Group{}, dbutil.ErrNotFound
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
		Total:  uint64(len(items)),
	}

	return page, nil
}

func (grm *groupRepositoryMock) RetrieveAll(_ context.Context, pm apiutil.PageMetadata) (things.GroupPage, error) {
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
		Total:  uint64(len(items)),
	}

	return page, nil
}
