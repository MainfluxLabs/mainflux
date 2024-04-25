package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/things/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveGroupPolicies(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)
	policiesRepo := postgres.NewPoliciesRepository(dbMiddleware)

	gr := things.Group{
		ID:      generateUUID(t),
		Name:    groupName,
		OwnerID: generateUUID(t),
		OrgID:   generateUUID(t),
	}

	group, err := groupRepo.Save(context.Background(), gr)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	memberID := generateUUID(t)
	memberID1 := generateUUID(t)

	gps := []things.GroupPolicyByID{
		{
			MemberID: memberID,
			Policy:   things.Read,
		},
		{
			MemberID: memberID1,
			Policy:   things.Read,
		},
	}

	gpsWithoutMemberIDs := []things.GroupPolicyByID{
		{
			MemberID: "",
			Policy:   things.Read,
		},
		{
			MemberID: "",
			Policy:   things.Read,
		},
	}

	cases := []struct {
		desc    string
		groupID string
		gps     []things.GroupPolicyByID
		err     error
	}{
		{
			desc:    "save group policies",
			gps:     gps,
			groupID: group.ID,
			err:     nil,
		},
		{
			desc:    "save group policies without group ids",
			gps:     gps,
			groupID: "",
			err:     errors.ErrMalformedEntity,
		},
		{
			desc:    "save group policies without member id",
			gps:     gpsWithoutMemberIDs,
			groupID: group.ID,
			err:     errors.ErrMalformedEntity,
		},
		{
			desc:    "save existing group policies",
			gps:     gps,
			groupID: group.ID,
			err:     errors.ErrConflict,
		},
	}

	for _, tc := range cases {
		err := policiesRepo.SaveGroupPolicies(context.Background(), tc.groupID, tc.gps...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveGroupPolicy(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)
	policiesRepo := postgres.NewPoliciesRepository(dbMiddleware)

	gr := things.Group{
		ID:      generateUUID(t),
		Name:    groupName,
		OwnerID: generateUUID(t),
		OrgID:   generateUUID(t),
	}

	group, err := groupRepo.Save(context.Background(), gr)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	memberID := generateUUID(t)

	gp := things.GroupPolicy{
		GroupID:  group.ID,
		Policy:   things.Read,
		MemberID: memberID,
	}

	gpByID := things.GroupPolicyByID{
		MemberID: memberID,
		Policy:   things.Read,
	}

	err = policiesRepo.SaveGroupPolicies(context.Background(), group.ID, gpByID)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc   string
		gp     things.GroupPolicy
		policy string
		err    error
	}{
		{
			desc:   "retrieve group policy",
			gp:     gp,
			policy: things.Read,
			err:    nil,
		},
		{
			desc: "retrieve group policy without group id",
			gp: things.GroupPolicy{
				GroupID:  "",
				MemberID: memberID,
			},
			policy: "",
			err:    errors.ErrRetrieveEntity,
		},
		{
			desc: "retrieve group policy without member id",
			gp: things.GroupPolicy{
				GroupID:  group.ID,
				MemberID: "",
			},
			policy: "",
			err:    errors.ErrRetrieveEntity,
		},
	}

	for _, tc := range cases {
		policy, err := policiesRepo.RetrieveGroupPolicy(context.Background(), tc.gp)
		assert.Equal(t, tc.policy, policy, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.policy, policy))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveGroupPolicies(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)
	policiesRepo := postgres.NewPoliciesRepository(dbMiddleware)

	gr := things.Group{
		ID:      generateUUID(t),
		Name:    groupName,
		OwnerID: generateUUID(t),
		OrgID:   generateUUID(t),
	}

	group, err := groupRepo.Save(context.Background(), gr)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	for i := uint64(0); i < n; i++ {
		memberID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		gp := things.GroupPolicyByID{
			MemberID: memberID,
			Policy:   things.Read,
		}
		err = policiesRepo.SaveGroupPolicies(context.Background(), group.ID, gp)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	}

	cases := []struct {
		desc     string
		groupID  string
		pageMeta things.PageMetadata
		size     uint64
		err      error
	}{
		{
			desc:    "retrieve group policies",
			groupID: group.ID,
			pageMeta: things.PageMetadata{
				Offset: 0,
				Limit:  5,
				Total:  n,
			},
			size: 5,
			err:  nil,
		},
		{
			desc:    "retrieve last group policy",
			groupID: group.ID,
			pageMeta: things.PageMetadata{
				Offset: n - 1,
				Limit:  1,
				Total:  n,
			},
			size: 1,
			err:  nil,
		},
		{
			desc:    "retrieve group policies with invalid group id",
			groupID: invalidID,
			pageMeta: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			err: errors.ErrRetrieveEntity,
		},
		{
			desc:    "retrieve group policies without group id",
			groupID: "",
			pageMeta: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			size: 0,

			err: errors.ErrRetrieveEntity,
		},
	}

	for _, tc := range cases {
		gpp, err := policiesRepo.RetrieveGroupPolicies(context.Background(), tc.groupID, tc.pageMeta)
		size := len(gpp.GroupPolicies)
		assert.Equal(t, tc.size, uint64(size), fmt.Sprintf("%v: expected size %v got %v\n", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveGroupPolicies(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)
	policiesRepo := postgres.NewPoliciesRepository(dbMiddleware)

	gr := things.Group{
		ID:      generateUUID(t),
		Name:    groupName,
		OwnerID: generateUUID(t),
		OrgID:   generateUUID(t),
	}

	group, err := groupRepo.Save(context.Background(), gr)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	var memberIDs []string
	for i := uint64(0); i < n; i++ {
		memberID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		gp := things.GroupPolicyByID{
			MemberID: memberID,
			Policy:   things.Read,
		}

		err = policiesRepo.SaveGroupPolicies(context.Background(), group.ID, gp)
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
			desc:      "remove group policies without group id",
			groupID:   "",
			memberIDs: memberIDs,
			err:       errors.ErrRemoveEntity,
		},
		{
			desc:      "remove group policies without member ids",
			groupID:   group.ID,
			memberIDs: []string{""},
			err:       errors.ErrRemoveEntity,
		},
		{
			desc:      "remove group policies",
			groupID:   group.ID,
			memberIDs: memberIDs,
			err:       nil,
		},
	}

	for _, tc := range cases {
		err := policiesRepo.RemoveGroupPolicies(context.Background(), tc.groupID, tc.memberIDs...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateGroupPolicies(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)
	policiesRepo := postgres.NewPoliciesRepository(dbMiddleware)

	memberID := generateUUID(t)
	memberID1 := generateUUID(t)

	gr := things.Group{
		ID:      generateUUID(t),
		Name:    groupName,
		OwnerID: generateUUID(t),
		OrgID:   generateUUID(t),
	}

	group, err := groupRepo.Save(context.Background(), gr)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	gpByIDs := []things.GroupPolicyByID{
		{
			MemberID: memberID,
			Policy:   things.Read,
		},
		{
			MemberID: memberID1,
			Policy:   things.Read,
		},
	}

	err = policiesRepo.SaveGroupPolicies(context.Background(), group.ID, gpByIDs...)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc    string
		groupID string
		gpByID  things.GroupPolicyByID
		err     error
	}{
		{
			desc:    "update group policies without group id",
			groupID: "",
			gpByID: things.GroupPolicyByID{
				MemberID: "",
				Policy:   things.Read,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc:    "update group policies without member id",
			groupID: group.ID,
			gpByID: things.GroupPolicyByID{
				MemberID: "",
				Policy:   things.Read,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc:    "update group policies",
			groupID: group.ID,
			gpByID: things.GroupPolicyByID{
				MemberID: memberID,
				Policy:   things.Read,
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		err := policiesRepo.UpdateGroupPolicies(context.Background(), tc.groupID, tc.gpByID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
