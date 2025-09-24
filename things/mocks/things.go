// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
)

var _ things.ThingRepository = (*thingRepositoryMock)(nil)

type thingRepositoryMock struct {
	mu           sync.Mutex
	counter      uint64
	things       map[string]things.Thing
	externalKeys map[string]string
}

// NewThingRepository creates in-memory thing repository.
func NewThingRepository() things.ThingRepository {
	repo := &thingRepositoryMock{
		things:       make(map[string]things.Thing),
		externalKeys: make(map[string]string),
	}

	return repo
}

func (trm *thingRepositoryMock) Save(_ context.Context, ths ...things.Thing) ([]things.Thing, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	for i := range ths {
		for _, th := range trm.things {
			if th.Key == ths[i].Key {
				return []things.Thing{}, dbutil.ErrConflict
			}
		}

		trm.counter++
		if ths[i].ID == "" {
			ths[i].ID = fmt.Sprintf("%03d", trm.counter)
		}
		trm.things[ths[i].ID] = ths[i]
	}

	return ths, nil
}

func (trm *thingRepositoryMock) Update(_ context.Context, thing things.Thing) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	if _, ok := trm.things[thing.ID]; !ok {
		return dbutil.ErrNotFound
	}
	thing.Key = trm.things[thing.ID].Key
	thing.GroupID = trm.things[thing.ID].GroupID

	trm.things[thing.ID] = thing

	return nil
}

func (trm *thingRepositoryMock) UpdateKey(_ context.Context, id, val string) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	for _, th := range trm.things {
		if th.Key == val {
			return dbutil.ErrConflict
		}
	}

	th, ok := trm.things[id]
	if !ok {
		return dbutil.ErrNotFound
	}

	th.Key = val
	trm.things[id] = th

	return nil
}

func (trm *thingRepositoryMock) RetrieveByID(_ context.Context, id string) (things.Thing, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	for _, th := range trm.things {
		if th.ID == id {
			return th, nil
		}
	}

	return things.Thing{}, dbutil.ErrNotFound
}

func (trm *thingRepositoryMock) RetrieveByGroups(_ context.Context, groupIDs []string, pm apiutil.PageMetadata) (things.ThingsPage, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	items := make([]things.Thing, 0)
	filteredItems := make([]things.Thing, 0)

	if pm.Limit == 0 {
		return things.ThingsPage{}, nil
	}

	first := uint64(pm.Offset) + 1
	last := first + pm.Limit

	for _, grID := range groupIDs {
		for _, v := range trm.things {
			if v.GroupID == grID {
				id := uuid.ParseID(v.ID)
				if id >= first && id < last {
					items = append(items, v)
				}
			}
		}
	}

	if pm.Name != "" {
		for _, v := range items {
			if v.Name == pm.Name {
				filteredItems = append(filteredItems, v)
			}
		}
		items = filteredItems
	}

	items = mocks.SortItems(pm.Order, pm.Dir, items, func(i int) (string, string) {
		return items[i].Name, items[i].ID
	})

	page := things.ThingsPage{
		Things: items,
		Total:  trm.counter,
	}

	return page, nil
}

func (trm *thingRepositoryMock) RetrieveByProfile(_ context.Context, chID string, pm apiutil.PageMetadata) (things.ThingsPage, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	first := uint64(pm.Offset) + 1
	last := first + uint64(pm.Limit)

	var ths []things.Thing

	for _, t := range trm.things {
		if t.ProfileID == chID {
			id := uuid.ParseID(t.ID)
			if id >= first && id < last || pm.Limit == 0 {
				ths = append(ths, t)
			}
		}
	}

	// Sort Things by Profile list
	ths = mocks.SortItems(pm.Order, pm.Dir, ths, func(i int) (string, string) {
		return ths[i].Name, ths[i].ID
	})

	page := things.ThingsPage{
		Things: ths,
		Total:  uint64(len(ths)),
	}

	return page, nil
}

func (trm *thingRepositoryMock) Remove(_ context.Context, ids ...string) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	for _, id := range ids {
		if _, ok := trm.things[id]; !ok {
			return dbutil.ErrNotFound
		}
		delete(trm.things, id)
	}

	return nil
}

func (trm *thingRepositoryMock) RetrieveByKey(_ context.Context, keyType, key string) (string, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	switch keyType {
	case things.KeyTypeInline:
		for _, thing := range trm.things {
			if thing.Key == key {
				return thing.ID, nil
			}
		}
	case things.KeyTypeExternal:
		if _, ok := trm.externalKeys[key]; !ok {
			return "", dbutil.ErrNotFound
		}

		return trm.externalKeys[key], nil
	default:
		return "", errors.New("invalid thing key type")
	}

	return "", dbutil.ErrNotFound
}

func (trm *thingRepositoryMock) SaveExternalKey(_ context.Context, key, thingID string) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	if _, ok := trm.externalKeys[key]; ok {
		return dbutil.ErrConflict
	}

	if _, ok := trm.things[thingID]; !ok {
		return dbutil.ErrNotFound
	}

	trm.externalKeys[key] = thingID

	return nil
}

func (trm *thingRepositoryMock) RemoveExternalKey(_ context.Context, key string) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	if _, ok := trm.externalKeys[key]; !ok {
		return dbutil.ErrNotFound
	}

	delete(trm.externalKeys, key)

	return nil
}

func (trm *thingRepositoryMock) RetrieveExternalKeysByThing(_ context.Context, thingID string) ([]string, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	var keys []string

	for key, keyThingID := range trm.externalKeys {
		if keyThingID == thingID {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

func (trm *thingRepositoryMock) BackupAll(_ context.Context) ([]things.Thing, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()
	var ths []things.Thing

	for _, th := range trm.things {
		ths = append(ths, th)
	}

	return ths, nil
}

func (trm *thingRepositoryMock) RetrieveAll(_ context.Context, pm apiutil.PageMetadata) (things.ThingsPage, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	i := uint64(0)
	var ths []things.Thing
	for _, th := range trm.things {
		if i >= pm.Offset && i < pm.Offset+pm.Limit {
			ths = append(ths, th)
		}
		i++
	}

	page := things.ThingsPage{
		Things: ths,
		Total:  trm.counter,
	}

	return page, nil
}

func (trm *thingRepositoryMock) BackupByGroups(_ context.Context, groupIDs []string) ([]things.Thing, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	var ths []things.Thing
	for _, grID := range groupIDs {
		for _, v := range trm.things {
			if v.GroupID == grID {
				ths = append(ths, v)
			}
		}
	}

	return ths, nil
}
