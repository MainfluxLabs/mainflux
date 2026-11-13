// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/auth/postgres"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const email = "user-save@example.com"

var (
	expTime    = time.Now().Add(5 * time.Minute)
	idProvider = uuid.New()
)

func TestKeySave(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repo := postgres.New(dbMiddleware)

	id, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc string
		key  auth.Key
		err  error
	}{
		{
			desc: "save a new key",
			key: auth.Key{
				Subject:   email,
				IssuedAt:  time.Now(),
				ExpiresAt: expTime,
				ID:        id,
				IssuerID:  id,
			},
			err: nil,
		},
		{
			desc: "save with duplicate id",
			key: auth.Key{
				Subject:   email,
				IssuedAt:  time.Now(),
				ExpiresAt: expTime,
				ID:        id,
				IssuerID:  id,
			},
			err: dbutil.ErrConflict,
		},
	}

	for _, tc := range cases {
		_, err := repo.Save(context.Background(), tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestKeyRetrieve(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repo := postgres.New(dbMiddleware)

	id, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	key := auth.Key{
		Subject:   email,
		IssuedAt:  time.Now(),
		ExpiresAt: expTime,
		ID:        id,
		IssuerID:  id,
	}
	_, err = repo.Save(context.Background(), key)
	assert.Nil(t, err, fmt.Sprintf("Storing Key expected to succeed: %s", err))
	cases := []struct {
		desc  string
		id    string
		owner string
		err   error
	}{
		{
			desc:  "retrieve an existing key",
			id:    key.ID,
			owner: key.IssuerID,
			err:   nil,
		},
		{
			desc:  "retrieve key with empty issuer id",
			id:    key.ID,
			owner: "",
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "retrieve non-existent key",
			id:    "",
			owner: key.IssuerID,
			err:   dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := repo.Retrieve(context.Background(), tc.owner, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestKeyRemoveAPIKeys(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repo := postgres.New(dbMiddleware)

	issuerID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	var keys []auth.Key
	for _, keyType := range []uint32{auth.APIKey, auth.APIKey, auth.RecoveryKey} {
		keyID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		key := auth.Key{
			ID:        keyID,
			Type:      keyType,
			IssuerID:  issuerID,
			Subject:   email,
			IssuedAt:  time.Now(),
			ExpiresAt: expTime,
		}
		_, err = repo.Save(context.Background(), key)
		require.Nil(t, err, fmt.Sprintf("Storing Key expected to succeed: %s", err))
		keys = append(keys, key)
	}

	cases := []struct {
		desc     string
		issuerID string
		err      error
	}{
		{
			desc:     "remove API keys",
			issuerID: issuerID,
			err:      nil,
		},
		{
			desc:     "remove API keys of issuer without API keys",
			issuerID: issuerID,
			err:      nil,
		},
		{
			desc:     "remove API keys with invalid issuer id",
			issuerID: "invalid",
			err:      dbutil.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		err := repo.RemoveAPIKeys(context.Background(), tc.issuerID)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)
	}

	for _, key := range keys {
		_, err := repo.Retrieve(context.Background(), issuerID, key.ID)
		if key.Type == auth.APIKey {
			msg := fmt.Sprintf("removed API key: expected %s got %s\n", dbutil.ErrNotFound, err)
			assert.True(t, errors.Contains(err, dbutil.ErrNotFound), msg)
			continue
		}
		assert.Nil(t, err, fmt.Sprintf("key of another type expected to survive: %s", err))
	}
}

func TestKeyRemove(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repo := postgres.New(dbMiddleware)

	id, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	key := auth.Key{
		Subject:   email,
		IssuedAt:  time.Now(),
		ExpiresAt: expTime,
		ID:        id,
		IssuerID:  id,
	}
	_, err = repo.Save(opentracing.ContextWithSpan(context.Background(), opentracing.StartSpan("")), key)
	assert.Nil(t, err, fmt.Sprintf("Storing Key expected to succeed: %s", err))
	cases := []struct {
		desc  string
		id    string
		owner string
		err   error
	}{
		{
			desc:  "remove an existing key",
			id:    key.ID,
			owner: key.IssuerID,
			err:   nil,
		},
		{
			desc:  "remove key that does not exist",
			id:    key.ID,
			owner: key.IssuerID,
			err:   nil,
		},
	}

	for _, tc := range cases {
		err := repo.Remove(context.Background(), tc.owner, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
