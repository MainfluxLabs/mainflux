package postgres_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/things/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	maxDescSize = 1024
	groupName   = "test-group"
	description = "description"
	n           = uint64(5)
	profileName = "test-profile"
	thingName   = "test-thing"
)

var (
	invalidDesc = strings.Repeat("m", maxDescSize+1)
	metadata    = things.Metadata{
		"field": "value",
	}
)

func TestSaveGroup(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)
	orgID := generateUUID(t)
	grID := generateUUID(t)

	cases := []struct {
		desc  string
		group things.Group
		err   error
	}{
		{
			desc: "save new group",
			group: things.Group{
				ID:    grID,
				OrgID: orgID,
				Name:  groupName,
			},
			err: nil,
		},
		{
			desc: "save new group with existing name",
			group: things.Group{
				ID:    grID,
				OrgID: orgID,
				Name:  groupName,
			},
			err: errors.ErrConflict,
		},
		{
			desc: "save group with invalid name",
			group: things.Group{
				ID:    grID,
				OrgID: orgID,
				Name:  invalidName,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "save group with invalid description",
			group: things.Group{
				ID:          grID,
				OrgID:       orgID,
				Name:        groupName,
				Description: invalidDesc,
			},
			err: errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		_, err := groupRepo.Save(context.Background(), tc.group)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveGroupByID(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)
	orgID := generateUUID(t)

	group1 := things.Group{
		ID:    generateUUID(t),
		Name:  fmt.Sprintf("%s-%d", groupName, 1),
		OrgID: orgID,
	}

	_, err := groupRepo.Save(context.Background(), group1)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	retrieved, err := groupRepo.RetrieveByID(context.Background(), group1.ID)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	assert.True(t, retrieved.ID == group1.ID, fmt.Sprintf("Save group, ID: expected %s got %s\n", group1.ID, retrieved.ID))

	// Round to milliseconds as otherwise saving and retriving from DB
	// adds rounding error.
	creationTime := time.Now().UTC().Round(time.Millisecond)
	group2 := things.Group{
		ID:          generateUUID(t),
		Name:        fmt.Sprintf("%s-%d", groupName, 2),
		OrgID:       orgID,
		CreatedAt:   creationTime,
		UpdatedAt:   creationTime,
		Description: description,
		Metadata:    metadata,
	}

	_, err = groupRepo.Save(context.Background(), group2)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	retrieved, err = groupRepo.RetrieveByID(context.Background(), group2.ID)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	assert.True(t, retrieved.ID == group2.ID, fmt.Sprintf("Save group, ID: expected %s got %s\n", group2.ID, retrieved.ID))
	assert.True(t, retrieved.CreatedAt.Equal(creationTime), fmt.Sprintf("Save group, CreatedAt: expected %s got %s\n", creationTime, retrieved.CreatedAt))
	assert.True(t, retrieved.UpdatedAt.Equal(creationTime), fmt.Sprintf("Save group, UpdatedAt: expected %s got %s\n", creationTime, retrieved.UpdatedAt))
	assert.True(t, retrieved.Description == description, fmt.Sprintf("Save group, Description: expected %v got %v\n", retrieved.Description, description))

	retrieved, err = groupRepo.RetrieveByID(context.Background(), generateUUID(t))
	assert.True(t, errors.Contains(err, errors.ErrNotFound), fmt.Sprintf("Retrieve group: expected %s got %s\n", errors.ErrNotFound, err))
}

func TestUpdateGroup(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)

	orgID := generateUUID(t)
	groupID := generateUUID(t)
	wrongUid := generateUUID(t)

	creationTime := time.Now().UTC()
	updateTime := time.Now().UTC()

	group := things.Group{
		ID:          groupID,
		Name:        groupName,
		OrgID:       orgID,
		CreatedAt:   creationTime,
		UpdatedAt:   creationTime,
		Description: description,
		Metadata:    metadata,
	}

	_, err := groupRepo.Save(context.Background(), group)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	retrieved, err := groupRepo.RetrieveByID(context.Background(), group.ID)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	cases := []struct {
		desc          string
		groupUpdate   things.Group
		groupExpected things.Group
		err           error
	}{
		{
			desc: "update group for existing id",
			groupUpdate: things.Group{
				ID:        groupID,
				Name:      groupName,
				UpdatedAt: updateTime,
				Metadata:  things.Metadata{"admin": "false"},
			},
			groupExpected: things.Group{
				Name:      groupName,
				UpdatedAt: updateTime,
				Metadata:  things.Metadata{"admin": "false"},
				CreatedAt: retrieved.CreatedAt,
				ID:        retrieved.ID,
			},
			err: nil,
		},
		{
			desc: "update group for non-existing id",
			groupUpdate: things.Group{
				ID:   wrongUid,
				Name: fmt.Sprintf("%s-%d", groupName, 2),
			},
			err: errors.ErrUpdateEntity,
		},
		{
			desc: "update group for invalid name",
			groupUpdate: things.Group{
				ID:   groupID,
				Name: invalidName,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "update group for invalid description",
			groupUpdate: things.Group{
				ID:          groupID,
				Description: invalidDesc,
			},
			err: errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		updated, err := groupRepo.Update(context.Background(), tc.groupUpdate)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.desc == "update group for existing id" {
			assert.True(t, updated.Name == tc.groupExpected.Name, fmt.Sprintf("%s:Name: expected %s got %s\n", tc.desc, tc.groupExpected.Name, updated.Name))
			assert.True(t, updated.Metadata["admin"] == tc.groupExpected.Metadata["admin"], fmt.Sprintf("%s:Level: expected %d got %d\n", tc.desc, tc.groupExpected.Metadata["admin"], updated.Metadata["admin"]))
		}
	}
}

func TestRemoveGroup(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)
	thingRepo := postgres.NewThingRepository(dbMiddleware)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	orgID := generateUUID(t)
	creationTime := time.Now().UTC()
	group1 := things.Group{
		ID:        generateUUID(t),
		Name:      fmt.Sprintf("%s-%d", groupName, 1),
		OrgID:     orgID,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
	}

	creationTime = time.Now().UTC()
	group2 := things.Group{
		ID:        generateUUID(t),
		Name:      fmt.Sprintf("%s-%d", groupName, 2),
		OrgID:     orgID,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
	}

	thID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("thing id create unexpected error: %s", err))
	key, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	group1, err = groupRepo.Save(context.Background(), group1)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	group2, err = groupRepo.Save(context.Background(), group2)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	pr, err := profileRepo.Save(context.Background(), things.Profile{
		ID:      thID,
		GroupID: group1.ID,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	_, err = thingRepo.Save(context.Background(), things.Thing{
		ID:        thID,
		ProfileID: pr[0].ID,
		GroupID:   group1.ID,
		Key:       key,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	grIDs := []string{group1.ID, group2.ID}

	err = groupRepo.Remove(context.Background(), grIDs...)
	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("delete non empty groups: expected %v got %v\n", nil, err))
}

func TestRetrieveByIDs(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)

	orgID := generateUUID(t)
	metadata := things.Metadata{
		"field": "value",
	}
	wrongMeta := things.Metadata{
		"wrong": "wrong",
	}

	malformedIDs := []string{"malformed1", "malformed2"}
	metaNum := uint64(3)
	offset := uint64(1)
	var ids []string
	for i := uint64(0); i < n; i++ {
		suffix := i + 1
		creationTime := time.Now().UTC()
		group := things.Group{
			ID:        fmt.Sprintf("%s%012d", prefixID, suffix),
			Name:      fmt.Sprintf("%s-%d", groupName, suffix),
			OrgID:     orgID,
			CreatedAt: creationTime,
			UpdatedAt: creationTime,
		}
		ids = append(ids, group.ID)
		// Create Groups with metadata.
		if i < metaNum {
			group.Metadata = metadata
		}

		_, err := groupRepo.Save(context.Background(), group)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	cases := map[string]struct {
		size         uint64
		ids          []string
		pageMetadata apiutil.PageMetadata
	}{
		"retrieve all groups": {
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  idOrder,
				Dir:    descDir,
			},
			size: n,
			ids:  ids,
		},
		"retrieve groups without ids": {
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Dir:    descDir,
				Order:  idOrder,
			},
			size: 0,
			ids:  []string{},
		},
		"retrieve groups with malformed ids": {
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Dir:    descDir,
				Order:  idOrder,
			},
			size: 0,
			ids:  malformedIDs,
		},

		"retrieve all groups by IDs without limit": {
			pageMetadata: apiutil.PageMetadata{
				Limit: 0,
				Dir:   descDir,
				Order: idOrder,
			},
			size: n,
			ids:  ids,
		},
		"retrieve subset of groups by IDs": {
			pageMetadata: apiutil.PageMetadata{
				Offset: offset,
				Limit:  n,
				Dir:    descDir,
				Order:  idOrder,
			},
			size: n - offset,
			ids:  ids,
		},
		"retrieve groups by IDs with existing name": {
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Name:   "test-group-5",
				Dir:    descDir,
				Order:  idOrder,
			},
			size: 1,
			ids:  ids,
		},
		"retrieve groups by IDs with non-existing name": {
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Name:   "wrong",
				Dir:    descDir,
				Order:  idOrder,
			},
			size: 0,
			ids:  ids,
		},
		"retrieve groups by IDs with existing metadata": {
			pageMetadata: apiutil.PageMetadata{
				Offset:   0,
				Limit:    n,
				Metadata: metadata,
				Dir:      descDir,
				Order:    idOrder,
			},
			size: metaNum,
			ids:  ids,
		},
		"retrieve groups by IDs with non-existing metadata": {
			pageMetadata: apiutil.PageMetadata{
				Offset:   0,
				Limit:    n,
				Metadata: wrongMeta,
				Dir:      descDir,
				Order:    idOrder,
			},
			ids: ids,
		},
		"retrieve groups by IDs sorted by name ascendant": {
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  nameOrder,
				Dir:    ascDir,
			},
			size: n,
			ids:  ids,
		},
		"retrieve groups by IDs sorted by name descendent": {
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  nameOrder,
				Dir:    descDir,
			},
			size: n,
			ids:  ids,
		},
	}

	for desc, tc := range cases {
		page, _ := groupRepo.RetrieveByIDs(context.Background(), tc.ids, tc.pageMetadata)
		size := len(page.Groups)
		assert.Equal(t, tc.size, uint64(size), fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))

		// Check if Groups list have been sorted properly
		testSortGroups(t, tc.pageMetadata, page.Groups)
	}
}

