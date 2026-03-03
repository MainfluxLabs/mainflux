// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/downlinks"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
)

var _ downlinks.DownlinkRepository = (*downlinkRepositoryMock)(nil)

type downlinkRepositoryMock struct {
	mu        sync.Mutex
	downlinks map[string]downlinks.Downlink
}

// NewDownlinkRepository creates an in-memory downlink repository.
func NewDownlinkRepository() downlinks.DownlinkRepository {
	return &downlinkRepositoryMock{
		downlinks: make(map[string]downlinks.Downlink),
	}
}

func (drm *downlinkRepositoryMock) Save(_ context.Context, dls ...downlinks.Downlink) ([]downlinks.Downlink, error) {
	drm.mu.Lock()
	defer drm.mu.Unlock()

	for _, dl := range dls {
		drm.downlinks[dl.ID] = dl
	}

	return dls, nil
}

func (drm *downlinkRepositoryMock) RetrieveByThing(_ context.Context, thingID string, pm apiutil.PageMetadata) (downlinks.DownlinksPage, error) {
	drm.mu.Lock()
	defer drm.mu.Unlock()

	var items []downlinks.Downlink

	first := uint64(pm.Offset) + 1
	last := first + pm.Limit

	for _, dl := range drm.downlinks {
		if dl.ThingID == thingID {
			id := uuid.ParseID(dl.ID)
			if id >= first && id < last || pm.Limit == 0 {
				items = append(items, dl)
			}
		}
	}

	return downlinks.DownlinksPage{
		Downlinks: items,
		PageMetadata: apiutil.PageMetadata{
			Total: uint64(len(items)),
		},
	}, nil
}

func (drm *downlinkRepositoryMock) RetrieveByGroup(_ context.Context, groupID string, pm apiutil.PageMetadata) (downlinks.DownlinksPage, error) {
	drm.mu.Lock()
	defer drm.mu.Unlock()

	var items []downlinks.Downlink

	first := uint64(pm.Offset) + 1
	last := first + pm.Limit

	for _, dl := range drm.downlinks {
		if dl.GroupID == groupID {
			id := uuid.ParseID(dl.ID)
			if id >= first && id < last || pm.Limit == 0 {
				items = append(items, dl)
			}
		}
	}

	return downlinks.DownlinksPage{
		Downlinks: items,
		PageMetadata: apiutil.PageMetadata{
			Total: uint64(len(items)),
		},
	}, nil
}

func (drm *downlinkRepositoryMock) RetrieveByID(_ context.Context, id string) (downlinks.Downlink, error) {
	drm.mu.Lock()
	defer drm.mu.Unlock()

	dl, ok := drm.downlinks[id]
	if !ok {
		return downlinks.Downlink{}, dbutil.ErrNotFound
	}

	return dl, nil
}

func (drm *downlinkRepositoryMock) RetrieveAll(_ context.Context) ([]downlinks.Downlink, error) {
	drm.mu.Lock()
	defer drm.mu.Unlock()

	var items []downlinks.Downlink
	for _, dl := range drm.downlinks {
		items = append(items, dl)
	}

	return items, nil
}

func (drm *downlinkRepositoryMock) Update(_ context.Context, dl downlinks.Downlink) error {
	drm.mu.Lock()
	defer drm.mu.Unlock()

	if _, ok := drm.downlinks[dl.ID]; !ok {
		return dbutil.ErrNotFound
	}

	drm.downlinks[dl.ID] = dl
	return nil
}

func (drm *downlinkRepositoryMock) Remove(_ context.Context, ids ...string) error {
	drm.mu.Lock()
	defer drm.mu.Unlock()

	for _, id := range ids {
		if _, ok := drm.downlinks[id]; !ok {
			return dbutil.ErrNotFound
		}
		delete(drm.downlinks, id)
	}

	return nil
}

func (drm *downlinkRepositoryMock) RemoveByThing(_ context.Context, thingID string) error {
	drm.mu.Lock()
	defer drm.mu.Unlock()

	for id, dl := range drm.downlinks {
		if dl.ThingID == thingID {
			delete(drm.downlinks, id)
		}
	}

	return nil
}

func (drm *downlinkRepositoryMock) RemoveByGroup(_ context.Context, groupID string) error {
	drm.mu.Lock()
	defer drm.mu.Unlock()

	for id, dl := range drm.downlinks {
		if dl.GroupID == groupID {
			delete(drm.downlinks, id)
		}
	}

	return nil
}
