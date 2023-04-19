package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/auth/postgres"
	"github.com/stretchr/testify/require"
)

const (
	orgName   = "test"
	orgDesc   = "test_description"
	adminRole = "admin"
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

	member1 := auth.MembersByID{
		ID:   memberID,
		Role: adminRole,
	}

	err = repo.AssignMembers(context.Background(), org.ID, []auth.MembersByID{member1})
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
			role:     "admin",
			err:      nil,
		},
	}

	for _, tc := range cases {
		role, _ := repo.RetrieveRole(context.Background(), tc.orgID, tc.memberID)
		fmt.Println("ROLE", tc.role)
		require.Equal(t, tc.role, role, fmt.Sprintf("expected %s got %s", tc.role, role))
	}
}
