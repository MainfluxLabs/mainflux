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

func TestSaveGroupPolicies(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	orgsRepo := postgres.NewOrgRepo(dbMiddleware)
	policiesRepo := postgres.NewPoliciesRepo(dbMiddleware)

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

	gps := []auth.GroupPolicyByID{
		{
			MemberID: memberID,
			Policy:   auth.RwPolicy,
		},
		{
			MemberID: memberID1,
			Policy:   auth.RPolicy,
		},
	}

	gpsWithoutMemberIDs := []auth.GroupPolicyByID{
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
		gps     []auth.GroupPolicyByID
		err     error
	}{
		{
			desc:    "save group policies",
			gps:     gps,
			groupID: groupID,
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
			groupID: groupID,
			err:     errors.ErrMalformedEntity,
		},
		{
			desc:    "save existing group policies",
			gps:     gps,
			groupID: groupID,
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
	orgsRepo := postgres.NewOrgRepo(dbMiddleware)
	policiesRepo := postgres.NewPoliciesRepo(dbMiddleware)

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

	gp := auth.GroupPolicy{
		GroupID:  groupID,
		Policy:   auth.RwPolicy,
		MemberID: memberID,
	}

	gpByID := auth.GroupPolicyByID{
		MemberID: memberID,
		Policy:   auth.RwPolicy,
	}

	err = policiesRepo.SaveGroupPolicies(context.Background(), groupID, gpByID)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc   string
		gp     auth.GroupPolicy
		policy string
		err    error
	}{
		{
			desc:   "retrieve group policy",
			gp:     gp,
			policy: auth.RwPolicy,
			err:    nil,
		},
		{
			desc: "retrieve group policy without group id",
			gp: auth.GroupPolicy{
				GroupID:  "",
				MemberID: memberID,
			},
			policy: "",
			err:    errors.ErrRetrieveEntity,
		},
		{
			desc: "retrieve group policy without member id",
			gp: auth.GroupPolicy{
				GroupID:  groupID,
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
	orgsRepo := postgres.NewOrgRepo(dbMiddleware)
	policiesRepo := postgres.NewPoliciesRepo(dbMiddleware)

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
		gp := auth.GroupPolicyByID{
			MemberID: memberID,
			Policy:   auth.RwPolicy,
		}
		err = policiesRepo.SaveGroupPolicies(context.Background(), groupID, gp)
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
			desc:    "retrieve group policies",
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
			desc:    "retrieve last group policy",
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
			desc:    "retrieve group policies with invalid group id",
			groupID: invalidID,
			pageMeta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			err: errors.ErrRetrieveEntity,
		},
		{
			desc:    "retrieve group policies without group id",
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
		gpp, err := policiesRepo.RetrieveGroupPolicies(context.Background(), tc.groupID, tc.pageMeta)
		size := len(gpp.GroupPolicies)
		assert.Equal(t, tc.size, uint64(size), fmt.Sprintf("%v: expected size %v got %v\n", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveGroupPolicies(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	orgsRepo := postgres.NewOrgRepo(dbMiddleware)
	policiesRepo := postgres.NewPoliciesRepo(dbMiddleware)

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

		gp := auth.GroupPolicyByID{
			MemberID: memberID,
			Policy:   auth.RwPolicy,
		}

		err = policiesRepo.SaveGroupPolicies(context.Background(), groupID, gp)
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
			groupID:   groupID,
			memberIDs: []string{""},
			err:       errors.ErrRemoveEntity,
		},
		{
			desc:      "remove group policies",
			groupID:   groupID,
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
	orgsRepo := postgres.NewOrgRepo(dbMiddleware)
	policiesRepo := postgres.NewPoliciesRepo(dbMiddleware)

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

	gpByIDs := []auth.GroupPolicyByID{
		{
			MemberID: memberID,
			Policy:   auth.RwPolicy,
		},
		{
			MemberID: memberID1,
			Policy:   auth.RPolicy,
		},
	}

	err = policiesRepo.SaveGroupPolicies(context.Background(), groupID, gpByIDs...)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc    string
		groupID string
		gpByID  auth.GroupPolicyByID
		err     error
	}{
		{
			desc:    "update group policies without group id",
			groupID: "",
			gpByID: auth.GroupPolicyByID{
				MemberID: "",
				Policy:   auth.RPolicy,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc:    "update group policies without member id",
			groupID: groupID,
			gpByID: auth.GroupPolicyByID{
				MemberID: "",
				Policy:   auth.RPolicy,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc:    "update group policies",
			groupID: groupID,
			gpByID: auth.GroupPolicyByID{
				MemberID: memberID,
				Policy:   auth.RPolicy,
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		err := policiesRepo.UpdateGroupPolicies(context.Background(), tc.groupID, tc.gpByID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
