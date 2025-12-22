package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/auth/postgres"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	membershipsTable = "org_memberships"
)

func TestSaveOrgMemberships(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repoOrg := postgres.NewOrgRepo(dbMiddleware)
	repoMembs := postgres.NewOrgMembershipsRepo(dbMiddleware)

	orgID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ownerID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	org := auth.Org{
		ID:          orgID,
		OwnerID:     ownerID,
		Name:        orgName,
		Description: orgDesc,
		Metadata:    map[string]any{"key": "value"},
	}

	err = repoOrg.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	var orgMemberships []auth.OrgMembership
	for i := uint64(0); i < n; i++ {
		memberID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		orgMembership := auth.OrgMembership{
			OrgID:     orgID,
			MemberID:  memberID,
			Role:      auth.Editor,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		orgMemberships = append(orgMemberships, orgMembership)
	}

	var invalidOrgIDmRel []auth.OrgMembership
	for _, m := range orgMemberships {
		m.OrgID = invalidID
		invalidOrgIDmRel = append(invalidOrgIDmRel, m)
	}

	var emptyOrgData []auth.OrgMembership
	for _, m := range orgMemberships {
		m.OrgID = ""
		emptyOrgData = append(emptyOrgData, m)
	}

	var noMembershipData []auth.OrgMembership
	for _, m := range orgMemberships {
		m.MemberID = ""
		noMembershipData = append(noMembershipData, m)
	}

	var invalidMembershipData []auth.OrgMembership
	for _, m := range orgMemberships {
		m.MemberID = invalidID
		invalidMembershipData = append(invalidMembershipData, m)
	}

	cases := []struct {
		desc           string
		orgMemberships []auth.OrgMembership
		err            error
	}{
		{
			desc:           "create org memberships",
			orgMemberships: orgMemberships,
			err:            nil,
		},
		{
			desc:           "create org memberships that already exist",
			orgMemberships: orgMemberships,
			err:            auth.ErrOrgMembershipExists,
		},
		{
			desc:           "create org memberships with invalid org id",
			orgMemberships: invalidOrgIDmRel,
			err:            dbutil.ErrMalformedEntity,
		},
		{
			desc:           "create org memberships without org id",
			orgMemberships: emptyOrgData,
			err:            dbutil.ErrMalformedEntity,
		},
		{
			desc:           "create org memberships with empty member ids",
			orgMemberships: noMembershipData,
			err:            dbutil.ErrMalformedEntity,
		},
		{
			desc:           "create org memberships with invalid member ids",
			orgMemberships: invalidMembershipData,
			err:            dbutil.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		err := repoMembs.Save(context.Background(), tc.orgMemberships...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveOrgMemberships(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repoOrg := postgres.NewOrgRepo(dbMiddleware)
	repoMembs := postgres.NewOrgMembershipsRepo(dbMiddleware)

	orgID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ownerID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	org := auth.Org{
		ID:          orgID,
		OwnerID:     ownerID,
		Name:        orgName,
		Description: orgDesc,
		Metadata:    map[string]any{"key": "value"},
	}

	err = repoOrg.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	var oms []auth.OrgMembership
	var memberIDs []string
	for i := uint64(0); i < n; i++ {
		memberID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		om := auth.OrgMembership{
			OrgID:     orgID,
			MemberID:  memberID,
			Role:      auth.Editor,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		oms = append(oms, om)
		memberIDs = append(memberIDs, memberID)
	}

	err = repoMembs.Save(context.Background(), oms...)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc      string
		orgID     string
		memberIDs []string
		err       error
	}{
		{
			desc:      "remove org memberships with invalid org id",
			orgID:     invalidID,
			memberIDs: memberIDs,
			err:       dbutil.ErrMalformedEntity,
		},
		{
			desc:      "remove org memberships without org id",
			orgID:     "",
			memberIDs: memberIDs,
			err:       dbutil.ErrMalformedEntity,
		},
		{
			desc:      "remove org memberships without member IDs",
			orgID:     orgID,
			memberIDs: []string{},
			err:       nil,
		},
		{
			desc:      "remove org memberships with invalid member id",
			orgID:     orgID,
			memberIDs: []string{invalidID},
			err:       dbutil.ErrMalformedEntity,
		},

		{
			desc:      "remove org memberships",
			orgID:     orgID,
			memberIDs: memberIDs,
			err:       nil,
		},
		{
			desc:      "remove already removed org memberships",
			orgID:     orgID,
			memberIDs: memberIDs,
			err:       nil,
		},
	}

	for _, tc := range cases {
		err := repoMembs.Remove(context.Background(), tc.orgID, tc.memberIDs...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveRole(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repoOrg := postgres.NewOrgRepo(dbMiddleware)
	repoMembs := postgres.NewOrgMembershipsRepo(dbMiddleware)

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
	err = repoOrg.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	orgMembership := auth.OrgMembership{
		OrgID:     org.ID,
		MemberID:  memberID,
		Role:      auth.Admin,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = repoMembs.Save(context.Background(), orgMembership)
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
			role:     auth.Admin,
			err:      nil,
		},
		{
			desc:     "retrieve role with invalid member id",
			orgID:    org.ID,
			memberID: invalidID,
			role:     "",
			err:      nil,
		},
		{
			desc:     "retrieve role without member id",
			orgID:    org.ID,
			memberID: "",
			role:     "",
			err:      nil,
		},
		{
			desc:     "retrieve role with invalid org id",
			orgID:    invalidID,
			memberID: memberID,
			role:     "",
			err:      nil,
		},
		{
			desc:     "retrieve role without org id",
			orgID:    "",
			memberID: memberID,
			role:     "",
			err:      nil,
		},
	}

	for _, tc := range cases {
		role, _ := repoMembs.RetrieveRole(context.Background(), tc.memberID, tc.orgID)
		require.Equal(t, tc.role, role, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.role, role))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateOrgMemberships(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repoOrg := postgres.NewOrgRepo(dbMiddleware)
	repoMembs := postgres.NewOrgMembershipsRepo(dbMiddleware)

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
	err = repoOrg.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	orgMembership := auth.OrgMembership{
		OrgID:     org.ID,
		MemberID:  memberID,
		Role:      auth.Editor,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = repoMembs.Save(context.Background(), orgMembership)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	updateMembership := auth.OrgMembership{
		OrgID:    org.ID,
		MemberID: memberID,
		Role:     auth.Viewer,
	}

	invalidOrgData := auth.OrgMembership{
		OrgID:    invalidID,
		MemberID: memberID,
		Role:     auth.Viewer,
	}

	unknownOrgData := auth.OrgMembership{
		OrgID:    unknownID,
		MemberID: memberID,
		Role:     auth.Viewer,
	}

	emptyOrgData := auth.OrgMembership{
		OrgID:    "",
		MemberID: memberID,
		Role:     auth.Viewer,
	}

	invalidMembershipData := auth.OrgMembership{
		OrgID:    org.ID,
		MemberID: invalidID,
		Role:     auth.Viewer,
	}

	unknownMembershipData := auth.OrgMembership{
		OrgID:    org.ID,
		MemberID: unknownID,
		Role:     auth.Viewer,
	}

	emptyMembershipData := auth.OrgMembership{
		OrgID:    org.ID,
		MemberID: "",
		Role:     auth.Viewer,
	}

	cases := []struct {
		desc          string
		orgMembership auth.OrgMembership
		err           error
	}{
		{
			desc:          "update membership",
			orgMembership: updateMembership,
			err:           nil,
		}, {
			desc:          "update membership with invalid org id",
			orgMembership: invalidOrgData,
			err:           dbutil.ErrMalformedEntity,
		}, {
			desc:          "update membership with unknown org id",
			orgMembership: unknownOrgData,
			err:           dbutil.ErrNotFound,
		}, {
			desc:          "update membership without org id",
			orgMembership: emptyOrgData,
			err:           dbutil.ErrMalformedEntity,
		}, {
			desc:          "update membership with invalid member id",
			orgMembership: invalidMembershipData,
			err:           dbutil.ErrMalformedEntity,
		}, {
			desc:          "update membership with unknown member id",
			orgMembership: unknownMembershipData,
			err:           dbutil.ErrNotFound,
		}, {
			desc:          "update membership with empty member",
			orgMembership: emptyMembershipData,
			err:           dbutil.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		err := repoMembs.Update(context.Background(), tc.orgMembership)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveOrgMemberships(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repoOrg := postgres.NewOrgRepo(dbMiddleware)
	repoMembs := postgres.NewOrgMembershipsRepo(dbMiddleware)

	orgID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ownerID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	unknownID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	org := auth.Org{
		ID:          orgID,
		OwnerID:     ownerID,
		Name:        orgName,
		Description: orgDesc,
		Metadata:    map[string]any{"key": "value"},
	}

	err = repoOrg.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	var orgMemberships []auth.OrgMembership
	for i := uint64(0); i < n; i++ {
		memberID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		orgMembership := auth.OrgMembership{
			OrgID:    orgID,
			MemberID: memberID,
			Role:     auth.Editor,
		}

		orgMemberships = append(orgMemberships, orgMembership)
	}

	err = repoMembs.Save(context.Background(), orgMemberships...)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc         string
		orgID        string
		pageMetadata apiutil.PageMetadata
		size         uint64
		err          error
	}{
		{
			desc:  "retrieve memberships by org",
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:  "retrieve memberships by org with unknown org id",
			orgID: unknownID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			size: 0,
			err:  nil,
		},
		{
			desc:  "retrieve memberships by org with invalid org id",
			orgID: invalidID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			size: 0,
			err:  auth.ErrRetrieveMembershipsByOrg,
		},
		{
			desc:  "retrieve memberships by org without org id",
			orgID: "",
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			size: 0,
			err:  auth.ErrRetrieveMembershipsByOrg,
		},
	}

	for desc, tc := range cases {
		page, err := repoMembs.RetrieveByOrg(context.Background(), tc.orgID, tc.pageMetadata)
		size := len(page.OrgMemberships)
		assert.Equal(t, tc.size, uint64(size), fmt.Sprintf("%v: expected size %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.pageMetadata.Total, page.Total, fmt.Sprintf("%v: expected total %d got %d\n", desc, tc.pageMetadata.Total, page.Total))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveAllMemberships(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repoOrg := postgres.NewOrgRepo(dbMiddleware)
	repoMembs := postgres.NewOrgMembershipsRepo(dbMiddleware)

	_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", membershipsTable))
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	orgID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ownerID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	org := auth.Org{
		ID:          orgID,
		OwnerID:     ownerID,
		Name:        orgName,
		Description: orgDesc,
		Metadata:    map[string]any{"key": "value"},
	}

	err = repoOrg.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	var orgMemberships []auth.OrgMembership
	for i := uint64(0); i < n; i++ {
		memberID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		orgMembership := auth.OrgMembership{
			OrgID:    org.ID,
			MemberID: memberID,
			Role:     auth.Editor,
		}

		orgMemberships = append(orgMemberships, orgMembership)
	}

	err = repoMembs.Save(context.Background(), orgMemberships...)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc string
		size uint64
		err  error
	}{
		{
			desc: "retrieve all memberships",
			size: n,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		page, err := repoMembs.BackupAll(context.Background())
		size := len(page)
		assert.Equal(t, tc.size, uint64(size), fmt.Sprintf("%v: expected size %v got %v\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
