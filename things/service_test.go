// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	authmock "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/things/mocks"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	wrongID        = ""
	wrongValue     = "wrong-value"
	adminEmail     = "admin@example.com"
	userEmail      = "user@example.com"
	otherUserEmail = "other.user@example.com"
	adminToken     = adminEmail
	token          = userEmail
	otherToken     = otherUserEmail
	password       = "password"
	n              = uint64(102)
	nUser          = n * 2
	nAdmin         = n * 3
	orgID          = "374106f7-030e-4881-8ab0-151195c29f92"
	prefixID       = "fe6b4e92-cc98-425e-b0aa-"
	prefixName     = "test-"
)

var (
	thing       = things.Thing{Name: "test"}
	thingList   = [n]things.Thing{}
	profileList = [n]things.Profile{}
	profile     = things.Profile{Name: "test"}
	thsExtID    = []things.Thing{{ID: prefixID + "000000000001", Name: "a"}, {ID: prefixID + "000000000002", Name: "b"}}
	chsExtID    = []things.Profile{{ID: prefixID + "000000000001", Name: "a"}, {ID: prefixID + "000000000002", Name: "b"}}
	user        = users.User{ID: "574106f7-030e-4881-8ab0-151195c29f94", Email: userEmail, Password: password, Role: auth.Editor}
	otherUser   = users.User{ID: "674106f7-030e-4881-8ab0-151195c29f95", Email: otherUserEmail, Password: password, Role: auth.Owner}
	admin       = users.User{ID: "874106f7-030e-4881-8ab0-151195c29f97", Email: adminEmail, Password: password, Role: auth.RootSub}
	usersList   = []users.User{admin, user, otherUser}
	group       = things.Group{OrgID: orgID, Name: "test-group", Description: "test-group-desc"}
)

func newService() things.Service {
	auth := authmock.NewAuthService(admin.ID, usersList)
	conns := make(chan mocks.Connection)
	thingsRepo := mocks.NewThingRepository(conns)
	profilesRepo := mocks.NewProfileRepository(thingsRepo, conns)
	groupsRepo := mocks.NewGroupRepository()
	rolesRepo := mocks.NewRolesRepository()
	profileCache := mocks.NewProfileCache()
	thingCache := mocks.NewThingCache()
	idProvider := uuid.NewMock()

	return things.New(auth, nil, thingsRepo, profilesRepo, groupsRepo, rolesRepo, profileCache, thingCache, idProvider)
}

func TestInit(t *testing.T) {
	for i := uint64(0); i < n; i++ {
		thingList[i].Name = fmt.Sprintf("name-%d", i+1)
		thingList[i].ID = fmt.Sprintf("%s%012d", prefixID, i+1)
		thingList[i].Key = fmt.Sprintf("%s1%011d", prefixID, i+1)
	}
}

