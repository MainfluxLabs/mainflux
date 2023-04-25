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
	orgName   = "test"
	orgDesc   = "test_description"
	invalidID = "invalid"
)

func TestRetrieveRole(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewOrgRepo(dbMiddleware)

	id, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ownerID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	memberID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	org := auth.Org{
		ID:          id,
		OwnerID:     ownerID,
		Name:        orgName,
		Description: orgDesc,
	}
	err = repo.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	member := auth.Member{
		ID:   memberID,
		Role: auth.AdminRole,
	}

	err = repo.AssignMembers(context.Background(), org.ID, member)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc     string
		orgID    string
		memberID string
		role     string
		err      error
	}{
		{
			desc:     "retrieve role",
			orgID:    org.ID,
			memberID: memberID,
			role:     auth.AdminRole,
			err:      nil,
		},
		{
			desc:     "retrieve role with non existing member",
			orgID:    org.ID,
			memberID: invalidID,
			role:     "",
			err:      nil,
		},
		{
			desc:     "retrieve role without member",
			orgID:    org.ID,
			memberID: "",
			role:     "",
			err:      nil,
		},
		{
			desc:     "retrieve role with non existing org",
			orgID:    invalidID,
			memberID: memberID,
			role:     "",
			err:      nil,
		},
		{
			desc:     "retrieve role without org",
			orgID:    "",
			memberID: memberID,
			role:     "",
			err:      nil,
		},
	}

	for _, tc := range cases {
		role, _ := repo.RetrieveRole(context.Background(), tc.memberID, tc.orgID)
		require.Equal(t, tc.role, role, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.role, role))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateRole(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewOrgRepo(dbMiddleware)

	id, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ownerID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	memberID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	unknownID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	org := auth.Org{
		ID:          id,
		OwnerID:     ownerID,
		Name:        orgName,
		Description: orgDesc,
	}
	err = repo.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	member := auth.Member{
		ID:   memberID,
		Role: auth.AdminRole,
	}

	err = repo.AssignMembers(context.Background(), org.ID, member)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc   string
		orgID  string
		member auth.Member
		err    error
	}{
		{
			desc:   "update member role",
			orgID:  org.ID,
			member: member,
			err:    nil,
		}, {
			desc:   "update role with invalid org",
			orgID:  invalidID,
			member: member,
			err:    errors.ErrMalformedEntity,
		}, {
			desc:   "update role with non-existing org",
			orgID:  unknownID,
			member: member,
			err:    errors.ErrNotFound,
		}, {
			desc:   "update role with empty org",
			orgID:  "",
			member: member,
			err:    errors.ErrMalformedEntity,
		}, {
			desc:   "update role with invalid member",
			orgID:  org.ID,
			member: auth.Member{ID: invalidID},
			err:    errors.ErrMalformedEntity,
		}, {
			desc:   "update role with non-existing member",
			orgID:  org.ID,
			member: auth.Member{ID: unknownID},
			err:    errors.ErrNotFound,
		}, {
			desc:   "update role with empty member",
			orgID:  org.ID,
			member: auth.Member{},
			err:    errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		err := repo.UpdateMembers(context.Background(), tc.orgID, tc.member)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
