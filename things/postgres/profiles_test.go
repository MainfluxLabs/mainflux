// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/things/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const prefixID = "fe6b4e92-cc98-425e-b0aa-"

func TestSaveProfiles(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)

	chs := []things.Profile{}
	for i := 1; i <= 5; i++ {
		id, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		ch := things.Profile{
			ID:      id,
			GroupID: group.ID,
			Name:    fmt.Sprintf("%s-%d", profileName, i),
		}
		chs = append(chs, ch)
	}
	id2, _ := idProvider.ID()
	chs = append(chs, things.Profile{ID: id2, GroupID: group.ID, Name: ""})
	id := chs[0].ID

	cases := []struct {
		desc     string
		profiles []things.Profile
		err      error
	}{
		{
			desc:     "save new profiles",
			profiles: chs,
			err:      nil,
		},
		{
			desc:     "save profiles that already exist",
			profiles: chs,
			err:      errors.ErrConflict,
		},
		{
			desc: "save profile with invalid ID",
			profiles: []things.Profile{
				{ID: "invalid", GroupID: group.ID, Name: profileName},
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "save profile with invalid name",
			profiles: []things.Profile{
				{ID: id, GroupID: group.ID, Name: invalidName},
			},
			err: errors.ErrMalformedEntity,
		},
	}

	for _, cc := range cases {
		_, err := profileRepo.Save(context.Background(), cc.profiles...)
		assert.True(t, errors.Contains(err, cc.err), fmt.Sprintf("%s: expected %s got %s\n", cc.desc, cc.err, err))
	}
}

func TestUpdateProfile(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)

	id, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ch := things.Profile{
		ID:      id,
		GroupID: group.ID,
		Name:    profileName,
	}

	chs, err := profileRepo.Save(context.Background(), ch)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch.ID = chs[0].ID

	nonexistentProfileID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc    string
		profile things.Profile
		err     error
	}{
		{
			desc:    "update existing profile",
			profile: ch,
			err:     nil,
		},
		{
			desc: "update non-existing profile",
			profile: things.Profile{
				ID: nonexistentProfileID,
			},
			err: errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := profileRepo.Update(context.Background(), tc.profile)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveProfileByID(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)

	thID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	th := things.Thing{
		ID:      thID,
		GroupID: group.ID,
		Name:    thingName,
		Key:     thkey,
	}
	ths, err := thingRepo.Save(context.Background(), th)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th = ths[0]

	chID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	c := things.Profile{
		ID:      chID,
		GroupID: group.ID,
		Name:    profileName,
	}
	chs, err := profileRepo.Save(context.Background(), c)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	ch := chs[0]

	err = profileRepo.Connect(context.Background(), ch.ID, []string{th.ID})
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	nonexistentProfileID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		ID  string
		err error
	}{
		"retrieve profile": {
			ID:  ch.ID,
			err: nil,
		},
		"retrieve profile with non-existing profile": {
			ID:  nonexistentProfileID,
			err: errors.ErrNotFound,
		},
		"retrieve profile with malformed ID": {
			ID:  wrongID,
			err: errors.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := profileRepo.RetrieveByID(context.Background(), tc.ID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRetrieveProfilesByGroupIDs(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	metadata := things.Metadata{
		"field": "value",
	}
	wrongMeta := things.Metadata{
		"wrong": "wrong",
	}

	offset := uint64(1)
	metaNum := uint64(3)
	group := createGroup(t, dbMiddleware)
	var chs []things.Profile
	n := uint64(101)

	for i := uint64(0); i < n; i++ {
		suffix := i + 1
		ch := things.Profile{
			ID:      fmt.Sprintf("%s%012d", prefixID, suffix),
			GroupID: group.ID,
			Name:    fmt.Sprintf("%s-%d", profileName, suffix),
		}
		if i < metaNum {
			ch.Metadata = metadata
		}

		chs = append(chs, ch)
	}

	_, err := profileRepo.Save(context.Background(), chs...)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		size         uint64
		pageMetadata things.PageMetadata
	}{
		"retrieve all profiles by group IDs": {
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
		},
		"retrieve all profiles by group IDs without limit": {
			pageMetadata: things.PageMetadata{
				Limit: 0,
			},
			size: n,
		},
		"retrieve subset of profiles by group IDs": {
			pageMetadata: things.PageMetadata{
				Offset: offset,
				Limit:  n,
			},
			size: n - offset,
		},
		"retrieve profiles by group IDs with existing name": {
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Name:   "test-profile-101",
			},
			size: 1,
		},
		"retrieve all profiles by group IDs with non-existing name": {
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Name:   "wrong",
			},
			size: 0,
		},
		"retrieve all profiles by group IDs with existing metadata": {
			pageMetadata: things.PageMetadata{
				Offset:   0,
				Limit:    n,
				Metadata: metadata,
			},
			size: metaNum,
		},
		"retrieve all profiles by group IDs with non-existing metadata": {
			pageMetadata: things.PageMetadata{
				Offset:   0,
				Limit:    n,
				Metadata: wrongMeta,
			},
		},
		"retrieve profiles by group IDs sorted by name ascendant": {
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "asc",
			},
			size: n,
		},
		"retrieve profiles by group IDs sorted by name descendent": {
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "desc",
			},
			size: n,
		},
	}

	for desc, tc := range cases {
		page, err := profileRepo.RetrieveByGroupIDs(context.Background(), []string{group.ID}, tc.pageMetadata)
		size := uint64(len(page.Profiles))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))

		// Check if Profiles list have been sorted properly
		testSortProfiles(t, tc.pageMetadata, page.Profiles)
	}
}

