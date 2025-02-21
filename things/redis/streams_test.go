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

	"github.com/MainfluxLabs/mainflux/auth"
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
	streamID       = "mainflux.things"
	email          = "user@example.com"
	adminEmail     = "admin@example.com"
	otherUserEmail = "other.user@example.com"
	password       = "password"
	token          = email
	thingPrefix    = "thing."
	thingCreate    = thingPrefix + "create"
	thingUpdate    = thingPrefix + "update"
	thingRemove    = thingPrefix + "remove"

	profilePrefix = "profile."
	profileCreate = profilePrefix + "create"
	profileUpdate = profilePrefix + "update"
	profileRemove = profilePrefix + "remove"
)

var (
	user      = users.User{ID: "574106f7-030e-4881-8ab0-151195c29f94", Email: email, Password: password, Role: auth.Owner}
	otherUser = users.User{ID: "674106f7-030e-4881-8ab0-151195c29f95", Email: otherUserEmail, Password: password, Role: auth.Editor}
	admin     = users.User{ID: "874106f7-030e-4881-8ab0-151195c29f97", Email: adminEmail, Password: password, Role: auth.RootSub}
	usersList = []users.User{admin, user, otherUser}
	group     = things.Group{OrgID: "374106f7-030e-4881-8ab0-151195c29f92", Name: "test-group", Description: "test-group-desc"}
	orgsList  = []auth.Org{{ID: "374106f7-030e-4881-8ab0-151195c29f92", OwnerID: user.ID}}
	profile   = things.Profile{Name: "test-profile"}
)

func newService(tokens map[string]string) things.Service {
	auth := mocks.NewAuthService("", usersList, orgsList)
	thingsRepo := thmocks.NewThingRepository()
	profilesRepo := thmocks.NewProfileRepository(thingsRepo)
	rolesRepo := thmocks.NewRolesRepository()
	groupsRepo := thmocks.NewGroupRepository(rolesRepo)
	profileCache := thmocks.NewProfileCache()
	thingCache := thmocks.NewThingCache()
	groupCache := thmocks.NewGroupCache()
	idProvider := uuid.NewMock()

	return things.New(auth, nil, thingsRepo, profilesRepo, groupsRepo, rolesRepo, profileCache, thingCache, groupCache, idProvider)
}

func TestCreateThings(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: email})
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	prID := prs[0].ID

	ths := []things.Thing{{
		Name:      "a",
		GroupID:   grID,
		ProfileID: prID,
		Metadata:  map[string]interface{}{"test": "test"},
	}}

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
			ths:  ths,
			key:  token,
			err:  nil,
			event: map[string]interface{}{
				"id":         "123e4567-e89b-12d3-a456-000000000003",
				"name":       "a",
				"group_id":   grID,
				"profile_id": prID,
				"metadata":   "{\"test\":\"test\"}",
				"operation":  thingCreate,
			},
		},
		{
			desc:  "create things with invalid credentials",
			ths:   ths,
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
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	prID := prs[0].ID

	// Create thing without sending event.
	th := things.Thing{Name: "a", GroupID: grID, ProfileID: prID, Metadata: map[string]interface{}{"test": "test"}}
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
				ID:        sth.ID,
				ProfileID: prID,
				Name:      "a",
				Metadata:  map[string]interface{}{"test": "test"},
			},
			key: token,
			err: nil,
			event: map[string]interface{}{
				"id":         sth.ID,
				"profile_id": sth.ProfileID,
				"name":       "a",
				"metadata":   "{\"test\":\"test\"}",
				"operation":  thingUpdate,
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
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	prID := prs[0].ID

	// Create thing without sending event.
	sths, err := svc.CreateThings(context.Background(), token, things.Thing{Name: "a", GroupID: grID, ProfileID: prID})
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
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	prID := prs[0].ID

	// Create thing without sending event.
	_, err = svc.CreateThings(context.Background(), token, things.Thing{Name: "a", GroupID: grID, ProfileID: prID})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	essvc := redis.NewEventStoreMiddleware(svc, redisClient)
	esths, eserr := essvc.ListThings(context.Background(), token, things.PageMetadata{Offset: 0, Limit: 10})
	ths, err := svc.ListThings(context.Background(), token, things.PageMetadata{Offset: 0, Limit: 10})
	assert.Equal(t, ths, esths, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", ths, esths))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", err, eserr))
}

