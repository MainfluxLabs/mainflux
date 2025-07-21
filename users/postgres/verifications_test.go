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
	"github.com/MainfluxLabs/mainflux/users/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerificationSave(t *testing.T) {
	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc         string
		verification users.EmailVerification
		err          error
	}{
		{
			desc: "save new verification",
			verification: users.EmailVerification{
				Token: uid,
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
				Token: uid,
				User: users.User{
					Email:    email,
					Password: password,
				},
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			},
			err: errors.ErrConflict,
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
				Token: "12b4ad3d-9563-408a-b9c2-425740823738",
			},
			err: errors.ErrRemoveEntity,
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
			desc: "retrieve existing verification",
			verification: users.EmailVerification{
				Token: uid,
			},
			err: nil,
		},
		{
			desc: "retrieve verification with non-existent token",
			verification: users.EmailVerification{
				Token: "12b4ad3d-9563-408a-b9c2-425740823738",
			},
			err: errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := repo.RetrieveByToken(context.Background(), tc.verification.Token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
