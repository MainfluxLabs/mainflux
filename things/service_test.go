// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

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
	prefix         = "fe6b4e92-cc98-425e-b0aa-"
)

var (
	thing     = things.Thing{Name: "test"}
	thingList = [n]things.Thing{}
	channel   = things.Channel{Name: "test"}
	thsExtID  = []things.Thing{{ID: prefix + "000000000001", Name: "a"}, {ID: prefix + "000000000002", Name: "b"}}
	chsExtID  = []things.Channel{{ID: prefix + "000000000001", Name: "a"}, {ID: prefix + "000000000002", Name: "b"}}
	user      = users.User{ID: "574106f7-030e-4881-8ab0-151195c29f94", Email: userEmail, Password: password}
	otherUser = users.User{Email: otherUserEmail, Password: password}
	admin     = users.User{Email: adminEmail, Password: password}
	usersList = []users.User{admin, user, otherUser}
	group     = things.Group{Name: "test-group", Description: "test-group-desc"}
)

func newService() things.Service {
	auth := authmock.NewAuthService(admin.ID, usersList)
	conns := make(chan mocks.Connection)
	thingsRepo := mocks.NewThingRepository(conns)
	channelsRepo := mocks.NewChannelRepository(thingsRepo, conns)
	groupsRepo := mocks.NewGroupRepository()
	rolesRepo := mocks.NewRolesRepository()
	chanCache := mocks.NewChannelCache()
	thingCache := mocks.NewThingCache()
	idProvider := uuid.NewMock()

	return things.New(auth, nil, thingsRepo, channelsRepo, groupsRepo, rolesRepo, chanCache, thingCache, idProvider)
}

func TestInit(t *testing.T) {
	for i := uint64(0); i < n; i++ {
		thingList[i].Name = fmt.Sprintf("name-%d", i+1)
		thingList[i].ID = fmt.Sprintf("%s%012d", prefix, i+1)
		thingList[i].Key = fmt.Sprintf("%s1%011d", prefix, i+1)
	}
}

func TestCreateThings(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc   string
		things []things.Thing
		token  string
		err    error
	}{
		{
			desc:   "create new things",
			things: []things.Thing{{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}},
			token:  token,
			err:    nil,
		},
		{
			desc:   "create thing with wrong credentials",
			things: []things.Thing{{Name: "e"}},
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
			things: []things.Thing{{ID: "b0aa-000000000001", Name: "a"}, {ID: "b0aa-000000000002", Name: "b"}},
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

	m := make(map[string]interface{})
	m["serial"] = "123456"
	thingList[0].Metadata = m

	var ths1 []things.Thing
	for i := uint64(0); i < n; i++ {
		th := thingList[i]
		ths1 = append(ths1, th)
	}

	var ths2 []things.Thing
	for i := uint64(0); i < n; i++ {
		th := thingList[i]
		th.OwnerID = userEmail
		th.Key = fmt.Sprintf("%s2%011d", prefix, i+1)
		th.ID = fmt.Sprintf("%s2%012d", prefix, i+1)
		ths2 = append(ths2, th)
	}

	_, err := svc.CreateThings(context.Background(), token, ths1...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	_, err = svc.CreateThings(context.Background(), otherToken, ths2...)
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
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		"list all things as admin": {
			token: adminToken,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n * 2,
				Total:  n * 2,
			},
			size: n * 2,
			err:  nil,
		},
		"list all things with no limit": {
			token: token,
			pageMetadata: things.PageMetadata{
				Limit: 0,
			},
			size: n,
			err:  nil,
		},
		"list half": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: n / 2,
				Limit:  n,
			},
			size: n / 2,
			err:  nil,
		},
		"list last thing": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: n - 1,
				Limit:  n,
			},
			size: 1,
			err:  nil,
		},
		"list empty set": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: n + 1,
				Limit:  n,
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
		"list with metadata": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset:   0,
				Limit:    n,
				Metadata: m,
			},
			size: n,
			err:  nil,
		},
		"list all things sorted by name ascendent": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "asc",
			},
			size: n,
			err:  nil,
		},
		"list all things sorted by name descendent": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "desc",
			},
			size: n,
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