func TestCreateThings(t *testing.T) {
	svc := newService()
	gr, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := gr[0].ID
	thsExtID[0].GroupID = grID
	thsExtID[1].GroupID = grID

	cases := []struct {
		desc   string
		things []things.Thing
		token  string
		err    error
	}{
		{
			desc:   "create new things",
			things: []things.Thing{{Name: "a", GroupID: grID}, {Name: "b", GroupID: grID}, {Name: "c", GroupID: grID}, {Name: "d", GroupID: grID}},
			token:  token,
			err:    nil,
		},
		{
			desc:   "create new thing with wrong group id",
			things: []things.Thing{{Name: "e", GroupID: wrongValue}},
			token:  token,
			err:    errors.ErrNotFound,
		},
		{
			desc:   "create thing with wrong credentials",
			things: []things.Thing{{Name: "f", GroupID: grID}},
			token:  wrongValue,
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "create new things with external UUID",
			things: thsExtID,
			token:  token,
			err:    nil,
		},
		{
			desc:   "create new things with external wrong UUID",
			things: []things.Thing{{ID: "b0aa-000000000001", Name: "a", GroupID: grID}, {ID: "b0aa-000000000002", Name: "b", GroupID: grID}},
			token:  token,
			err:    nil,
		},
	}

	for _, tc := range cases {
		_, err := svc.CreateThings(context.Background(), tc.token, tc.things...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateThing(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	thing.GroupID = gr.ID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]
	other := things.Thing{ID: wrongID, Key: "x"}

	cases := []struct {
		desc  string
		thing things.Thing
		token string
		err   error
	}{
		{
			desc:  "update existing thing",
			thing: th,
			token: token,
			err:   nil,
		},
		{
			desc:  "update thing with wrong credentials",
			thing: th,
			token: wrongValue,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "update non-existing thing",
			thing: other,
			token: token,
			err:   errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateThing(context.Background(), tc.token, tc.thing)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateKey(t *testing.T) {
	key := "new-key"
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	thing.GroupID = gr.ID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	cases := []struct {
		desc  string
		token string
		id    string
		key   string
		err   error
	}{
		{
			desc:  "update key of an existing thing",
			token: token,
			id:    th.ID,
			key:   key,
			err:   nil,
		},
		{
			desc:  "update key with invalid credentials",
			token: wrongValue,
			id:    th.ID,
			key:   key,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "update key of non-existing thing",
			token: token,
			id:    wrongID,
			key:   wrongValue,
			err:   errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateKey(context.Background(), tc.token, tc.id, tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewThing(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	thing.GroupID = gr.ID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	cases := map[string]struct {
		id    string
		token string
		err   error
	}{
		"view existing thing": {
			id:    th.ID,
			token: token,
			err:   nil,
		},
		"view existing thing as admin": {
			id:    th.ID,
			token: adminToken,
			err:   nil,
		},
		"view thing with wrong credentials": {
			id:    th.ID,
			token: wrongValue,
			err:   errors.ErrAuthentication,
		},
		"view non-existing thing": {
			id:    wrongID,
			token: token,
			err:   errors.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := svc.ViewThing(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListThings(t *testing.T) {
	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	grs2, err := svc.CreateGroups(context.Background(), otherToken, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr2 := grs2[0]

	m := make(map[string]interface{})
	m["serial"] = "123456"
	thingList[0].Metadata = m

	var ths1 []things.Thing
	var suffix uint64
	for i := uint64(0); i < n; i++ {
		suffix = i + 1
		th := thingList[i]
		th.GroupID = gr.ID
		th.Name = fmt.Sprintf("%s%d", prefixName, suffix)
		th.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		th.Key = fmt.Sprintf("%s%d", prefixID, suffix)

		ths1 = append(ths1, th)
	}

	var ths2 []things.Thing
	for i := uint64(0); i < n; i++ {
		suffix = n + i + 1
		th := thingList[i]
		th.GroupID = gr.ID
		th.Name = fmt.Sprintf("%s%d", prefixName, suffix)
		th.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		th.Key = fmt.Sprintf("%s%d", prefixID, suffix)

		ths2 = append(ths2, th)
	}

	var ths3 []things.Thing
	for i := uint64(0); i < n; i++ {
		suffix = (n * 2) + i + 1
		th := thingList[i]
		th.GroupID = gr2.ID
		th.Name = fmt.Sprintf("%s%d", prefixName, suffix)
		th.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		th.Key = fmt.Sprintf("%s%d", prefixID, suffix)

		ths3 = append(ths3, th)
	}

	_, err = svc.CreateThings(context.Background(), token, ths1...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	_, err = svc.CreateThings(context.Background(), otherToken, ths2...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	_, err = svc.CreateThings(context.Background(), otherToken, ths3...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		token        string
		pageMetadata things.PageMetadata
		size         uint64
		err          error
	}{
		"list all things": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  nUser,
			},
			size: nUser,
			err:  nil,
		},
		"list all things as admin": {
			token: adminToken,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  nAdmin,
			},
			size: nAdmin,
			err:  nil,
		},
		"list all things with no limit": {
			token: token,
			pageMetadata: things.PageMetadata{
				Limit: 0,
			},
			size: 0,
			err:  nil,
		},
		"list half": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: nUser / 2,
				Limit:  nUser,
			},
			size: nUser / 2,
			err:  nil,
		},
		"list last thing": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: uint64(nUser) - 1,
				Limit:  nUser,
			},
			size: 1,
			err:  nil,
		},
		"list empty set": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: uint64(nUser) + 1,
				Limit:  nUser,
			},
			size: 0,
			err:  nil,
		},
		"list with wrong credentials": {
			token: wrongValue,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  0,
			},
			size: 0,
			err:  errors.ErrAuthentication,
		},
		"list with existing name": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  nUser,
				Name:   "test-1",
			},
			size: 1,
			err:  nil,
		},
		"list with non-existent name": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  nUser,
				Name:   "wrong",
			},
			size: 0,
			err:  nil,
		},
		"list with metadata": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset:   0,
				Limit:    nUser,
				Metadata: m,
			},
			size: nUser,
			err:  nil,
		},
		"list all things sorted by name ascendant": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  nUser,
				Order:  "name",
				Dir:    "asc",
			},
			size: nUser,
			err:  nil,
		},
		"list all things sorted by name descendent": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  nUser,
				Order:  "name",
				Dir:    "desc",
			},
			size: nUser,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListThings(context.Background(), tc.token, tc.pageMetadata)
		size := uint64(len(page.Things))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

		// Check if Things list have been sorted properly
		testSortThings(t, tc.pageMetadata, page.Things)
	}
}

