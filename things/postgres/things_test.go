// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/things/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	maxNameSize = 1024
	descDir     = "desc"
	ascDir      = "asc"
	idOrder     = "id"
	nameOrder   = "name"
)

var (
	invalidName = strings.Repeat("m", maxNameSize+1)
	idProvider  = uuid.New()
)

func TestSaveThings(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	nonexistentThingKey, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	prID := generateUUID(t)
	group := createGroup(t, dbMiddleware)

	p := things.Profile{
		ID:      prID,
		GroupID: group.ID,
		Name:    profileName,
	}
	_, err = profileRepo.Save(context.Background(), p)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	ths := []things.Thing{}
	for i := 1; i <= 5; i++ {
		thkey := generateUUID(t)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		thing := things.Thing{
			ID:        fmt.Sprintf("%s%012d", prefixID, i),
			GroupID:   group.ID,
			ProfileID: prID,
			Name:      fmt.Sprintf("%s-%d", thingName, i),
			Key:       thkey,
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
			desc:   "save new things",
			things: ths,
			err:    nil,
		},
		{
			desc:   "save things that already exist",
			things: ths,
			err:    errors.ErrConflict,
		},
		{
			desc: "save thing with invalid ID",
			things: []things.Thing{
				{ID: "invalid", GroupID: group.ID, Key: thkey},
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "save thing with invalid name",
			things: []things.Thing{
				{ID: thID, GroupID: group.ID, Key: thkey, Name: invalidName},
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "save thing with invalid Key",
			things: []things.Thing{
				{ID: thID, GroupID: group.ID, Key: nonexistentThingKey},
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc:   "save things with conflicting keys",
			things: ths,
			err:    errors.ErrConflict,
		},
	}

	for _, tc := range cases {
		_, err := thingRepo.Save(context.Background(), tc.things...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateThing(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	validName := "mfx_device"

	group := createGroup(t, dbMiddleware)

	prID := generateUUID(t)
	thID := generateUUID(t)
	thkey := generateUUID(t)

	p := things.Profile{
		ID:      prID,
		GroupID: group.ID,
		Name:    profileName,
	}
	_, err := profileRepo.Save(context.Background(), p)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	thing := things.Thing{
		ID:        thID,
		GroupID:   group.ID,
		ProfileID: prID,
		Name:      thingName,
		Key:       thkey,
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
			desc: "update non-existing thing",
			thing: things.Thing{
				ID:      nonexistentThingID,
				GroupID: group.ID,
			},
			err: errors.ErrNotFound,
		},
		{
			desc: "update existing thing ID",
			thing: things.Thing{
				ID:      thing.ID,
				GroupID: group.ID,
			},
			err: nil,
		},
		{
			desc: "update thing with valid name",
			thing: things.Thing{
				ID:      thID,
				GroupID: group.ID,
				Key:     thkey,
				Name:    validName,
			},
			err: nil,
		},
		{
			desc: "update thing with invalid name",
			thing: things.Thing{
				ID:   thID,
				Key:  thkey,
				Name: invalidName,
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
	dbMiddleware := dbutil.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)
	prID := generateUUID(t)
	id := generateUUID(t)
	key := generateUUID(t)

	p := things.Profile{
		ID:      prID,
		GroupID: group.ID,
		Name:    profileName,
	}
	_, err := profileRepo.Save(context.Background(), p)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	th1 := things.Thing{
		ID:        id,
		GroupID:   group.ID,
		ProfileID: prID,
		Name:      fmt.Sprintf("%s-%d", thingName, 1),
		Key:       key,
	}
	ths, err := thingRepo.Save(context.Background(), th1)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th1.ID = ths[0].ID

	id, err = idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	key, err = idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	th2 := things.Thing{
		ID:        id,
		GroupID:   group.ID,
		ProfileID: prID,
		Name:      fmt.Sprintf("%s-%d", thingName, 2),
		Key:       key,
	}
	ths, err = thingRepo.Save(context.Background(), th2)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th2.ID = ths[0].ID

	nonexistentThingID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		key  string
		err  error
	}{
		{
			desc: "update key of an existing thing",
			id:   th2.ID,
			key:  newKey,
			err:  nil,
		},
		{
			desc: "update key of a non-existing thing",
			id:   nonexistentThingID,
			key:  newKey,
			err:  errors.ErrNotFound,
		},
		{
			desc: "update key with existing key value",
			id:   th2.ID,
			key:  th1.Key,
			err:  errors.ErrConflict,
		},
	}

	for _, tc := range cases {
		err := thingRepo.UpdateKey(context.Background(), tc.id, tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveThingByID(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)
	prID := generateUUID(t)
	id := generateUUID(t)
	key := generateUUID(t)

	p := things.Profile{
		ID:      prID,
		GroupID: group.ID,
		Name:    profileName,
	}
	_, err := profileRepo.Save(context.Background(), p)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	th := things.Thing{
		ID:        id,
		GroupID:   group.ID,
		ProfileID: prID,
		Name:      thingName,
		Key:       key,
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

func TestRetrieveByKey(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)
	prID := generateUUID(t)
	id := generateUUID(t)
	key := generateUUID(t)

	p := things.Profile{
		ID:      prID,
		GroupID: group.ID,
		Name:    profileName,
	}
	_, err := profileRepo.Save(context.Background(), p)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	th := things.Thing{
		ID:        id,
		GroupID:   group.ID,
		ProfileID: prID,
		Name:      thingName,
		Key:       key,
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

func TestRetrieveThingsByGroupIDs(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	err := cleanTestTable(context.Background(), "things", dbMiddleware)
	assert.Nil(t, err, fmt.Sprintf("cleaning table 'things' expected to success %v", err))
	thingRepo := postgres.NewThingRepository(dbMiddleware)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	prID := generateUUID(t)
	group := createGroup(t, dbMiddleware)

	p := things.Profile{
		ID:      prID,
		GroupID: group.ID,
		Name:    profileName,
	}
	_, err = profileRepo.Save(context.Background(), p)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	metadata := things.Metadata{
		"field": "value",
	}

	wrongMeta := things.Metadata{
		"wrong": "wrong",
	}

	offset := uint64(1)
	metaNum := uint64(3)
	var ths []things.Thing
	n := uint64(101)

	for i := uint64(0); i < n; i++ {
		suffix := i + 1
		key := generateUUID(t)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		th := things.Thing{
			ID:        fmt.Sprintf("%s%012d", prefixID, suffix),
			GroupID:   group.ID,
			ProfileID: prID,
			Name:      fmt.Sprintf("%s-%d", thingName, suffix),
			Key:       key,
		}

		if i < metaNum {
			th.Metadata = metadata
		}

		ths = append(ths, th)
	}

	_, err = thingRepo.Save(context.Background(), ths...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := map[string]struct {
		pageMetadata apiutil.PageMetadata
		size         uint64
	}{
		"retrieve all things by group IDs": {
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Dir:    descDir,
				Order:  idOrder,
			},
			size: n,
		},
		"retrieve all things by group IDs without limit": {
			pageMetadata: apiutil.PageMetadata{
				Limit: 0,
				Dir:   descDir,
				Order: idOrder,
			},
			size: n,
		},
		"retrieve subset of things by group IDs": {
			pageMetadata: apiutil.PageMetadata{
				Offset: offset,
				Limit:  n,
				Dir:    descDir,
				Order:  idOrder,
			},
			size: n - offset,
		},
		"retrieve things by group IDs with existing name": {
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Name:   "test-thing-101",
				Dir:    descDir,
				Order:  idOrder,
			},
			size: 1,
		},
		"retrieve things by group IDs with non-existing name": {
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Name:   "wrong",
				Order:  nameOrder,
				Dir:    descDir,
			},
			size: 0,
		},
		"retrieve things by group IDs with existing metadata": {
			pageMetadata: apiutil.PageMetadata{
				Offset:   0,
				Limit:    n,
				Metadata: metadata,
				Dir:      descDir,
				Order:    idOrder,
			},
			size: metaNum,
		},
		"retrieve things by group IDs with non-existing metadata": {
			pageMetadata: apiutil.PageMetadata{
				Offset:   0,
				Limit:    n,
				Metadata: wrongMeta,
				Order:    nameOrder,
				Dir:      descDir,
			},
			size: 0,
		},
		"retrieve things by group IDs sorted by name ascendant": {
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  nameOrder,
				Dir:    ascDir,
			},
			size: n,
		},
		"retrieve things by group IDs sorted by name descendent": {
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  nameOrder,
				Dir:    descDir,
			},
			size: n,
		},
	}

	for desc, tc := range cases {
		page, err := thingRepo.RetrieveByGroupIDs(context.Background(), []string{group.ID}, tc.pageMetadata)
		size := uint64(len(page.Things))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))

		// Check if Things list have been sorted properly
		testSortThings(t, tc.pageMetadata, page.Things)
	}
}

func TestRetrieveAllThings(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	err := cleanTestTable(context.Background(), "things", dbMiddleware)
	assert.Nil(t, err, fmt.Sprintf("cleaning table 'things' expected to success %v", err))
	thingRepo := postgres.NewThingRepository(dbMiddleware)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)
	prID := generateUUID(t)

	p := things.Profile{
		ID:      prID,
		GroupID: group.ID,
		Name:    profileName,
	}
	_, err = profileRepo.Save(context.Background(), p)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	metaStr := `{"field1":"value1","field2":{"subfield11":"value2","subfield12":{"subfield121":"value3","subfield122":"value4"}}}`
	subMetaStr := `{"field2":{"subfield12":{"subfield121":"value3"}}}`

	metadata := things.Metadata{}
	json.Unmarshal([]byte(metaStr), &metadata)

	subMeta := things.Metadata{}
	json.Unmarshal([]byte(subMetaStr), &subMeta)

	metaNum := uint64(3)
	ths := []things.Thing{}
	n := uint64(101)
	for i := uint64(0); i < n; i++ {
		suffix := i + 1
		key := generateUUID(t)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		th := things.Thing{
			ID:        fmt.Sprintf("%s%012d", prefixID, suffix),
			GroupID:   group.ID,
			ProfileID: prID,
			Name:      fmt.Sprintf("%s-%d", thingName, suffix),
			Key:       key,
		}
		if i < metaNum {
			th.Metadata = metadata
		}

		ths = append(ths, th)
	}

	_, err = thingRepo.Save(context.Background(), ths...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

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

func TestRetrieveByProfile(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	n := uint64(101)
	group := createGroup(t, dbMiddleware)
	prID := generateUUID(t)

	p := things.Profile{
		ID:      prID,
		GroupID: group.ID,
		Name:    profileName,
	}
	_, err := profileRepo.Save(context.Background(), p)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var ths []things.Thing
	for i := uint64(0); i < n; i++ {
		suffix := n + i + 1
		thkey, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		th := things.Thing{
			ID:        fmt.Sprintf("%s%012d", prefixID, suffix),
			GroupID:   group.ID,
			ProfileID: prID,
			Name:      fmt.Sprintf("%s-%d", thingName, suffix),
			Key:       thkey,
		}
		ths = append(ths, th)
	}

	_, err = thingRepo.Save(context.Background(), ths...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	nonexistentProfileID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		prID         string
		pageMetadata apiutil.PageMetadata
		size         uint64
		err          error
	}{
		"retrieve all things by profile": {
			prID: prID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Dir:    descDir,
				Order:  idOrder,
			},
			size: n,
		},

		"retrieve all things by profile without limit": {
			prID: prID,
			pageMetadata: apiutil.PageMetadata{
				Limit: 0,
				Dir:   descDir,
				Order: idOrder,
			},
			size: n,
		},
		"retrieve subset of things by profile": {
			prID: prID,
			pageMetadata: apiutil.PageMetadata{
				Offset: n / 2,
				Limit:  n,
				Dir:    descDir,
				Order:  idOrder,
			},
			size: n - (n / 2),
		},
		"retrieve things by non-existing profile": {
			prID: nonexistentProfileID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Dir:    descDir,
				Order:  idOrder,
			},
			size: 0,
		},
		"retrieve things with malformed UUID": {
			prID: "wrong",
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: 0,
			err:  errors.ErrNotFound,
		},
		"retrieve all things by profile sorted by name ascendant": {
			prID: prID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  nameOrder,
				Dir:    ascDir,
			},
			size: n,
		},
		"retrieve all things by profile sorted by name descendent": {
			prID: prID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  nameOrder,
				Dir:    descDir,
			},
			size: n,
		},
	}

	for desc, tc := range cases {
		page, err := thingRepo.RetrieveByProfile(context.Background(), tc.prID, tc.pageMetadata)
		size := uint64(len(page.Things))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected no error got %d\n", desc, err))

		// Check if Things by Profile list have been sorted properly
		testSortThings(t, tc.pageMetadata, page.Things)
	}
}

func TestRemoveThing(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)
	id := generateUUID(t)
	key := generateUUID(t)
	prID := generateUUID(t)

	p := things.Profile{
		ID:      prID,
		GroupID: group.ID,
		Name:    profileName,
	}
	_, err := profileRepo.Save(context.Background(), p)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	thing := things.Thing{
		ID:        id,
		GroupID:   group.ID,
		ProfileID: prID,
		Name:      thingName,
		Key:       key,
	}

	ths, _ := thingRepo.Save(context.Background(), thing)
	thing.ID = ths[0].ID

	cases := map[string]struct {
		thingID string
		err     error
	}{
		"remove non-existing thing": {
			thingID: "wrong",
			err:     errors.ErrRemoveEntity,
		},
		"remove thing": {
			thingID: thing.ID,
			err:     nil,
		},
	}

	for desc, tc := range cases {
		err := thingRepo.Remove(context.Background(), tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func testSortThings(t *testing.T, pm apiutil.PageMetadata, ths []things.Thing) {
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

func cleanTestTable(ctx context.Context, table string, db dbutil.Database) error {
	q := fmt.Sprintf(`DELETE FROM %s CASCADE;`, table)
	_, err := db.NamedExecContext(ctx, q, map[string]interface{}{})
	return err
}
