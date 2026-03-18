// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
)

var _ auth.KeyRepository = (*keyRepositoryMock)(nil)

type keyRepositoryMock struct {
	mu   sync.Mutex
	keys map[string]auth.Key
}

// NewKeyRepository creates in-memory user repository
func NewKeyRepository() auth.KeyRepository {
	return &keyRepositoryMock{
		keys: make(map[string]auth.Key),
	}
}

func (krm *keyRepositoryMock) Save(ctx context.Context, key auth.Key) (string, error) {
	krm.mu.Lock()
	defer krm.mu.Unlock()

	if _, ok := krm.keys[key.ID]; ok {
		return "", dbutil.ErrConflict
	}

	krm.keys[key.ID] = key
	return key.ID, nil
}

func (krm *keyRepositoryMock) Retrieve(ctx context.Context, issuerID, id string) (auth.Key, error) {
	krm.mu.Lock()
	defer krm.mu.Unlock()

	if key, ok := krm.keys[id]; ok && key.IssuerID == issuerID {
		return key, nil
	}

	return auth.Key{}, dbutil.ErrNotFound
}

func (krm *keyRepositoryMock) Remove(ctx context.Context, issuerID, id string) error {
	krm.mu.Lock()
	defer krm.mu.Unlock()
	if key, ok := krm.keys[id]; ok && key.IssuerID == issuerID {
		delete(krm.keys, id)
	}
	return nil
}

func (krm *keyRepositoryMock) RetrieveAPIKeys(ctx context.Context, issuerID string, pm apiutil.PageMetadata) (auth.KeysPage, error) {
	krm.mu.Lock()
	defer krm.mu.Unlock()

	var all []auth.Key
	for _, key := range krm.keys {
		if key.IssuerID == issuerID && key.Type == auth.APIKey {
			all = append(all, key)
		}
	}

	total := uint64(len(all))
	if pm.Limit == 0 || pm.Offset >= total {
		return auth.KeysPage{
			Total: total,
			Keys:  []auth.Key{},
		}, nil
	}

	start := pm.Offset
	end := pm.Offset + pm.Limit
	if end > total {
		end = total
	}

	page := make([]auth.Key, 0, end-start)
	for i := start; i < end; i++ {
		page = append(page, all[i])
	}

	return auth.KeysPage{
		Total: total,
		Keys:  page,
	}, nil
}