func TestListThingsByProfile(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profile.GroupID = gr.ID
	chs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	ch := chs[0]

	thsDisconNum := uint64(4)

	var ths []things.Thing
	for i := uint64(0); i < n; i++ {
		suffix := i + 1
		th := thingList[i]
		th.GroupID = gr.ID
		th.Name = fmt.Sprintf("%s%012d", prefixName, suffix)
		th.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		ths = append(ths, th)
	}

	thsc, err := svc.CreateThings(context.Background(), token, ths...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var thIDs []string
	for _, thID := range thsc {
		thIDs = append(thIDs, thID.ID)
	}

	err = svc.Connect(context.Background(), token, ch.ID, thIDs[0:n-thsDisconNum])
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	// Wait for things and profiles to connect
	time.Sleep(time.Second)

	cases := map[string]struct {
		token        string
		chID         string
		pageMetadata things.PageMetadata
		size         uint64
		err          error
	}{
		"list all things by existing profile": {
			token: token,
			chID:  ch.ID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n - thsDisconNum,
			err:  nil,
		},
		"list all things by existing profile with no limit": {
			token: token,
			chID:  ch.ID,
			pageMetadata: things.PageMetadata{
				Limit: 0,
			},
			size: n - thsDisconNum,
			err:  nil,
		},
		"list half of things by existing profile": {
			token: token,
			chID:  ch.ID,
			pageMetadata: things.PageMetadata{
				Offset: n / 2,
				Limit:  n,
			},
			size: (n / 2) - thsDisconNum,
			err:  nil,
		},
		"list last thing by existing profile": {
			token: token,
			chID:  ch.ID,
			pageMetadata: things.PageMetadata{
				Offset: n - 1 - thsDisconNum,
				Limit:  n,
			},
			size: 1,
			err:  nil,
		},
		"list empty set of things by existing profile": {
			token: token,
			chID:  ch.ID,
			pageMetadata: things.PageMetadata{
				Offset: n + 1,
				Limit:  n,
			},
			size: 0,
			err:  nil,
		},
		"list things by existing profile with wrong credentials": {
			token: wrongValue,
			chID:  ch.ID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  0,
			},
			size: 0,
			err:  errors.ErrAuthentication,
		},
		"list things by non-existent profile with wrong credentials": {
			token: token,
			chID:  "non-existent",
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: 0,
			err:  errors.ErrNotFound,
		},
		"list all things by profile sorted by name ascendant": {
			token: token,
			chID:  ch.ID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "asc",
			},
			size: n - thsDisconNum,
			err:  nil,
		},
		"list all things by profile sorted by name descendent": {
			token: token,
			chID:  ch.ID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "desc",
			},
			size: n - thsDisconNum,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListThingsByProfile(context.Background(), tc.token, tc.chID, tc.pageMetadata)
		size := uint64(len(page.Things))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

		// Check if Things by Profile list have been sorted properly
		testSortThings(t, tc.pageMetadata, page.Things)
	}
}

