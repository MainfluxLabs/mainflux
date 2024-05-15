// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis_test

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	thmocks "github.com/MainfluxLabs/mainflux/things/mocks"
	"github.com/MainfluxLabs/mainflux/things/redis"
	"github.com/MainfluxLabs/mainflux/users"
	r "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	streamID        = "mainflux.things"
	email           = "user@example.com"
	adminEmail      = "admin@example.com"
	password        = "password"
	token           = email
	thingPrefix     = "thing."
	thingCreate     = thingPrefix + "create"
	thingUpdate     = thingPrefix + "update"
	thingRemove     = thingPrefix + "remove"
	thingConnect    = thingPrefix + "connect"
	thingDisconnect = thingPrefix + "disconnect"

	channelPrefix = "channel."
	channelCreate = channelPrefix + "create"
	channelUpdate = channelPrefix + "update"
	channelRemove = channelPrefix + "remove"
)

var (
	user      = users.User{Email: email, Password: password}
	admin     = users.User{Email: adminEmail, Password: password}
	usersList = []users.User{admin, user}
	group     = things.Group{Name: "test-group", Description: "test-group-desc"}
)

func newService(tokens map[string]string) things.Service {
	auth := mocks.NewAuthService("", usersList)
	conns := make(chan thmocks.Connection)
	thingsRepo := thmocks.NewThingRepository(conns)
	channelsRepo := thmocks.NewChannelRepository(thingsRepo, conns)
	groupsRepo := thmocks.NewGroupRepository()
	rolesRepo := thmocks.NewRolesRepository()
	chanCache := thmocks.NewChannelCache()
	thingCache := thmocks.NewThingCache()
	idProvider := uuid.NewMock()

	return things.New(auth, nil, thingsRepo, channelsRepo, groupsRepo, rolesRepo, chanCache, thingCache, idProvider)
}

func TestCreateThings(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: email})
	svc = redis.NewEventStoreMiddleware(svc, redisClient)

	cases := []struct {
		desc  string
		ths   []things.Thing
		key   string
		err   error
		event map[string]interface{}
	}{
		{
			desc: "create things successfully",
			ths: []things.Thing{{
				Name:     "a",
				Metadata: map[string]interface{}{"test": "test"},
			}},
			key: token,
			err: nil,
			event: map[string]interface{}{
				"id":        "123e4567-e89b-12d3-a456-000000000001",
				"name":      "a",
				"owner_id":  user.ID,
				"group_id":  group.ID,
				"metadata":  "{\"test\":\"test\"}",
				"operation": thingCreate,
			},
		},
		{
			desc:  "create things with invalid credentials",
			ths:   []things.Thing{{Name: "a", Metadata: map[string]interface{}{"test": "test"}}},
			key:   "",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, err := svc.CreateThings(context.Background(), tc.key, tc.ths...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &r.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			lastID = msg.ID
		}

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}

func TestUpdateThing(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: email})
	// Create thing without sending event.
	th := things.Thing{Name: "a", Metadata: map[string]interface{}{"test": "test"}}
	sths, err := svc.CreateThings(context.Background(), token, th)
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sth := sths[0]

	svc = redis.NewEventStoreMiddleware(svc, redisClient)

	cases := []struct {
		desc  string
		thing things.Thing
		key   string
		err   error
		event map[string]interface{}
	}{
		{
			desc: "update existing thing successfully",
			thing: things.Thing{
				ID:       sth.ID,
				Name:     "a",
				Metadata: map[string]interface{}{"test": "test"},
			},
			key: token,
			err: nil,
			event: map[string]interface{}{
				"id":        sth.ID,
				"name":      "a",
				"metadata":  "{\"test\":\"test\"}",
				"operation": thingUpdate,
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.UpdateThing(context.Background(), tc.key, tc.thing)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		streams := redisClient.XRead(context.Background(), &r.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			lastID = msg.ID
		}

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}

func TestViewThing(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: email})
	// Create thing without sending event.
	sths, err := svc.CreateThings(context.Background(), token, things.Thing{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sth := sths[0]

	essvc := redis.NewEventStoreMiddleware(svc, redisClient)
	esth, eserr := essvc.ViewThing(context.Background(), token, sth.ID)
	th, err := svc.ViewThing(context.Background(), token, sth.ID)
	assert.Equal(t, th, esth, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", th, esth))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", err, eserr))
}

func TestListThings(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: email})
	// Create thing without sending event.
	_, err := svc.CreateThings(context.Background(), token, things.Thing{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	essvc := redis.NewEventStoreMiddleware(svc, redisClient)
	esths, eserr := essvc.ListThings(context.Background(), token, things.PageMetadata{Offset: 0, Limit: 10})
	ths, err := svc.ListThings(context.Background(), token, things.PageMetadata{Offset: 0, Limit: 10})
	assert.Equal(t, ths, esths, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", ths, esths))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", err, eserr))
}

