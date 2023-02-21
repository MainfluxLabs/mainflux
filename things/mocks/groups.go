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
	// Map of groups (with group id as key) which
	// represent memberships is element in
	// memberships' map where member id is a key.
	// memberships map[MemberID]map[GroupID]auth.Group
	memberships map[string]map[string]things.Group
	// Map of group members where member id is a key
	// is an element in the map members where group id is a key.
	// members     map[type][GroupID]map[MemberID]MemberID
	members map[string]map[string]map[string]string
}

// NewGroupRepository creates in-memory user repository
func NewGroupRepository() things.GroupRepository {
	return &groupRepositoryMock{
		groups:      make(map[string]things.Group),
		memberships: make(map[string]map[string]things.Group),
		members:     make(map[string]map[string]map[string]string),
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

func (grm *groupRepositoryMock) Delete(ctx context.Context, id string) error {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	if _, ok := grm.groups[id]; !ok {
		return errors.ErrNotFound
	}

	if len(grm.members[id]) > 0 {
		return things.ErrGroupNotEmpty
	}
	// This is not quite exact, it should go in depth
	delete(grm.groups, id)

	return nil

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

func (grm *groupRepositoryMock) RetrieveAll(ctx context.Context, pm things.PageMetadata) (things.GroupPage, error) {
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

func (grm *groupRepositoryMock) Unassign(ctx context.Context, groupID string, memberIDs ...string) error {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	if _, ok := grm.groups[groupID]; !ok {
		return errors.ErrNotFound
	}
	for _, memberID := range memberIDs {
		for typ, m := range grm.members[groupID] {
			_, ok := m[memberID]
			if !ok {
				return errors.ErrNotFound
			}
			delete(grm.members[groupID][typ], memberID)
			delete(grm.memberships[memberID], groupID)
		}

	}
	return nil
}

func (grm *groupRepositoryMock) Assign(ctx context.Context, groupID string, memberIDs ...string) error {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	if _, ok := grm.groups[groupID]; !ok {
		return errors.ErrNotFound
	}

	if _, ok := grm.members[groupID]; !ok {
		grm.members[groupID] = make(map[string]map[string]string)
	}

	for _, memberID := range memberIDs {
		if _, ok := grm.members[groupID][groupID]; !ok {
			grm.members[groupID][groupID] = make(map[string]string)
		}
		if _, ok := grm.memberships[memberID]; !ok {
			grm.memberships[memberID] = make(map[string]things.Group)
		}

		grm.members[groupID][groupID][memberID] = memberID
		grm.memberships[memberID][groupID] = grm.groups[groupID]
	}
	return nil

}

func (grm *groupRepositoryMock) Memberships(ctx context.Context, memberID string, pm things.PageMetadata) (things.GroupPage, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	var items []things.Group

	first := uint64(pm.Offset)
	last := first + uint64(pm.Limit)

	i := uint64(0)
	for _, g := range grm.memberships[memberID] {
		if i >= first && i < last {
			items = append(items, g)
		}
		i++
	}

	return things.GroupPage{
		Groups: items,
		PageMetadata: things.PageMetadata{
			Limit:  pm.Limit,
			Offset: pm.Offset,
			Total:  uint64(len(items)),
		},
	}, nil
}

func (grm *groupRepositoryMock) Members(ctx context.Context, groupID string, pm things.PageMetadata) (things.MemberPage, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	var items []string
	members, ok := grm.members[groupID]
	if !ok {
		return things.MemberPage{}, errors.ErrNotFound
	}

	first := uint64(pm.Offset)
	last := first + uint64(pm.Limit)

	i := uint64(0)
	for _, g := range members {
		if i >= first && i < last {
			items = append(items, g[groupID])
		}
		i++
	}
	return things.MemberPage{
		Members: items,
		PageMetadata: things.PageMetadata{
			Total: uint64(len(items)),
		},
	}, nil
}
