package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/auth/postgres"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	password   = "password"
	invalid    = "invalid"
	rolesTable = "users_roles"
)

func TestSaveRole(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repo := postgres.NewRolesRepo(dbMiddleware)

	userID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		role string
		err  error
	}{
		{
			desc: "save role",
			id:   userID,
			role: auth.RoleRootAdmin,
			err:  nil,
		},
		{
			desc: "save already existing role",
			id:   userID,
			role: auth.RoleRootAdmin,
			err:  dbutil.ErrConflict,
		},
		{
			desc: "save invalid role",
			id:   userID,
			role: invalid,
			err:  dbutil.ErrCreateEntity,
		},
		{
			desc: "save with invalid user id",
			id:   invalid,
			role: auth.RoleRootAdmin,
			err:  dbutil.ErrCreateEntity,
		},
		{
			desc: "save without user id",
			id:   "",
			role: auth.RoleRootAdmin,
			err:  dbutil.ErrCreateEntity,
		},
		{
			desc: "save without user role",
			id:   userID,
			role: "",
			err:  dbutil.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		err := repo.SaveRole(context.Background(), tc.id, tc.role)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieve(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repo := postgres.NewRolesRepo(dbMiddleware)

	userID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	unknownUid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	err = repo.SaveRole(context.Background(), userID, auth.RoleAdmin)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		role string
		err  error
	}{
		{
			desc: "retrieve role",
			id:   userID,
			role: auth.RoleAdmin,
			err:  nil,
		},
		{
			desc: "retrieve role for non existing user",
			id:   unknownUid,
			role: "",
			err:  nil,
		},
		{
			desc: "retrieve role for invalid user id",
			id:   invalid,
			role: "",
			err:  dbutil.ErrRetrieveEntity,
		},
	}

	for _, tc := range cases {
		role, err := repo.RetrieveRole(context.Background(), tc.id)
		assert.Equal(t, tc.role, role, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.role, role))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateRole(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repo := postgres.NewRolesRepo(dbMiddleware)

	userID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	err = repo.SaveRole(context.Background(), userID, auth.RoleAdmin)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		role string
		err  error
	}{
		{
			desc: "update role",
			id:   userID,
			role: auth.RoleRootAdmin,
			err:  nil,
		},
		{
			desc: "update role for invalid user id",
			id:   invalid,
			role: auth.RoleRootAdmin,
			err:  dbutil.ErrUpdateEntity,
		},
		{
			desc: "update with empty role",
			id:   userID,
			role: "",
			err:  dbutil.ErrUpdateEntity,
		},
		{
			desc: "update with empty user id",
			id:   "",
			role: auth.RoleRootAdmin,
			err:  dbutil.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		err := repo.UpdateRole(context.Background(), tc.id, tc.role)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestDeleteRole(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repo := postgres.NewRolesRepo(dbMiddleware)

	userID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	err = repo.SaveRole(context.Background(), userID, auth.RoleAdmin)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "delete role",
			id:   userID,
			err:  nil,
		},
		{
			desc: "delete non-existing role",
			id:   userID,
			err:  dbutil.ErrNotFound,
		},
		{
			desc: "delete role for invalid user id",
			id:   invalid,
			err:  dbutil.ErrRemoveEntity,
		},
		{
			desc: "delete role without user id",
			id:   "",
			err:  dbutil.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		err := repo.RemoveRole(context.Background(), tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveRoles(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repo := postgres.NewRolesRepo(dbMiddleware)

	rootID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	adminID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	userID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	err = repo.SaveRole(context.Background(), rootID, auth.RoleRootAdmin)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	err = repo.SaveRole(context.Background(), adminID, auth.RoleAdmin)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc  string
		ids   []string
		roles map[string]string
		err   error
	}{
		{
			desc: "retrieve roles",
			ids:  []string{rootID, adminID, userID},
			roles: map[string]string{
				rootID:  auth.RoleRootAdmin,
				adminID: auth.RoleAdmin,
			},
			err: nil,
		},
		{
			desc:  "retrieve roles for users without roles",
			ids:   []string{userID},
			roles: map[string]string{},
			err:   nil,
		},
		{
			desc:  "retrieve roles for invalid user id",
			ids:   []string{invalid},
			roles: nil,
			err:   dbutil.ErrRetrieveEntity,
		},
	}

	for _, tc := range cases {
		roles, err := repo.RetrieveRoles(context.Background(), tc.ids)
		msg := fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.roles, roles)
		assert.Equal(t, tc.roles, roles, msg)
		msg = fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)
	}
}

func TestBackupAllRoles(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repo := postgres.NewRolesRepo(dbMiddleware)

	_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", rolesTable))
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	for i := uint64(0); i < n; i++ {
		userID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		err = repo.SaveRole(context.Background(), userID, auth.RoleAdmin)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	urs, err := repo.BackupAll(context.Background())
	assert.Nil(t, err, fmt.Sprintf("backup all roles: unexpected error: %s", err))
	size := uint64(len(urs))
	assert.Equal(t, n, size, fmt.Sprintf("backup all roles: expected size %d got %d", n, size))
}

func TestRestoreAllRoles(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repo := postgres.NewRolesRepo(dbMiddleware)

	rootID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	adminID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	err = repo.SaveRole(context.Background(), rootID, auth.RoleRootAdmin)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		urs  []auth.UserRole
		err  error
	}{
		{
			desc: "restore roles",
			urs: []auth.UserRole{
				{
					UserID: rootID,
					Role:   auth.RoleAdmin,
				},
				{
					UserID: adminID,
					Role:   auth.RoleAdmin,
				},
			},
			err: nil,
		},
		{
			desc: "restore role with invalid user id",
			urs: []auth.UserRole{
				{
					UserID: invalid,
					Role:   auth.RoleAdmin,
				},
			},
			err: dbutil.ErrMalformedEntity,
		},
		{
			desc: "restore role with invalid role",
			urs: []auth.UserRole{
				{
					UserID: adminID,
					Role:   invalid,
				},
			},
			err: dbutil.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		err := repo.RestoreAll(context.Background(), tc.urs...)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)
	}

	roles := map[string]string{
		rootID:  auth.RoleRootAdmin,
		adminID: auth.RoleAdmin,
	}

	for userID, expected := range roles {
		role, err := repo.RetrieveRole(context.Background(), userID)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		msg := fmt.Sprintf("%s: expected role %s got %s", userID, expected, role)
		assert.Equal(t, expected, role, msg)
	}
}
