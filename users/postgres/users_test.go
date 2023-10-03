// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/MainfluxLabs/mainflux/users/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	usersNum   = 101
	usersTable = "users"
	email      = "user@test.com"
	password   = "password"
)

var idProvider = uuid.New()

func TestUserSave(t *testing.T) {
	email := "user-save@example.com"

	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		user users.User
		err  error
	}{
		{
			desc: "new user",
			user: users.User{
				ID:       uid,
				Email:    email,
				Password: "password",
				Status:   users.EnabledStatusKey,
			},
			err: nil,
		},
		{
			desc: "duplicate user",
			user: users.User{
				ID:       uid,
				Email:    email,
				Password: "password",
				Status:   users.EnabledStatusKey,
			},
			err: errors.ErrConflict,
		},
		{
			desc: "invalid user status",
			user: users.User{
				ID:       uid,
				Email:    email,
				Password: "password",
				Status:   "invalid",
			},
			err: errors.ErrMalformedEntity,
		},
	}

	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewUserRepo(dbMiddleware)

	for _, tc := range cases {
		_, err := repo.Save(context.Background(), tc.user)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSingleUserRetrieval(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewUserRepo(dbMiddleware)

	email := "user-retrieval@example.com"

	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	user := users.User{
		ID:       uid,
		Email:    email,
		Password: "password",
		Status:   users.EnabledStatusKey,
	}

	_, err = repo.Save(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		email string
		err   error
	}{
		"existing user":     {email, nil},
		"non-existing user": {"unknown@example.com", errors.ErrNotFound},
	}

	for desc, tc := range cases {
		_, err := repo.RetrieveByEmail(context.Background(), tc.email)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRetrieveByIDs(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	userRepo := postgres.NewUserRepo(dbMiddleware)
	metaNum := uint64(2)
	var nUsers = uint64(usersNum)

	meta := users.Metadata{
		"admin": "true",
	}

	wrongMeta := users.Metadata{
		"wrong": "true",
	}

	var ids []string
	for i := uint64(0); i < nUsers; i++ {
		uid, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		email := fmt.Sprintf("TestRetrieveAll%d@example.com", i)
		user := users.User{
			ID:       uid,
			Email:    email,
			Password: "password",
			Status:   users.EnabledStatusKey,
		}
		if i < metaNum {
			user.Metadata = meta
		}
		ids = append(ids, uid)
		_, err = userRepo.Save(context.Background(), user)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	cases := map[string]struct {
		pm   users.PageMetadata
		size uint64
		ids  []string
	}{
		"retrieve all users filtered by email": {
			pm: users.PageMetadata{
				Email:  "All",
				Status: users.EnabledStatusKey,
				Offset: 0,
				Limit:  nUsers,
				Total:  nUsers,
			},
			size: nUsers,
			ids:  ids,
		},
		"retrieve all users by email with limit and offset": {
			pm: users.PageMetadata{
				Email:  "All",
				Status: users.EnabledStatusKey,
				Offset: 2,
				Limit:  5,
				Total:  nUsers,
			},
			size: 5,
			ids:  ids,
		},
		"retrieve all users by email without limit": {
			pm: users.PageMetadata{
				Email:  "All",
				Status: users.EnabledStatusKey,
				Limit:  0,
				Total:  nUsers,
			},
			size: nUsers,
			ids:  ids,
		},
		"retrieve all users by metadata": {
			pm: users.PageMetadata{
				Email:    "All",
				Status:   users.EnabledStatusKey,
				Offset:   0,
				Limit:    nUsers,
				Total:    nUsers,
				Metadata: meta,
			},
			size: metaNum,
			ids:  ids,
		},
		"retrieve users by metadata and ids": {
			pm: users.PageMetadata{
				Email:    "All",
				Status:   users.EnabledStatusKey,
				Offset:   0,
				Limit:    nUsers,
				Total:    nUsers,
				Metadata: meta,
			},
			size: 1,
			ids:  []string{ids[0]},
		},
		"retrieve users by wrong metadata": {
			pm: users.PageMetadata{
				Email:    "All",
				Status:   users.EnabledStatusKey,
				Offset:   0,
				Limit:    nUsers,
				Total:    nUsers,
				Metadata: wrongMeta,
			},
			size: 0,
			ids:  ids,
		},
		"retrieve users by wrong metadata and ids": {
			pm: users.PageMetadata{
				Email:    "All",
				Status:   users.EnabledStatusKey,
				Offset:   0,
				Limit:    nUsers,
				Total:    nUsers,
				Metadata: wrongMeta,
			},
			size: 0,
			ids:  []string{ids[0]},
		},
		"retrieve all users by list of ids with limit and offset": {
			pm: users.PageMetadata{
				Email:  "All",
				Status: users.EnabledStatusKey,
				Offset: 2,
				Limit:  5,
				Total:  nUsers,
			},
			size: 5,
			ids:  ids,
		},
		"retrieve all users by list of ids with limit and offset and metadata": {
			pm: users.PageMetadata{
				Email:    "All",
				Status:   users.EnabledStatusKey,
				Offset:   1,
				Limit:    5,
				Total:    nUsers,
				Metadata: meta,
			},
			size: 1,
			ids:  ids[0:5],
		},
		"retrieve all users from empty ids": {
			pm: users.PageMetadata{
				Email:  "All",
				Status: users.EnabledStatusKey,
				Offset: 0,
				Limit:  nUsers,
				Total:  nUsers,
			},
			size: nUsers,
			ids:  []string{},
		},
		"retrieve all users from empty ids with offset": {
			pm: users.PageMetadata{
				Email:  "All",
				Status: users.EnabledStatusKey,
				Offset: 1,
				Limit:  5,
				Total:  nUsers,
			},
			size: 5,
			ids:  []string{},
		},
	}
	for desc, tc := range cases {
		page, err := userRepo.RetrieveByIDs(context.Background(), tc.ids, tc.pm)
		size := uint64(len(page.Users))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestRetrieveAll(t *testing.T) {
	_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", usersTable))
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	dbMiddleware := postgres.NewDatabase(db)
	userRepo := postgres.NewUserRepo(dbMiddleware)
	metaNum := uint64(2)
	var nUsers = uint64(usersNum)

	meta := users.Metadata{
		"field": "value",
	}

	var ids []string
	for i := uint64(0); i < nUsers; i++ {
		uid, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		email := fmt.Sprintf("TestRetrieveAll%d@example.com", i)
		user := users.User{
			ID:       uid,
			Email:    email,
			Password: "password",
			Status:   users.EnabledStatusKey,
		}
		if i < metaNum {
			user.Metadata = meta
		}
		ids = append(ids, uid)
		_, err = userRepo.Save(context.Background(), user)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	cases := map[string]struct {
		size uint64
	}{
		"retrieve all users": {
			size: nUsers,
		},
	}
	for desc, tc := range cases {
		users, err := userRepo.RetrieveAll(context.Background())
		size := uint64(len(users))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestUpdateUser(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	userRepo := postgres.NewUserRepo(dbMiddleware)

	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	updateID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	user := users.User{
		ID:       uid,
		Email:    email,
		Password: password,
		Metadata: map[string]interface{}{"metadata": "test"},
		Status:   users.EnabledStatusKey,
	}

	_, err = userRepo.Save(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	updtUser := user
	updtUser.ID = updateID
	updtUser.Metadata = map[string]interface{}{"updated": "metadata"}

	wrongEmailUser := user
	wrongEmailUser.Email = "wrong@email.com"

	disabledUser := user
	disabledUser.Status = users.DisabledStatusKey

	cases := map[string]struct {
		user  users.User
		email string
		err   error
	}{
		"update user with invalid email": {
			user:  wrongEmailUser,
			email: wrongEmailUser.Email,
			err:   errors.ErrNotFound,
		},
		"update disabled user": {
			user:  disabledUser,
			email: disabledUser.Email,
			err:   errors.ErrNotFound,
		},
		"update existing user": {
			user:  updtUser,
			email: updtUser.Email,
			err:   nil,
		},
	}
	for desc, tc := range cases {
		err := userRepo.UpdateUser(context.Background(), tc.user)
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}