func TestListThingsByChannel(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	channel.GroupID = gr.ID
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	ch := chs[0]

	thsDisconNum := uint64(4)

	var ths []things.Thing
	for i := uint64(0); i < n; i++ {
		th := thingList[i]
		th.GroupID = gr.ID
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

	// Wait for things and channels to connect
	time.Sleep(time.Second)

	cases := map[string]struct {
		token        string
		chID         string
		pageMetadata things.PageMetadata
		size         uint64
		err          error
	}{
		"list all things by existing channel": {
			token: token,
			chID:  ch.ID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n - thsDisconNum,
			err:  nil,
		},
		"list all things by existing channel with no limit": {
			token: token,
			chID:  ch.ID,
			pageMetadata: things.PageMetadata{
				Limit: 0,
			},
			size: n - thsDisconNum,
			err:  nil,
		},
		"list half of things by existing channel": {
			token: token,
			chID:  ch.ID,
			pageMetadata: things.PageMetadata{
				Offset: n / 2,
				Limit:  n,
			},
			size: (n / 2) - thsDisconNum,
			err:  nil,
		},
		"list last thing by existing channel": {
			token: token,
			chID:  ch.ID,
			pageMetadata: things.PageMetadata{
				Offset: n - 1 - thsDisconNum,
				Limit:  n,
			},
			size: 1,
			err:  nil,
		},
		"list empty set of things by existing channel": {
			token: token,
			chID:  ch.ID,
			pageMetadata: things.PageMetadata{
				Offset: n + 1,
				Limit:  n,
			},
			size: 0,
			err:  nil,
		},
		"list things by existing channel with wrong credentials": {
			token: wrongValue,
			chID:  ch.ID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  0,
			},
			size: 0,
			err:  errors.ErrAuthentication,
		},
		"list things by non-existent channel with wrong credentials": {
			token: token,
			chID:  "non-existent",
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: 0,
			err:  errors.ErrNotFound,
		},
		"list all things by channel sorted by name ascendent": {
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
		"list all things by channel sorted by name descendent": {
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
		page, err := svc.ListThingsByChannel(context.Background(), tc.token, tc.chID, tc.pageMetadata)
		size := uint64(len(page.Things))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

		// Check if Things by Channel list have been sorted properly
		testSortThings(t, tc.pageMetadata, page.Things)
	}
}

func TestRemoveThings(t *testing.T) {
	svc := newService()
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

func TestCreateChannels(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc     string
		channels []things.Channel
		token    string
		err      error
	}{
		{
			desc:     "create new channels",
			channels: []things.Channel{{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}},
			token:    token,
			err:      nil,
		},
		{
			desc:     "create channel with wrong credentials",
			channels: []things.Channel{{Name: "e"}},
			token:    wrongValue,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "create new channels with external UUID",
			channels: chsExtID,
			token:    token,
			err:      nil,
		},
		{
			desc:     "create new channels with invalid external UUID",
			channels: []things.Channel{{ID: "b0aa-000000000001", Name: "a"}, {ID: "b0aa-000000000002", Name: "b"}},
			token:    token,
			err:      nil,
		},
	}

	for _, cc := range cases {
		_, err := svc.CreateChannels(context.Background(), cc.token, cc.channels...)
		assert.True(t, errors.Contains(err, cc.err), fmt.Sprintf("%s: expected %s got %s\n", cc.desc, cc.err, err))
	}
}

func TestUpdateChannel(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	channel.GroupID = gr.ID
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]
	other := things.Channel{ID: wrongID}

	cases := []struct {
		desc    string
		channel things.Channel
		token   string
		err     error
	}{
		{
			desc:    "update existing channel",
			channel: ch,
			token:   token,
			err:     nil,
		},
		{
			desc:    "update channel with wrong credentials",
			channel: ch,
			token:   wrongValue,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "update non-existing channel",
			channel: other,
			token:   token,
			err:     errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateChannel(context.Background(), tc.token, tc.channel)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewChannel(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	channel.GroupID = gr.ID
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]

	cases := map[string]struct {
		id       string
		token    string
		err      error
		metadata map[string]interface{}
	}{
		"view existing channel": {
			id:    ch.ID,
			token: token,
			err:   nil,
		},
		"view existing channel as admin": {
			id:    ch.ID,
			token: adminToken,
			err:   nil,
		},
		"view channel with wrong credentials": {
			id:    ch.ID,
			token: wrongValue,
			err:   errors.ErrAuthentication,
		},
		"view non-existing channel": {
			id:    wrongID,
			token: token,
			err:   errors.ErrNotFound,
		},
		"view channel with metadata": {
			id:    wrongID,
			token: token,
			err:   errors.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := svc.ViewChannel(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListChannels(t *testing.T) {
	svc := newService()
	meta := things.Metadata{}
	meta["name"] = "test-channel"
	channel.Metadata = meta

	var chs []things.Channel
	for i := uint64(0); i < n; i++ {
		ch := channel
		ch.Name = fmt.Sprintf("name-%d", i)
		chs = append(chs, ch)
	}

	_, err := svc.CreateChannels(context.Background(), token, chs...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		token        string
		pageMetadata things.PageMetadata
		size         uint64
		err          error
	}{
		"list all channels": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		"list all channels as admin": {
			token: adminToken,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		"list all channels with no limit": {
			token: token,
			pageMetadata: things.PageMetadata{
				Limit: 0,
			},
			size: n,
			err:  nil,
		},
		"list half": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: n / 2,
				Limit:  n,
			},
			size: n / 2,
			err:  nil,
		},
		"list last channel": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: n - 1,
				Limit:  n,
			},
			size: 1,
			err:  nil,
		},
		"list empty set": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: n + 1,
				Limit:  n,
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
				Limit:  n,
				Name:   "chanel_name",
			},
			size: n,
			err:  nil,
		},
		"list with non-existent name": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Name:   "wrong",
			},
			size: n,
			err:  nil,
		},
		"list all channels with metadata": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset:   0,
				Limit:    n,
				Metadata: meta,
			},
			size: n,
			err:  nil,
		},
		"list all channels sorted by name ascendent": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "asc",
			},
			size: n,
			err:  nil,
		},
		"list all channels sorted by name descendent": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "desc",
			},
			size: n,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListChannels(context.Background(), tc.token, tc.pageMetadata)
		size := uint64(len(page.Channels))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

		// Check if channels list have been sorted properly
		testSortChannels(t, tc.pageMetadata, page.Channels)
	}
}