func TestListThingsByProfile(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: email})

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	sprs, err := svc.CreateProfiles(context.Background(), token, things.Profile{Name: "a", GroupID: gr.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	pr := sprs[0]

	// Create thing without sending event.
	_, err = svc.CreateThings(context.Background(), token, things.Thing{Name: "a", GroupID: gr.ID, ProfileID: pr.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	essvc := redis.NewEventStoreMiddleware(svc, redisClient)
	esths, eserr := essvc.ListThingsByProfile(context.Background(), token, pr.ID, things.PageMetadata{Offset: 0, Limit: 10})
	thps, err := svc.ListThingsByProfile(context.Background(), token, pr.ID, things.PageMetadata{Offset: 0, Limit: 10})
	assert.Equal(t, thps, esths, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", thps, esths))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", err, eserr))
}

func TestRemoveThing(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: email})
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	prID := prs[0].ID

	// Create thing without sending event.
	sths, err := svc.CreateThings(context.Background(), token, things.Thing{Name: "a", GroupID: grID, ProfileID: prID})
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
			desc: "remove existing thing successfully",
			id:   sth.ID,
			key:  token,
			err:  nil,
			event: map[string]interface{}{
				"id":        sth.ID,
				"operation": thingRemove,
			},
		},
		{
			desc:  "remove non-existent thing",
			id:    strconv.FormatUint(math.MaxUint64, 10),
			key:   "",
			err:   errors.ErrNotFound,
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

func TestCreateProfiles(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: email})
	svc = redis.NewEventStoreMiddleware(svc, redisClient)
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	cases := []struct {
		desc  string
		prs   []things.Profile
		key   string
		err   error
		event map[string]interface{}
	}{
		{
			desc: "create profiles successfully",
			prs:  []things.Profile{{GroupID: gr.ID, Name: "a", Metadata: map[string]interface{}{"test": "test"}}},
			key:  token,
			err:  nil,
			event: map[string]interface{}{
				"id":        "123e4567-e89b-12d3-a456-000000000002",
				"name":      "a",
				"metadata":  "{\"test\":\"test\"}",
				"group_id":  gr.ID,
				"operation": profileCreate,
			},
		},
		{
			desc:  "create profiles with invalid credentials",
			prs:   []things.Profile{{GroupID: gr.ID, Name: "a", Metadata: map[string]interface{}{"test": "test"}}},
			key:   "",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, err := svc.CreateProfiles(context.Background(), tc.key, tc.prs...)
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

func TestUpdateProfile(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: adminEmail})
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]
	// Create profile without sending event.
	sprs, err := svc.CreateProfiles(context.Background(), token, things.Profile{Name: "a", GroupID: gr.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	spr := sprs[0]

	svc = redis.NewEventStoreMiddleware(svc, redisClient)

	cases := []struct {
		desc    string
		profile things.Profile
		key     string
		err     error
		event   map[string]interface{}
	}{
		{
			desc: "update profile successfully",
			profile: things.Profile{
				ID:       spr.ID,
				Name:     "b",
				Metadata: map[string]interface{}{"test": "test"},
			},
			key: token,
			err: nil,
			event: map[string]interface{}{
				"id":        spr.ID,
				"name":      "b",
				"metadata":  "{\"test\":\"test\"}",
				"operation": profileUpdate,
			},
		},
		{
			desc: "update non-existent profile",
			profile: things.Profile{
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
		err := svc.UpdateProfile(context.Background(), tc.key, tc.profile)
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

func TestViewProfile(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: email})
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]
	// Create profile without sending event.
	sprs, err := svc.CreateProfiles(context.Background(), token, things.Profile{Name: "a", GroupID: gr.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	spr := sprs[0]

	essvc := redis.NewEventStoreMiddleware(svc, redisClient)
	espr, eserr := essvc.ViewProfile(context.Background(), token, spr.ID)
	pr, err := svc.ViewProfile(context.Background(), token, spr.ID)
	assert.Equal(t, pr, espr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", pr, espr))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", err, eserr))
}

func TestListProfiles(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: email})
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]
	// Create thing without sending event.
	_, err = svc.CreateProfiles(context.Background(), token, things.Profile{Name: "a", GroupID: gr.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	essvc := redis.NewEventStoreMiddleware(svc, redisClient)
	esprs, eserr := essvc.ListProfiles(context.Background(), token, things.PageMetadata{Offset: 0, Limit: 10})
	prs, err := svc.ListProfiles(context.Background(), token, things.PageMetadata{Offset: 0, Limit: 10})
	assert.Equal(t, prs, esprs, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", prs, esprs))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", err, eserr))
}

func TestListProfilesByThing(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: email})

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	prID := prs[0].ID

	// Create thing without sending event.
	sths, err := svc.CreateThings(context.Background(), token, things.Thing{Name: "a", GroupID: grID, ProfileID: prID})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sth := sths[0]

	essvc := redis.NewEventStoreMiddleware(svc, redisClient)
	esprs, eserr := essvc.ViewProfileByThing(context.Background(), token, sth.ID)
	prps, err := svc.ViewProfileByThing(context.Background(), token, sth.ID)
	assert.Equal(t, prps, esprs, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", prps, esprs))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", err, eserr))
}

func TestRemoveProfile(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService(map[string]string{token: adminEmail})
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]
	// Create profile without sending event.
	sprs, err := svc.CreateProfiles(context.Background(), token, things.Profile{Name: "a", GroupID: gr.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	spr := sprs[0]

	svc = redis.NewEventStoreMiddleware(svc, redisClient)

	cases := []struct {
		desc  string
		id    string
		key   string
		err   error
		event map[string]interface{}
	}{
		{
			desc: "remove profile successfully",
			id:   spr.ID,
			key:  token,
			err:  nil,
			event: map[string]interface{}{
				"id":        spr.ID,
				"operation": profileRemove,
			},
		},
		{
			desc:  "remove non-existent profile",
			id:    strconv.FormatUint(math.MaxUint64, 10),
			key:   "",
			err:   errors.ErrNotFound,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.RemoveProfiles(context.Background(), tc.key, tc.id)
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
