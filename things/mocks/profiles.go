// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"fmt"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
)

var _ things.ProfileRepository = (*profileRepositoryMock)(nil)

type profileRepositoryMock struct {
	mu       sync.Mutex
	counter  uint64
	profiles map[string]things.Profile
	things   things.ThingRepository
}

// NewProfileRepository creates in-memory profile repository.
func NewProfileRepository(repo things.ThingRepository) things.ProfileRepository {
	return &profileRepositoryMock{
		profiles: make(map[string]things.Profile),
		things:   repo,
	}
}

func (crm *profileRepositoryMock) Save(_ context.Context, profiles ...things.Profile) ([]things.Profile, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	for i := range profiles {
		crm.counter++
		if profiles[i].ID == "" {
			profiles[i].ID = fmt.Sprintf("%03d", crm.counter)
		}
		crm.profiles[profiles[i].ID] = profiles[i]
	}

	return profiles, nil
}

func (crm *profileRepositoryMock) Update(_ context.Context, profile things.Profile) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	if _, ok := crm.profiles[profile.ID]; !ok {
		return errors.ErrNotFound
	}
	profile.GroupID = crm.profiles[profile.ID].GroupID

	crm.profiles[profile.ID] = profile
	return nil
}

func (crm *profileRepositoryMock) RetrieveByID(_ context.Context, id string) (things.Profile, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	for _, pr := range crm.profiles {
		if pr.ID == id {
			return pr, nil
		}
	}

	return things.Profile{}, errors.ErrNotFound
}

func (crm *profileRepositoryMock) RetrieveByGroupIDs(_ context.Context, groupIDs []string, pm apiutil.PageMetadata) (things.ProfilesPage, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	items := make([]things.Profile, 0)
	filteredItems := make([]things.Profile, 0)

	if pm.Limit == 0 {
		return things.ProfilesPage{}, nil
	}

	first := uint64(pm.Offset) + 1
	last := first + pm.Limit

	for _, grID := range groupIDs {
		for _, v := range crm.profiles {
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

	page := things.ProfilesPage{
		Profiles: items,
		PageMetadata: apiutil.PageMetadata{
			Total:  crm.counter,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (crm *profileRepositoryMock) RetrieveByAdmin(_ context.Context, pm apiutil.PageMetadata) (things.ProfilesPage, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	i := uint64(0)
	var prs []things.Profile
	for _, pr := range crm.profiles {
		if i >= pm.Offset && i < pm.Offset+pm.Limit {
			prs = append(prs, pr)
		}
		i++
	}

	page := things.ProfilesPage{
		Profiles: prs,
		PageMetadata: apiutil.PageMetadata{
			Total:  crm.counter,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (crm *profileRepositoryMock) RetrieveByThing(_ context.Context, thID string) (things.Profile, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	thing, _ := crm.things.RetrieveByID(context.Background(), thID)
	for _, pr := range crm.profiles {
		if pr.ID == thing.ProfileID {
			return pr, nil
		}
	}

	return things.Profile{}, errors.ErrNotFound
}

func (crm *profileRepositoryMock) Remove(_ context.Context, ids ...string) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	for _, id := range ids {
		if _, ok := crm.profiles[id]; !ok {
			return errors.ErrNotFound
		}

		delete(crm.profiles, id)
	}

	return nil
}

func (crm *profileRepositoryMock) RetrieveAll(_ context.Context) ([]things.Profile, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	var prs []things.Profile
	for _, v := range crm.profiles {
		prs = append(prs, v)
	}

	return prs, nil
}