func TestViewChannelByThing(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	thingList[0].GroupID = grs[0].ID

	ths, err := svc.CreateThings(context.Background(), token, thingList[0])
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th := ths[0]

	c := channel
	c.Name = "test-channel"
	c.GroupID = grs[0].ID

	chs, err := svc.CreateChannels(context.Background(), token, c)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	ch := chs[0]

	err = svc.Connect(context.Background(), token, ch.ID, []string{th.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	// Wait for things and channels to connect.
	time.Sleep(time.Second)

	cases := map[string]struct {
		token   string
		thID    string
		channel things.Channel
		err     error
	}{
		"view channel by existing thing": {
			token:   token,
			thID:    th.ID,
			channel: ch,
			err:     nil,
		},
		"view channel by existing thing as admin": {
			token:   adminToken,
			thID:    th.ID,
			channel: ch,
			err:     nil,
		},
		"view channel by existing thing with wrong credentials": {
			token:   wrongValue,
			thID:    th.ID,
			channel: things.Channel{},
			err:     errors.ErrAuthentication,
		},
		"view channel by non-existent thing": {
			token:   token,
			thID:    "non-existent",
			channel: things.Channel{},
			err:     errors.ErrNotFound,
		},
		"view channel by existent thing with invalid token": {
			token:   wrongValue,
			thID:    th.ID,
			channel: things.Channel{},
			err:     errors.ErrAuthentication,
		},
	}

	for desc, tc := range cases {
		ch, err := svc.ViewChannelByThing(context.Background(), tc.token, tc.thID)
		assert.Equal(t, tc.channel, ch, fmt.Sprintf("%s: expected %v got %v\n", desc, tc.channel, ch))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRemoveChannel(t *testing.T) {
	svc := newService()
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "remove channel with wrong credentials",
			id:    ch.ID,
			token: wrongValue,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "remove existing channel",
			id:    ch.ID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove removed channel",
			id:    ch.ID,
			token: token,
			err:   errors.ErrNotFound,
		},
		{
			desc:  "remove non-existing channel",
			id:    wrongID,
			token: token,
			err:   errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveChannels(context.Background(), tc.token, tc.id)
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

	channel.GroupID = gr.ID
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]

	cases := []struct {
		desc    string
		token   string
		chanID  string
		thingID string
		err     error
	}{
		{
			desc:    "connect thing",
			token:   token,
			chanID:  ch.ID,
			thingID: th.ID,
			err:     nil,
		},
		{
			desc:    "connect thing with wrong credentials",
			token:   wrongValue,
			chanID:  ch.ID,
			thingID: th.ID,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "connect thing to non-existing channel",
			token:   token,
			chanID:  wrongID,
			thingID: th.ID,
			err:     errors.ErrNotFound,
		},
		{
			desc:    "connect non-existing thing to channel",
			token:   token,
			chanID:  ch.ID,
			thingID: wrongID,
			err:     errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.Connect(context.Background(), tc.token, tc.chanID, []string{tc.thingID})
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

	channel.GroupID = gr.ID
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]

	err = svc.Connect(context.Background(), token, ch.ID, []string{th.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc    string
		token   string
		chanID  string
		thingID string
		err     error
	}{
		{
			desc:    "disconnect connected thing",
			token:   token,
			chanID:  ch.ID,
			thingID: th.ID,
			err:     nil,
		},
		{
			desc:    "disconnect with wrong credentials",
			token:   wrongValue,
			chanID:  ch.ID,
			thingID: th.ID,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "disconnect from non-existing channel",
			token:   token,
			chanID:  wrongID,
			thingID: th.ID,
			err:     errors.ErrNotFound,
		},
		{
			desc:    "disconnect non-existing thing",
			token:   token,
			chanID:  ch.ID,
			thingID: wrongID,
			err:     errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.Disconnect(context.Background(), tc.token, tc.chanID, []string{tc.thingID})
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

	channel.GroupID = gr.ID
	chs, err := svc.CreateChannels(context.Background(), token, channel, channel)
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

func TestIsChannelOwner(t *testing.T) {
	svc := newService()
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ownedCh := chs[0]
	chs, err = svc.CreateChannels(context.Background(), otherToken, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	nonOwnedCh := chs[0]

	cases := map[string]struct {
		channel string
		err     error
	}{
		"user owns channel": {
			channel: ownedCh.ID,
			err:     nil,
		},
		"user does not own channel": {
			channel: nonOwnedCh.ID,
			err:     errors.ErrAuthorization,
		},
		"access to non-existing channel": {
			channel: wrongID,
			err:     errors.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		err := svc.IsChannelOwner(context.Background(), token, tc.channel)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestIdentify(t *testing.T) {
	svc := newService()

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

	var chs []things.Channel
	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		ch := channel
		ch.Name = fmt.Sprintf("name-%d", i)
		ch.GroupID = gr.ID
		chs = append(chs, ch)
	}

	chsc, err := svc.CreateChannels(context.Background(), token, chs...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	ch := chsc[0]

	err = svc.Connect(context.Background(), token, ch.ID, []string{th.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	// Wait for things and channels to connect.
	time.Sleep(time.Second)

	connections := []things.Connection{}
	connections = append(connections, things.Connection{
		ChannelID: ch.ID,
		ThingID:   th.ID,
	})

	backup := things.Backup{
		Groups:      groups,
		Things:      ths,
		Channels:    chsc,
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
		channelsSize := len(backup.Channels)
		connectionsSize := len(backup.Connections)

		assert.Equal(t, len(tc.backup.Groups), groupSize, fmt.Sprintf("%s: expected %v got %d\n", desc, len(tc.backup.Groups), groupSize))
		assert.Equal(t, len(tc.backup.Things), thingsSize, fmt.Sprintf("%s: expected %v got %d\n", desc, len(tc.backup.Things), thingsSize))
		assert.Equal(t, len(tc.backup.Channels), channelsSize, fmt.Sprintf("%s: expected %v got %d\n", desc, len(tc.backup.Channels), channelsSize))
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
			ID:          fmt.Sprintf("%s%012d", prefix, i+1),
			Name:        "test-group-" + num,
			Description: "test group desc",
		}

		groups = append(groups, gr)
	}

	ths := []things.Thing{
		{
			ID:       thID,
			Name:     "testThing",
			OwnerID:  adminEmail,
			Key:      thkey,
			Metadata: map[string]interface{}{},
		},
	}
	th := ths[0]

	var chs []things.Channel
	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		ch := things.Channel{
			ID:       chID,
			Name:     "testChannel",
			OwnerID:  adminEmail,
			Metadata: map[string]interface{}{},
		}
		ch.Name = fmt.Sprintf("name-%d", i)
		chs = append(chs, ch)
	}
	ch := chs[0]

	var connections []things.Connection
	conn := things.Connection{
		ChannelID: ch.ID,
		ThingID:   th.ID,
	}

	connections = append(connections, conn)

	backup := things.Backup{
		Groups:      groups,
		Things:      ths,
		Channels:    chs,
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

func testSortChannels(t *testing.T, pm things.PageMetadata, chs []things.Channel) {
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
