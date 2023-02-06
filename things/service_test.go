// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/things/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	wrongID    = ""
	wrongValue = "wrong-value"
	adminEmail = "admin@example.com"
	email      = "user@example.com"
	email2     = "user2@example.com"
	token      = "token"
	token2     = "token2"
	n          = uint64(102)
	prefix     = "fe6b4e92-cc98-425e-b0aa-"
)

var (
	thing     = things.Thing{Name: "test"}
	thingList = [n]things.Thing{}
	channel   = things.Channel{Name: "test"}
	thsExtID  = []things.Thing{{ID: prefix + "000000000001", Name: "a"}, {ID: prefix + "000000000002", Name: "b"}}
	chsExtID  = []things.Channel{{ID: prefix + "000000000001", Name: "a"}, {ID: prefix + "000000000002", Name: "b"}}
)

func newService(tokens map[string]string) things.Service {
	userPolicy := mocks.MockSubjectSet{Object: "users", Relation: "member"}
	adminPolicy := mocks.MockSubjectSet{Object: "authorities", Relation: "member"}
	auth := mocks.NewAuthService(tokens, map[string][]mocks.MockSubjectSet{
		adminEmail: {userPolicy, adminPolicy}, email: {userPolicy}})
	conns := make(chan mocks.Connection)
	thingsRepo := mocks.NewThingRepository(conns)
	channelsRepo := mocks.NewChannelRepository(thingsRepo, conns)
	chanCache := mocks.NewChannelCache()
	thingCache := mocks.NewThingCache()
	idProvider := uuid.NewMock()

	return things.New(auth, thingsRepo, channelsRepo, chanCache, thingCache, idProvider)
}

func TestInit(t *testing.T) {
	for i := uint64(0); i < n; i++ {
		thingList[i].Name = fmt.Sprintf("name-%d", i+1)
		thingList[i].ID = fmt.Sprintf("%s%012d", prefix, i+1)
		thingList[i].Key = fmt.Sprintf("%s1%011d", prefix, i+1)
	}
}

