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

func TestSaveGroupMembers(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	orgsRepo := postgres.NewOrgRepo(dbMiddleware)
	membersRepo := postgres.NewMembersRepo(dbMiddleware)

	_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", groupRelationsTable))
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	memberID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	memberID1, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	orgID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	org := auth.Org{
		ID:          orgID,
		OwnerID:     memberID,
		Name:        orgName,
		Description: orgDesc,
		Metadata:    map[string]interface{}{"key": "value"},
	}

	err = orgsRepo.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	groupID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	og := auth.OrgGroup{
		OrgID:   orgID,
		GroupID: groupID,
	}
	err = orgsRepo.AssignGroups(context.Background(), og)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	giByIDs := []auth.GroupInvitationByID{
		{
			MemberID: memberID,
			Policy:   auth.RwPolicy,
		},
		{
			MemberID: memberID1,
			Policy:   auth.RPolicy,
		},
	}

	giWithoutMemberIDs := []auth.GroupInvitationByID{
		{
			MemberID: "",
			Policy:   auth.RwPolicy,
		},
		{
			MemberID: "",
			Policy:   auth.RPolicy,
		},
	}

	cases := []struct {
		desc    string
		groupID string
		giByIDs []auth.GroupInvitationByID
		err     error
	}{
		{
			desc:    "save group members",
			giByIDs: giByIDs,
			groupID: groupID,
			err:     nil,
		},
		{
			desc:    "save group members without group ids",
			giByIDs: giByIDs,
			groupID: "",
			err:     errors.ErrMalformedEntity,
		},
		{
			desc:    "save group members without member id",
			giByIDs: giWithoutMemberIDs,
			groupID: groupID,
			err:     errors.ErrMalformedEntity,
		},
		{
			desc:    "save existing group members",
			giByIDs: giByIDs,
			groupID: groupID,
			err:     errors.ErrConflict,
		},
	}

	for _, tc := range cases {
		err := membersRepo.SaveGroupMembers(context.Background(), tc.groupID, tc.giByIDs...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveGroupMemberPolicy(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	orgsRepo := postgres.NewOrgRepo(dbMiddleware)
	membersRepo := postgres.NewMembersRepo(dbMiddleware)

	_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", groupRelationsTable))
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	groupID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	memberID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	orgID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	org := auth.Org{
		ID:          orgID,
		OwnerID:     memberID,
		Name:        orgName,
		Description: orgDesc,
		Metadata:    map[string]interface{}{"key": "value"},
	}

	err = orgsRepo.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	og := auth.OrgGroup{
		OrgID:   orgID,
		GroupID: groupID,
	}
	err = orgsRepo.AssignGroups(context.Background(), og)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	gp := auth.GroupsPolicy{
		GroupID:  groupID,
		Policy:   auth.RwPolicy,
		MemberID: memberID,
	}

	giByID := auth.GroupInvitationByID{
		MemberID: memberID,
		Policy:   auth.RwPolicy,
	}

	err = membersRepo.SaveGroupMembers(context.Background(), groupID, giByID)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc   string
		gp     auth.GroupsPolicy
		policy string
		err    error
	}{
		{
			desc:   "retrieve policy",
			gp:     gp,
			policy: auth.RwPolicy,
			err:    nil,
		},
		{
			desc: "retrieve policy without group id",
			gp: auth.GroupsPolicy{
				GroupID:  "",
				MemberID: memberID,
			},
			policy: "",
			err:    errors.ErrRetrieveEntity,
		},
		{
			desc: "retrieve policy without member id",
			gp: auth.GroupsPolicy{
				GroupID:  groupID,
				MemberID: "",
			},
			policy: "",
			err:    errors.ErrRetrieveEntity,
		},
	}

	for _, tc := range cases {
		policy, err := membersRepo.RetrieveGroupMemberPolicy(context.Background(), tc.gp)
		assert.Equal(t, tc.policy, policy, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.policy, policy))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveGroupMembers(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	orgsRepo := postgres.NewOrgRepo(dbMiddleware)
	membersRepo := postgres.NewMembersRepo(dbMiddleware)

	_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", groupRelationsTable))
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	groupID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ownerID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	orgID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	org := auth.Org{
		ID:          orgID,
		OwnerID:     ownerID,
		Name:        orgName,
		Description: orgDesc,
		Metadata:    map[string]interface{}{"key": "value"},
	}

	err = orgsRepo.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	og := auth.OrgGroup{
		OrgID:   orgID,
		GroupID: groupID,
	}
	err = orgsRepo.AssignGroups(context.Background(), og)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	for i := uint64(0); i < n; i++ {
		memberID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		giByID := auth.GroupInvitationByID{
			MemberID: memberID,
			Policy:   auth.RwPolicy,
		}
		err = membersRepo.SaveGroupMembers(context.Background(), groupID, giByID)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	}

	cases := []struct {
		desc     string
		groupID  string
		pageMeta auth.PageMetadata
		size     uint64
		err      error
	}{
		{
			desc:    "retrieve policies",
			groupID: groupID,
			pageMeta: auth.PageMetadata{
				Offset: 0,
				Limit:  5,
				Total:  n,
			},
			size: 5,
			err:  nil,
		},
		{
			desc:    "retrieve last policy",
			groupID: groupID,
			pageMeta: auth.PageMetadata{
				Offset: n - 1,
				Limit:  1,
				Total:  n,
			},
			size: 1,
			err:  nil,
		},
		{
			desc:    "retrieve policies with invalid group id",
			groupID: invalidID,
			pageMeta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			err: errors.ErrRetrieveEntity,
		},
		{
			desc:    "retrieve policies without group id",
			groupID: "",
			pageMeta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			size: 0,

			err: errors.ErrRetrieveEntity,
		},
	}

	for _, tc := range cases {
		mpp, err := membersRepo.RetrieveGroupMembers(context.Background(), tc.groupID, tc.pageMeta)
		size := len(mpp.GroupMembers)
		assert.Equal(t, tc.size, uint64(size), fmt.Sprintf("%v: expected size %v got %v\n", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveGroupMembers(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	orgsRepo := postgres.NewOrgRepo(dbMiddleware)
	membersRepo := postgres.NewMembersRepo(dbMiddleware)

	_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", groupRelationsTable))
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	groupID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ownerID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	orgID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	org := auth.Org{
		ID:          orgID,
		OwnerID:     ownerID,
		Name:        orgName,
		Description: orgDesc,
		Metadata:    map[string]interface{}{"key": "value"},
	}

	err = orgsRepo.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	og := auth.OrgGroup{
		OrgID:   orgID,
		GroupID: groupID,
	}
	err = orgsRepo.AssignGroups(context.Background(), og)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	var memberIDs []string
	for i := uint64(0); i < n; i++ {
		memberID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		giByID := auth.GroupInvitationByID{
			MemberID: memberID,
			Policy:   auth.RwPolicy,
		}

		err = membersRepo.SaveGroupMembers(context.Background(), groupID, giByID)
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
			desc:      "remove policies without group id",
			groupID:   "",
			memberIDs: memberIDs,
			err:       errors.ErrRemoveEntity,
		},
		{
			desc:      "remove policies without member id",
			groupID:   groupID,
			memberIDs: []string{""},
			err:       errors.ErrRemoveEntity,
		},
		{
			desc:      "remove policies",
			groupID:   groupID,
			memberIDs: memberIDs,
			err:       nil,
		},
	}

	for _, tc := range cases {
		err := membersRepo.RemoveGroupMembers(context.Background(), tc.groupID, tc.memberIDs...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateGroupMembers(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	orgsRepo := postgres.NewOrgRepo(dbMiddleware)
	membersRepo := postgres.NewMembersRepo(dbMiddleware)

	_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", groupRelationsTable))
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	groupID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	memberID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	memberID1, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	orgID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	org := auth.Org{
		ID:          orgID,
		OwnerID:     memberID,
		Name:        orgName,
		Description: orgDesc,
		Metadata:    map[string]interface{}{"key": "value"},
	}

	err = orgsRepo.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	og := auth.OrgGroup{
		OrgID:   orgID,
		GroupID: groupID,
	}
	err = orgsRepo.AssignGroups(context.Background(), og)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	giByIDs := []auth.GroupInvitationByID{
		{
			MemberID: memberID,
			Policy:   auth.RwPolicy,
		},
		{
			MemberID: memberID1,
			Policy:   auth.RPolicy,
		},
	}

	err = membersRepo.SaveGroupMembers(context.Background(), groupID, giByIDs...)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc    string
		groupID string
		giByID  auth.GroupInvitationByID
		err     error
	}{
		{
			desc:    "update policy without group id",
			groupID: "",
			giByID: auth.GroupInvitationByID{
				MemberID: "",
				Policy:   auth.RPolicy,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc:    "update policy without member id",
			groupID: groupID,
			giByID: auth.GroupInvitationByID{
				MemberID: "",
				Policy:   auth.RPolicy,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc:    "update policy",
			groupID: groupID,
			giByID: auth.GroupInvitationByID{
				MemberID: memberID,
				Policy:   auth.RPolicy,
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		err := membersRepo.UpdateGroupMembers(context.Background(), tc.groupID, tc.giByID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