func TestRemoveThings(t *testing.T) {
	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	thingList[0].GroupID = gr.ID
	ths, err := svc.CreateThings(context.Background(), token, thingList[0])
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	sth := ths[0]

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "remove thing with wrong credentials",
			id:    sth.ID,
			token: wrongValue,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "remove existing thing",
			id:    sth.ID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove removed thing",
			id:    sth.ID,
			token: token,
			err:   errors.ErrNotFound,
		},
		{
			desc:  "remove non-existing thing",
			id:    wrongID,
			token: token,
			err:   errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveThings(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestCreateProfiles(t *testing.T) {
	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID
	chsExtID[0].GroupID = grID
	chsExtID[1].GroupID = grID
	cases := []struct {
		desc     string
		profiles []things.Profile
		token    string
		err      error
	}{
		{
			desc:     "create new profiles",
			profiles: []things.Profile{{Name: "a", GroupID: grID}, {Name: "b", GroupID: grID}, {Name: "c", GroupID: grID}, {Name: "d", GroupID: grID}},
			token:    token,
			err:      nil,
		},
		{
			desc:     "create new profile with wrong group id",
			profiles: []things.Profile{{Name: "e", GroupID: wrongValue}},
			token:    token,
			err:      errors.ErrNotFound,
		},
		{
			desc:     "create profile with wrong credentials",
			profiles: []things.Profile{{Name: "f", GroupID: grID}},
			token:    wrongValue,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "create new profiles with external UUID",
			profiles: chsExtID,
			token:    token,
			err:      nil,
		},
		{
			desc:     "create new profiles with invalid external UUID",
			profiles: []things.Profile{{ID: "b0aa-000000000001", Name: "a", GroupID: grID}, {ID: "b0aa-000000000002", Name: "b", GroupID: grID}},
			token:    token,
			err:      nil,
		},
	}

	for _, cc := range cases {
		_, err := svc.CreateProfiles(context.Background(), cc.token, cc.profiles...)
		assert.True(t, errors.Contains(err, cc.err), fmt.Sprintf("%s: expected %s got %s\n", cc.desc, cc.err, err))
	}
}

func TestUpdateProfile(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profile.GroupID = gr.ID
	chs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]
	other := things.Profile{ID: wrongID}

	cases := []struct {
		desc    string
		profile things.Profile
		token   string
		err     error
	}{
		{
			desc:    "update existing profile",
			profile: ch,
			token:   token,
			err:     nil,
		},
		{
			desc:    "update profile with wrong credentials",
			profile: ch,
			token:   wrongValue,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "update non-existing profile",
			profile: other,
			token:   token,
			err:     errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateProfile(context.Background(), tc.token, tc.profile)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewProfile(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profile.GroupID = gr.ID
	chs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]

	cases := map[string]struct {
		id       string
		token    string
		err      error
		metadata map[string]interface{}
	}{
		"view existing profile": {
			id:    ch.ID,
			token: token,
			err:   nil,
		},
		"view existing profile as admin": {
			id:    ch.ID,
			token: adminToken,
			err:   nil,
		},
		"view profile with wrong credentials": {
			id:    ch.ID,
			token: wrongValue,
			err:   errors.ErrAuthentication,
		},
		"view non-existing profile": {
			id:    wrongID,
			token: token,
			err:   errors.ErrNotFound,
		},
		"view profile with metadata": {
			id:    wrongID,
			token: token,
			err:   errors.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := svc.ViewProfile(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListProfiles(t *testing.T) {
	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	grs2, err := svc.CreateGroups(context.Background(), otherToken, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr2 := grs2[0]

	meta := things.Metadata{}
	meta["name"] = "test-profile"
	profile.Metadata = meta

	var chs1 []things.Profile
	var suffix uint64
	for i := uint64(0); i < n; i++ {
		suffix = i + 1
		ch := profileList[i]
		ch.GroupID = gr.ID
		ch.Name = fmt.Sprintf("%s%d", prefixName, suffix)
		ch.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		chs1 = append(chs1, ch)
	}

	var chs2 []things.Profile
	for i := uint64(0); i < n; i++ {
		suffix = n + i + 1
		ch := profileList[i]
		ch.GroupID = gr.ID
		ch.Name = fmt.Sprintf("%s%d", prefixName, suffix)
		ch.ID = fmt.Sprintf("%s%012d", prefixID, suffix)

		chs2 = append(chs2, ch)
	}

	var chs3 []things.Profile
	for i := uint64(0); i < n; i++ {
		suffix = (n * 2) + i + 1
		ch := profileList[i]
		ch.GroupID = gr2.ID
		ch.Name = fmt.Sprintf("%s%d", prefixName, suffix)
		ch.ID = fmt.Sprintf("%s%012d", prefixID, suffix)

		chs3 = append(chs3, ch)
	}

	_, err = svc.CreateProfiles(context.Background(), token, chs1...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	_, err = svc.CreateProfiles(context.Background(), otherToken, chs2...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	_, err = svc.CreateProfiles(context.Background(), otherToken, chs3...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		token        string
		pageMetadata things.PageMetadata
		size         uint64
		err          error
	}{
		"list all profiles": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  nUser,
			},
			size: nUser,
			err:  nil,
		},
		"list all profiles as admin": {
			token: adminToken,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  nAdmin,
			},
			size: nAdmin,
			err:  nil,
		},
		"list all profiles with no limit": {
			token: token,
			pageMetadata: things.PageMetadata{
				Limit: 0,
			},
			size: 0,
			err:  nil,
		},
		"list half": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: nUser / 2,
				Limit:  nUser,
			},
			size: nUser / 2,
			err:  nil,
		},
		"list last profile": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: nUser - 1,
				Limit:  nUser,
			},
			size: 1,
			err:  nil,
		},
		"list empty set": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: uint64(nUser) + 1,
				Limit:  nUser,
			},
			size: 0,
			err:  nil,
		},
		"list with wrong credentials": {
			token: wrongValue,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  0,
			},
			size: 0,
			err:  errors.ErrAuthentication,
		},
		"list with existing name": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  nUser,
				Name:   "test-1",
			},
			size: 1,
			err:  nil,
		},
		"list with non-existent name": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  nUser,
				Name:   "wrong",
			},
			size: 0,
			err:  nil,
		},
		"list all profiles with metadata": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset:   0,
				Limit:    nUser,
				Metadata: meta,
			},
			size: nUser,
			err:  nil,
		},
		"list all profiles sorted by name ascendant": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  nUser,
				Order:  "name",
				Dir:    "asc",
			},
			size: nUser,
			err:  nil,
		},
		"list all profiles sorted by name descendent": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  nUser,
				Order:  "name",
				Dir:    "desc",
			},
			size: nUser,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListProfiles(context.Background(), tc.token, tc.pageMetadata)
		size := uint64(len(page.Profiles))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

		// Check if profiles list have been sorted properly
		testSortProfiles(t, tc.pageMetadata, page.Profiles)
	}
}

