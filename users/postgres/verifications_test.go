// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/MainfluxLabs/mainflux/users/argon2"
	"github.com/MainfluxLabs/mainflux/users/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	nonExistentToken = "12b4ad3d-9563-408a-b9c2-425740823738"
)

func TestVerificationSave(t *testing.T) {
	id, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc         string
		verification users.EmailVerification
		err          error
	}{
		{
			desc: "save new verification",
			verification: users.EmailVerification{
				Token: id,
				User: users.User{
					Email:    email,
					Password: password,
				},
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			},
			err: nil,
		},
		{
			desc: "save verification with duplicate token",
			verification: users.EmailVerification{
				Token: id,
				User: users.User{
					Email:    email,
					Password: password,
				},
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			},
			err: dbutil.ErrConflict,
		},
	}

	dbMiddleware := dbutil.NewDatabase(db)
	repo := postgres.NewEmailVerificationRepo(dbMiddleware)

	for _, tc := range cases {
		_, err := repo.Save(context.Background(), tc.verification)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestVerificationRemove(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repo := postgres.NewEmailVerificationRepo(dbMiddleware)

	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	verification := users.EmailVerification{
		User:      users.User{Email: email, Password: password},
		Token:     uid,
		CreatedAt: time.Now().Add(-7 * 24 * time.Hour),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	_, err = repo.Save(context.Background(), verification)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc         string
		verification users.EmailVerification
		err          error
	}{
		{
			desc: "remove existing verification",
			verification: users.EmailVerification{
				Token: uid,
			},
			err: nil,
		},
		{
			desc: "remove verification with invalid token",
			verification: users.EmailVerification{
				Token: nonExistentToken,
			},
			err: dbutil.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		err := repo.Remove(context.Background(), tc.verification.Token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestVerificationRetrieve(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repo := postgres.NewEmailVerificationRepo(dbMiddleware)

	id, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	verification := users.EmailVerification{
		User:      users.User{Email: email, Password: password},
		Token:     id,
		CreatedAt: time.Now().Add(-7 * 24 * time.Hour),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	_, err = repo.Save(context.Background(), verification)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc         string
		verification users.EmailVerification
		err          error
	}{
		{
			desc: "retrieve existing verification",
			verification: users.EmailVerification{
				Token: id,
			},
			err: nil,
		},
		{
			desc: "retrieve verification with non-existent token",
			verification: users.EmailVerification{
				Token: nonExistentToken,
			},
			err: dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := repo.RetrieveByToken(context.Background(), tc.verification.Token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestVerificationArgon2idHashStorage(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repo := postgres.NewEmailVerificationRepo(dbMiddleware)

	hash, err := argon2.New().Hash("correct horse battery staple")
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	token, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	verification := users.EmailVerification{
		Token:     token,
		User:      users.User{Email: "argon2-verification@example.com", Password: hash},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	_, err = repo.Save(context.Background(), verification)
	require.Nil(t, err, fmt.Sprintf("save verification with Argon2id hash: unexpected error: %s", err))

	saved, err := repo.RetrieveByToken(context.Background(), token)
	require.Nil(t, err, fmt.Sprintf("retrieve verification: unexpected error: %s", err))
	assert.Equal(t, hash, saved.User.Password, "retrieved hash expected to equal the saved hash")
}
