package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/auth/postgres"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	orgName              = "test"
	orgDesc              = "test_description"
	invalidID            = "invalid"
	n                    = uint64(10)
	orgsTable            = "orgs"
	memberRelationsTable = "member_relations"
)

func TestSave(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewOrgRepo(dbMiddleware)

	id, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ownerID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	org := auth.Org{
		ID:          id,
		OwnerID:     ownerID,
		Name:        orgName,
		Description: orgDesc,
	}

	invalidOwnerOrg := auth.Org{
		ID:          id,
		OwnerID:     invalidID,
		Name:        orgName,
		Description: orgDesc,
	}

	invalidIDOrg := auth.Org{
		ID:          invalidID,
		OwnerID:     ownerID,
		Name:        orgName,
		Description: orgDesc,
	}

	cases := []struct {
		desc string
		org  auth.Org
		err  error
	}{
		{
			desc: "save org with invalid owner id",
			org:  invalidOwnerOrg,
			err:  errors.ErrMalformedEntity,
		},
		{
			desc: "save org with invalid org id",
			org:  invalidIDOrg,
			err:  errors.ErrMalformedEntity,
		},
		{
			desc: "save empty org",
			org:  auth.Org{},
			err:  errors.ErrMalformedEntity,
		},
		{
			desc: "save org",
			org:  org,
			err:  nil,
		},
		{
			desc: "save existing org",
			org:  org,
			err:  errors.ErrConflict,
		},
	}

	for _, tc := range cases {
		err := repo.Save(context.Background(), tc.org)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdate(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewOrgRepo(dbMiddleware)

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

	err = repo.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	updateOrg := auth.Org{
		ID:          orgID,
		Name:        "updated-name",
		Description: "updated-description",
		Metadata:    map[string]interface{}{"updated": "metadata"},
	}
	updateOwnerOrg := auth.Org{ID: orgID, OwnerID: unknownID}
	nonExistingOrg := auth.Org{ID: unknownID}
	invalidIDOrg := auth.Org{ID: invalidID}

	cases := []struct {
		desc string
		org  auth.Org
		err  error
	}{
		{
			desc: "update org owner",
			org:  updateOwnerOrg,
			err:  nil,
		},
		{
			desc: "update non existing org",
			org:  nonExistingOrg,
			err:  nil,
		},
		{
			desc: "update org with invalid org id",
			org:  invalidIDOrg,
			err:  errors.ErrMalformedEntity,
		},
		{
			desc: "update with empty org",
			org:  auth.Org{},
			err:  errors.ErrMalformedEntity,
		},
		{
			desc: "update org",
			org:  updateOrg,
			err:  nil,
		},
	}

	for _, tc := range cases {
		err := repo.Update(context.Background(), tc.org)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestDelete(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewOrgRepo(dbMiddleware)

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

	err = repo.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc    string
		orgID   string
		ownerID string
		err     error
	}{
		{
			desc:    "remove org with invalid org id",
			orgID:   invalidID,
			ownerID: ownerID,
			err:     errors.ErrMalformedEntity,
		},
		{
			desc:    "remove org with unknown org id",
			orgID:   unknownID,
			ownerID: ownerID,
			err:     errors.ErrRemoveEntity,
		},
		{
			desc:    "remove org with invalid owner id",
			orgID:   orgID,
			ownerID: invalidID,
			err:     errors.ErrMalformedEntity,
		},
		{
			desc:    "remove org with unknown owner id",
			orgID:   orgID,
			ownerID: unknownID,
			err:     errors.ErrRemoveEntity,
		},
		{
			desc:    "remove org",
			orgID:   orgID,
			ownerID: ownerID,
			err:     nil,
		},
		{
			desc:    "remove removed org",
			orgID:   orgID,
			ownerID: ownerID,
			err:     errors.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		err := repo.Delete(context.Background(), tc.ownerID, tc.orgID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveByID(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewOrgRepo(dbMiddleware)

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

	err = repo.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc  string
		orgID string
		err   error
	}{
		{
			desc:  "retrieve org by id",
			orgID: orgID,
			err:   nil,
		},
		{
			desc:  "retrieve org with unknown org id",
			orgID: unknownID,
			err:   errors.ErrNotFound,
		},
		{
			desc:  "retrieve org with invalid org id",
			orgID: invalidID,
			err:   errors.ErrRetrieveEntity,
		},
		{
			desc:  "retrieve org without org id",
			orgID: "",
			err:   errors.ErrRetrieveEntity,
		},
	}

	for _, tc := range cases {
		_, err := repo.RetrieveByID(context.Background(), tc.orgID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveByOwner(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewOrgRepo(dbMiddleware)

	ownerID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	unknownID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	for i := uint64(0); i < n; i++ {
		orgID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		org := auth.Org{
			ID:          orgID,
			OwnerID:     ownerID,
			Name:        fmt.Sprintf("%s-%d", orgName, i),
			Description: fmt.Sprintf("%s-%d", orgDesc, i),
			Metadata:    map[string]interface{}{fmt.Sprintf("key-%d", i): fmt.Sprintf("value-%d", i)},
		}

		err = repo.Save(context.Background(), org)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	}

	cases := []struct {
		desc         string
		ownerID      string
		pageMetadata auth.PageMetadata
		size         uint64
		err          error
	}{
		{
			desc:    "retrieve orgs by owner",
			ownerID: ownerID,
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:    "retrieve half of orgs by owner",
			ownerID: ownerID,
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n / 2,
				Total:  n,
			},
			size: n / 2,
			err:  nil,
		},
		{
			desc:    "retrieve last org by owner",
			ownerID: ownerID,
			pageMetadata: auth.PageMetadata{
				Offset: n - 1,
				Limit:  1,
				Total:  n,
			},
			size: 1,
			err:  nil,
		},
		{
			desc:    "retrieve orgs by owner filtered by name",
			ownerID: ownerID,
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  1,
				Name:   orgName + "-1",
			},
			size: 1,
			err:  nil,
		},
		{
			desc:    "retrieve orgs by owner filtered by part of name",
			ownerID: ownerID,
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  n,
				Name:   orgName,
			},
			size: n,
			err:  nil,
		},
		{
			desc:    "retrieve orgs by owner filtered by metadata",
			ownerID: ownerID,
			pageMetadata: auth.PageMetadata{
				Offset:   0,
				Limit:    n,
				Total:    1,
				Metadata: map[string]interface{}{"key-1": "value-1"},
			},
			size: 1,
			err:  nil,
		},
		{
			desc:    "retrieve orgs by owner with invalid owner id",
			ownerID: invalidID,
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			size: 0,
			err:  errors.ErrRetrieveEntity,
		},
		{
			desc:    "retrieve orgs by owner without owner id",
			ownerID: "",
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			size: 0,
			err:  errors.ErrRetrieveEntity,
		},
		{
			desc:    "retrieve orgs by owner with unknown owner id",
			ownerID: unknownID,
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			size: 0,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		page, err := repo.RetrieveByOwner(context.Background(), tc.ownerID, tc.pageMetadata)
		size := len(page.Orgs)
		assert.Equal(t, tc.size, uint64(size), fmt.Sprintf("%v: expected size %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.pageMetadata.Total, page.Total, fmt.Sprintf("%v: expected total %d got %d\n", desc, tc.pageMetadata.Total, page.Total))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveAll(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewOrgRepo(dbMiddleware)

	_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", orgsTable))
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	ownerID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	for i := uint64(0); i < n; i++ {
		orgID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		org := auth.Org{
			ID:          orgID,
			OwnerID:     ownerID,
			Name:        fmt.Sprintf("%s-%d", orgName, i),
			Description: fmt.Sprintf("%s-%d", orgDesc, i),
			Metadata:    map[string]interface{}{fmt.Sprintf("key-%d", i): fmt.Sprintf("value-%d", i)},
		}

		err = repo.Save(context.Background(), org)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	}

	cases := []struct {
		desc string
		size uint64
		err  error
	}{
		{
			desc: "retrieve all orgs",
			size: n,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		orgs, err := repo.RetrieveAll(context.Background())
		size := len(orgs)
		assert.Equal(t, tc.size, uint64(size), fmt.Sprintf("%v: expected size %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveOrgsByMember(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewOrgRepo(dbMiddleware)

	ownerID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	memberID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	unknownID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	for i := uint64(0); i < n; i++ {
		orgID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		org := auth.Org{
			ID:          orgID,
			OwnerID:     ownerID,
			Name:        fmt.Sprintf("%s-%d", orgName, i),
			Description: orgDesc,
			Metadata:    map[string]interface{}{fmt.Sprintf("key-%d", i): fmt.Sprintf("value-%d", i)},
		}

		err = repo.Save(context.Background(), org)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		orgMember := auth.OrgMember{
			OrgID:     orgID,
			MemberID:  memberID,
			Role:      auth.Editor,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = repo.AssignMembers(context.Background(), orgMember)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	}

	cases := []struct {
		desc         string
		memberID     string
		pageMetadata auth.PageMetadata
		size         uint64
		err          error
	}{
		{
			desc:     "retrieve orgs by member",
			memberID: memberID,
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:     "retrieve orgs by member filtered by metadata",
			memberID: memberID,
			pageMetadata: auth.PageMetadata{
				Offset:   0,
				Limit:    n,
				Total:    1,
				Metadata: map[string]interface{}{"key-1": "value-1"},
			},
			size: 1,
			err:  nil,
		},
		{
			desc:     "retrieve orgs by member filter by name",
			memberID: memberID,
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  1,
				Name:   orgName + "-1",
			},
			size: 1,
			err:  nil,
		},
		{
			desc:     "retrieve orgs by member filter by part of the name",
			memberID: memberID,
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  n,
				Name:   orgName,
			},
			size: n,
			err:  nil,
		},
		{
			desc:     "retrieve orgs by member with unknown member id",
			memberID: unknownID,
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			size: 0,
			err:  nil,
		},
		{
			desc:     "retrieve orgs by member with invalid member id",
			memberID: invalidID,
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			size: 0,
			err:  auth.ErrFailedToRetrieveOrgsByMember,
		},
		{
			desc:     "retrieve orgs by member without member id",
			memberID: "",
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			size: 0,
			err:  auth.ErrFailedToRetrieveOrgsByMember,
		},
	}

	for desc, tc := range cases {
		page, err := repo.RetrieveOrgsByMember(context.Background(), tc.memberID, tc.pageMetadata)
		size := len(page.Orgs)
		assert.Equal(t, tc.size, uint64(size), fmt.Sprintf("%v: expected size %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.pageMetadata.Total, page.Total, fmt.Sprintf("%v: expected total %d got %d\n", desc, tc.pageMetadata.Total, page.Total))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestAssignMembers(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewOrgRepo(dbMiddleware)

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

	err = repo.Save(context.Background(), org)
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
		err := repo.AssignMembers(context.Background(), tc.orgMembers...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUnassignMembers(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewOrgRepo(dbMiddleware)

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

	err = repo.Save(context.Background(), org)
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

	err = repo.AssignMembers(context.Background(), orgMembers...)
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
		err := repo.UnassignMembers(context.Background(), tc.orgID, tc.memberIDs...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

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

	orgMember := auth.OrgMember{
		OrgID:     org.ID,
		MemberID:  memberID,
		Role:      auth.Admin,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = repo.AssignMembers(context.Background(), orgMember)
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
		role, _ := repo.RetrieveRole(context.Background(), tc.memberID, tc.orgID)
		require.Equal(t, tc.role, role, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.role, role))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateMembers(t *testing.T) {
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

	orgMember := auth.OrgMember{
		OrgID:     org.ID,
		MemberID:  memberID,
		Role:      auth.Editor,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = repo.AssignMembers(context.Background(), orgMember)
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
		err := repo.UpdateMembers(context.Background(), tc.orgMember)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveMembersByOrg(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewOrgRepo(dbMiddleware)

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

	err = repo.Save(context.Background(), org)
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

	err = repo.AssignMembers(context.Background(), orgMembers...)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc         string
		orgID        string
		pageMetadata auth.PageMetadata
		size         uint64
		err          error
	}{
		{
			desc:  "retrieve members by org",
			orgID: orgID,
			pageMetadata: auth.PageMetadata{
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
			pageMetadata: auth.PageMetadata{
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
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			size: 0,
			err:  auth.ErrFailedToRetrieveMembersByOrg,
		},
		{
			desc:  "retrieve members by org without org id",
			orgID: "",
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			size: 0,
			err:  auth.ErrFailedToRetrieveMembersByOrg,
		},
	}

	for desc, tc := range cases {
		page, err := repo.RetrieveMembersByOrg(context.Background(), tc.orgID, tc.pageMetadata)
		size := len(page.OrgMembers)
		assert.Equal(t, tc.size, uint64(size), fmt.Sprintf("%v: expected size %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.pageMetadata.Total, page.Total, fmt.Sprintf("%v: expected total %d got %d\n", desc, tc.pageMetadata.Total, page.Total))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveAllMembersByOrg(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewOrgRepo(dbMiddleware)

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

	err = repo.Save(context.Background(), org)
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

	err = repo.AssignMembers(context.Background(), orgMembers...)
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
		page, err := repo.RetrieveAllMembersByOrg(context.Background())
		size := len(page)
		assert.Equal(t, tc.size, uint64(size), fmt.Sprintf("%v: expected size %v got %v\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
