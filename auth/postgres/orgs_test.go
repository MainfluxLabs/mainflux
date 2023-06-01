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
	orgName              = "test"
	orgDesc              = "test_description"
	invalidID            = "invalid"
	n                    = uint64(10)
	orgsTable            = "orgs"
	memberRelationsTable = "member_relations"
	groupRelationsTable  = "group_relations"
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

func TestRetrieveMemberships(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewOrgRepo(dbMiddleware)

	ownerID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	memberID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	unknownID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	member := auth.Member{
		ID:    memberID,
		Email: email,
		Role:  auth.EditorRole,
	}

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

		err = repo.AssignMembers(context.Background(), orgID, member)
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
			desc:     "retrieve memberships",
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
			desc:     "retrieve memberships filtered by metadata",
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
			desc:     "retrieve memberships filter by name",
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
			desc:     "retrieve memberships filter by part of the name",
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
			desc:     "retrieve memberships with unknown member id",
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
			desc:     "retrieve memberships with invalid member id",
			memberID: invalidID,
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			size: 0,
			err:  auth.ErrFailedToRetrieveMembership,
		},
		{
			desc:     "retrieve memberships without member id",
			memberID: "",
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			size: 0,
			err:  auth.ErrFailedToRetrieveMembership,
		},
	}

	for desc, tc := range cases {
		page, err := repo.RetrieveMemberships(context.Background(), tc.memberID, tc.pageMetadata)
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

	var members []auth.Member
	for i := uint64(0); i < n; i++ {
		memberID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		email := fmt.Sprintf("member%d@email.com", i)
		member := auth.Member{
			ID:    memberID,
			Email: email,
			Role:  auth.EditorRole,
		}

		members = append(members, member)
	}

	cases := []struct {
		desc    string
		orgID   string
		members []auth.Member
		err     error
	}{
		{
			desc:    "assign members to org",
			orgID:   orgID,
			members: members,
			err:     nil,
		},
		{
			desc:    "assign already assigned members to org",
			orgID:   orgID,
			members: members,
			err:     auth.ErrOrgMemberAlreadyAssigned,
		},
		{
			desc:    "assign members to org with invalid org id",
			orgID:   invalidID,
			members: members,
			err:     errors.ErrMalformedEntity,
		},
		{
			desc:    "assign members to org without org id",
			orgID:   "",
			members: members,
			err:     errors.ErrMalformedEntity,
		},
		{
			desc:    "assign members to org with empty members",
			orgID:   orgID,
			members: []auth.Member{},
			err:     nil,
		},
		{
			desc:    "assign members to org with invalid member id",
			orgID:   orgID,
			members: []auth.Member{{ID: invalidID}},
			err:     errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		err := repo.AssignMembers(context.Background(), tc.orgID, tc.members...)
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

	var members []auth.Member
	var memberIDs []string
	for i := uint64(0); i < n; i++ {
		memberID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		email := fmt.Sprintf("member%d@email.com", i)
		member := auth.Member{
			ID:    memberID,
			Email: email,
			Role:  auth.ViewerRole,
		}

		members = append(members, member)
		memberIDs = append(memberIDs, memberID)
	}

	err = repo.AssignMembers(context.Background(), orgID, members...)
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
			desc:   "update role with invalid org id",
			orgID:  invalidID,
			member: member,
			err:    errors.ErrMalformedEntity,
		}, {
			desc:   "update role with unknown org id",
			orgID:  unknownID,
			member: member,
			err:    errors.ErrNotFound,
		}, {
			desc:   "update role without org id",
			orgID:  "",
			member: member,
			err:    errors.ErrMalformedEntity,
		}, {
			desc:   "update role with invalid member id",
			orgID:  org.ID,
			member: auth.Member{ID: invalidID},
			err:    errors.ErrMalformedEntity,
		}, {
			desc:   "update role with unknown member id",
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

func TestRetrieveMembers(t *testing.T) {
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

	var members []auth.Member
	for i := uint64(0); i < n; i++ {
		memberID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		email := fmt.Sprintf("member%d@email.com", i)
		member := auth.Member{
			ID:    memberID,
			Email: email,
			Role:  auth.EditorRole,
		}

		members = append(members, member)
	}

	err = repo.AssignMembers(context.Background(), orgID, members...)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc         string
		orgID        string
		pageMetadata auth.PageMetadata
		size         uint64
		err          error
	}{
		{
			desc:  "retrieve org members",
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
			desc:  "retrieve org members with unknown org id",
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
			desc:  "retrieve org members with invalid org id",
			orgID: invalidID,
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			size: 0,
			err:  auth.ErrFailedToRetrieveMembers,
		},
		{
			desc:  "retrieve org members without org id",
			orgID: "",
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			size: 0,
			err:  auth.ErrFailedToRetrieveMembers,
		},
	}

	for desc, tc := range cases {
		page, err := repo.RetrieveMembers(context.Background(), tc.orgID, tc.pageMetadata)
		size := len(page.Members)
		assert.Equal(t, tc.size, uint64(size), fmt.Sprintf("%v: expected size %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.pageMetadata.Total, page.Total, fmt.Sprintf("%v: expected total %d got %d\n", desc, tc.pageMetadata.Total, page.Total))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestAssignGroups(t *testing.T) {
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

	var groupIDs []string
	for i := uint64(0); i < n; i++ {
		groupID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		groupIDs = append(groupIDs, groupID)
	}

	cases := []struct {
		desc     string
		orgID    string
		groupIDs []string
		err      error
	}{
		{
			desc:     "assign groups to org",
			orgID:    orgID,
			groupIDs: groupIDs,
			err:      nil,
		},
		{
			desc:     "assign already assigned groups to org",
			orgID:    orgID,
			groupIDs: groupIDs,
			err:      auth.ErrOrgMemberAlreadyAssigned,
		},
		{
			desc:     "assign groups to org with invalid org id",
			orgID:    invalidID,
			groupIDs: groupIDs,
			err:      errors.ErrMalformedEntity,
		},
		{
			desc:     "assign groups to org without org id",
			orgID:    "",
			groupIDs: groupIDs,
			err:      errors.ErrMalformedEntity,
		},
		{
			desc:     "assign groups to org with unknown org id",
			orgID:    unknownID,
			groupIDs: groupIDs,
			err:      nil,
		},
		{
			desc:     "assign groups to org without group ids",
			orgID:    orgID,
			groupIDs: []string{},
			err:      nil,
		},
		{
			desc:     "assign groups to org with invalid group id",
			orgID:    orgID,
			groupIDs: []string{invalidID},
			err:      errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		err := repo.AssignGroups(context.Background(), tc.orgID, tc.groupIDs...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUnassignGroups(t *testing.T) {
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

	var groupIDs []string
	for i := uint64(0); i < n; i++ {
		groupID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		groupIDs = append(groupIDs, groupID)
	}

	err = repo.AssignGroups(context.Background(), orgID, groupIDs...)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc     string
		orgID    string
		groupIDs []string
		err      error
	}{
		{
			desc:     "unassign groups from org with invalid org id",
			orgID:    invalidID,
			groupIDs: groupIDs,
			err:      errors.ErrMalformedEntity,
		},
		{
			desc:     "unassign groups from org without org id",
			orgID:    "",
			groupIDs: groupIDs,
			err:      errors.ErrMalformedEntity,
		},
		{
			desc:     "unassign empty group list from org",
			orgID:    orgID,
			groupIDs: []string{},
			err:      nil,
		},
		{
			desc:     "unassign groups from org with invalid group id",
			orgID:    orgID,
			groupIDs: []string{invalidID},
			err:      errors.ErrMalformedEntity,
		},
		{
			desc:     "unassign groups from org with unknown org id",
			orgID:    unknownID,
			groupIDs: groupIDs,
			err:      nil,
		},
		{
			desc:     "unassign groups from org",
			orgID:    orgID,
			groupIDs: groupIDs,
			err:      nil,
		},
		{
			desc:     "unassign already unassigned groups from org",
			orgID:    orgID,
			groupIDs: groupIDs,
			err:      nil,
		},
	}

	for _, tc := range cases {
		err := repo.UnassignGroups(context.Background(), tc.orgID, tc.groupIDs...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveGroups(t *testing.T) {
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

	var groupIDs []string
	for i := uint64(0); i < n; i++ {
		groupID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		groupIDs = append(groupIDs, groupID)
	}

	err = repo.AssignGroups(context.Background(), orgID, groupIDs...)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc         string
		orgID        string
		pageMetadata auth.PageMetadata
		groupIDs     []string
		size         uint64
		err          error
	}{
		{
			desc:  "retrieve groups",
			orgID: orgID,
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  n,
			},
			groupIDs: groupIDs,
			size:     n,
			err:      nil,
		},
		{
			desc:  "retrieve groups with invalid org id",
			orgID: invalidID,
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			groupIDs: nil,
			size:     0,
			err:      errors.ErrRetrieveEntity,
		},
		{
			desc:  "retrieve groups without org id",
			orgID: "",
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			groupIDs: nil,
			size:     0,
			err:      errors.ErrRetrieveEntity,
		},
		{
			desc:  "retrieve groups with unknown org id",
			orgID: unknownID,
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
			},
			groupIDs: nil,
			size:     0,
			err:      nil,
		},
	}

	for desc, tc := range cases {
		page, err := repo.RetrieveGroups(context.Background(), tc.orgID, tc.pageMetadata)
		size := len(page.GroupIDs)
		assert.Equal(t, tc.pageMetadata.Total, uint64(size), fmt.Sprintf("%v: expected size %v got %v\n", desc, tc.pageMetadata.Total, size))
		assert.Equal(t, tc.groupIDs, page.GroupIDs, fmt.Sprintf("%v: expected size %v got %v\n", desc, tc.groupIDs, page.GroupIDs))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveByGroupID(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewOrgRepo(dbMiddleware)

	ownerID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	groupID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	unknownID, err := idProvider.ID()
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

	err = repo.Save(context.Background(), org)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	err = repo.AssignGroups(context.Background(), orgID, groupID)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc    string
		groupID string
		org     auth.Org
		err     error
	}{
		{
			desc:    "retrieve orgs by group",
			groupID: groupID,
			org:     org,
			err:     nil,
		},
		{
			desc:    "retrieve orgs by invalid group id",
			groupID: invalidID,
			org:     auth.Org{},
			err:     errors.ErrRetrieveEntity,
		},
		{
			desc:    "retrieve orgs by empty group id",
			groupID: "",
			org:     auth.Org{},
			err:     errors.ErrRetrieveEntity,
		},
		{
			desc:    "retrieve orgs by unknown group id",
			groupID: unknownID,
			org:     auth.Org{},
			err:     nil,
		},
	}

	for _, tc := range cases {
		or, err := repo.RetrieveByGroupID(context.Background(), tc.groupID)
		assert.Equal(t, tc.org, or, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.org, or))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveAllMemberRelations(t *testing.T) {
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

	for i := uint64(0); i < n; i++ {
		memberID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		member := auth.Member{
			ID:   memberID,
			Role: auth.EditorRole,
		}

		err = repo.AssignMembers(context.Background(), org.ID, member)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	}

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
		page, err := repo.RetrieveAllMemberRelations(context.Background())
		size := len(page)
		assert.Equal(t, tc.size, uint64(size), fmt.Sprintf("%v: expected size %v got %v\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveAllGroupRelations(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewOrgRepo(dbMiddleware)

	_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", groupRelationsTable))
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

	for i := uint64(0); i < n; i++ {
		groupID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		err = repo.AssignGroups(context.Background(), org.ID, groupID)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	}

	cases := []struct {
		desc string
		size uint64
		err  error
	}{
		{
			desc: "retrieve all group relations",
			size: n,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		page, err := repo.RetrieveAllGroupRelations(context.Background())
		size := len(page)
		assert.Equal(t, tc.size, uint64(size), fmt.Sprintf("%v: expected size %v got %v\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