func TestRetrieveAllGroups(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)

	err := cleanTestTable(context.Background(), "groups", dbMiddleware)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	groupRepo := postgres.NewGroupRepository(dbMiddleware)

	orgID := generateUUID(t)
	metadata := apiutil.PageMetadata{
		Metadata: things.Metadata{
			"field": "value",
		},
	}

	metaNum := uint64(3)
	for i := uint64(0); i < n; i++ {
		creationTime := time.Now().UTC()
		group := things.Group{
			ID:        generateUUID(t),
			Name:      fmt.Sprintf("%s-%d", groupName, i),
			OrgID:     orgID,
			CreatedAt: creationTime,
			UpdatedAt: creationTime,
		}
		// Create Groups with metadata.
		if i < metaNum {
			group.Metadata = metadata.Metadata
		}

		_, err = groupRepo.Save(context.Background(), group)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	cases := map[string]struct {
		Size uint64
	}{
		"retrieve all groups": {
			Size: n,
		},
	}

	for desc, tc := range cases {
		gr, err := groupRepo.RetrieveAll(context.Background())
		size := len(gr)
		assert.Equal(t, tc.Size, uint64(size), fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.Size, size))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func generateUUID(t *testing.T) string {
	id, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	return id
}

func createGroup(t *testing.T, dbMiddleware dbutil.Database) things.Group {
	groupRepo := postgres.NewGroupRepository(dbMiddleware)

	grID := generateUUID(t)
	orgID := generateUUID(t)

	group, err := groupRepo.Save(context.Background(), things.Group{
		ID:    grID,
		OrgID: orgID,
		Name:  groupName,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	return group
}

func testSortGroups(t *testing.T, pm apiutil.PageMetadata, grs []things.Group) {
	if len(grs) < 1 {
		return
	}

	switch pm.Order {
	case "name":
		current := grs[0]
		for _, res := range grs {
			if pm.Dir == "asc" {
				assert.GreaterOrEqual(t, res.Name, current.Name)
			}
			if pm.Dir == "desc" {
				assert.GreaterOrEqual(t, current.Name, res.Name)
			}
			current = res
		}
	default:
		break
	}
}
