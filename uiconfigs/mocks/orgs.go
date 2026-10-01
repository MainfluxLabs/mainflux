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

var _ uiconfigs.OrgConfigRepository = (*orgConfigRepositoryMock)(nil)

type orgConfigRepositoryMock struct {
	mu      sync.Mutex
	configs map[string]uiconfigs.OrgConfig
}

// NewOrgConfigRepository creates in-memory org config repository used for testing.
func NewOrgConfigRepository() uiconfigs.OrgConfigRepository {
	return &orgConfigRepositoryMock{
		configs: make(map[string]uiconfigs.OrgConfig),
	}
}

func (ocrm *orgConfigRepositoryMock) Save(_ context.Context, oc uiconfigs.OrgConfig) (uiconfigs.OrgConfig, error) {
	ocrm.mu.Lock()
	defer ocrm.mu.Unlock()

	if _, ok := ocrm.configs[oc.OrgID]; ok {
		return uiconfigs.OrgConfig{}, dbutil.ErrConflict
	}

	ocrm.configs[oc.OrgID] = oc

	return oc, nil
}

func (ocrm *orgConfigRepositoryMock) RetrieveByOrg(_ context.Context, orgID string) (uiconfigs.OrgConfig, error) {
	ocrm.mu.Lock()
	defer ocrm.mu.Unlock()

	oc, ok := ocrm.configs[orgID]
	if !ok {
		return uiconfigs.OrgConfig{OrgID: orgID, Config: make(uiconfigs.Config)}, nil
	}

	return oc, nil
}

func (ocrm *orgConfigRepositoryMock) RetrieveAll(_ context.Context, pm apiutil.PageMetadata) (uiconfigs.OrgConfigPage, error) {
	ocrm.mu.Lock()
	defer ocrm.mu.Unlock()

	ids := make([]string, 0, len(ocrm.configs))
	for id := range ocrm.configs {
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

	configs := make([]uiconfigs.OrgConfig, 0, len(page))
	for _, id := range page {
		configs = append(configs, ocrm.configs[id])
	}

	return uiconfigs.OrgConfigPage{
		Total:       uint64(len(ids)),
		OrgsConfigs: configs,
	}, nil
}

func (ocrm *orgConfigRepositoryMock) Update(ctx context.Context, oc uiconfigs.OrgConfig) error {
	ocrm.mu.Lock()

	existing, ok := ocrm.configs[oc.OrgID]
	if !ok {
		ocrm.mu.Unlock()
		_, err := ocrm.Save(ctx, oc)
		return err
	}

	existing.Config = oc.Config
	ocrm.configs[oc.OrgID] = existing
	ocrm.mu.Unlock()

	return nil
}

func (ocrm *orgConfigRepositoryMock) Remove(_ context.Context, orgID string) error {
	ocrm.mu.Lock()
	defer ocrm.mu.Unlock()

	delete(ocrm.configs, orgID)

	return nil
}

func (ocrm *orgConfigRepositoryMock) BackupAll(_ context.Context) (uiconfigs.OrgConfigBackup, error) {
	ocrm.mu.Lock()
	defer ocrm.mu.Unlock()

	configs := make([]uiconfigs.OrgConfig, 0, len(ocrm.configs))
	for _, oc := range ocrm.configs {
		configs = append(configs, oc)
	}

	return uiconfigs.OrgConfigBackup{OrgsConfigs: configs}, nil
}
