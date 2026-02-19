// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/users"
)

var _ users.IdentityRepository = (*identityRepositoryMock)(nil)

type identityRepositoryMock struct {
	mu       sync.Mutex
	identity map[string]users.Identity
}

func NewIdentityRepository() users.IdentityRepository {
	return &identityRepositoryMock{
		identity: make(map[string]users.Identity),
	}
}

func (irm *identityRepositoryMock) Save(ctx context.Context, identity users.Identity) error {
	irm.mu.Lock()
	defer irm.mu.Unlock()
	if irm.identity == nil {
		irm.identity = make(map[string]users.Identity)
	}
	key := identity.Provider + ":" + identity.ProviderUserID
	irm.identity[key] = identity
	return nil
}

func (irm *identityRepositoryMock) Retrieve(ctx context.Context, provider, providerUserID string) (users.Identity, error) {
	irm.mu.Lock()
	defer irm.mu.Unlock()
	key := provider + ":" + providerUserID
	identity, ok := irm.identity[key]
	if !ok {
		return users.Identity{}, nil
	}
	return identity, nil
}
