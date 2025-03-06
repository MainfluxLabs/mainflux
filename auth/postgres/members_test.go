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
	memberRelationsTable = "member_relations"
)

func TestAssignMembers(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repoOrg := postgres.NewOrgRepo(dbMiddleware)
	repoMembs := postgres.NewMembersRepo(dbMiddleware)

	orgID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ownerID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	org := auth.Org{
		ID:          orgID,
		OwnerID:     ownerID,
		Name:        orgName,
		Description: orgDesc,
		Metadata:    map[string]interface{}{"key": "value"},
	}

	err = repoOrg.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	var orgMembers []auth.OrgMember
	for i := uint64(0); i < n; i++ {
		memberID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		orgMember := auth.OrgMember{
			OrgID:     orgID,
			MemberID:  memberID,
			Role:      auth.Editor,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		orgMembers = append(orgMembers, orgMember)
	}

	var invalidOrgIDmRel []auth.OrgMember
	for _, m := range orgMembers {
		m.OrgID = invalidID
		invalidOrgIDmRel = append(invalidOrgIDmRel, m)
	}

	var emptyOrgIDmRel []auth.OrgMember
	for _, m := range orgMembers {
		m.OrgID = ""
		emptyOrgIDmRel = append(emptyOrgIDmRel, m)
	}

	var noMemberIDmRel []auth.OrgMember
	for _, m := range orgMembers {
		m.MemberID = ""
		noMemberIDmRel = append(noMemberIDmRel, m)
	}

	var invalidMemberIDmRel []auth.OrgMember
	for _, m := range orgMembers {
		m.MemberID = invalidID
		invalidMemberIDmRel = append(invalidMemberIDmRel, m)
	}

	cases := []struct {
		desc       string
		orgMembers []auth.OrgMember
		err        error
	}{
		{
			desc:       "assign members to org",
			orgMembers: orgMembers,
			err:        nil,
		},
		{
			desc:       "assign already assigned members to org",
			orgMembers: orgMembers,
			err:        auth.ErrOrgMemberAlreadyAssigned,
		},
		{
			desc:       "assign members to org with invalid org id",
			orgMembers: invalidOrgIDmRel,
			err:        errors.ErrMalformedEntity,
		},
		{
			desc:       "assign members to org without org id",
			orgMembers: emptyOrgIDmRel,
			err:        errors.ErrMalformedEntity,
		},
		{
			desc:       "assign members to org with empty member ids",
			orgMembers: noMemberIDmRel,
			err:        errors.ErrMalformedEntity,
		},
		{
			desc:       "assign members to org with invalid member ids",
			orgMembers: invalidMemberIDmRel,
			err:        errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		err := repoMembs.Save(context.Background(), tc.orgMembers...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUnassignMembers(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repoOrg := postgres.NewOrgRepo(dbMiddleware)
	repoMembs := postgres.NewMembersRepo(dbMiddleware)

	orgID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ownerID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	org := auth.Org{
		ID:          orgID,
		OwnerID:     ownerID,
		Name:        orgName,
		Description: orgDesc,
		Metadata:    map[string]interface{}{"key": "value"},
	}

	err = repoOrg.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	var orgMembers []auth.OrgMember
	var memberIDs []string
	for i := uint64(0); i < n; i++ {
		memberID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		orgMember := auth.OrgMember{
			OrgID:     orgID,
			MemberID:  memberID,
			Role:      auth.Editor,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		orgMembers = append(orgMembers, orgMember)
		memberIDs = append(memberIDs, memberID)
	}

	err = repoMembs.Save(context.Background(), orgMembers...)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc      string
		orgID     string
		memberIDs []string
		err       error
	}{
		{
			desc:      "unassign members from org with invalid org id",
			orgID:     invalidID,
			memberIDs: memberIDs,
			err:       errors.ErrMalformedEntity,
		},
		{
			desc:      "unassign members from org without org id",
			orgID:     "",
			memberIDs: memberIDs,
			err:       errors.ErrMalformedEntity,
		},
		{
			desc:      "unassign members from org without members",
			orgID:     orgID,
			memberIDs: []string{},
			err:       nil,
		},
		{
			desc:      "unassign members from org with invalid member id",
			orgID:     orgID,
			memberIDs: []string{invalidID},
			err:       errors.ErrMalformedEntity,
		},

		{
			desc:      "unassign members from org",
			orgID:     orgID,
			memberIDs: memberIDs,
			err:       nil,
		},
		{
			desc:      "unassign already unassigned members from org",
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
	repoMembs := postgres.NewMembersRepo(dbMiddleware)

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

	orgMember := auth.OrgMember{
		OrgID:     org.ID,
		MemberID:  memberID,
		Role:      auth.Admin,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = repoMembs.Save(context.Background(), orgMember)
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

func TestUpdateMembers(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repoOrg := postgres.NewOrgRepo(dbMiddleware)
	repoMembs := postgres.NewMembersRepo(dbMiddleware)

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

	orgMember := auth.OrgMember{
		OrgID:     org.ID,
		MemberID:  memberID,
		Role:      auth.Editor,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = repoMembs.Save(context.Background(), orgMember)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	updateMrel := auth.OrgMember{
		OrgID:    org.ID,
		MemberID: memberID,
		Role:     auth.Viewer,
	}

	invalidOrgIDmRel := auth.OrgMember{
		OrgID:    invalidID,
		MemberID: memberID,
		Role:     auth.Viewer,
	}

	unknownOrgIDmRel := auth.OrgMember{
		OrgID:    unknownID,
		MemberID: memberID,
		Role:     auth.Viewer,
	}

	emptyOrgIDmRel := auth.OrgMember{
		OrgID:    "",
		MemberID: memberID,
		Role:     auth.Viewer,
	}

	invalidMemberIDmRel := auth.OrgMember{
		OrgID:    org.ID,
		MemberID: invalidID,
		Role:     auth.Viewer,
	}

	unknownMemberIDmRel := auth.OrgMember{
		OrgID:    org.ID,
		MemberID: unknownID,
		Role:     auth.Viewer,
	}

	emptyMemberIDmRel := auth.OrgMember{
		OrgID:    org.ID,
		MemberID: "",
		Role:     auth.Viewer,
	}

	cases := []struct {
		desc      string
		orgMember auth.OrgMember
		err       error
	}{
		{
			desc:      "update member role",
			orgMember: updateMrel,
			err:       nil,
		}, {
			desc:      "update role with invalid org id",
			orgMember: invalidOrgIDmRel,
			err:       errors.ErrMalformedEntity,
		}, {
			desc:      "update role with unknown org id",
			orgMember: unknownOrgIDmRel,
			err:       errors.ErrNotFound,
		}, {
			desc:      "update role without org id",
			orgMember: emptyOrgIDmRel,
			err:       errors.ErrMalformedEntity,
		}, {
			desc:      "update role with invalid member id",
			orgMember: invalidMemberIDmRel,
			err:       errors.ErrMalformedEntity,
		}, {
			desc:      "update role with unknown member id",
			orgMember: unknownMemberIDmRel,
			err:       errors.ErrNotFound,
		}, {
			desc:      "update role with empty member",
			orgMember: emptyMemberIDmRel,
			err:       errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		err := repoMembs.Update(context.Background(), tc.orgMember)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveMembersByOrg(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repoOrg := postgres.NewOrgRepo(dbMiddleware)
	repoMembs := postgres.NewMembersRepo(dbMiddleware)

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
		Metadata:    map[string]interface{}{"key": "value"},
	}

	err = repoOrg.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	var orgMembers []auth.OrgMember
	for i := uint64(0); i < n; i++ {
		memberID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		orgMember := auth.OrgMember{
			OrgID:    orgID,
			MemberID: memberID,
			Role:     auth.Editor,
		}

		orgMembers = append(orgMembers, orgMember)
	}

	err = repoMembs.Save(context.Background(), orgMembers...)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc         string
		orgID        string
		pageMetadata apiutil.PageMetadata
		size         uint64
		err          error
	}{
		{
			desc:  "retrieve members by org",
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
			desc:  "retrieve members by org with unknown org id",
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
			desc:  "retrieve members by org with invalid org id",
			orgID: invalidID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			size: 0,
			err:  auth.ErrRetrieveMembersByOrg,
		},
		{
			desc:  "retrieve members by org without org id",
			orgID: "",
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			size: 0,
			err:  auth.ErrRetrieveMembersByOrg,
		},
	}

	for desc, tc := range cases {
		page, err := repoMembs.RetrieveByOrgID(context.Background(), tc.orgID, tc.pageMetadata)
		size := len(page.OrgMembers)
		assert.Equal(t, tc.size, uint64(size), fmt.Sprintf("%v: expected size %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.pageMetadata.Total, page.Total, fmt.Sprintf("%v: expected total %d got %d\n", desc, tc.pageMetadata.Total, page.Total))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveAllMembersByOrg(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repoOrg := postgres.NewOrgRepo(dbMiddleware)
	repoMembs := postgres.NewMembersRepo(dbMiddleware)

	_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", memberRelationsTable))
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
		Metadata:    map[string]interface{}{"key": "value"},
	}

	err = repoOrg.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	var orgMembers []auth.OrgMember
	for i := uint64(0); i < n; i++ {
		memberID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		orgMember := auth.OrgMember{
			OrgID:    org.ID,
			MemberID: memberID,
			Role:     auth.Editor,
		}

		orgMembers = append(orgMembers, orgMember)
	}

	err = repoMembs.Save(context.Background(), orgMembers...)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc string
		size uint64
		err  error
	}{
		{
			desc: "retrieve all member relations",
			size: n,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		page, err := repoMembs.RetrieveAll(context.Background())
		size := len(page)
		assert.Equal(t, tc.size, uint64(size), fmt.Sprintf("%v: expected size %v got %v\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
