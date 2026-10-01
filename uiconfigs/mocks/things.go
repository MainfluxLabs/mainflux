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

var _ uiconfigs.ThingConfigRepository = (*thingConfigRepositoryMock)(nil)

type thingConfigRepositoryMock struct {
	mu      sync.Mutex
	configs map[string]uiconfigs.ThingConfig
}

// NewThingConfigRepository creates in-memory thing config repository used for testing.
func NewThingConfigRepository() uiconfigs.ThingConfigRepository {
	return &thingConfigRepositoryMock{
		configs: make(map[string]uiconfigs.ThingConfig),
	}
}

func (tcrm *thingConfigRepositoryMock) Save(_ context.Context, tc uiconfigs.ThingConfig) (uiconfigs.ThingConfig, error) {
	tcrm.mu.Lock()
	defer tcrm.mu.Unlock()

	if _, ok := tcrm.configs[tc.ThingID]; ok {
		return uiconfigs.ThingConfig{}, dbutil.ErrConflict
	}

	tcrm.configs[tc.ThingID] = tc

	return tc, nil
}

func (tcrm *thingConfigRepositoryMock) RetrieveByThing(_ context.Context, thingID string) (uiconfigs.ThingConfig, error) {
	tcrm.mu.Lock()
	defer tcrm.mu.Unlock()

	tc, ok := tcrm.configs[thingID]
	if !ok {
		return uiconfigs.ThingConfig{ThingID: thingID, Config: make(uiconfigs.Config)}, nil
	}

	return tc, nil
}

func (tcrm *thingConfigRepositoryMock) RetrieveAll(_ context.Context, pm apiutil.PageMetadata) (uiconfigs.ThingConfigPage, error) {
	tcrm.mu.Lock()
	defer tcrm.mu.Unlock()

	ids := make([]string, 0, len(tcrm.configs))
	for id := range tcrm.configs {
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

	configs := make([]uiconfigs.ThingConfig, 0, len(page))
	for _, id := range page {
		configs = append(configs, tcrm.configs[id])
	}

	return uiconfigs.ThingConfigPage{
		Total:         uint64(len(ids)),
		ThingsConfigs: configs,
	}, nil
}

// Update mirrors the Postgres repository, which only writes the config
// column on an existing row - the stored GroupID is left untouched.
func (tcrm *thingConfigRepositoryMock) Update(ctx context.Context, tc uiconfigs.ThingConfig) error {
	tcrm.mu.Lock()

	existing, ok := tcrm.configs[tc.ThingID]
	if !ok {
		tcrm.mu.Unlock()
		_, err := tcrm.Save(ctx, tc)
		return err
	}

	existing.Config = tc.Config
	tcrm.configs[tc.ThingID] = existing
	tcrm.mu.Unlock()

	return nil
}

func (tcrm *thingConfigRepositoryMock) Remove(_ context.Context, thingID string) error {
	tcrm.mu.Lock()
	defer tcrm.mu.Unlock()

	delete(tcrm.configs, thingID)

	return nil
}

func (tcrm *thingConfigRepositoryMock) RemoveByGroup(_ context.Context, groupID string) error {
	tcrm.mu.Lock()
	defer tcrm.mu.Unlock()

	for id, tc := range tcrm.configs {
		if tc.GroupID == groupID {
			delete(tcrm.configs, id)
		}
	}

	return nil
}

func (tcrm *thingConfigRepositoryMock) BackupAll(_ context.Context) (uiconfigs.ThingConfigBackup, error) {
	tcrm.mu.Lock()
	defer tcrm.mu.Unlock()

	configs := make([]uiconfigs.ThingConfig, 0, len(tcrm.configs))
	for _, tc := range tcrm.configs {
		configs = append(configs, tc)
	}

	return uiconfigs.ThingConfigBackup{ThingsConfigs: configs}, nil
}
