// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/things/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const prefixID = "fe6b4e92-cc98-425e-b0aa-"

func TestSaveProfiles(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)

	prs := []things.Profile{}
	for i := 1; i <= 5; i++ {
		id, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		pr := things.Profile{
			ID:      id,
			GroupID: group.ID,
			Name:    fmt.Sprintf("%s-%d", profileName, i),
		}
		prs = append(prs, pr)
	}
	id2, _ := idProvider.ID()
	prs = append(prs, things.Profile{ID: id2, GroupID: group.ID, Name: ""})
	id := prs[0].ID

	cases := []struct {
		desc     string
		profiles []things.Profile
		err      error
	}{
		{
			desc:     "save new profiles",
			profiles: prs,
			err:      nil,
		},
		{
			desc:     "save profiles that already exist",
			profiles: prs,
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
	dbMiddleware := dbutil.NewDatabase(db)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)

	id, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pr := things.Profile{
		ID:      id,
		GroupID: group.ID,
		Name:    profileName,
	}

	prs, err := profileRepo.Save(context.Background(), pr)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	pr.ID = prs[0].ID

	nonexistentProfileID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc    string
		profile things.Profile
		err     error
	}{
		{
			desc:    "update existing profile",
			profile: pr,
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
	dbMiddleware := dbutil.NewDatabase(db)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)
	prID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	nonexistentProfileID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	c := things.Profile{
		ID:      prID,
		GroupID: group.ID,
		Name:    profileName,
	}
	prs, err := profileRepo.Save(context.Background(), c)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	th := things.Thing{
		ID:        thID,
		GroupID:   group.ID,
		ProfileID: prID,
		Name:      thingName,
		Key:       thkey,
	}
	_, err = thingRepo.Save(context.Background(), th)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		ID  string
		err error
	}{
		"retrieve profile": {
			ID:  pr.ID,
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
	dbMiddleware := dbutil.NewDatabase(db)
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
	var prs []things.Profile
	n := uint64(101)

	for i := uint64(0); i < n; i++ {
		suffix := i + 1
		pr := things.Profile{
			ID:      fmt.Sprintf("%s%012d", prefixID, suffix),
			GroupID: group.ID,
			Name:    fmt.Sprintf("%s-%d", profileName, suffix),
		}
		if i < metaNum {
			pr.Metadata = metadata
		}

		prs = append(prs, pr)
	}

	_, err := profileRepo.Save(context.Background(), prs...)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		size         uint64
		pageMetadata apiutil.PageMetadata
	}{
		"retrieve all profiles by group IDs": {
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Dir:    descDir,
				Order:  idOrder,
			},
			size: n,
		},
		"retrieve all profiles by group IDs without limit": {
			pageMetadata: apiutil.PageMetadata{
				Limit: 0,
				Dir:   descDir,
				Order: idOrder,
			},
			size: n,
		},
		"retrieve subset of profiles by group IDs": {
			pageMetadata: apiutil.PageMetadata{
				Offset: offset,
				Limit:  n,
				Dir:    descDir,
				Order:  idOrder,
			},
			size: n - offset,
		},
		"retrieve profiles by group IDs with existing name": {
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Name:   "test-profile-101",
				Dir:    descDir,
				Order:  idOrder,
			},
			size: 1,
		},
		"retrieve all profiles by group IDs with non-existing name": {
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Name:   "wrong",
				Dir:    descDir,
				Order:  idOrder,
			},
			size: 0,
		},
		"retrieve all profiles by group IDs with existing metadata": {
			pageMetadata: apiutil.PageMetadata{
				Offset:   0,
				Limit:    n,
				Metadata: metadata,
				Dir:      descDir,
				Order:    idOrder,
			},
			size: metaNum,
		},
		"retrieve all profiles by group IDs with non-existing metadata": {
			pageMetadata: apiutil.PageMetadata{
				Offset:   0,
				Limit:    n,
				Metadata: wrongMeta,
				Dir:      descDir,
				Order:    idOrder,
			},
		},
		"retrieve profiles by group IDs sorted by name ascendant": {
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  nameOrder,
				Dir:    ascDir,
			},
			size: n,
		},
		"retrieve profiles by group IDs sorted by name descendent": {
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
		page, err := profileRepo.RetrieveByGroupIDs(context.Background(), []string{group.ID}, tc.pageMetadata)
		size := uint64(len(page.Profiles))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))

		// Check if Profiles list have been sorted properly
		testSortProfiles(t, tc.pageMetadata, page.Profiles)
	}
}

