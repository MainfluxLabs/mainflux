// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/modbus"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
)

var _ modbus.ClientRepository = (*clientRepositoryMock)(nil)

type clientRepositoryMock struct {
	mu      sync.Mutex
	clients map[string]modbus.Client
}

// NewClientRepository creates an in-memory Modbus client repository.
func NewClientRepository() modbus.ClientRepository {
	return &clientRepositoryMock{
		clients: make(map[string]modbus.Client),
	}
}

func (crm *clientRepositoryMock) Save(_ context.Context, cls ...modbus.Client) ([]modbus.Client, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	for _, c := range cls {
		crm.clients[c.ID] = c
	}

	return cls, nil
}

func (crm *clientRepositoryMock) RetrieveByThing(_ context.Context, thingID string, pm apiutil.PageMetadata) (modbus.ClientsPage, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	var items []modbus.Client

	first := uint64(pm.Offset) + 1
	last := first + pm.Limit

	for _, c := range crm.clients {
		if c.ThingID == thingID {
			id := uuid.ParseID(c.ID)
			if id >= first && id < last || pm.Limit == 0 {
				items = append(items, c)
			}
		}
	}

	return modbus.ClientsPage{
		Clients: items,
		PageMetadata: apiutil.PageMetadata{
			Total: uint64(len(items)),
		},
	}, nil
}

func (crm *clientRepositoryMock) RetrieveByGroup(_ context.Context, groupID string, pm apiutil.PageMetadata) (modbus.ClientsPage, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	var items []modbus.Client

	first := uint64(pm.Offset) + 1
	last := first + pm.Limit

	for _, c := range crm.clients {
		if c.GroupID == groupID {
			id := uuid.ParseID(c.ID)
			if id >= first && id < last || pm.Limit == 0 {
				items = append(items, c)
			}
		}
	}

	return modbus.ClientsPage{
		Clients: items,
		PageMetadata: apiutil.PageMetadata{
			Total: uint64(len(items)),
		},
	}, nil
}

func (crm *clientRepositoryMock) RetrieveByID(_ context.Context, id string) (modbus.Client, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	c, ok := crm.clients[id]
	if !ok {
		return modbus.Client{}, dbutil.ErrNotFound
	}

	return c, nil
}

func (crm *clientRepositoryMock) RetrieveAll(_ context.Context) ([]modbus.Client, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	var items []modbus.Client
	for _, c := range crm.clients {
		items = append(items, c)
	}

	return items, nil
}

func (crm *clientRepositoryMock) Update(_ context.Context, c modbus.Client) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	if _, ok := crm.clients[c.ID]; !ok {
		return dbutil.ErrNotFound
	}

	crm.clients[c.ID] = c
	return nil
}

func (crm *clientRepositoryMock) Remove(_ context.Context, ids ...string) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	for _, id := range ids {
		if _, ok := crm.clients[id]; !ok {
			return dbutil.ErrNotFound
		}
		delete(crm.clients, id)
	}

	return nil
}

func (crm *clientRepositoryMock) RemoveByThing(_ context.Context, thingID string) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	for id, c := range crm.clients {
		if c.ThingID == thingID {
			delete(crm.clients, id)
		}
	}

	return nil
}

func (crm *clientRepositoryMock) RemoveByGroup(_ context.Context, groupID string) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	for id, c := range crm.clients {
		if c.GroupID == groupID {
			delete(crm.clients, id)
		}
	}

	return nil
}
