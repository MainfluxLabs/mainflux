package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/auth"
)

type rolesRepositoryMock struct {
	mu    sync.Mutex
	roles map[string]string
}

func NewRolesRepository() auth.RolesRepository {
	return &rolesRepositoryMock{
		roles: make(map[string]string),
	}
}

func (rrm *rolesRepositoryMock) SaveRole(ctx context.Context, id, role string) error {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	rrm.roles[id] = role

	return nil
}

func (rrm *rolesRepositoryMock) RetrieveRole(ctx context.Context, id string) (string, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	if role, ok := rrm.roles[id]; ok {
		return role, nil
	}

	return "", nil
}

func (rrm *rolesRepositoryMock) UpdateRole(ctx context.Context, id, role string) error {
	panic("not implemented")
}

func (rrm *rolesRepositoryMock) RemoveRole(ctx context.Context, id string) error {
	panic("not implemented")
}
