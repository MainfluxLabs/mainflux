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
	groupMembersRepo := postgres.NewGroupMembersRepository(dbMiddleware)

	gr := things.Group{
		ID:    generateUUID(t),
		Name:  groupName,
		OrgID: generateUUID(t),
	}

	group, err := groupRepo.Save(context.Background(), gr)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	memberID := generateUUID(t)
	memberID1 := generateUUID(t)

	gms := []things.GroupMember{
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

	gmsWithoutMemberIDs := []things.GroupMember{
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

	gmsWithoutGroupIDs := []things.GroupMember{
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
		gms  []things.GroupMember
		err  error
	}{
		{
			desc: "save group members",
			gms:  gms,
			err:  nil,
		},
		{
			desc: "save group members without group ids",
			gms:  gmsWithoutGroupIDs,
			err:  errors.ErrMalformedEntity,
		},
		{
			desc: "save group members without member id",
			gms:  gmsWithoutMemberIDs,
			err:  errors.ErrMalformedEntity,
		},
		{
			desc: "save existing group members",
			gms:  gms,
			err:  errors.ErrConflict,
		},
	}

	for _, tc := range cases {
		err := groupMembersRepo.Save(context.Background(), tc.gms...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveRole(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)
	groupMembersRepo := postgres.NewGroupMembersRepository(dbMiddleware)

	gr := things.Group{
		ID:    generateUUID(t),
		Name:  groupName,
		OrgID: generateUUID(t),
	}

	group, err := groupRepo.Save(context.Background(), gr)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	memberID := generateUUID(t)

	gm := things.GroupMember{
		GroupID:  group.ID,
		Role:     things.Viewer,
		MemberID: memberID,
	}

	err = groupMembersRepo.Save(context.Background(), gm)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc string
		gp   things.GroupMember
		role string
		err  error
	}{
		{
			desc: "retrieve group role",
			gp:   gm,
			role: things.Viewer,
			err:  nil,
		},
		{
			desc: "retrieve group role without group id",
			gp: things.GroupMember{
				GroupID:  "",
				MemberID: memberID,
			},
			role: "",
			err:  errors.ErrNotFound,
		},
		{
			desc: "retrieve group role without member id",
			gp: things.GroupMember{
				GroupID:  group.ID,
				MemberID: "",
			},
			role: "",
			err:  errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		role, err := groupMembersRepo.RetrieveRole(context.Background(), tc.gp)
		assert.Equal(t, tc.role, role, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.role, role))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveByGroup(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)
	groupMembersRepo := postgres.NewGroupMembersRepository(dbMiddleware)

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
		gm := things.GroupMember{
			MemberID: memberID,
			GroupID:  gr.ID,
			Role:     things.Viewer,
		}
		err = groupMembersRepo.Save(context.Background(), gm)
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
			desc:    "retrieve group members",
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
			desc:    "retrieve last group member",
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
			desc:    "retrieve group members with invalid group id",
			groupID: invalidID,
			pageMeta: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			err: errors.ErrRetrieveEntity,
		},
		{
			desc:    "retrieve group members without group id",
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
		gpp, err := groupMembersRepo.RetrieveByGroup(context.Background(), tc.groupID, tc.pageMeta)
		size := len(gpp.GroupMembers)
		assert.Equal(t, tc.size, uint64(size), fmt.Sprintf("%v: expected size %v got %v\n", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveGroupMembers(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)
	groupMembersRepo := postgres.NewGroupMembersRepository(dbMiddleware)

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

		gp := things.GroupMember{
			MemberID: memberID,
			GroupID:  group.ID,
			Role:     things.Viewer,
		}

		err = groupMembersRepo.Save(context.Background(), gp)
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
			desc:      "remove group members without group id",
			groupID:   "",
			memberIDs: memberIDs,
			err:       errors.ErrRemoveEntity,
		},
		{
			desc:      "remove group members without member ids",
			groupID:   group.ID,
			memberIDs: []string{""},
			err:       errors.ErrRemoveEntity,
		},
		{
			desc:      "remove group members",
			groupID:   group.ID,
			memberIDs: memberIDs,
			err:       nil,
		},
	}

	for _, tc := range cases {
		err := groupMembersRepo.Remove(context.Background(), tc.groupID, tc.memberIDs...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateGroupMembers(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)
	groupMembersRepo := postgres.NewGroupMembersRepository(dbMiddleware)

	memberID := generateUUID(t)
	memberID1 := generateUUID(t)

	gr := things.Group{
		ID:    generateUUID(t),
		Name:  groupName,
		OrgID: generateUUID(t),
	}

	group, err := groupRepo.Save(context.Background(), gr)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	gpByIDs := []things.GroupMember{
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

	err = groupMembersRepo.Save(context.Background(), gpByIDs...)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc   string
		gpByID things.GroupMember
		err    error
	}{
		{
			desc: "update group members without group id",
			gpByID: things.GroupMember{
				MemberID: memberID,
				GroupID:  "",
				Role:     things.Viewer,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "update group members without member id",
			gpByID: things.GroupMember{
				MemberID: "",
				GroupID:  group.ID,
				Role:     things.Viewer,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "update group members",
			gpByID: things.GroupMember{
				MemberID: memberID,
				GroupID:  group.ID,
				Role:     things.Viewer,
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		err := groupMembersRepo.Update(context.Background(), tc.gpByID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
