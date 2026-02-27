// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/users"
)

var _ users.IdentityRepository = (*identityRepositoryMock)(nil)

type identityRepositoryMock struct {
	mu         sync.Mutex
	identities map[string]users.Identity
}

func NewIdentityRepository() users.IdentityRepository {
	return &identityRepositoryMock{
		identities: make(map[string]users.Identity),
	}
}

func (irm *identityRepositoryMock) Save(ctx context.Context, identity users.Identity) error {
	irm.mu.Lock()
	defer irm.mu.Unlock()
	key := identity.Provider + ":" + identity.ProviderUserID
	irm.identities[key] = identity
	return nil
}

func (irm *identityRepositoryMock) Retrieve(ctx context.Context, provider, providerUserID string) (users.Identity, error) {
	irm.mu.Lock()
	defer irm.mu.Unlock()
	key := provider + ":" + providerUserID
	identity, ok := irm.identities[key]
	if !ok {
		return users.Identity{}, dbutil.ErrNotFound
	}
	return identity, nil
}

func (irm *identityRepositoryMock) BackupAll(ctx context.Context) ([]users.Identity, error) {
	irm.mu.Lock()
	defer irm.mu.Unlock()
	var identities []users.Identity
	for _, identity := range irm.identities {
		identities = append(identities, identity)
	}
	return identities, nil
}
