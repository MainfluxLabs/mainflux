package postgres_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/things/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	maxNameSize1 = 254
	maxDescSize  = 1024
	groupName    = "Mainflux"
	description  = "description"
)

var (
	invalidDesc = strings.Repeat("m", maxDescSize+1)
	metadata    = things.GroupMetadata{
		"admin": "true",
	}
)

func generateGroupID(t *testing.T) string {
	grpID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	return grpID
}

func TestGroupSave(t *testing.T) {
	t.Cleanup(func() { cleanUp(t) })
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepo(dbMiddleware)

	usrID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	grpID := generateGroupID(t)

	cases := []struct {
		desc  string
		group things.Group
		err   error
	}{
		{
			desc: "create new group",
			group: things.Group{
				ID:      grpID,
				OwnerID: usrID,
				Name:    groupName,
			},
			err: nil,
		},
		{
			desc: "create new group with existing name",
			group: things.Group{
				ID:      grpID,
				OwnerID: usrID,
				Name:    groupName,
			},
			err: errors.ErrConflict,
		},
		{
			desc: "create group with invalid name",
			group: things.Group{
				ID:      grpID,
				OwnerID: usrID,
				Name:    invalidName,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "create group with invalid description",
			group: things.Group{
				ID:          grpID,
				OwnerID:     usrID,
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

func TestGroupRetrieveByID(t *testing.T) {
	t.Cleanup(func() { cleanUp(t) })
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepo(dbMiddleware)

	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	require.Nil(t, err, fmt.Sprintf("group id unexpected error: %s", err))
	group1 := things.Group{
		ID:      generateGroupID(t),
		Name:    groupName + "TestGroupRetrieveByID1",
		OwnerID: uid,
	}

	_, err = groupRepo.Save(context.Background(), group1)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	retrieved, err := groupRepo.RetrieveByID(context.Background(), group1.ID)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	assert.True(t, retrieved.ID == group1.ID, fmt.Sprintf("Save group, ID: expected %s got %s\n", group1.ID, retrieved.ID))

	// Round to milliseconds as otherwise saving and retriving from DB
	// adds rounding error.
	creationTime := time.Now().UTC().Round(time.Millisecond)
	group2 := things.Group{
		ID:          generateGroupID(t),
		Name:        groupName + "TestGroupRetrieveByID",
		OwnerID:     uid,
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

	retrieved, err = groupRepo.RetrieveByID(context.Background(), generateGroupID(t))
	assert.True(t, errors.Contains(err, errors.ErrNotFound), fmt.Sprintf("Retrieve group: expected %s got %s\n", errors.ErrNotFound, err))
}

func TestGroupUpdate(t *testing.T) {
	t.Cleanup(func() { cleanUp(t) })
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepo(dbMiddleware)

	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	wrongUid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	creationTime := time.Now().UTC()
	updateTime := time.Now().UTC()
	groupID := generateGroupID(t)

	group := things.Group{
		ID:          groupID,
		Name:        groupName + "TestGroupUpdate",
		OwnerID:     uid,
		CreatedAt:   creationTime,
		UpdatedAt:   creationTime,
		Description: description,
		Metadata:    metadata,
	}

	_, err = groupRepo.Save(context.Background(), group)
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
				Name:      groupName + "Updated",
				UpdatedAt: updateTime,
				Metadata:  things.GroupMetadata{"admin": "false"},
			},
			groupExpected: things.Group{
				Name:      groupName + "Updated",
				UpdatedAt: updateTime,
				Metadata:  things.GroupMetadata{"admin": "false"},
				CreatedAt: retrieved.CreatedAt,
				ID:        retrieved.ID,
			},
			err: nil,
		},
		{
			desc: "update group for non-existing id",
			groupUpdate: things.Group{
				ID:   wrongUid,
				Name: groupName + "-2",
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

func TestGroupRemove(t *testing.T) {
	t.Cleanup(func() { cleanUp(t) })
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepo(dbMiddleware)

	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	creationTime := time.Now().UTC()
	group1 := things.Group{
		ID:        generateGroupID(t),
		Name:      groupName + "child1",
		OwnerID:   uid,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
	}

	creationTime = time.Now().UTC()
	group2 := things.Group{
		ID:        generateGroupID(t),
		Name:      groupName + "child2",
		OwnerID:   uid,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
	}

	group1, err = groupRepo.Save(context.Background(), group1)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	group2, err = groupRepo.Save(context.Background(), group2)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	thingID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("thing id create unexpected error: %s", err))

	err = groupRepo.AssignMember(context.Background(), group1.ID, thingID)
	require.Nil(t, err, fmt.Sprintf("thing assign got unexpected error: %s", err))

	err = groupRepo.Remove(context.Background(), group1.ID)
	assert.True(t, errors.Contains(err, things.ErrGroupNotEmpty), fmt.Sprintf("delete non empty group: expected %v got %v\n", things.ErrGroupNotEmpty, err))

	err = groupRepo.Remove(context.Background(), group2.ID)
	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("delete empty group: expected %v got %v\n", nil, err))
}

func TestRetrieveByOwner(t *testing.T) {
	t.Cleanup(func() { cleanUp(t) })
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepo(dbMiddleware)
	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	metadata := things.PageMetadata{
		Metadata: things.GroupMetadata{
			"field": "value",
		},
	}
	wrongMeta := things.PageMetadata{
		Metadata: things.GroupMetadata{
			"wrong": "wrong",
		},
	}

	metaNum := uint64(3)

	var n uint64 = 5
	for i := uint64(0); i < n; i++ {
		creationTime := time.Now().UTC()
		group := things.Group{
			ID:        generateGroupID(t),
			Name:      fmt.Sprintf("%s-%d", groupName, i),
			OwnerID:   uid,
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
		Size     uint64
		Metadata things.PageMetadata
	}{
		"retrieve all groups": {
			Metadata: things.PageMetadata{
				Total: n,
				Limit: n,
			},
			Size: n,
		},
		"retrieve groups with existing metadata": {
			Metadata: things.PageMetadata{
				Total:    metaNum,
				Limit:    n,
				Metadata: metadata.Metadata,
			},
			Size: metaNum,
		},
		"retrieve groups with non-existing metadata": {
			Metadata: things.PageMetadata{
				Total:    uint64(0),
				Limit:    n,
				Metadata: wrongMeta.Metadata,
			},
			Size: uint64(0),
		},
	}

	for desc, tc := range cases {
		page, err := groupRepo.RetrieveByOwner(context.Background(), uid, tc.Metadata)
		size := len(page.Groups)
		assert.Equal(t, tc.Size, uint64(size), fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.Size, size))
		assert.Equal(t, tc.Metadata.Total, page.Total, fmt.Sprintf("%s: expected total %d got %d\n", desc, tc.Metadata.Total, page.Total))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestRetrieveByIDs(t *testing.T) {
	t.Cleanup(func() { cleanUp(t) })
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepo(dbMiddleware)
	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	metadata := things.PageMetadata{
		Metadata: things.GroupMetadata{
			"field": "value",
		},
	}
	wrongMeta := things.PageMetadata{
		Metadata: things.GroupMetadata{
			"wrong": "wrong",
		},
	}
	malformedIDs := []string{"malformed1", "malformed2"}

	metaNum := uint64(3)

	var n uint64 = 5
	var ids []string
	for i := uint64(0); i < n; i++ {
		creationTime := time.Now().UTC()
		group := things.Group{
			ID:        generateGroupID(t),
			Name:      fmt.Sprintf("%s-%d", groupName, i),
			OwnerID:   uid,
			CreatedAt: creationTime,
			UpdatedAt: creationTime,
		}
		ids = append(ids, group.ID)
		// Create Groups with metadata.
		if i < metaNum {
			group.Metadata = metadata.Metadata
		}

		_, err = groupRepo.Save(context.Background(), group)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	cases := map[string]struct {
		Size     uint64
		IDs      []string
		Metadata things.PageMetadata
		err      error
	}{
		"retrieve all groups": {
			Metadata: things.PageMetadata{
				Offset: 0,
				Total:  n,
				Limit:  n,
			},
			Size: n,
			IDs:  ids,
			err:  nil,
		},
		"retrieve groups without ids": {
			Metadata: things.PageMetadata{
				Offset: 0,
				Total:  0,
				Limit:  n,
			},
			Size: 0,
			IDs:  []string{},
			err:  nil,
		},
		"retrieve groups with malformed ids": {
			Metadata: things.PageMetadata{
				Offset: 0,
				Total:  0,
				Limit:  n,
			},
			Size: 0,
			IDs:  malformedIDs,
			err:  errors.ErrRetrieveEntity,
		},
		"retrieve groups with non-existing metadata": {
			Metadata: wrongMeta,
			Size:     0,
			IDs:      ids,
			err:      errors.ErrRetrieveEntity,
		},
		"retrieve groups sorted by name ascendent": {
			Metadata: things.PageMetadata{
				Offset: 0,
				Total:  n,
				Limit:  n,
				Order:  "name",
				Dir:    "asc",
			},
			Size: n,
			IDs:  ids,
			err:  nil,
		},
		"retrieve groups sorted by name descendent": {
			Metadata: things.PageMetadata{
				Offset: 0,
				Total:  n,
				Limit:  n,
				Order:  "name",
				Dir:    "desc",
			},
			Size: n,
			IDs:  ids,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		page, err := groupRepo.RetrieveByIDs(context.Background(), tc.IDs, tc.Metadata)
		size := len(page.Groups)
		assert.Equal(t, tc.Size, uint64(size), fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.Size, size))
		assert.Equal(t, tc.Metadata.Total, page.Total, fmt.Sprintf("%s: expected total %d got %d\n", desc, tc.Metadata.Total, page.Total))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

		testSortGroups(t, tc.Metadata, page.Groups)
	}
}

func TestRetrieveAllGroups(t *testing.T) {
	t.Cleanup(func() { cleanUp(t) })
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepo(dbMiddleware)
	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	metadata := things.PageMetadata{
		Metadata: things.GroupMetadata{
			"field": "value",
		},
	}

	metaNum := uint64(3)

	var n uint64 = 5
	for i := uint64(0); i < n; i++ {
		creationTime := time.Now().UTC()
		group := things.Group{
			ID:        generateGroupID(t),
			Name:      fmt.Sprintf("%s-%d", groupName, i),
			OwnerID:   uid,
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

func TestAssign(t *testing.T) {
	t.Cleanup(func() { cleanUp(t) })
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepo(dbMiddleware)

	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	creationTime := time.Now().UTC()
	group := things.Group{
		ID:        generateGroupID(t),
		Name:      groupName + "Updated",
		OwnerID:   uid,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
	}

	pm := things.PageMetadata{
		Offset: 0,
		Limit:  10,
	}

	group, err = groupRepo.Save(context.Background(), group)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	mid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	err = groupRepo.AssignMember(context.Background(), group.ID, mid)
	require.Nil(t, err, fmt.Sprintf("member assign save unexpected error: %s", err))

	mp, err := groupRepo.RetrieveMembers(context.Background(), group.ID, pm)
	require.Nil(t, err, fmt.Sprintf("member assign save unexpected error: %s", err))
	assert.True(t, mp.Total == 1, fmt.Sprintf("retrieve members of a group: expected %d got %d\n", 1, mp.Total))

	err = groupRepo.AssignMember(context.Background(), group.ID, mid)
	assert.True(t, errors.Contains(err, things.ErrMemberAlreadyAssigned), fmt.Sprintf("assign member again: expected %v got %v\n", things.ErrMemberAlreadyAssigned, err))
}

func TestUnassign(t *testing.T) {
	t.Cleanup(func() { cleanUp(t) })
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepo(dbMiddleware)

	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	creationTime := time.Now().UTC()
	group := things.Group{
		ID:        generateGroupID(t),
		Name:      groupName + "Updated",
		OwnerID:   uid,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
	}

	pm := things.PageMetadata{
		Offset: 0,
		Limit:  10,
	}

	group, err = groupRepo.Save(context.Background(), group)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	mid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	err = groupRepo.AssignMember(context.Background(), group.ID, mid)
	require.Nil(t, err, fmt.Sprintf("member assign unexpected error: %s", err))

	mid, err = idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	err = groupRepo.AssignMember(context.Background(), group.ID, mid)
	require.Nil(t, err, fmt.Sprintf("member assign unexpected error: %s", err))

	mp, err := groupRepo.RetrieveMembers(context.Background(), group.ID, pm)
	require.Nil(t, err, fmt.Sprintf("member assign save unexpected error: %s", err))
	assert.True(t, mp.Total == 2, fmt.Sprintf("retrieve members of a group: expected %d got %d\n", 2, mp.Total))

	err = groupRepo.UnassignMember(context.Background(), group.ID, mid)
	require.Nil(t, err, fmt.Sprintf("member unassign save unexpected error: %s", err))

	mp, err = groupRepo.RetrieveMembers(context.Background(), group.ID, pm)
	require.Nil(t, err, fmt.Sprintf("members retrieve unexpected error: %s", err))
	assert.True(t, mp.Total == 1, fmt.Sprintf("retrieve members of a group: expected %d got %d\n", 1, mp.Total))
}

func cleanUp(t *testing.T) {
	_, err := db.Exec("delete from group_relations")
	require.Nil(t, err, fmt.Sprintf("clean relations unexpected error: %s", err))
	_, err = db.Exec("delete from groups")
	require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
}

func testSortGroups(t *testing.T, pm things.PageMetadata, groups []things.Group) {
	if len(groups) < 1 {
		return
	}

	switch pm.Order {
	case "name":
		current := groups[0]
		for _, res := range groups {
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
