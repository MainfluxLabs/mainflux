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
	n            = uint64(5)
	invalid      = "invalid"
	channelName  = "channel"
)

var (
	invalidDesc = strings.Repeat("m", maxDescSize+1)
	metadata    = things.GroupMetadata{
		"field": "value",
	}
)

func TestGroupSave(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)

	owID := generateUUID(t)
	orgID := generateUUID(t)
	grID := generateUUID(t)

	cases := []struct {
		desc  string
		group things.Group
		err   error
	}{
		{
			desc: "create new group",
			group: things.Group{
				ID:      grID,
				OwnerID: owID,
				OrgID:   orgID,
				Name:    groupName,
			},
			err: nil,
		},
		{
			desc: "create new group with existing name",
			group: things.Group{
				ID:      grID,
				OwnerID: owID,
				OrgID:   orgID,
				Name:    groupName,
			},
			err: errors.ErrConflict,
		},
		{
			desc: "create group with invalid name",
			group: things.Group{
				ID:      grID,
				OwnerID: owID,
				OrgID:   orgID,
				Name:    invalidName,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "create group with invalid description",
			group: things.Group{
				ID:          grID,
				OwnerID:     owID,
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

func TestGroupRetrieveByID(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)

	owID := generateUUID(t)
	orgID := generateUUID(t)

	group1 := things.Group{
		ID:      generateUUID(t),
		Name:    groupName + "TestGroupRetrieveByID1",
		OwnerID: owID,
		OrgID:   orgID,
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
		Name:        groupName + "TestGroupRetrieveByID",
		OwnerID:     owID,
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

func TestGroupUpdate(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)

	owID := generateUUID(t)
	orgID := generateUUID(t)

	wrongUid := generateUUID(t)

	creationTime := time.Now().UTC()
	updateTime := time.Now().UTC()
	groupID := generateUUID(t)

	group := things.Group{
		ID:          groupID,
		Name:        groupName + "TestGroupUpdate",
		OwnerID:     owID,
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
				Metadata:  things.GroupMetadata{"admin": "false"},
			},
			groupExpected: things.Group{
				Name:      groupName,
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
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	owID := generateUUID(t)
	orgID := generateUUID(t)

	creationTime := time.Now().UTC()
	group1 := things.Group{
		ID:        generateUUID(t),
		Name:      groupName + "child1",
		OwnerID:   owID,
		OrgID:     orgID,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
	}

	creationTime = time.Now().UTC()
	group2 := things.Group{
		ID:        generateUUID(t),
		Name:      groupName + "child2",
		OwnerID:   owID,
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

	_, err = thingRepo.Save(context.Background(), things.Thing{
		ID:      thID,
		OwnerID: owID,
		GroupID: group1.ID,
		Key:     key,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	grIDs := []string{group1.ID, group2.ID}

	err = groupRepo.Remove(context.Background(), grIDs...)
	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("delete non empty groups: expected %v got %v\n", nil, err))
}

func TestRetrieveByOwner(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)

	owID := generateUUID(t)
	orgID := generateUUID(t)

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
	for i := uint64(0); i < n; i++ {
		creationTime := time.Now().UTC()
		group := things.Group{
			ID:        generateUUID(t),
			Name:      fmt.Sprintf("%s-%d", groupName, i),
			OwnerID:   owID,
			OrgID:     orgID,
			CreatedAt: creationTime,
			UpdatedAt: creationTime,
		}
		// Create Groups with metadata.
		if i < metaNum {
			group.Metadata = metadata.Metadata
		}

		_, err := groupRepo.Save(context.Background(), group)
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
		page, err := groupRepo.RetrieveByOwner(context.Background(), owID, "", tc.Metadata)
		size := len(page.Groups)
		assert.Equal(t, tc.Size, uint64(size), fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.Size, size))
		assert.Equal(t, tc.Metadata.Total, page.Total, fmt.Sprintf("%s: expected total %d got %d\n", desc, tc.Metadata.Total, page.Total))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestRetrieveByIDs(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)

	owID := generateUUID(t)
	orgID := generateUUID(t)

	metadata := things.PageMetadata{
		Metadata: things.GroupMetadata{
			"field": "value",
		},
	}

	malformedIDs := []string{"malformed1", "malformed2"}
	metaNum := uint64(3)
	var ids []string
	for i := uint64(0); i < n; i++ {
		creationTime := time.Now().UTC()
		group := things.Group{
			ID:        generateUUID(t),
			Name:      fmt.Sprintf("%s-%d", groupName, i),
			OwnerID:   owID,
			OrgID:     orgID,
			CreatedAt: creationTime,
			UpdatedAt: creationTime,
		}
		ids = append(ids, group.ID)
		// Create Groups with metadata.
		if i < metaNum {
			group.Metadata = metadata.Metadata
		}

		_, err := groupRepo.Save(context.Background(), group)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	cases := map[string]struct {
		Size     uint64
		IDs      []string
		Metadata things.PageMetadata
		err      error
	}{
		"retrieve all groups": {
			Size: n,
			IDs:  ids,
			err:  nil,
		},
		"retrieve groups without ids": {
			Size: 0,
			IDs:  []string{},
			err:  nil,
		},
		"retrieve groups with malformed ids": {
			Size: 0,
			IDs:  malformedIDs,
			err:  errors.ErrRetrieveEntity,
		},
	}

	for desc, tc := range cases {
		page, err := groupRepo.RetrieveByIDs(context.Background(), tc.IDs)
		size := len(page.Groups)
		assert.Equal(t, tc.Size, uint64(size), fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.Size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRetrieveAllGroups(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)

	err := cleanTestTable(context.Background(), "groups", dbMiddleware)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	groupRepo := postgres.NewGroupRepository(dbMiddleware)

	owID := generateUUID(t)
	orgID := generateUUID(t)

	metadata := things.PageMetadata{
		Metadata: things.GroupMetadata{
			"field": "value",
		},
	}

	metaNum := uint64(3)
	for i := uint64(0); i < n; i++ {
		creationTime := time.Now().UTC()
		group := things.Group{
			ID:        generateUUID(t),
			Name:      fmt.Sprintf("%s-%d", groupName, i),
			OwnerID:   owID,
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

func TestRetrieveGroupThings(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	owID := generateUUID(t)
	orgID := generateUUID(t)

	creationTime := time.Now().UTC()
	group := things.Group{
		ID:        generateUUID(t),
		Name:      groupName,
		OwnerID:   owID,
		OrgID:     orgID,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
	}
	group2 := things.Group{
		ID:        generateUUID(t),
		Name:      groupName,
		OwnerID:   owID,
		OrgID:     orgID,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
	}

	group, err := groupRepo.Save(context.Background(), group)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))
	group2, err = groupRepo.Save(context.Background(), group2)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	thID1, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thID2, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thID3, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	ths := []things.Thing{
		{
			ID:       thID1,
			OwnerID:  owID,
			GroupID:  group.ID,
			Key:      "key1",
			Metadata: map[string]interface{}{},
		},
		{
			ID:       thID2,
			OwnerID:  owID,
			GroupID:  group.ID,
			Key:      "key2",
			Metadata: map[string]interface{}{},
		},
		{
			ID:       thID3,
			OwnerID:  owID,
			GroupID:  group2.ID,
			Key:      "key3",
			Metadata: map[string]interface{}{},
		},
	}
	_, err = thingRepo.Save(context.Background(), ths...)
	require.Nil(t, err, fmt.Sprintf("channel save got unexpected error: %s", err))

	cases := map[string]struct {
		pagemeta things.PageMetadata
		groupID  string
		things   []things.Thing
		err      error
	}{
		"retrieve group things": {
			pagemeta: things.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			groupID: group.ID,
			things:  ths[:2],
			err:     nil,
		},
		"retrieve group things without group id": {
			pagemeta: things.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			groupID: "",
			things:  nil,
			err:     things.ErrRetrieveGroupThings,
		},
		"retrieve last group thing": {
			pagemeta: things.PageMetadata{
				Offset: 1,
				Limit:  1,
			},
			groupID: group.ID,
			things:  ths[1:2],
			err:     nil,
		},
	}

	for desc, tc := range cases {
		ths, err := groupRepo.RetrieveThingsByGroup(context.Background(), tc.groupID, tc.pagemeta)
		assert.Equal(t, tc.things, ths.Things, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.things, ths.Things))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRetrieveGroupChannels(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepository(dbMiddleware)
	channelRepo := postgres.NewChannelRepository(dbMiddleware)

	owID := generateUUID(t)
	orgID := generateUUID(t)

	creationTime := time.Now().UTC()
	group := things.Group{
		ID:        generateUUID(t),
		Name:      groupName,
		OwnerID:   owID,
		OrgID:     orgID,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
	}
	group2 := things.Group{
		ID:        generateUUID(t),
		Name:      groupName + "2",
		OwnerID:   owID,
		OrgID:     orgID,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
	}

	group, err := groupRepo.Save(context.Background(), group)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))
	group2, err = groupRepo.Save(context.Background(), group2)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	chID1, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	chID2, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	chID3, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	channels := []things.Channel{
		{
			ID:       chID1,
			Name:     channelName,
			OwnerID:  owID,
			GroupID:  group.ID,
			Metadata: map[string]interface{}{},
		},
		{
			ID:       chID2,
			Name:     channelName,
			OwnerID:  owID,
			GroupID:  group.ID,
			Metadata: map[string]interface{}{},
		},
		{
			ID:       chID3,
			Name:     channelName,
			OwnerID:  owID,
			GroupID:  group2.ID,
			Metadata: map[string]interface{}{},
		},
	}
	_, err = channelRepo.Save(context.Background(), channels...)
	require.Nil(t, err, fmt.Sprintf("channel save got unexpected error: %s", err))

	cases := map[string]struct {
		pagemeta things.PageMetadata
		groupID  string
		channels []things.Channel
		err      error
	}{
		"retrieve group channels": {
			pagemeta: things.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			groupID:  group.ID,
			channels: channels[:2],
			err:      nil,
		},
		"retrieve group channels without group id": {
			pagemeta: things.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			groupID:  "",
			channels: nil,
			err:      things.ErrRetrieveGroupChannels,
		},
		"retrieve last group channel": {
			pagemeta: things.PageMetadata{
				Offset: 1,
				Limit:  1,
			},
			groupID:  group.ID,
			channels: channels[1:2],
			err:      nil,
		},
	}

	for desc, tc := range cases {
		chs, err := groupRepo.RetrieveChannelsByGroup(context.Background(), tc.groupID, tc.pagemeta)
		assert.Equal(t, tc.channels, chs.Channels, fmt.Sprintf("%s: expected %v got %v\n", desc, tc.channels, chs.Channels))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func generateUUID(t *testing.T) string {
	id, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	return id
}

func createGroup(t *testing.T, dbMiddleware postgres.Database) things.Group {
	groupRepo := postgres.NewGroupRepository(dbMiddleware)

	grID := generateUUID(t)
	owID := generateUUID(t)
	orgID := generateUUID(t)

	group, err := groupRepo.Save(context.Background(), things.Group{
		ID:      grID,
		OwnerID: owID,
		OrgID:   orgID,
		Name:    "gr-name",
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	return group
}