func TestRetrieveProfileByThing(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)
	prID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thKey, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	nonexistentThingID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	c := things.Profile{
		ID:       prID,
		GroupID:  group.ID,
		Name:     profileName,
		Config:   map[string]interface{}{},
		Metadata: things.Metadata{},
	}
	prs, err := profileRepo.Save(context.Background(), c)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	th := things.Thing{
		ID:        thID,
		GroupID:   group.ID,
		ProfileID: prID,
		Name:      thingName,
		Key:       thKey,
	}
	_, err = thingRepo.Save(context.Background(), th)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		thID    string
		profile things.Profile
		err     error
	}{
		"retrieve profile by thing": {
			thID:    thID,
			profile: pr,
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
		pr, err := profileRepo.RetrieveByThing(context.Background(), tc.thID)
		assert.Equal(t, tc.profile, pr, fmt.Sprintf("%s: expected %v got %v\n", desc, tc.profile, pr))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestRemoveProfile(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	group := createGroup(t, dbMiddleware)

	prID1 := generateUUID(t)
	prID2 := generateUUID(t)
	thID := generateUUID(t)
	thKey := generateUUID(t)

	_, err := profileRepo.Save(context.Background(), things.Profile{
		ID:      prID1,
		GroupID: group.ID,
		Name:    profileName,
	},
		things.Profile{
			ID:      prID2,
			GroupID: group.ID,
			Name:    profileName + "2",
		})

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	th := things.Thing{
		ID:        thID,
		GroupID:   group.ID,
		ProfileID: prID2,
		Name:      thingName,
		Key:       thKey,
	}
	_, err = thingRepo.Save(context.Background(), th)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := map[string]struct {
		prID string
		err  error
	}{
		"remove non-existing profile": {
			prID: "wrong",
			err:  errors.ErrRemoveEntity,
		},
		"remove profile": {
			prID: prID1,
			err:  nil,
		},
		"remove assigned profile": {
			prID: prID2,
			err:  errors.ErrRemoveEntity,
		},
	}

	for desc, tc := range cases {
		err := profileRepo.Remove(context.Background(), tc.prID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRetrieveAllProfiles(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	profileRepo := postgres.NewProfileRepository(dbMiddleware)

	err := cleanTestTable(context.Background(), "things", dbMiddleware)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	err = cleanTestTable(context.Background(), "profiles", dbMiddleware)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	metadata := things.Metadata{
		"field": "value",
	}
	metaNum := uint64(3)
	group := createGroup(t, dbMiddleware)
	prs := []things.Profile{}
	n := uint64(101)
	for i := uint64(0); i < n; i++ {
		suffix := i + 1
		pr := things.Profile{
			ID:      fmt.Sprintf("%s%012d", prefixID, suffix),
			GroupID: group.ID,
			Name:    fmt.Sprintf("%s-%d", profileName, suffix),
		}
		if i < metaNum {
			pr.Metadata = metadata
		}

		prs = append(prs, pr)
	}

	_, err = profileRepo.Save(context.Background(), prs...)
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

func testSortProfiles(t *testing.T, pm apiutil.PageMetadata, prs []things.Profile) {
	switch pm.Order {
	case "name":
		current := prs[0]
		for _, res := range prs {
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