func TestListThingsByChannel(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: email})

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	// Create thing without sending event.
	sths, err := svc.CreateThings(context.Background(), token, things.Thing{Name: "a", GroupID: gr.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sth := sths[0]

	schs, err := svc.CreateChannels(context.Background(), token, things.Channel{Name: "a", GroupID: gr.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sch := schs[0]

	err = svc.Connect(context.Background(), token, sch.ID, []string{sth.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	essvc := redis.NewEventStoreMiddleware(svc, redisClient)
	esths, eserr := essvc.ListThingsByChannel(context.Background(), token, sch.ID, things.PageMetadata{Offset: 0, Limit: 10})
	thps, err := svc.ListThingsByChannel(context.Background(), token, sch.ID, things.PageMetadata{Offset: 0, Limit: 10})
	assert.Equal(t, thps, esths, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", thps, esths))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", err, eserr))
}

func TestRemoveThing(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: email})
	// Create thing without sending event.
	sths, err := svc.CreateThings(context.Background(), token, things.Thing{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sth := sths[0]

	svc = redis.NewEventStoreMiddleware(svc, redisClient)

	cases := []struct {
		desc  string
		id    string
		key   string
		err   error
		event map[string]interface{}
	}{
		{
			desc: "delete existing thing successfully",
			id:   sth.ID,
			key:  token,
			err:  nil,
			event: map[string]interface{}{
				"id":        sth.ID,
				"operation": thingRemove,
			},
		},
		{
			desc:  "delete thing with invalid credentials",
			id:    strconv.FormatUint(math.MaxUint64, 10),
			key:   "",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.RemoveThings(context.Background(), tc.key, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &r.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			lastID = msg.ID
		}

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}

func TestCreateChannels(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: email})
	svc = redis.NewEventStoreMiddleware(svc, redisClient)

	cases := []struct {
		desc  string
		chs   []things.Channel
		key   string
		err   error
		event map[string]interface{}
	}{
		{
			desc: "create channels successfully",
			chs:  []things.Channel{{GroupID: group.ID, Name: "a", Metadata: map[string]interface{}{"test": "test"}}},
			key:  token,
			err:  nil,
			event: map[string]interface{}{
				"id":        "123e4567-e89b-12d3-a456-000000000001",
				"name":      "a",
				"metadata":  "{\"test\":\"test\"}",
				"owner_id":  user.ID,
				"group_id":  group.ID,
				"operation": channelCreate,
			},
		},
		{
			desc:  "create channels with invalid credentials",
			chs:   []things.Channel{{GroupID: group.ID, Name: "a", Metadata: map[string]interface{}{"test": "test"}}},
			key:   "",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, err := svc.CreateChannels(context.Background(), tc.key, tc.chs...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &r.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			lastID = msg.ID
		}

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}

func TestUpdateChannel(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: adminEmail})
	// Create channel without sending event.
	schs, err := svc.CreateChannels(context.Background(), token, things.Channel{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sch := schs[0]

	svc = redis.NewEventStoreMiddleware(svc, redisClient)

	cases := []struct {
		desc    string
		channel things.Channel
		key     string
		err     error
		event   map[string]interface{}
	}{
		{
			desc: "update channel successfully",
			channel: things.Channel{
				ID:       sch.ID,
				Name:     "b",
				Metadata: map[string]interface{}{"test": "test"},
			},
			key: token,
			err: nil,
			event: map[string]interface{}{
				"id":        sch.ID,
				"name":      "b",
				"metadata":  "{\"test\":\"test\"}",
				"operation": channelUpdate,
			},
		},
		{
			desc: "create non-existent channel",
			channel: things.Channel{
				ID:   strconv.FormatUint(math.MaxUint64, 10),
				Name: "c",
			},
			key:   token,
			err:   errors.ErrNotFound,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.UpdateChannel(context.Background(), tc.key, tc.channel)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &r.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			lastID = msg.ID
		}

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}

func TestViewChannel(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: email})
	// Create channel without sending event.
	schs, err := svc.CreateChannels(context.Background(), token, things.Channel{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sch := schs[0]

	essvc := redis.NewEventStoreMiddleware(svc, redisClient)
	esch, eserr := essvc.ViewChannel(context.Background(), token, sch.ID)
	ch, err := svc.ViewChannel(context.Background(), token, sch.ID)
	assert.Equal(t, ch, esch, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", ch, esch))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", err, eserr))
}

func TestListChannels(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: email})
	// Create thing without sending event.
	_, err := svc.CreateChannels(context.Background(), token, things.Channel{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	essvc := redis.NewEventStoreMiddleware(svc, redisClient)
	eschs, eserr := essvc.ListChannels(context.Background(), token, things.PageMetadata{Offset: 0, Limit: 10})
	chs, err := svc.ListChannels(context.Background(), token, things.PageMetadata{Offset: 0, Limit: 10})
	assert.Equal(t, chs, eschs, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", chs, eschs))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", err, eserr))
}

func TestListChannelsByThing(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: email})

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	// Create thing without sending event.
	sths, err := svc.CreateThings(context.Background(), token, things.Thing{Name: "a", GroupID: gr.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sth := sths[0]

	schs, err := svc.CreateChannels(context.Background(), token, things.Channel{Name: "a", GroupID: gr.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sch := schs[0]

	err = svc.Connect(context.Background(), token, sch.ID, []string{sth.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	essvc := redis.NewEventStoreMiddleware(svc, redisClient)
	eschs, eserr := essvc.ViewChannelByThing(context.Background(), token, sth.ID)
	chps, err := svc.ViewChannelByThing(context.Background(), token, sth.ID)
	assert.Equal(t, chps, eschs, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", chps, eschs))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", err, eserr))
}

func TestRemoveChannel(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: adminEmail})
	// Create channel without sending event.
	schs, err := svc.CreateChannels(context.Background(), token, things.Channel{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sch := schs[0]

	svc = redis.NewEventStoreMiddleware(svc, redisClient)

	cases := []struct {
		desc  string
		id    string
		key   string
		err   error
		event map[string]interface{}
	}{
		{
			desc: "update channel successfully",
			id:   sch.ID,
			key:  token,
			err:  nil,
			event: map[string]interface{}{
				"id":        sch.ID,
				"operation": channelRemove,
			},
		},
		{
			desc:  "create non-existent channel",
			id:    strconv.FormatUint(math.MaxUint64, 10),
			key:   "",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.RemoveChannels(context.Background(), tc.key, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &r.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			lastID = msg.ID
		}

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}

func TestConnectEvent(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: email})

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	// Create thing and channel that will be connected.
	sths, err := svc.CreateThings(context.Background(), token, things.Thing{Name: "a", GroupID: gr.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sth := sths[0]
	schs, err := svc.CreateChannels(context.Background(), token, things.Channel{Name: "a", GroupID: gr.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sch := schs[0]

	svc = redis.NewEventStoreMiddleware(svc, redisClient)

	cases := []struct {
		desc    string
		thingID string
		chanID  string
		key     string
		err     error
		event   map[string]interface{}
	}{
		{
			desc:    "connect existing thing to existing channel",
			thingID: sth.ID,
			chanID:  sch.ID,
			key:     token,
			err:     nil,
			event: map[string]interface{}{
				"chan_id":   sch.ID,
				"thing_id":  sth.ID,
				"operation": thingConnect,
			},
		},
		{
			desc:    "connect non-existent thing to channel",
			thingID: strconv.FormatUint(math.MaxUint64, 10),
			chanID:  sch.ID,
			key:     token,
			err:     errors.ErrNotFound,
			event:   nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.Connect(context.Background(), tc.key, tc.chanID, []string{tc.thingID})
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &r.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			lastID = msg.ID
		}

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}

func TestDisconnectEvent(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: email})

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	// Create thing and channel that will be connected.
	sths, err := svc.CreateThings(context.Background(), token, things.Thing{Name: "a", GroupID: gr.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sth := sths[0]

	schs, err := svc.CreateChannels(context.Background(), token, things.Channel{Name: "a", GroupID: gr.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sch := schs[0]

	err = svc.Connect(context.Background(), token, sch.ID, []string{sth.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	svc = redis.NewEventStoreMiddleware(svc, redisClient)

	cases := []struct {
		desc    string
		thingID string
		chanID  string
		key     string
		err     error
		event   map[string]interface{}
	}{
		{
			desc:    "disconnect thing from channel",
			thingID: sth.ID,
			chanID:  sch.ID,
			key:     token,
			err:     nil,
			event: map[string]interface{}{
				"chan_id":   sch.ID,
				"thing_id":  sth.ID,
				"operation": thingDisconnect,
			},
		},
		{
			desc:    "disconnect non-existent thing from channel",
			thingID: strconv.FormatUint(math.MaxUint64, 10),
			chanID:  sch.ID,
			key:     token,
			err:     errors.ErrNotFound,
			event:   nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.Disconnect(context.Background(), tc.key, tc.chanID, []string{tc.thingID})
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &r.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			lastID = msg.ID
		}

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}
