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
	// Map of group membership where member id is a key and group id is a value.
	membership map[string]string
	// Map of group member where group id is a key and member ids are values.
	members map[string][]string
}

// NewGroupRepository creates in-memory user repository
func NewGroupRepository() things.GroupRepository {
	return &groupRepositoryMock{
		groups:     make(map[string]things.Group),
		membership: make(map[string]string),
		members:    make(map[string][]string),
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

func (grm *groupRepositoryMock) Remove(ctx context.Context, id string) error {
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

func (grm *groupRepositoryMock) RetrieveByOwner(ctx context.Context, ownerID string, pm things.PageMetadata) (things.GroupPage, error) {
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

func (grm *groupRepositoryMock) UnassignMember(ctx context.Context, groupID string, memberIDs ...string) error {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	if _, ok := grm.groups[groupID]; !ok {
		return errors.ErrNotFound
	}

	for _, memberID := range memberIDs {
		members, ok := grm.members[groupID]
		if !ok {
			return errors.ErrNotFound
		}

		for i, member := range members {
			if member == memberID {
				grm.members[groupID] = append(members[:i], members[i+1:]...)
				delete(grm.membership, memberID)
				break
			}
		}
	}

	return nil
}

func (grm *groupRepositoryMock) AssignMember(ctx context.Context, groupID string, memberIDs ...string) error {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	if _, ok := grm.groups[groupID]; !ok {
		return errors.ErrNotFound
	}

	if _, ok := grm.members[groupID]; !ok {
		grm.members[groupID] = []string{}
	}

	for _, memberID := range memberIDs {
		grm.members[groupID] = append(grm.members[groupID], memberID)
		grm.membership[memberID] = groupID
	}

	return nil
}

func (grm *groupRepositoryMock) RetrieveMembership(ctx context.Context, memberID string) (string, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	groupID, ok := grm.membership[memberID]
	if !ok {
		return "", errors.ErrNotFound
	}
	return groupID, nil
}

func (grm *groupRepositoryMock) RetrieveMembers(ctx context.Context, groupID string, pm things.PageMetadata) (things.MemberPage, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	var items []things.Thing
	members, ok := grm.members[groupID]
	if !ok {
		return things.MemberPage{}, errors.ErrNotFound
	}

	first := uint64(pm.Offset)
	last := first + uint64(pm.Limit)

	if last > uint64(len(members)) {
		last = uint64(len(members))
	}

	for i := first; i < last; i++ {
		items = append(items, things.Thing{ID: members[i]})
	}

	return things.MemberPage{
		Members: items,
		PageMetadata: things.PageMetadata{
			Total: uint64(len(items)),
		},
	}, nil
}

func (grm *groupRepositoryMock) RetrieveAllGroupRelations(ctx context.Context) ([]things.GroupRelation, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	var groupRelations []things.GroupRelation
	for groupID, members := range grm.members {
		for _, memberID := range members {
			groupRelations = append(groupRelations, things.GroupRelation{
				GroupID:  groupID,
				MemberID: memberID,
			})
		}
	}

	return groupRelations, nil
}

func (grm *groupRepositoryMock) RetrieveByAdmin(ctx context.Context, pm things.PageMetadata) (things.GroupPage, error) {
	panic("not implemented")
}
