// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sort"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/uiconfigs"
)

var _ uiconfigs.GroupConfigRepository = (*groupConfigRepositoryMock)(nil)

type groupConfigRepositoryMock struct {
	mu      sync.Mutex
	configs map[string]uiconfigs.GroupConfig
}

// NewGroupConfigRepository creates in-memory group config repository used for testing.
func NewGroupConfigRepository() uiconfigs.GroupConfigRepository {
	return &groupConfigRepositoryMock{
		configs: make(map[string]uiconfigs.GroupConfig),
	}
}

func (gcrm *groupConfigRepositoryMock) Save(_ context.Context, gc uiconfigs.GroupConfig) (uiconfigs.GroupConfig, error) {
	gcrm.mu.Lock()
	defer gcrm.mu.Unlock()

	if _, ok := gcrm.configs[gc.GroupID]; ok {
		return uiconfigs.GroupConfig{}, dbutil.ErrConflict
	}

	gcrm.configs[gc.GroupID] = gc

	return gc, nil
}

func (gcrm *groupConfigRepositoryMock) RetrieveByGroup(_ context.Context, groupID string) (uiconfigs.GroupConfig, error) {
	gcrm.mu.Lock()
	defer gcrm.mu.Unlock()

	gc, ok := gcrm.configs[groupID]
	if !ok {
		return uiconfigs.GroupConfig{GroupID: groupID, Config: make(uiconfigs.Config)}, nil
	}

	return gc, nil
}

func (gcrm *groupConfigRepositoryMock) RetrieveAll(_ context.Context, pm apiutil.PageMetadata) (uiconfigs.GroupConfigPage, error) {
	gcrm.mu.Lock()
	defer gcrm.mu.Unlock()

	ids := make([]string, 0, len(gcrm.configs))
	for id := range gcrm.configs {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var page []string
	if pm.Offset < uint64(len(ids)) {
		end := uint64(len(ids))
		if pm.Limit > 0 && pm.Offset+pm.Limit < end {
			end = pm.Offset + pm.Limit
		}
		page = ids[pm.Offset:end]
	}

	configs := make([]uiconfigs.GroupConfig, 0, len(page))
	for _, id := range page {
		configs = append(configs, gcrm.configs[id])
	}

	return uiconfigs.GroupConfigPage{
		Total:         uint64(len(ids)),
		GroupsConfigs: configs,
	}, nil
}

func (gcrm *groupConfigRepositoryMock) Update(ctx context.Context, gc uiconfigs.GroupConfig) error {
	gcrm.mu.Lock()

	existing, ok := gcrm.configs[gc.GroupID]
	if !ok {
		gcrm.mu.Unlock()
		_, err := gcrm.Save(ctx, gc)
		return err
	}

	existing.Config = gc.Config
	gcrm.configs[gc.GroupID] = existing
	gcrm.mu.Unlock()

	return nil
}

func (gcrm *groupConfigRepositoryMock) Remove(_ context.Context, groupID string) error {
	gcrm.mu.Lock()
	defer gcrm.mu.Unlock()

	delete(gcrm.configs, groupID)

	return nil
}

func (gcrm *groupConfigRepositoryMock) BackupAll(_ context.Context) (uiconfigs.GroupConfigBackup, error) {
	gcrm.mu.Lock()
	defer gcrm.mu.Unlock()

	configs := make([]uiconfigs.GroupConfig, 0, len(gcrm.configs))
	for _, gc := range gcrm.configs {
		configs = append(configs, gc)
	}

	return uiconfigs.GroupConfigBackup{GroupsConfigs: configs}, nil
}
