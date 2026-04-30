// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/filestore"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
)

// ThingsRepository is an in-memory filestore.ThingsRepository for tests.
type ThingsRepository struct {
	mu    sync.Mutex
	byKey map[string]filestore.FileInfo
	group map[string]string // thingID -> groupID
}

func NewThingsRepository() *ThingsRepository {
	return &ThingsRepository{
		byKey: map[string]filestore.FileInfo{},
		group: map[string]string{},
	}
}

func thingKey(thingID string, fi filestore.FileInfo) string {
	return thingID + "|" + fi.Class + "|" + fi.Format + "|" + fi.Name
}

func (r *ThingsRepository) Save(_ context.Context, thingID, groupID string, fi filestore.FileInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byKey[thingKey(thingID, fi)] = fi
	r.group[thingID] = groupID
	return nil
}

func (r *ThingsRepository) Update(_ context.Context, thingID string, fi filestore.FileInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := thingKey(thingID, fi)
	if _, ok := r.byKey[k]; !ok {
		return dbutil.ErrNotFound
	}
	r.byKey[k] = fi
	return nil
}

func (r *ThingsRepository) Retrieve(_ context.Context, thingID string, fi filestore.FileInfo) (filestore.FileInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	got, ok := r.byKey[thingKey(thingID, fi)]
	if !ok {
		return filestore.FileInfo{}, dbutil.ErrNotFound
	}
	return got, nil
}

func (r *ThingsRepository) RetrieveByThing(_ context.Context, _ string, _ filestore.FileInfo, _ filestore.PageMetadata) (filestore.FileThingsPage, error) {
	return filestore.FileThingsPage{}, nil
}

func (r *ThingsRepository) Remove(_ context.Context, thingID string, fi filestore.FileInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byKey, thingKey(thingID, fi))
	return nil
}

func (r *ThingsRepository) RemoveByThing(_ context.Context, thingID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k := range r.byKey {
		if len(k) > len(thingID) && k[:len(thingID)+1] == thingID+"|" {
			delete(r.byKey, k)
		}
	}
	delete(r.group, thingID)
	return nil
}

func (r *ThingsRepository) RemoveByGroup(_ context.Context, groupID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for thID, gID := range r.group {
		if gID == groupID {
			delete(r.group, thID)
		}
	}
	return nil
}

func (r *ThingsRepository) RetrieveThingIDsByGroup(_ context.Context, groupID string) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var ids []string
	for thID, gID := range r.group {
		if gID == groupID {
			ids = append(ids, thID)
		}
	}
	return ids, nil
}

// GroupsRepository is an in-memory filestore.GroupsRepository for tests.
type GroupsRepository struct {
	mu     sync.Mutex
	byKey  map[string]filestore.FileInfo
	FailOn string // name that triggers a synthetic Save failure
}

func NewGroupsRepository() *GroupsRepository {
	return &GroupsRepository{byKey: map[string]filestore.FileInfo{}}
}

func groupKey(groupID string, fi filestore.FileInfo) string {
	return groupID + "|" + fi.Class + "|" + fi.Format + "|" + fi.Name
}

func (r *GroupsRepository) Save(_ context.Context, groupID string, fi filestore.FileInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.FailOn != "" && fi.Name == r.FailOn {
		return dbutil.ErrCreateEntity
	}
	r.byKey[groupKey(groupID, fi)] = fi
	return nil
}

func (r *GroupsRepository) Update(_ context.Context, groupID string, fi filestore.FileInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := groupKey(groupID, fi)
	if _, ok := r.byKey[k]; !ok {
		return dbutil.ErrNotFound
	}
	r.byKey[k] = fi
	return nil
}

func (r *GroupsRepository) Retrieve(_ context.Context, groupID string, fi filestore.FileInfo) (filestore.FileInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	got, ok := r.byKey[groupKey(groupID, fi)]
	if !ok {
		return filestore.FileInfo{}, dbutil.ErrNotFound
	}
	return got, nil
}

func (r *GroupsRepository) RetrieveByGroup(_ context.Context, _ string, _ filestore.FileInfo, _ filestore.PageMetadata) (filestore.FileGroupsPage, error) {
	return filestore.FileGroupsPage{}, nil
}

func (r *GroupsRepository) Remove(_ context.Context, groupID string, fi filestore.FileInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := groupKey(groupID, fi)
	if _, ok := r.byKey[k]; !ok {
		return dbutil.ErrNotFound
	}
	delete(r.byKey, k)
	return nil
}

func (r *GroupsRepository) RemoveByGroup(_ context.Context, groupID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k := range r.byKey {
		if len(k) > len(groupID) && k[:len(groupID)+1] == groupID+"|" {
			delete(r.byKey, k)
		}
	}
	return nil
}

func (r *GroupsRepository) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.byKey)
}