func TestCreateThings(t *testing.T) {
	svc := newService(map[string]string{token: email})

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
	svc := newService(map[string]string{token: email})
	ths, err := svc.CreateThings(context.Background(), token, thingList[0])
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
			err:   errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateThing(context.Background(), tc.token, tc.thing)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateKey(t *testing.T) {
	key := "new-key"
	svc := newService(map[string]string{token: email})
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
			err:   errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateKey(context.Background(), tc.token, tc.id, tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestShareThing(t *testing.T) {
	svc := newService(map[string]string{token: email, token2: email2})
	ths, err := svc.CreateThings(context.Background(), token, thingList[0])
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]
	policies := []string{"read"}

	cases := []struct {
		desc     string
		token    string
		thingID  string
		policies []string
		userIDs  []string
		err      error
	}{
		{
			desc:     "share a thing with a valid user",
			token:    token,
			thingID:  th.ID,
			policies: policies,
			userIDs:  []string{email2},
			err:      nil,
		},
		{
			desc:     "share a thing via unauthorized access",
			token:    token2,
			thingID:  th.ID,
			policies: policies,
			userIDs:  []string{email2},
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "share a thing with invalid token",
			token:    wrongValue,
			thingID:  th.ID,
			policies: policies,
			userIDs:  []string{email2},
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "share a thing with partially invalid policies",
			token:    token,
			thingID:  th.ID,
			policies: []string{"", "read"},
			userIDs:  []string{email2},
			err:      fmt.Errorf("cannot claim ownership on object '%s' by user '%s': %s", th.ID, email2, errors.ErrMalformedEntity),
		},
	}

	for _, tc := range cases {
		err := svc.ShareThing(context.Background(), tc.token, tc.thingID, tc.policies, tc.userIDs)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}

}

func TestViewThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ths, err := svc.CreateThings(context.Background(), token, thingList[0])
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
	svc := newService(map[string]string{token: email})

	m := make(map[string]interface{})
	m["serial"] = "123456"
	thingList[0].Metadata = m

	var ths []things.Thing
	for i := uint64(0); i < n; i++ {
		th := thingList[i]
		ths = append(ths, th)
	}

	_, err := svc.CreateThings(context.Background(), token, ths...)
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
	svc := newService(map[string]string{token: email})

	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	ch := chs[0]

	thsDisconNum := uint64(4)

	var ths []things.Thing
	for i := uint64(0); i < n; i++ {
		th := thingList[i]
		ths = append(ths, th)
	}

	thsc, err := svc.CreateThings(context.Background(), token, ths...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var thIDs []string
	for _, thID := range thsc {
		thIDs = append(thIDs, thID.ID)
	}
	chIDs := []string{chs[0].ID}

	err = svc.Connect(context.Background(), token, chIDs, thIDs[0:n-thsDisconNum])
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
			err:  nil,
		},
		"list all non connected things by existing channel": {
			token: token,
			chID:  ch.ID,
			pageMetadata: things.PageMetadata{
				Offset:       0,
				Limit:        n,
				Disconnected: true,
			},
			size: thsDisconNum,
			err:  nil,
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
		"list all non-connected things by channel sorted by name ascendent": {
			token: token,
			chID:  ch.ID,
			pageMetadata: things.PageMetadata{
				Offset:       0,
				Limit:        n,
				Disconnected: true,
				Order:        "name",
				Dir:          "asc",
			},
			size: thsDisconNum,
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
		"list all non-connected things by channel sorted by name descendent": {
			token: token,
			chID:  ch.ID,
			pageMetadata: things.PageMetadata{
				Offset:       0,
				Limit:        n,
				Disconnected: true,
				Order:        "name",
				Dir:          "desc",
			},
			size: thsDisconNum,
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

func TestRemoveThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
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
			err:   nil,
		},
		{
			desc:  "remove non-existing thing",
			id:    wrongID,
			token: token,
			err:   errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveThing(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestCreateChannels(t *testing.T) {
	svc := newService(map[string]string{token: email})

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
	svc := newService(map[string]string{token: adminEmail})
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
	svc := newService(map[string]string{token: adminEmail})
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]

	cases := map[string]struct {
		id       string
		token    string
		err      error
		metadata things.Metadata
	}{
		"view existing channel": {
			id:    ch.ID,
			token: token,
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
	svc := newService(map[string]string{token: email})
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

func TestListChannelsByThing(t *testing.T) {
	svc := newService(map[string]string{token: email})

	ths, err := svc.CreateThings(context.Background(), token, thingList[0])
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th := ths[0]

	chsDisconNum := uint64(4)

	var chs []things.Channel
	for i := uint64(0); i < n; i++ {
		ch := channel
		ch.Name = fmt.Sprintf("name-%d", i)
		chs = append(chs, ch)
	}

	chsc, err := svc.CreateChannels(context.Background(), token, chs...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var chIDs []string
	for _, chID := range chsc {
		chIDs = append(chIDs, chID.ID)
	}
	thIDs := []string{ths[0].ID}

	err = svc.Connect(context.Background(), token, chIDs[0:n-chsDisconNum], thIDs)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	// Wait for things and channels to connect.
	time.Sleep(time.Second)

	cases := map[string]struct {
		token        string
		thID         string
		pageMetadata things.PageMetadata
		size         uint64
		err          error
	}{
		"list all channels by existing thing": {
			token: token,
			thID:  th.ID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n - chsDisconNum,
			err:  nil,
		},
		"list all channels by existing thing with no limit": {
			token: token,
			thID:  th.ID,
			pageMetadata: things.PageMetadata{
				Limit: 0,
			},
			size: n - chsDisconNum,
			err:  nil,
		},
		"list half of channels by existing thing": {
			token: token,
			thID:  th.ID,
			pageMetadata: things.PageMetadata{
				Offset: (n - chsDisconNum) / 2,
				Limit:  n,
			},
			size: (n - chsDisconNum) / 2,
			err:  nil,
		},
		"list last channel by existing thing": {
			token: token,
			thID:  th.ID,
			pageMetadata: things.PageMetadata{
				Offset: n - 1 - chsDisconNum,
				Limit:  n,
			},
			size: 1,
			err:  nil,
		},
		"list empty set of channels by existing thing": {
			token: token,
			thID:  th.ID,
			pageMetadata: things.PageMetadata{
				Offset: n + 1,
				Limit:  n,
			},
			size: 0,
			err:  nil,
		},
		"list channels by existing thing with wrong credentials": {
			token: wrongValue,
			thID:  th.ID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  0,
			},
			size: 0,
			err:  errors.ErrAuthentication,
		},
		"list channels by non-existent thing": {
			token: token,
			thID:  "non-existent",
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: 0,
			err:  nil,
		},
		"list all non-connected channels by existing thing": {
			token: token,
			thID:  th.ID,
			pageMetadata: things.PageMetadata{
				Offset:       0,
				Limit:        n,
				Disconnected: true,
			},
			size: chsDisconNum,
			err:  nil,
		},
		"list all non-connected channels by existing thing with no limit": {
			token: token,
			thID:  th.ID,
			pageMetadata: things.PageMetadata{
				Limit:        0,
				Disconnected: true,
			},
			size: chsDisconNum,
			err:  nil,
		},
		"list all channels by thing sorted by name ascendent": {
			token: token,
			thID:  th.ID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "asc",
			},
			size: n - chsDisconNum,
			err:  nil,
		},
		"list all non-connected channels by thing sorted by name ascendent": {
			token: token,
			thID:  th.ID,
			pageMetadata: things.PageMetadata{
				Offset:       0,
				Limit:        n,
				Disconnected: true,
				Order:        "name",
				Dir:          "asc",
			},
			size: chsDisconNum,
			err:  nil,
		},
		"list all channels by thing sorted by name descendent": {
			token: token,
			thID:  th.ID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "desc",
			},
			size: n - chsDisconNum,
			err:  nil,
		},
		"list all non-connected channels by thing sorted by name descendent": {
			token: token,
			thID:  th.ID,
			pageMetadata: things.PageMetadata{
				Offset:       0,
				Limit:        n,
				Disconnected: true,
				Order:        "name",
				Dir:          "desc",
			},
			size: chsDisconNum,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListChannelsByThing(context.Background(), tc.token, tc.thID, tc.pageMetadata)
		size := uint64(len(page.Channels))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

		// Check if Channels by Thing list have been sorted properly
		testSortChannels(t, tc.pageMetadata, page.Channels)
	}
}

func TestRemoveChannel(t *testing.T) {
	svc := newService(map[string]string{token: adminEmail})
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
			err:   nil,
		},
		{
			desc:  "remove non-existing channel",
			id:    ch.ID,
			token: token,
			err:   nil,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveChannel(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestConnect(t *testing.T) {
	svc := newService(map[string]string{token: email})

	ths, err := svc.CreateThings(context.Background(), token, thingList[0])
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]
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
		err := svc.Connect(context.Background(), tc.token, []string{tc.chanID}, []string{tc.thingID})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestDisconnect(t *testing.T) {
	svc := newService(map[string]string{token: email})

	ths, err := svc.CreateThings(context.Background(), token, thingList[0])
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]
	err = svc.Connect(context.Background(), token, []string{ch.ID}, []string{th.ID})
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
			desc:    "disconnect disconnected thing",
			token:   token,
			chanID:  ch.ID,
			thingID: th.ID,
			err:     errors.ErrNotFound,
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
		err := svc.Disconnect(context.Background(), tc.token, []string{tc.chanID}, []string{tc.thingID})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}

}

func TestCanAccessByKey(t *testing.T) {
	svc := newService(map[string]string{token: email})

	ths, err := svc.CreateThings(context.Background(), token, thingList[0])
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chs, err := svc.CreateChannels(context.Background(), token, channel, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	err = svc.Connect(context.Background(), token, []string{chs[0].ID}, []string{ths[0].ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := map[string]struct {
		token   string
		channel string
		err     error
	}{
		"allowed access": {
			token:   ths[0].Key,
			channel: chs[0].ID,
			err:     nil,
		},
		"non-existing thing": {
			token:   wrongValue,
			channel: chs[0].ID,
			err:     errors.ErrNotFound,
		},
		"non-existing chan": {
			token:   ths[0].Key,
			channel: wrongValue,
			err:     errors.ErrAuthorization,
		},
		"non-connected channel": {
			token:   ths[0].Key,
			channel: chs[1].ID,
			err:     errors.ErrAuthorization,
		},
	}

	for desc, tc := range cases {
		_, err := svc.CanAccessByKey(context.Background(), tc.channel, tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected '%s' got '%s'\n", desc, tc.err, err))
	}
}

func TestCanAccessByID(t *testing.T) {
	svc := newService(map[string]string{token: email})

	ths, err := svc.CreateThings(context.Background(), token, thingList[0], thingList[1])
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]
	err = svc.Connect(context.Background(), token, []string{ch.ID}, []string{th.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := map[string]struct {
		thingID string
		channel string
		err     error
	}{
		"allowed access": {
			thingID: th.ID,
			channel: ch.ID,
			err:     nil,
		},
		"access to non-existing thing": {
			thingID: wrongValue,
			channel: ch.ID,
			err:     errors.ErrAuthorization,
		},
		"access to non-existing channel": {
			thingID: th.ID,
			channel: wrongID,
			err:     errors.ErrAuthorization,
		},
		"access to not-connected thing": {
			thingID: ths[1].ID,
			channel: ch.ID,
			err:     errors.ErrAuthorization,
		},
	}

	for desc, tc := range cases {
		err := svc.CanAccessByID(context.Background(), tc.channel, tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestIsChannelOwner(t *testing.T) {
	svc := newService(map[string]string{token: email, token2: "john.doe@email.net"})

	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ownedCh := chs[0]
	chs, err = svc.CreateChannels(context.Background(), token2, channel)
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
			err:     errors.ErrNotFound,
		},
		"access to non-existing channel": {
			channel: wrongID,
			err:     errors.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		err := svc.IsChannelOwner(context.Background(), email, tc.channel)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestIdentify(t *testing.T) {
	svc := newService(map[string]string{token: email})

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
	svc := newService(map[string]string{token: adminEmail})

	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var chs []things.Channel
	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		ch := channel
		ch.Name = fmt.Sprintf("name-%d", i)
		chs = append(chs, ch)
	}

	chsc, err := svc.CreateChannels(context.Background(), token, chs...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var chIDs []string
	for _, chID := range chsc {
		chIDs = append(chIDs, chID.ID)
	}
	thIDs := []string{ths[0].ID}

	err = svc.Connect(context.Background(), token, chIDs[0:10], thIDs)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	// Wait for things and channels to connect.
	time.Sleep(time.Second)

	connections := []things.Connection{}
	for _, ch := range chsc {
		connections = append(connections, things.Connection{
			ChannelID:    ch.ID,
			ChannelOwner: ths[0].Owner,
			ThingID:      ths[0].ID,
			ThingOwner:   ths[0].Owner,
		})
	}

	backup := things.Backup{
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
			token:  token,
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
		thingsSize := len(backup.Things)
		channelsSize := len(backup.Channels)
		connectionsSize := len(backup.Connections)
		assert.Equal(t, len(tc.backup.Things), thingsSize, fmt.Sprintf("%s: expected %v got %d\n", desc, len(tc.backup.Things), thingsSize))
		assert.Equal(t, len(tc.backup.Channels), channelsSize, fmt.Sprintf("%s: expected %v got %d\n", desc, len(tc.backup.Channels), channelsSize))
		assert.Equal(t, len(tc.backup.Connections), connectionsSize, fmt.Sprintf("%s: expected %v got %d\n", desc, len(tc.backup.Connections), connectionsSize))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

	}
}

func TestRestore(t *testing.T) {
	svc := newService(map[string]string{token: adminEmail})
	idProvider := uuid.New()

	thkey, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	thID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	chID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	ths := []things.Thing{
		{
			ID:       thID,
			Name:     "testThing",
			Owner:    adminEmail,
			Key:      thkey,
			Metadata: map[string]interface{}{},
		},
	}

	var chs []things.Channel
	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		ch := things.Channel{
			ID:       chID,
			Name:     "testChannel",
			Owner:    adminEmail,
			Metadata: map[string]interface{}{},
		}
		ch.Name = fmt.Sprintf("name-%d", i)
		chs = append(chs, ch)
	}

	var connections []things.Connection
	for _, ch := range chs {
		conn := things.Connection{
			ChannelID:    ch.ID,
			ChannelOwner: ch.Owner,
			ThingID:      ths[0].ID,
			ThingOwner:   ths[0].Owner,
		}

		connections = append(connections, conn)
	}

	backup := things.Backup{
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
			token:  token,
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
