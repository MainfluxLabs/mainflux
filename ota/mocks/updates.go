// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/ota"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
)

var _ ota.UpdateRepository = (*updateRepositoryMock)(nil)

type updateRepositoryMock struct {
	mu           sync.Mutex
	updates      map[string]ota.Update
	thingUpdates map[string]ota.ThingUpdate
}

// NewUpdateRepository returns an in-memory update repository.
func NewUpdateRepository() ota.UpdateRepository {
	return &updateRepositoryMock{
		updates:      make(map[string]ota.Update),
		thingUpdates: make(map[string]ota.ThingUpdate),
	}
}

func thingUpdateKey(updateID, thingID string) string {
	return updateID + "/" + thingID
}

func isActive(status string) bool {
	return slices.Contains(ota.ActiveStatuses, status)
}

func (urm *updateRepositoryMock) Save(_ context.Context, u ota.Update, thingIDs ...string) error {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	if _, ok := urm.updates[u.ID]; ok {
		return dbutil.ErrConflict
	}

	for _, thingID := range thingIDs {
		for k, old := range urm.thingUpdates {
			if old.ThingID == thingID && isActive(old.Status) {
				old.Status = ota.StatusFailed
				old.Error = fmt.Sprintf(ota.ErrMsgSuperseded, u.ID)
				old.UpdatedAt = u.CreatedAt
				urm.thingUpdates[k] = old
			}
		}
	}

	urm.updates[u.ID] = u
	for _, thingID := range thingIDs {
		urm.thingUpdates[thingUpdateKey(u.ID, thingID)] = ota.ThingUpdate{
			UpdateID:  u.ID,
			ThingID:   thingID,
			Status:    ota.StatusPending,
			UpdatedAt: u.CreatedAt,
		}
	}

	return nil
}

func (urm *updateRepositoryMock) RetrieveByID(_ context.Context, id string) (ota.Update, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	u, ok := urm.updates[id]
	if !ok {
		return ota.Update{}, dbutil.ErrNotFound
	}
	u.Stats = urm.stats(id)

	return u, nil
}

func (urm *updateRepositoryMock) RetrieveByGroup(_ context.Context, groupID string, pm ota.PageMetadata) (ota.UpdatesPage, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	var items []ota.Update
	for _, u := range urm.updates {
		if u.GroupID == groupID {
			u.Stats = urm.stats(u.ID)
			items = append(items, u)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })

	total := uint64(len(items))
	start, end := pageBounds(total, pm)

	return ota.UpdatesPage{
		Total:   total,
		Updates: items[start:end],
	}, nil
}

func (urm *updateRepositoryMock) Remove(_ context.Context, id string) error {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	delete(urm.updates, id)
	for k, tu := range urm.thingUpdates {
		if tu.UpdateID == id {
			delete(urm.thingUpdates, k)
		}
	}

	return nil
}

func (urm *updateRepositoryMock) RemoveByGroup(_ context.Context, groupID string) error {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	for id, u := range urm.updates {
		if u.GroupID != groupID {
			continue
		}
		delete(urm.updates, id)
		for k, tu := range urm.thingUpdates {
			if tu.UpdateID == id {
				delete(urm.thingUpdates, k)
			}
		}
	}

	return nil
}

func (urm *updateRepositoryMock) RetrieveThingUpdates(_ context.Context, updateID string, pm ota.PageMetadata) (ota.ThingUpdatesPage, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	var items []ota.ThingUpdate
	for _, tu := range urm.thingUpdates {
		if tu.UpdateID == updateID && (pm.Status == "" || tu.Status == pm.Status) {
			items = append(items, tu)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ThingID < items[j].ThingID })

	total := uint64(len(items))
	start, end := pageBounds(total, pm)

	return ota.ThingUpdatesPage{
		Total:        total,
		ThingUpdates: items[start:end],
	}, nil
}

func (urm *updateRepositoryMock) RetrieveThingUpdate(_ context.Context, updateID, thingID string) (ota.ThingUpdate, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	tu, ok := urm.thingUpdates[thingUpdateKey(updateID, thingID)]
	if !ok {
		return ota.ThingUpdate{}, dbutil.ErrNotFound
	}

	return tu, nil
}

func (urm *updateRepositoryMock) RetrieveActiveThingUpdate(_ context.Context, thingID string) (ota.ThingUpdate, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	var newest ota.ThingUpdate
	found := false
	for _, tu := range urm.thingUpdates {
		if tu.ThingID != thingID || !isActive(tu.Status) {
			continue
		}
		if !found || urm.updates[tu.UpdateID].CreatedAt.After(urm.updates[newest.UpdateID].CreatedAt) {
			newest = tu
			found = true
		}
	}

	if !found {
		return ota.ThingUpdate{}, dbutil.ErrNotFound
	}

	return newest, nil
}

func (urm *updateRepositoryMock) UpdateThingUpdate(_ context.Context, tu ota.ThingUpdate) error {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	k := thingUpdateKey(tu.UpdateID, tu.ThingID)
	if _, ok := urm.thingUpdates[k]; !ok {
		return dbutil.ErrNotFound
	}
	urm.thingUpdates[k] = tu

	return nil
}

func (urm *updateRepositoryMock) RetryFailed(_ context.Context, updateID string) ([]string, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	var thingIDs []string
	for k, tu := range urm.thingUpdates {
		if tu.UpdateID != updateID || tu.Status != ota.StatusFailed {
			continue
		}
		tu.Status = ota.StatusPending
		tu.Error = ""
		tu.UpdatedAt = time.Now().UTC()
		urm.thingUpdates[k] = tu
		thingIDs = append(thingIDs, tu.ThingID)
	}

	return thingIDs, nil
}

func (urm *updateRepositoryMock) FailTimedOut(_ context.Context, now time.Time) error {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	for k, tu := range urm.thingUpdates {
		if isActive(tu.Status) && tu.UpdatedAt.Add(urm.updates[tu.UpdateID].Timeout).Before(now) {
			tu.Status = ota.StatusFailed
			tu.Error = ota.ErrMsgTimedOut
			tu.UpdatedAt = now
			urm.thingUpdates[k] = tu
		}
	}

	return nil
}

func (urm *updateRepositoryMock) RemoveThingUpdatesByThing(_ context.Context, thingID string) error {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	for k, tu := range urm.thingUpdates {
		if tu.ThingID == thingID {
			delete(urm.thingUpdates, k)
		}
	}

	return nil
}

func (urm *updateRepositoryMock) stats(updateID string) ota.UpdateStats {
	var s ota.UpdateStats
	for _, tu := range urm.thingUpdates {
		if tu.UpdateID != updateID {
			continue
		}
		s.Targets++
		switch tu.Status {
		case ota.StatusPending:
			s.Pending++
		case ota.StatusDownloading:
			s.Downloading++
		case ota.StatusInstalling:
			s.Installing++
		case ota.StatusSucceeded:
			s.Succeeded++
		case ota.StatusFailed:
			s.Failed++
		}
	}

	return s
}

func pageBounds(total uint64, pm ota.PageMetadata) (uint64, uint64) {
	start := min(pm.Offset, total)
	end := total
	if pm.Limit > 0 {
		end = min(start+pm.Limit, total)
	}

	return start, end
}
