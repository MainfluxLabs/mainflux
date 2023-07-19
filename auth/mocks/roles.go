package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/auth"
)

type roleRepositoryMock struct {
	mu    sync.Mutex
	roles map[string]string
}

func NewRoleRepository() auth.RoleRepository {
	return &roleRepositoryMock{
		roles: make(map[string]string),
	}
}

func (rrm *roleRepositoryMock) SaveRole(ctx context.Context, id, role string) error {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	rrm.roles[id] = role

	return nil
}

func (rrm *roleRepositoryMock) RetrieveRole(ctx context.Context, id string) (string, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	if role, ok := rrm.roles[id]; ok {
		return role, nil
	}

	return "", nil
}

func (rrm *roleRepositoryMock) UpdateRole(ctx context.Context, id, role string) error {
	panic("not implemented")
}

func (rrm *roleRepositoryMock) RemoveRole(ctx context.Context, id string) error {
	panic("not implemented")
}