func TestViewProfileByThing(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	thingList[0].GroupID = grs[0].ID

	ths, err := svc.CreateThings(context.Background(), token, thingList[0])
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th := ths[0]

	c := profile
	c.Name = "test-profile"
	c.GroupID = grs[0].ID

	chs, err := svc.CreateProfiles(context.Background(), token, c)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	ch := chs[0]

	err = svc.Connect(context.Background(), token, ch.ID, []string{th.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	// Wait for things and profiles to connect.
	time.Sleep(time.Second)

	cases := map[string]struct {
		token   string
		thID    string
		profile things.Profile
		err     error
	}{
		"view profile by existing thing": {
			token:   token,
			thID:    th.ID,
			profile: ch,
			err:     nil,
		},
		"view profile by existing thing as admin": {
			token:   adminToken,
			thID:    th.ID,
			profile: ch,
			err:     nil,
		},
		"view profile by existing thing with wrong credentials": {
			token:   wrongValue,
			thID:    th.ID,
			profile: things.Profile{},
			err:     errors.ErrAuthentication,
		},
		"view profile by non-existent thing": {
			token:   token,
			thID:    "non-existent",
			profile: things.Profile{},
			err:     errors.ErrNotFound,
		},
		"view profile by existent thing with invalid token": {
			token:   wrongValue,
			thID:    th.ID,
			profile: things.Profile{},
			err:     errors.ErrAuthentication,
		},
	}

	for desc, tc := range cases {
		ch, err := svc.ViewProfileByThing(context.Background(), tc.token, tc.thID)
		assert.Equal(t, tc.profile, ch, fmt.Sprintf("%s: expected %v got %v\n", desc, tc.profile, ch))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRemoveProfile(t *testing.T) {
	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profile.GroupID = gr.ID
	chs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "remove profile with wrong credentials",
			id:    ch.ID,
			token: wrongValue,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "remove existing profile",
			id:    ch.ID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove removed profile",
			id:    ch.ID,
			token: token,
			err:   errors.ErrNotFound,
		},
		{
			desc:  "remove non-existing profile",
			id:    wrongID,
			token: token,
			err:   errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveProfiles(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestConnect(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	thingList[0].GroupID = gr.ID
	ths, err := svc.CreateThings(context.Background(), token, thingList[0])
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	profile.GroupID = gr.ID
	chs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]

	cases := []struct {
		desc      string
		token     string
		profileID string
		thingID   string
		err       error
	}{
		{
			desc:      "connect thing",
			token:     token,
			profileID: ch.ID,
			thingID:   th.ID,
			err:       nil,
		},
		{
			desc:      "connect thing with wrong credentials",
			token:     wrongValue,
			profileID: ch.ID,
			thingID:   th.ID,
			err:       errors.ErrAuthentication,
		},
		{
			desc:      "connect thing to non-existing profile",
			token:     token,
			profileID: wrongID,
			thingID:   th.ID,
			err:       errors.ErrNotFound,
		},
		{
			desc:      "connect non-existing thing to profile",
			token:     token,
			profileID: ch.ID,
			thingID:   wrongID,
			err:       errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.Connect(context.Background(), tc.token, tc.profileID, []string{tc.thingID})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestDisconnect(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	thingList[0].GroupID = gr.ID
	ths, err := svc.CreateThings(context.Background(), token, thingList[0])
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	profile.GroupID = gr.ID
	chs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]

	err = svc.Connect(context.Background(), token, ch.ID, []string{th.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc      string
		token     string
		profileID string
		thingID   string
		err       error
	}{
		{
			desc:      "disconnect connected thing",
			token:     token,
			profileID: ch.ID,
			thingID:   th.ID,
			err:       nil,
		},
		{
			desc:      "disconnect with wrong credentials",
			token:     wrongValue,
			profileID: ch.ID,
			thingID:   th.ID,
			err:       errors.ErrAuthentication,
		},
		{
			desc:      "disconnect from non-existing profile",
			token:     token,
			profileID: wrongID,
			thingID:   th.ID,
			err:       errors.ErrNotFound,
		},
		{
			desc:      "disconnect non-existing thing",
			token:     token,
			profileID: ch.ID,
			thingID:   wrongID,
			err:       errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.Disconnect(context.Background(), tc.token, tc.profileID, []string{tc.thingID})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}

}

func TestGetConnByKey(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	thingList[0].GroupID = gr.ID
	ths, err := svc.CreateThings(context.Background(), token, thingList[0])
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	profile.GroupID = gr.ID
	chs, err := svc.CreateProfiles(context.Background(), token, profile, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]

	err = svc.Connect(context.Background(), token, ch.ID, []string{th.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := map[string]struct {
		key string
		err error
	}{
		"allowed access": {
			key: th.Key,
			err: nil,
		},
		"non-existing thing": {
			key: wrongValue,
			err: errors.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := svc.GetConnByKey(context.Background(), tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected '%s' got '%s'\n", desc, tc.err, err))
	}
}

func TestIdentify(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	thingList[0].GroupID = gr.ID
	ths, err := svc.CreateThings(context.Background(), token, thingList[0])
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	cases := map[string]struct {
		token string
		id    string
		err   error
	}{
		"identify existing thing": {
			token: th.Key,
			id:    th.ID,
			err:   nil,
		},
		"identify non-existing thing": {
			token: wrongValue,
			id:    wrongID,
			err:   errors.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		id, err := svc.Identify(context.Background(), tc.token)
		assert.Equal(t, tc.id, id, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.id, id))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestBackup(t *testing.T) {
	svc := newService()

	var groups []things.Group
	for i := uint64(0); i < 10; i++ {
		num := strconv.FormatUint(i, 10)
		group := things.Group{
			Name:        "test-group-" + num,
			Description: "test group desc",
		}

		grs, err := svc.CreateGroups(context.Background(), token, group)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		gr := grs[0]

		groups = append(groups, gr)
	}
	gr := groups[0]

	thing.GroupID = gr.ID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th := ths[0]

	var chs []things.Profile
	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		ch := profile
		ch.Name = fmt.Sprintf("name-%d", i)
		ch.GroupID = gr.ID
		chs = append(chs, ch)
	}

	chsc, err := svc.CreateProfiles(context.Background(), token, chs...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	ch := chsc[0]

	err = svc.Connect(context.Background(), token, ch.ID, []string{th.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	// Wait for things and profiles to connect.
	time.Sleep(time.Second)

	connections := []things.Connection{}
	connections = append(connections, things.Connection{
		ProfileID: ch.ID,
		ThingID:   th.ID,
	})

	backup := things.Backup{
		Groups:      groups,
		Things:      ths,
		Profiles:    chsc,
		Connections: connections,
	}

	cases := map[string]struct {
		token  string
		backup things.Backup
		err    error
	}{
		"list backups": {
			token:  adminToken,
			backup: backup,
			err:    nil,
		},
		"list backups with invalid token": {
			token:  wrongValue,
			backup: things.Backup{},
			err:    errors.ErrAuthentication,
		},
		"list backups with empty token": {
			token:  "",
			backup: things.Backup{},
			err:    errors.ErrAuthentication,
		},
	}

	for desc, tc := range cases {
		backup, err := svc.Backup(context.Background(), tc.token)
		groupSize := len(backup.Groups)
		thingsSize := len(backup.Things)
		profilesSize := len(backup.Profiles)
		connectionsSize := len(backup.Connections)

		assert.Equal(t, len(tc.backup.Groups), groupSize, fmt.Sprintf("%s: expected %v got %d\n", desc, len(tc.backup.Groups), groupSize))
		assert.Equal(t, len(tc.backup.Things), thingsSize, fmt.Sprintf("%s: expected %v got %d\n", desc, len(tc.backup.Things), thingsSize))
		assert.Equal(t, len(tc.backup.Profiles), profilesSize, fmt.Sprintf("%s: expected %v got %d\n", desc, len(tc.backup.Profiles), profilesSize))
		assert.Equal(t, len(tc.backup.Connections), connectionsSize, fmt.Sprintf("%s: expected %v got %d\n", desc, len(tc.backup.Connections), connectionsSize))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

	}
}

func TestRestore(t *testing.T) {
	svc := newService()
	idProvider := uuid.New()

	thkey, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	thID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	chID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var groups []things.Group
	for i := uint64(0); i < 10; i++ {
		num := strconv.FormatUint(i, 10)
		gr := things.Group{
			ID:          fmt.Sprintf("%s%012d", prefixID, i+1),
			Name:        "test-group-" + num,
			Description: "test group desc",
		}

		groups = append(groups, gr)
	}

	ths := []things.Thing{
		{
			ID:       thID,
			Name:     "testThing",
			Key:      thkey,
			Metadata: map[string]interface{}{},
		},
	}
	th := ths[0]

	var chs []things.Profile
	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		ch := things.Profile{
			ID:       chID,
			Name:     "testProfile",
			Metadata: map[string]interface{}{},
		}
		ch.Name = fmt.Sprintf("name-%d", i)
		chs = append(chs, ch)
	}
	ch := chs[0]

	var connections []things.Connection
	conn := things.Connection{
		ProfileID: ch.ID,
		ThingID:   th.ID,
	}

	connections = append(connections, conn)

	backup := things.Backup{
		Groups:      groups,
		Things:      ths,
		Profiles:    chs,
		Connections: connections,
	}

	cases := map[string]struct {
		token  string
		backup things.Backup
		err    error
	}{
		"Restore backup": {
			token:  adminToken,
			backup: backup,
			err:    nil,
		},
		"Restore backup with invalid token": {
			token: wrongValue,
			err:   errors.ErrAuthentication,
		},
		"Restore backup with empty token": {
			token: "",
			err:   errors.ErrAuthentication,
		},
	}

	for desc, tc := range cases {
		err := svc.Restore(context.Background(), tc.token, tc.backup)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func testSortThings(t *testing.T, pm things.PageMetadata, ths []things.Thing) {
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
