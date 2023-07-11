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
	email      = "user-save@example.com"
	password   = "password"
	invalid    = "invalid"
)

var (
	idProvider = uuid.New()
	user       = users.User{
		Email:    email,
		Password: password,
		Status:   users.EnabledStatusKey,
	}
)

func TestUserSave(t *testing.T) {
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

	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	user.ID = uid
	user.Email = "0" + email

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
		u := user
		u.ID = uid
		u.Email = email
		if i < metaNum {
			u.Metadata = meta
		}
		ids = append(ids, uid)
		_, err = userRepo.Save(context.Background(), u)
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
		u := user
		u.ID = uid
		u.Email = email

		if i < metaNum {
			u.Metadata = meta
		}

		ids = append(ids, uid)
		_, err = userRepo.Save(context.Background(), u)
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

func TestSaveRole(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewUserRepo(dbMiddleware)

	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	unknownUid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	user.ID = uid
	user.Email = "1" + email

	id, err := repo.Save(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		role string
		err  error
	}{
		{
			desc: "save role",
			id:   id,
			role: users.MakerRole,
			err:  nil,
		},
		{
			desc: "save already existing role",
			id:   id,
			role: users.MakerRole,
			err:  errors.ErrCreateEntity,
		},
		{
			desc: "save invalid role",
			id:   id,
			role: invalid,
			err:  errors.ErrCreateEntity,
		},
		{
			desc: "save with invalid user id",
			id:   invalid,
			role: users.MakerRole,
			err:  errors.ErrCreateEntity,
		},
		{
			desc: "save role for non existing user",
			id:   unknownUid,
			role: users.MakerRole,
			err:  errors.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		err := repo.SaveRole(context.Background(), tc.id, tc.role)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveRole(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewUserRepo(dbMiddleware)

	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	unknownUid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	user.ID = uid
	user.Email = "2" + email

	_, err = repo.Save(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	err = repo.SaveRole(context.Background(), user.ID, users.AdminRole)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		role string
		err  error
	}{
		{
			desc: "retrieve role",
			id:   user.ID,
			role: users.AdminRole,
			err:  nil,
		},
		{
			desc: "retrieve role for non existing user",
			id:   unknownUid,
			role: "",
			err:  errors.ErrNotFound,
		},
		{
			desc: "retrieve role for invalid user id",
			id:   invalid,
			role: "",
			err:  errors.ErrRetrieveEntity,
		},
	}

	for _, tc := range cases {
		role, err := repo.RetrieveRole(context.Background(), tc.id)
		assert.Equal(t, tc.role, role, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.role, role))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateRole(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewUserRepo(dbMiddleware)

	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	user.ID = uid
	user.Email = "3" + email

	_, err = repo.Save(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	err = repo.SaveRole(context.Background(), user.ID, users.AdminRole)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		role string
		err  error
	}{
		{
			desc: "update role",
			id:   user.ID,
			role: users.MakerRole,
			err:  nil,
		},
		{
			desc: "update role for invalid user id",
			id:   invalid,
			role: users.MakerRole,
			err:  errors.ErrUpdateEntity,
		},
		{
			desc: "update with empty role",
			id:   user.ID,
			role: "",
			err:  errors.ErrUpdateEntity,
		},
		{
			desc: "update with empty user id",
			id:   "",
			role: users.MakerRole,
			err:  errors.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		err := repo.UpdateRole(context.Background(), tc.id, tc.role)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestDeleteRole(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewUserRepo(dbMiddleware)

	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	user.ID = uid
	user.Email = "4" + email

	_, err = repo.Save(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	err = repo.SaveRole(context.Background(), user.ID, users.AdminRole)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "delete role",
			id:   user.ID,
			err:  nil,
		},
		{
			desc: "delete role for invalid user id",
			id:   invalid,
			err:  errors.ErrRemoveEntity,
		},
		{
			desc: "delete role without user id",
			id:   "",
			err:  errors.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		err := repo.RemoveRole(context.Background(), tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