func TestRetrieveProfileByThing(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)

	thID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	th, err := thingRepo.Save(context.Background(), things.Thing{
		ID:      thID,
		GroupID: group.ID,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	thID = th[0].ID

	chID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ch := things.Profile{
		ID:       chID,
		GroupID:  group.ID,
		Config:   things.Metadata{},
		Metadata: things.Metadata{},
	}

	_, err = profileRepo.Save(context.Background(), ch)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	err = profileRepo.Connect(context.Background(), chID, []string{thID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	nonexistentThingID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		thID    string
		profile things.Profile
		err     error
	}{
		"retrieve profile by thing": {
			thID:    thID,
			profile: ch,
			err:     nil,
		},
		"retrieve profile by non-existent thing": {
			thID:    nonexistentThingID,
			profile: things.Profile{},
			err:     nil,
		},
		"retrieve profile with malformed UUID": {
			thID:    "wrong",
			profile: things.Profile{},
			err:     errors.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		ch, err := profileRepo.RetrieveByThing(context.Background(), tc.thID)
		assert.Equal(t, tc.profile, ch, fmt.Sprintf("%s: expected %v got %v\n", desc, tc.profile, ch))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestRemoveProfile(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)

	chID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	chs, err := profileRepo.Save(context.Background(), things.Profile{
		ID:      chID,
		GroupID: group.ID,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chID = chs[0].ID

	cases := map[string]struct {
		chID string
		err  error
	}{
		"remove non-existing profile": {
			chID: "wrong",
			err:  errors.ErrRemoveEntity,
		},
		"remove profile": {
			chID: chID,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		err := profileRepo.Remove(context.Background(), tc.chID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestConnect(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)

	thID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thID1, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey1, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	th := []things.Thing{
		{
			ID:       thID,
			GroupID:  group.ID,
			Name:     fmt.Sprintf("%s-%d", thingName, 1),
			Key:      thkey,
			Metadata: things.Metadata{},
		},
		{
			ID:       thID1,
			GroupID:  group.ID,
			Name:     fmt.Sprintf("%s-%d", thingName, 2),
			Key:      thkey1,
			Metadata: things.Metadata{},
		}}

	ths, err := thingRepo.Save(context.Background(), th...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	thID = ths[0].ID

	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	chID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	chs, err := profileRepo.Save(context.Background(), things.Profile{
		ID:      chID,
		GroupID: group.ID,
		Name:    profileName,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chID = chs[0].ID

	nonexistentThingID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	nonexistentProfileID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc string
		chID string
		thID string
		err  error
	}{
		{
			desc: "connect existing profile and thing",
			chID: chID,
			thID: thID,
			err:  nil,
		},
		{
			desc: "connect connected profile and thing",
			chID: chID,
			thID: thID,
			err:  errors.ErrConflict,
		},
		{
			desc: "connect non-existing profile",
			chID: nonexistentProfileID,
			thID: thID1,
			err:  errors.ErrNotFound,
		},
		{
			desc: "connect non-existing thing",
			chID: chID,
			thID: nonexistentThingID,
			err:  errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := profileRepo.Connect(context.Background(), tc.chID, []string{tc.thID})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestDisconnect(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)

	thID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	th := things.Thing{
		ID:       thID,
		GroupID:  group.ID,
		Name:     thingName,
		Key:      thkey,
		Metadata: map[string]interface{}{},
	}
	ths, err := thingRepo.Save(context.Background(), th)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	thID = ths[0].ID

	profileRepo := postgres.NewProfileRepository(dbMiddleware)
	chID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	chs, err := profileRepo.Save(context.Background(), things.Profile{
		ID:      chID,
		GroupID: group.ID,
		Name:    profileName,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]
	err = profileRepo.Connect(context.Background(), ch.ID, []string{thID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	nonexistentThingID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	nonexistentProfileID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc string
		chID string
		thID string
		err  error
	}{
		{
			desc: "disconnect connected thing",
			chID: chID,
			thID: thID,
			err:  nil,
		},
		{
			desc: "disconnect non-connected thing",
			chID: chID,
			thID: thID,
			err:  errors.ErrNotFound,
		},
		{
			desc: "disconnect non-existing user",
			chID: chID,
			thID: thID,
			err:  errors.ErrNotFound,
		},
		{
			desc: "disconnect non-existing profile",
			chID: nonexistentProfileID,
			thID: thID,
			err:  errors.ErrNotFound,
		},
		{
			desc: "disconnect non-existing thing",
			chID: chID,
			thID: nonexistentThingID,
			err:  errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := profileRepo.Disconnect(context.Background(), tc.chID, []string{tc.thID})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveConnByThingKey(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)

	thID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	th := things.Thing{
		ID:      thID,
		GroupID: group.ID,
		Name:    thingName,
		Key:     thkey,
	}
	ths, err := thingRepo.Save(context.Background(), th)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	thID = ths[0].ID

	profileRepo := postgres.NewProfileRepository(dbMiddleware)
	chID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	chs, err := profileRepo.Save(context.Background(), things.Profile{
		ID:      chID,
		GroupID: group.ID,
		Name:    profileName,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chID = chs[0].ID
	err = profileRepo.Connect(context.Background(), chID, []string{thID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		chID      string
		key       string
		hasAccess bool
	}{
		"access check for thing that has access": {
			key:       th.Key,
			hasAccess: true,
		},
		"access check for thing without access": {
			key:       wrongID,
			hasAccess: false,
		},
	}

	for desc, tc := range cases {
		_, err := profileRepo.RetrieveConnByThingKey(context.Background(), tc.key)
		hasAccess := err == nil
		assert.Equal(t, tc.hasAccess, hasAccess, fmt.Sprintf("%s: expected %t got %t\n", desc, tc.hasAccess, hasAccess))
	}
}

func TestRetrieveAllProfiles(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	err := cleanTestTable(context.Background(), "profiles", dbMiddleware)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	metadata := things.Metadata{
		"field": "value",
	}
	metaNum := uint64(3)
	group := createGroup(t, dbMiddleware)
	chs := []things.Profile{}
	n := uint64(101)
	for i := uint64(0); i < n; i++ {
		suffix := i + 1
		ch := things.Profile{
			ID:      fmt.Sprintf("%s%012d", prefixID, suffix),
			GroupID: group.ID,
			Name:    fmt.Sprintf("%s-%d", profileName, suffix),
		}
		if i < metaNum {
			ch.Metadata = metadata
		}

		chs = append(chs, ch)
	}

	_, err = profileRepo.Save(context.Background(), chs...)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		size uint64
		err  error
	}{
		"retrieve all profiles without limit": {
			size: n,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		profiles, err := profileRepo.RetrieveAll(context.Background())
		size := uint64(len(profiles))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestRetrieveAllConnections(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	err := cleanTestTable(context.Background(), "connections", dbMiddleware)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	group := createGroup(t, dbMiddleware)

	n := uint64(101)
	for i := uint64(0); i < n; i++ {
		suffix := n + i + 1
		thkey, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		th := things.Thing{
			ID:       fmt.Sprintf("%s%012d", prefixID, suffix),
			GroupID:  group.ID,
			Name:     fmt.Sprintf("%s-%d", thingName, suffix),
			Key:      thkey,
			Metadata: things.Metadata{},
		}
		ths, err := thingRepo.Save(context.Background(), th)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		thID := ths[0].ID

		chs, err := profileRepo.Save(context.Background(), things.Profile{
			ID:      fmt.Sprintf("%s%012d", prefixID, suffix),
			GroupID: group.ID,
			Name:    fmt.Sprintf("%s-%d", profileName, suffix),
		})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		chID := chs[0].ID

		err = profileRepo.Connect(context.Background(), chID, []string{thID})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	cases := map[string]struct {
		size uint64
		err  error
	}{
		"retrieve all profiles": {
			size: n,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		connections, err := profileRepo.RetrieveAllConnections(context.Background())
		size := uint64(len(connections))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func testSortProfiles(t *testing.T, pm things.PageMetadata, chs []things.Profile) {
	switch pm.Order {
	case "name":
		current := chs[0]
		for _, res := range chs {
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
