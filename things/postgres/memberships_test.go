package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/things/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSave(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)
	groupMembershipsRepo := postgres.NewGroupMembershipsRepository(dbMiddleware)

	gr := things.Group{
		ID:    generateUUID(t),
		Name:  groupName,
		OrgID: generateUUID(t),
	}

	group, err := groupRepo.Save(context.Background(), gr)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	memberID := generateUUID(t)
	memberID1 := generateUUID(t)

	gms := []things.GroupMembership{
		{
			MemberID: memberID,
			GroupID:  group.ID,
			Role:     things.Viewer,
		},
		{
			MemberID: memberID1,
			GroupID:  group.ID,
			Role:     things.Viewer,
		},
	}

	gmsWithoutMemberIDs := []things.GroupMembership{
		{
			MemberID: "",
			GroupID:  group.ID,
			Role:     things.Viewer,
		},
		{
			MemberID: "",
			GroupID:  group.ID,
			Role:     things.Viewer,
		},
	}

	gmsWithoutGroupIDs := []things.GroupMembership{
		{
			MemberID: memberID,
			GroupID:  "",
			Role:     things.Viewer,
		},
		{
			MemberID: memberID1,
			GroupID:  "",
			Role:     things.Viewer,
		},
	}

	cases := []struct {
		desc string
		gms  []things.GroupMembership
		err  error
	}{
		{
			desc: "save group memberships",
			gms:  gms,
			err:  nil,
		},
		{
			desc: "save existing group memberships",
			gms:  gms,
			err:  things.ErrGroupMembershipExists,
		},
		{
			desc: "save group memberships without group ids",
			gms:  gmsWithoutGroupIDs,
			err:  errors.ErrMalformedEntity,
		},
		{
			desc: "save group memberships without member id",
			gms:  gmsWithoutMemberIDs,
			err:  errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		err := groupMembershipsRepo.Save(context.Background(), tc.gms...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveRole(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)
	groupMembershipsRepo := postgres.NewGroupMembershipsRepository(dbMiddleware)

	gr := things.Group{
		ID:    generateUUID(t),
		Name:  groupName,
		OrgID: generateUUID(t),
	}

	group, err := groupRepo.Save(context.Background(), gr)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	memberID := generateUUID(t)

	gm := things.GroupMembership{
		GroupID:  group.ID,
		Role:     things.Viewer,
		MemberID: memberID,
	}

	err = groupMembershipsRepo.Save(context.Background(), gm)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc string
		gm   things.GroupMembership
		role string
		err  error
	}{
		{
			desc: "retrieve member role",
			gm:   gm,
			role: things.Viewer,
			err:  nil,
		},
		{
			desc: "retrieve member role without group id",
			gm: things.GroupMembership{
				GroupID:  "",
				MemberID: memberID,
			},
			role: "",
			err:  errors.ErrNotFound,
		},
		{
			desc: "retrieve member role without member id",
			gm: things.GroupMembership{
				GroupID:  group.ID,
				MemberID: "",
			},
			role: "",
			err:  errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		role, err := groupMembershipsRepo.RetrieveRole(context.Background(), tc.gm)
		assert.Equal(t, tc.role, role, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.role, role))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveByGroup(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)
	groupMembershipsRepo := postgres.NewGroupMembershipsRepository(dbMiddleware)

	gr := things.Group{
		ID:    generateUUID(t),
		Name:  groupName,
		OrgID: generateUUID(t),
	}

	group, err := groupRepo.Save(context.Background(), gr)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	for i := uint64(0); i < n; i++ {
		memberID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		gm := things.GroupMembership{
			MemberID: memberID,
			GroupID:  gr.ID,
			Role:     things.Viewer,
		}
		err = groupMembershipsRepo.Save(context.Background(), gm)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	}

	cases := []struct {
		desc     string
		groupID  string
		pageMeta apiutil.PageMetadata
		size     uint64
		err      error
	}{
		{
			desc:    "retrieve group memberships",
			groupID: group.ID,
			pageMeta: apiutil.PageMetadata{
				Offset: 0,
				Limit:  5,
				Total:  n,
			},
			size: 5,
			err:  nil,
		},
		{
			desc:    "retrieve last group membership",
			groupID: group.ID,
			pageMeta: apiutil.PageMetadata{
				Offset: n - 1,
				Limit:  1,
				Total:  n,
			},
			size: 1,
			err:  nil,
		},
		{
			desc:    "retrieve group memberships with invalid group id",
			groupID: invalidID,
			pageMeta: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			err: errors.ErrRetrieveEntity,
		},
		{
			desc:    "retrieve group memberships without group id",
			groupID: "",
			pageMeta: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			size: 0,

			err: errors.ErrRetrieveEntity,
		},
	}

	for _, tc := range cases {
		gmp, err := groupMembershipsRepo.RetrieveByGroup(context.Background(), tc.groupID, tc.pageMeta)
		size := len(gmp.GroupMemberships)
		assert.Equal(t, tc.size, uint64(size), fmt.Sprintf("%v: expected size %v got %v\n", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveGroupMemberships(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)
	groupMembershipsRepo := postgres.NewGroupMembershipsRepository(dbMiddleware)

	gr := things.Group{
		ID:    generateUUID(t),
		Name:  groupName,
		OrgID: generateUUID(t),
	}

	group, err := groupRepo.Save(context.Background(), gr)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	var memberIDs []string
	for i := uint64(0); i < n; i++ {
		memberID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		gm := things.GroupMembership{
			MemberID: memberID,
			GroupID:  group.ID,
			Role:     things.Viewer,
		}

		err = groupMembershipsRepo.Save(context.Background(), gm)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		memberIDs = append(memberIDs, memberID)
	}

	cases := []struct {
		desc      string
		groupID   string
		memberIDs []string
		err       error
	}{
		{
			desc:      "remove group memberships without group id",
			groupID:   "",
			memberIDs: memberIDs,
			err:       errors.ErrRemoveEntity,
		},
		{
			desc:      "remove group memberships without member ids",
			groupID:   group.ID,
			memberIDs: []string{""},
			err:       errors.ErrRemoveEntity,
		},
		{
			desc:      "remove group memberships",
			groupID:   group.ID,
			memberIDs: memberIDs,
			err:       nil,
		},
	}

	for _, tc := range cases {
		err := groupMembershipsRepo.Remove(context.Background(), tc.groupID, tc.memberIDs...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateGroupMemberships(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)
	groupMembershipsRepo := postgres.NewGroupMembershipsRepository(dbMiddleware)

	memberID := generateUUID(t)
	memberID1 := generateUUID(t)

	gr := things.Group{
		ID:    generateUUID(t),
		Name:  groupName,
		OrgID: generateUUID(t),
	}

	group, err := groupRepo.Save(context.Background(), gr)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	gms := []things.GroupMembership{
		{
			MemberID: memberID,
			GroupID:  gr.ID,
			Role:     things.Viewer,
		},
		{
			MemberID: memberID1,
			GroupID:  gr.ID,
			Role:     things.Viewer,
		},
	}

	err = groupMembershipsRepo.Save(context.Background(), gms...)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc string
		gm   things.GroupMembership
		err  error
	}{
		{
			desc: "update group membership without group id",
			gm: things.GroupMembership{
				MemberID: memberID,
				GroupID:  "",
				Role:     things.Viewer,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "update group membership without member id",
			gm: things.GroupMembership{
				MemberID: "",
				GroupID:  group.ID,
				Role:     things.Viewer,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "update group membership",
			gm: things.GroupMembership{
				MemberID: memberID,
				GroupID:  group.ID,
				Role:     things.Viewer,
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		err := groupMembershipsRepo.Update(context.Background(), tc.gm)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
