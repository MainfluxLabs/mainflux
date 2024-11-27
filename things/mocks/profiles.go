// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"fmt"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
)

// Connection represents connection between profile and thing that is used for
// testing purposes.
type Connection struct {
	profileID string
	thing     things.Thing
	connected bool
}

var _ things.ProfileRepository = (*profileRepositoryMock)(nil)

type profileRepositoryMock struct {
	mu       sync.Mutex
	counter  uint64
	profiles map[string]things.Profile
	tconns   chan Connection                      // used for synchronization with thing repo
	cconns   map[string]map[string]things.Profile // used to track connections
	conns    map[string]string                    // used to track connections
	things   things.ThingRepository
}

// NewProfileRepository creates in-memory profile repository.
func NewProfileRepository(repo things.ThingRepository, tconns chan Connection) things.ProfileRepository {
	return &profileRepositoryMock{
		profiles: make(map[string]things.Profile),
		tconns:   tconns,
		cconns:   make(map[string]map[string]things.Profile),
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

func (crm *profileRepositoryMock) RetrieveByGroupIDs(_ context.Context, groupIDs []string, pm things.PageMetadata) (things.ProfilesPage, error) {
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

	items = sortItems(pm, items, func(i int) (string, string) {
		return items[i].Name, items[i].ID
	})

	page := things.ProfilesPage{
		Profiles: items,
		PageMetadata: things.PageMetadata{
			Total:  crm.counter,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (crm *profileRepositoryMock) RetrieveByAdmin(_ context.Context, pm things.PageMetadata) (things.ProfilesPage, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	if pm.Limit < 0 {
		return things.ProfilesPage{}, nil
	}

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
		PageMetadata: things.PageMetadata{
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

	for _, pr := range crm.profiles {
		for _, co := range crm.cconns[thID] {
			if pr.ID == co.ID {
				return pr, nil
			}
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

		for thk := range crm.cconns {
			delete(crm.cconns[thk], id)
		}
		crm.tconns <- Connection{
			profileID: id,
			connected: false,
		}
	}

	return nil
}

func (crm *profileRepositoryMock) Connect(_ context.Context, prID string, thIDs []string) error {
	pr, err := crm.RetrieveByID(context.Background(), prID)
	if err != nil {
		return err
	}

	for _, thID := range thIDs {
		if _, ok := crm.cconns[thID]; ok {
			return errors.ErrConflict
		}
		th, err := crm.things.RetrieveByID(context.Background(), thID)
		if err != nil {
			return err
		}
		crm.tconns <- Connection{
			profileID: prID,
			thing:     th,
			connected: true,
		}
		if _, ok := crm.cconns[thID]; !ok {
			crm.cconns[thID] = make(map[string]things.Profile)
		}
		crm.cconns[thID][prID] = pr
	}

	return nil
}

func (crm *profileRepositoryMock) Disconnect(_ context.Context, prID string, thIDs []string) error {
	for _, thID := range thIDs {
		if _, ok := crm.cconns[thID]; !ok {
			return errors.ErrNotFound
		}

		if _, ok := crm.cconns[thID][prID]; !ok {
			return errors.ErrNotFound
		}

		crm.tconns <- Connection{
			profileID: prID,
			thing:     things.Thing{ID: thID},
			connected: false,
		}
		delete(crm.cconns[thID], prID)
	}

	return nil
}

func (crm *profileRepositoryMock) RetrieveConnByThingKey(_ context.Context, token string) (things.Connection, error) {
	tid, err := crm.things.RetrieveByKey(context.Background(), token)
	if err != nil {
		return things.Connection{}, err
	}

	profiles, ok := crm.cconns[tid]
	if !ok {
		return things.Connection{}, errors.ErrAuthorization
	}

	if len(profiles) == 0 {
		return things.Connection{}, errors.ErrAuthorization
	}

	for _, v := range profiles {
		return things.Connection{ThingID: tid, ProfileID: v.ID}, nil
	}

	return things.Connection{}, errors.ErrNotFound
}

func (crm *profileRepositoryMock) HasThingByID(_ context.Context, profileID, thingID string) error {
	profiles, ok := crm.cconns[thingID]
	if !ok {
		return errors.ErrAuthorization
	}

	if _, ok := profiles[profileID]; !ok {
		return errors.ErrAuthorization
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

func (crm *profileRepositoryMock) RetrieveAllConnections(_ context.Context) ([]things.Connection, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()
	var conns []things.Connection

	for thingID, con := range crm.cconns {
		for _, v := range con {
			con := things.Connection{
				ProfileID: v.ID,
				ThingID:   thingID,
			}
			conns = append(conns, con)
		}
	}

	return conns, nil

}

type profileCacheMock struct {
	mu       sync.Mutex
	profiles map[string]string
}

// NewProfileCache returns mock cache instance.
func NewProfileCache() things.ProfileCache {
	return &profileCacheMock{
		profiles: make(map[string]string),
	}
}

func (ccm *profileCacheMock) Connect(_ context.Context, profileID, thingID string) error {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	ccm.profiles[profileID] = thingID
	return nil
}

func (ccm *profileCacheMock) HasThing(_ context.Context, profileID, thingID string) bool {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	return ccm.profiles[profileID] == thingID
}

func (ccm *profileCacheMock) Disconnect(_ context.Context, profileID, thingID string) error {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	delete(ccm.profiles, profileID)
	return nil
}

func (ccm *profileCacheMock) Remove(_ context.Context, profileID string) error {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	delete(ccm.profiles, profileID)
	return nil
}
