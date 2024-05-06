// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/things/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const maxNameSize = 1024

var (
	invalidName = strings.Repeat("m", maxNameSize+1)
	idProvider  = uuid.New()
)

func TestThingsSave(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	nonexistentThingKey, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	group := createGroup(t, dbMiddleware)

	ths := []things.Thing{}
	for i := 1; i <= 5; i++ {
		thID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		thkey := generateUUID(t)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		thing := things.Thing{
			ID:      thID,
			OwnerID: group.OwnerID,
			GroupID: group.ID,
			Key:     thkey,
		}
		ths = append(ths, thing)
	}
	thkey := ths[0].Key
	thID := ths[0].ID

	cases := []struct {
		desc   string
		things []things.Thing
		err    error
	}{
		{
			desc:   "create new things",
			things: ths,
			err:    nil,
		},
		{
			desc:   "create things that already exist",
			things: ths,
			err:    errors.ErrConflict,
		},
		{
			desc: "create thing with invalid ID",
			things: []things.Thing{
				{ID: "invalid", OwnerID: group.OwnerID, GroupID: group.ID, Key: thkey},
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "create thing with invalid name",
			things: []things.Thing{
				{ID: thID, OwnerID: group.OwnerID, GroupID: group.ID, Key: thkey, Name: invalidName},
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "create thing with invalid Key",
			things: []things.Thing{
				{ID: thID, OwnerID: group.OwnerID, GroupID: group.ID, Key: nonexistentThingKey},
			},
			err: errors.ErrConflict,
		},
		{
			desc:   "create things with conflicting keys",
			things: ths,
			err:    errors.ErrConflict,
		},
	}

	for _, tc := range cases {
		_, err := thingRepo.Save(context.Background(), tc.things...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestThingUpdate(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	validName := "mfx_device"

	group := createGroup(t, dbMiddleware)

	thID := generateUUID(t)
	thkey := generateUUID(t)

	thing := things.Thing{
		ID:      thID,
		OwnerID: group.OwnerID,
		GroupID: group.ID,
		Key:     thkey,
	}

	sths, err := thingRepo.Save(context.Background(), thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	thing.ID = sths[0].ID

	nonexistentThingID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc  string
		thing things.Thing
		err   error
	}{
		{
			desc:  "update existing thing",
			thing: thing,
			err:   nil,
		},
		{
			desc: "update non-existing thing with existing user",
			thing: things.Thing{
				ID:      nonexistentThingID,
				OwnerID: group.OwnerID,
				GroupID: group.ID,
			},
			err: errors.ErrNotFound,
		},
		{
			desc: "update existing thing ID with non-existing user",
			thing: things.Thing{
				ID:      thing.ID,
				OwnerID: wrongID,
				GroupID: group.ID,
			},
			err: nil,
		},
		{
			desc: "update non-existing thing with non-existing user",
			thing: things.Thing{
				ID:      nonexistentThingID,
				OwnerID: wrongID,
				GroupID: group.ID,
			},
			err: errors.ErrNotFound,
		},
		{
			desc: "update thing with valid name",
			thing: things.Thing{
				ID:      thID,
				OwnerID: group.OwnerID,
				GroupID: group.ID,
				Key:     thkey,
				Name:    validName,
			},
			err: nil,
		},
		{
			desc: "update thing with invalid name",
			thing: things.Thing{
				ID:      thID,
				OwnerID: group.OwnerID,
				Key:     thkey,
				Name:    invalidName,
			},
			err: errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		err := thingRepo.Update(context.Background(), tc.thing)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateKey(t *testing.T) {
	newKey := "new-key"
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)

	id := generateUUID(t)
	key := generateUUID(t)

	th1 := things.Thing{
		ID:      id,
		OwnerID: group.OwnerID,
		GroupID: group.ID,
		Key:     key,
	}
	ths, err := thingRepo.Save(context.Background(), th1)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th1.ID = ths[0].ID

	id, err = idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	key, err = idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	th2 := things.Thing{
		ID:      id,
		OwnerID: group.OwnerID,
		GroupID: group.ID,
		Key:     key,
	}
	ths, err = thingRepo.Save(context.Background(), th2)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th2.ID = ths[0].ID

	nonexistentThingID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc    string
		ownerID string
		id      string
		key     string
		err     error
	}{
		{
			desc:    "update key of an existing thing",
			ownerID: th2.OwnerID,
			id:      th2.ID,
			key:     newKey,
			err:     nil,
		},
		{
			desc:    "update key of a non-existing thing with existing user",
			ownerID: th2.OwnerID,
			id:      nonexistentThingID,
			key:     newKey,
			err:     errors.ErrNotFound,
		},
		{
			desc:    "update key of an existing thing with non-existing user",
			ownerID: wrongID,
			id:      th2.ID,
			key:     newKey,
			err:     errors.ErrNotFound,
		},
		{
			desc:    "update key of a non-existing thing with non-existing user",
			ownerID: wrongID,
			id:      nonexistentThingID,
			key:     newKey,
			err:     errors.ErrNotFound,
		},
		{
			desc:    "update key with existing key value",
			ownerID: th2.OwnerID,
			id:      th2.ID,
			key:     th1.Key,
			err:     errors.ErrConflict,
		},
	}

	for _, tc := range cases {
		err := thingRepo.UpdateKey(context.Background(), tc.ownerID, tc.id, tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSingleThingRetrieval(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)

	id := generateUUID(t)
	key := generateUUID(t)

	th := things.Thing{
		ID:      id,
		OwnerID: group.OwnerID,
		GroupID: group.ID,
		Key:     key,
	}

	ths, err := thingRepo.Save(context.Background(), th)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th.ID = ths[0].ID

	nonexistentThingID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		ID  string
		err error
	}{
		"retrieve thing with existing user": {
			ID:  th.ID,
			err: nil,
		},
		"retrieve non-existing thing with existing user": {
			ID:  nonexistentThingID,
			err: errors.ErrNotFound,
		},
		"retrieve thing with malformed ID": {
			ID:  wrongID,
			err: errors.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := thingRepo.RetrieveByID(context.Background(), tc.ID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestThingRetrieveByKey(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)

	id := generateUUID(t)
	key := generateUUID(t)

	th := things.Thing{
		ID:      id,
		OwnerID: group.OwnerID,
		GroupID: group.ID,
		Key:     key,
	}

	ths, err := thingRepo.Save(context.Background(), th)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th.ID = ths[0].ID

	cases := map[string]struct {
		key string
		ID  string
		err error
	}{
		"retrieve existing thing by key": {
			key: th.Key,
			ID:  th.ID,
			err: nil,
		},
		"retrieve non-existent thing by key": {
			key: wrongID,
			ID:  "",
			err: errors.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		id, err := thingRepo.RetrieveByKey(context.Background(), tc.key)
		assert.Equal(t, tc.ID, id, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.ID, id))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestMultiThingRetrieval(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	err := cleanTestTable(context.Background(), "things", dbMiddleware)
	assert.Nil(t, err, fmt.Sprintf("cleaning table 'things' expected to success %v", err))
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	name := "thing_name"
	metaStr := `{"field1":"value1","field2":{"subfield11":"value2","subfield12":{"subfield121":"value3","subfield122":"value4"}}}`
	subMetaStr := `{"field2":{"subfield12":{"subfield121":"value3"}}}`

	metadata := things.Metadata{}
	json.Unmarshal([]byte(metaStr), &metadata)

	subMeta := things.Metadata{}
	json.Unmarshal([]byte(subMetaStr), &subMeta)

	wrongMeta := things.Metadata{
		"field": "value1",
	}

	offset := uint64(1)
	nameNum := uint64(3)
	metaNum := uint64(3)
	nameMetaNum := uint64(2)

	group := createGroup(t, dbMiddleware)

	n := uint64(101)
	for i := uint64(0); i < n; i++ {
		id := generateUUID(t)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		key := generateUUID(t)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		th := things.Thing{
			OwnerID: group.OwnerID,
			GroupID: group.ID,
			ID:      id,
			Key:     key,
		}

		// Create Things with name.
		if i < nameNum {
			th.Name = fmt.Sprintf("%s-%d", name, i)
		}
		// Create Things with metadata.
		if i >= nameNum && i < nameNum+metaNum {
			th.Metadata = metadata
		}
		// Create Things with name and metadata.
		if i >= n-nameMetaNum {
			th.Metadata = metadata
			th.Name = name
		}

		_, err = thingRepo.Save(context.Background(), th)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	cases := map[string]struct {
		ownerID      string
		pageMetadata things.PageMetadata
		size         uint64
	}{
		"retrieve all things": {
			ownerID: group.OwnerID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  n,
			},
			size: n,
		},
		"retrieve all things with no limit": {
			ownerID: group.OwnerID,
			pageMetadata: things.PageMetadata{
				Limit: 0,
				Total: n,
			},
			size: n,
		},
		"retrieve subset of things with existing owner": {
			ownerID: group.OwnerID,
			pageMetadata: things.PageMetadata{
				Offset: offset,
				Limit:  n,
				Total:  n,
			},
			size: n - offset,
		},
		"retrieve things with existing name": {
			ownerID: group.OwnerID,
			pageMetadata: things.PageMetadata{
				Offset: offset,
				Limit:  n,
				Name:   name,
				Total:  nameNum + nameMetaNum,
			},
			size: nameNum + nameMetaNum - offset,
		},
		"retrieve things with non-existing name": {
			ownerID: group.OwnerID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Name:   "wrong",
				Total:  0,
			},
			size: 0,
		},
		"retrieve things with existing metadata": {
			ownerID: group.OwnerID,
			pageMetadata: things.PageMetadata{
				Offset:   0,
				Limit:    n,
				Total:    metaNum + nameMetaNum,
				Metadata: metadata,
			},
			size: metaNum + nameMetaNum,
		},
		"retrieve things with partial metadata": {
			ownerID: group.OwnerID,
			pageMetadata: things.PageMetadata{
				Offset:   0,
				Limit:    n,
				Total:    metaNum + nameMetaNum,
				Metadata: subMeta,
			},
			size: metaNum + nameMetaNum,
		},
		"retrieve things with non-existing metadata": {
			ownerID: group.OwnerID,
			pageMetadata: things.PageMetadata{
				Offset:   0,
				Limit:    n,
				Total:    0,
				Metadata: wrongMeta,
			},
			size: 0,
		},
		"retrieve all things with existing name and metadata": {
			ownerID: group.OwnerID,
			pageMetadata: things.PageMetadata{
				Offset:   0,
				Limit:    n,
				Total:    nameMetaNum,
				Name:     name,
				Metadata: metadata,
			},
			size: nameMetaNum,
		},
		"retrieve things sorted by name ascendent": {
			ownerID: group.OwnerID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  n,
				Order:  "name",
				Dir:    "asc",
			},
			size: n,
		},
		"retrieve things sorted by name descendent": {
			ownerID: group.OwnerID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  n,
				Order:  "name",
				Dir:    "desc",
			},
			size: n,
		},
	}

	for desc, tc := range cases {
		page, err := thingRepo.RetrieveByOwner(context.Background(), tc.ownerID, tc.pageMetadata)
		size := uint64(len(page.Things))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.pageMetadata.Total, page.Total, fmt.Sprintf("%s: expected total %d got %d\n", desc, tc.pageMetadata.Total, page.Total))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))

		// Check if Things list have been sorted properly
		testSortThings(t, tc.pageMetadata, page.Things)
	}
}

func TestBackupThings(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	err := cleanTestTable(context.Background(), "things", dbMiddleware)
	assert.Nil(t, err, fmt.Sprintf("cleaning table 'things' expected to success %v", err))
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	name := "thing_name"
	metaStr := `{"field1":"value1","field2":{"subfield11":"value2","subfield12":{"subfield121":"value3","subfield122":"value4"}}}`
	subMetaStr := `{"field2":{"subfield12":{"subfield121":"value3"}}}`

	metadata := things.Metadata{}
	json.Unmarshal([]byte(metaStr), &metadata)

	subMeta := things.Metadata{}
	json.Unmarshal([]byte(subMetaStr), &subMeta)

	nameNum := uint64(3)
	metaNum := uint64(3)
	nameMetaNum := uint64(2)

	group := createGroup(t, dbMiddleware)

	n := uint64(101)
	for i := uint64(0); i < n; i++ {
		id := generateUUID(t)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		key := generateUUID(t)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		th := things.Thing{
			OwnerID: group.OwnerID,
			GroupID: group.ID,
			ID:      id,
			Key:     key,
		}

		// Create Things with name.
		if i < nameNum {
			th.Name = fmt.Sprintf("%s-%d", name, i)
		}
		// Create Things with metadata.
		if i >= nameNum && i < nameNum+metaNum {
			th.Metadata = metadata
		}
		// Create Things with name and metadata.
		if i >= n-nameMetaNum {
			th.Metadata = metadata
			th.Name = name
		}

		_, err = thingRepo.Save(context.Background(), th)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	cases := map[string]struct {
		size uint64
		err  error
	}{
		"retrieve all things": {
			size: n,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		things, err := thingRepo.RetrieveAll(context.Background())
		size := uint64(len(things))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestRetrieveByChannel(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)
	channelRepo := postgres.NewChannelRepository(dbMiddleware)

	n := uint64(101)
	thsDisconNum := uint64(1)

	group := createGroup(t, dbMiddleware)

	chID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	_, err = channelRepo.Save(context.Background(), things.Channel{
		ID:      chID,
		OwnerID: group.OwnerID,
		GroupID: group.ID,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	for i := uint64(0); i < n; i++ {
		thID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		thkey := generateUUID(t)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		th := things.Thing{
			ID:      thID,
			OwnerID: group.OwnerID,
			GroupID: group.ID,
			Key:     thkey,
		}

		ths, err := thingRepo.Save(context.Background(), th)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		thID = ths[0].ID

		// Don't connnect last Thing
		if i == n-thsDisconNum {
			break
		}

		err = channelRepo.Connect(context.Background(), chID, []string{thID})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	nonexistentChanID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		chID         string
		pageMetadata things.PageMetadata
		size         uint64
		err          error
	}{
		"retrieve all things by channel with existing owner": {
			chID: chID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n - thsDisconNum,
		},
		"retrieve all things by channel without limit": {
			chID: chID,
			pageMetadata: things.PageMetadata{
				Limit: 0,
			},
			size: n - thsDisconNum,
		},
		"retrieve subset of things by channel with existing owner": {
			chID: chID,
			pageMetadata: things.PageMetadata{
				Offset: n / 2,
				Limit:  n,
			},
			size: (n - (n / 2)) - thsDisconNum,
		},
		"retrieve things by non-existing channel": {
			chID: nonexistentChanID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: 0,
		},
		"retrieve things with malformed UUID": {
			chID: "wrong",
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: 0,
			err:  errors.ErrNotFound,
		},
		"retrieve all things by channel sorted by name ascendent": {
			chID: chID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "asc",
			},
			size: n - thsDisconNum,
		},
		"retrieve all things by channel sorted by name descendent": {
			chID: chID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "desc",
			},
			size: n - thsDisconNum,
		},
	}

	for desc, tc := range cases {
		page, err := thingRepo.RetrieveByChannel(context.Background(), tc.chID, tc.pageMetadata)
		size := uint64(len(page.Things))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected no error got %d\n", desc, err))

		// Check if Things by Channel list have been sorted properly
		testSortThings(t, tc.pageMetadata, page.Things)
	}
}

func TestThingRemoval(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)

	id := generateUUID(t)
	key := generateUUID(t)

	thing := things.Thing{
		ID:      id,
		OwnerID: group.OwnerID,
		GroupID: group.ID,
		Key:     key,
	}

	ths, _ := thingRepo.Save(context.Background(), thing)
	thing.ID = ths[0].ID

	cases := map[string]struct {
		owner   string
		thingID string
		err     error
	}{
		"remove non-existing thing": {
			owner:   group.OwnerID,
			thingID: "wrong",
			err:     errors.ErrRemoveEntity,
		},
		"remove thing": {
			owner:   group.OwnerID,
			thingID: thing.ID,
			err:     nil,
		},
	}

	for desc, tc := range cases {
		err := thingRepo.Remove(context.Background(), tc.owner, tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func testSortThings(t *testing.T, pm things.PageMetadata, ths []things.Thing) {
	if len(ths) < 1 {
		return
	}

	switch pm.Order {
	case "name":
		current := ths[0]
		for _, res := range ths {
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

func cleanTestTable(ctx context.Context, table string, db postgres.Database) error {
	q := fmt.Sprintf(`DELETE FROM %s CASCADE;`, table)
	_, err := db.NamedExecContext(ctx, q, map[string]interface{}{})
	return err
}
