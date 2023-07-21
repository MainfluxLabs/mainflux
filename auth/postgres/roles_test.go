package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/auth/postgres"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	password = "password"
	invalid  = "invalid"
)

func TestSaveRole(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
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
			err:  errors.ErrCreateEntity,
		},
		{
			desc: "save invalid role",
			id:   userID,
			role: invalid,
			err:  errors.ErrCreateEntity,
		},
		{
			desc: "save with invalid user id",
			id:   invalid,
			role: auth.RoleRootAdmin,
			err:  errors.ErrCreateEntity,
		},
		{
			desc: "save without user id",
			id:   "",
			role: auth.RoleRootAdmin,
			err:  errors.ErrCreateEntity,
		},
		{
			desc: "save without user role",
			id:   userID,
			role: "",
			err:  errors.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		err := repo.SaveRole(context.Background(), tc.id, tc.role)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieve(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
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
			err:  errors.ErrUpdateEntity,
		},
		{
			desc: "update with empty role",
			id:   userID,
			role: "",
			err:  errors.ErrUpdateEntity,
		},
		{
			desc: "update with empty user id",
			id:   "",
			role: auth.RoleRootAdmin,
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
