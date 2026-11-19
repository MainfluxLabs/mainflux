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
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	mfevents "github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	thmocks "github.com/MainfluxLabs/mainflux/things/mocks"
	"github.com/MainfluxLabs/mainflux/things/redis/events"
	"github.com/MainfluxLabs/mainflux/users"
	r "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	email          = "user@example.com"
	adminEmail     = "admin@example.com"
	otherUserEmail = "other.user@example.com"
	password       = "password"
	token          = email
)

var (
	user      = users.User{ID: "574106f7-030e-4881-8ab0-151195c29f94", Email: email, Password: password}
	otherUser = users.User{ID: "674106f7-030e-4881-8ab0-151195c29f95", Email: otherUserEmail, Password: password}
	admin     = users.User{ID: "874106f7-030e-4881-8ab0-151195c29f97", Email: adminEmail, Password: password, Role: auth.RootSub}
	usersList = []users.User{admin, user, otherUser}
	orgID     = "374106f7-030e-4881-8ab0-151195c29f92"
	group     = things.Group{Name: "test-group", Description: "test-group-desc"}
	orgsList  = []auth.Org{{ID: "374106f7-030e-4881-8ab0-151195c29f92", OwnerID: user.ID}}
	profile   = things.Profile{Name: "test-profile"}

	orgMembership = auth.OrgMembership{OrgID: orgID, MemberID: otherUser.ID, Role: auth.Viewer}
	usersByIDs    = map[string]users.User{user.ID: user, otherUser.ID: otherUser}
	usersByEmails = map[string]users.User{email: user, otherUserEmail: otherUser}
)

// withEventStore wraps svc in the event-store middleware backed by a fresh
// Publisher pointed at the test Redis. The publisher is closed via
// t.Cleanup, which drains any in-flight events before the test ends.
func withEventStore(t *testing.T, svc things.Service) things.Service {
	t.Helper()
	pub, err := mfevents.NewPublisher(mfevents.PublisherConfig{
		URL:                  redisURL,
		Stream:               mfevents.ThingsStream,
		BufferSize:           mfevents.DefBufferSize,
		DrainInitialInterval: 10 * time.Millisecond,
		DrainMaxBackoff:      100 * time.Millisecond,
		ShutdownDrainTimeout: 2 * time.Second,
	}, logger.NewMock())
	require.NoError(t, err)
	t.Cleanup(func() { _ = pub.Close() })
	return events.NewEventStoreMiddleware(svc, pub)
}

func newService() things.Service {
	auth := mocks.NewAuthService("", usersList, orgsList, orgMembership)
	uc := thmocks.NewUsersService(usersByIDs, usersByEmails)
	thingsRepo := thmocks.NewThingRepository()
	profilesRepo := thmocks.NewProfileRepository(thingsRepo)
	groupMembershipsRepo := thmocks.NewGroupMembershipsRepository()
	groupsRepo := thmocks.NewGroupRepository(groupMembershipsRepo)
	profileCache := thmocks.NewProfileCache()
	thingCache := thmocks.NewThingCache()
	groupCache := thmocks.NewGroupCache()
	idProvider := uuid.NewMock()
	emailerMock := thmocks.NewEmailer()

	return things.New(
		auth,
		uc,
		thingsRepo,
		profilesRepo,
		groupsRepo,
		groupMembershipsRepo,
		profileCache,
		thingCache,
		groupCache,
		idProvider,
		emailerMock,
	)
}

func TestCreateThings(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	ths := []things.Thing{{
		Name:     "a",
		Metadata: map[string]any{"test": "test"},
	}}

	svc = withEventStore(t, svc)

	cases := []struct {
		desc  string
		ths   []things.Thing
		key   string
		err   error
		event map[string]any
	}{
		{
			desc: "create things successfully",
			ths:  ths,
			key:  token,
			err:  nil,
			event: map[string]any{
				"id":           "123e4567-e89b-12d3-a456-000000000003",
				"name":         "a",
				"group_id":     grID,
				"profile_id":   prID,
				"metadata":     "{\"test\":\"test\"}",
				"operation":    mfevents.ThingCreate,
				"evt_org_id":   orgID,
				"evt_group_id": grID,
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
		_, err := svc.CreateThings(context.Background(), tc.key, prID, tc.ths...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &r.XReadArgs{
			Streams: []string{mfevents.ThingsStream, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]any
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			lastID = msg.ID

			assert.NotEmpty(t, event["occurred_at"], tc.desc)
			delete(event, "occurred_at")
		}

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}

func TestUpdateThing(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	// Create thing without sending event.
	th := things.Thing{Name: "a", Metadata: map[string]any{"test": "test"}}
	sths, err := svc.CreateThings(context.Background(), token, prID, th)
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sth := sths[0]

	svc = withEventStore(t, svc)

	cases := []struct {
		desc  string
		thing things.Thing
		key   string
		err   error
		event map[string]any
	}{
		{
			desc: "update existing thing successfully",
			thing: things.Thing{
				ID:        sth.ID,
				ProfileID: prID,
				Name:      "a",
				Metadata:  map[string]any{"test": "test"},
			},
			key: token,
			err: nil,
			event: map[string]any{
				"id":           sth.ID,
				"profile_id":   sth.ProfileID,
				"name":         "a",
				"metadata":     "{\"test\":\"test\"}",
				"operation":    mfevents.ThingUpdate,
				"evt_org_id":   orgID,
				"evt_group_id": grID,
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.UpdateThing(context.Background(), tc.key, tc.thing)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		streams := redisClient.XRead(context.Background(), &r.XReadArgs{
			Streams: []string{mfevents.ThingsStream, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]any
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			lastID = msg.ID

			assert.NotEmpty(t, event["occurred_at"], tc.desc)
			delete(event, "occurred_at")
		}

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}

func TestViewThing(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	// Create thing without sending event.
	sths, err := svc.CreateThings(context.Background(), token, prID, things.Thing{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sth := sths[0]

	essvc := withEventStore(t, svc)
	esth, eserr := essvc.ViewThing(context.Background(), token, sth.ID)
	th, err := svc.ViewThing(context.Background(), token, sth.ID)
	assert.Equal(t, th, esth, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", th, esth))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", err, eserr))
}

func TestListThings(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	// Create thing without sending event.
	_, err = svc.CreateThings(context.Background(), token, prID, things.Thing{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	essvc := withEventStore(t, svc)
	esths, eserr := essvc.ListThings(context.Background(), token, things.PageMetadata{Offset: 0, Limit: 10})
	ths, err := svc.ListThings(context.Background(), token, things.PageMetadata{Offset: 0, Limit: 10})
	assert.Equal(t, ths, esths, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", ths, esths))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", err, eserr))
}

func TestListThingsByProfile(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	sprs, err := svc.CreateProfiles(context.Background(), token, gr.ID, things.Profile{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	pr := sprs[0]

	// Create thing without sending event.
	_, err = svc.CreateThings(context.Background(), token, pr.ID, things.Thing{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	essvc := withEventStore(t, svc)
	esths, eserr := essvc.ListThingsByProfile(context.Background(), token, pr.ID, things.PageMetadata{Offset: 0, Limit: 10})
	thps, err := svc.ListThingsByProfile(context.Background(), token, pr.ID, things.PageMetadata{Offset: 0, Limit: 10})
	assert.Equal(t, thps, esths, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", thps, esths))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", err, eserr))
}

func TestRemoveThing(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	// Create thing without sending event.
	sths, err := svc.CreateThings(context.Background(), token, prID, things.Thing{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sth := sths[0]

	svc = withEventStore(t, svc)

	cases := []struct {
		desc  string
		id    string
		key   string
		err   error
		event map[string]any
	}{
		{
			desc: "remove existing thing successfully",
			id:   sth.ID,
			key:  token,
			err:  nil,
			event: map[string]any{
				"id":           sth.ID,
				"operation":    mfevents.ThingRemove,
				"evt_org_id":   orgID,
				"evt_group_id": grID,
			},
		},
		{
			desc:  "remove non-existent thing",
			id:    strconv.FormatUint(math.MaxUint64, 10),
			key:   "",
			err:   dbutil.ErrNotFound,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.RemoveThings(context.Background(), tc.key, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &r.XReadArgs{
			Streams: []string{mfevents.ThingsStream, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]any
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			lastID = msg.ID

			assert.NotEmpty(t, event["occurred_at"], tc.desc)
			delete(event, "occurred_at")
		}

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}

func TestRemoveThings(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	otherGrs, err := svc.CreateGroups(context.Background(), token, orgID, things.Group{Name: "other-test-group"})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	otherGrID := otherGrs[0].ID

	otherPrs, err := svc.CreateProfiles(context.Background(), token, otherGrID, things.Profile{Name: "other-test-profile"})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	otherPrID := otherPrs[0].ID

	sths, err := svc.CreateThings(context.Background(), token, prID, things.Thing{Name: "a"}, things.Thing{Name: "b"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	otherSths, err := svc.CreateThings(context.Background(), token, otherPrID, things.Thing{Name: "c"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	ids := []string{sths[0].ID, sths[1].ID, otherSths[0].ID}

	svc = withEventStore(t, svc)

	err = svc.RemoveThings(context.Background(), token, ids...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	expected := []map[string]any{
		{
			"id":           sths[0].ID,
			"operation":    mfevents.ThingRemove,
			"evt_org_id":   orgID,
			"evt_group_id": grID,
		},
		{
			"id":           sths[1].ID,
			"operation":    mfevents.ThingRemove,
			"evt_org_id":   orgID,
			"evt_group_id": grID,
		},
		{
			"id":           otherSths[0].ID,
			"operation":    mfevents.ThingRemove,
			"evt_org_id":   orgID,
			"evt_group_id": otherGrID,
		},
	}

	lastID := "0"
	for i, exp := range expected {
		streams := redisClient.XRead(context.Background(), &r.XReadArgs{
			Streams: []string{mfevents.ThingsStream, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		require.NotEmpty(t, streams, fmt.Sprintf("event %d: expected an event, got none", i))
		require.NotEmpty(t, streams[0].Messages, fmt.Sprintf("event %d: expected an event, got none", i))

		msg := streams[0].Messages[0]
		event := msg.Values
		lastID = msg.ID

		assert.NotEmpty(t, event["occurred_at"], fmt.Sprintf("event %d", i))
		delete(event, "occurred_at")

		assert.Equal(t, exp, event, fmt.Sprintf("event %d: expected %v got %v\n", i, exp, event))
	}
}

func TestCreateProfiles(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	svc = withEventStore(t, svc)

	cases := []struct {
		desc    string
		prs     []things.Profile
		token   string
		groupID string
		err     error
		event   map[string]any
	}{
		{
			desc:    "create profiles successfully",
			prs:     []things.Profile{{Name: "a", Metadata: map[string]any{"test": "test"}}},
			token:   token,
			groupID: gr.ID,
			err:     nil,
			event: map[string]any{
				"id":           "123e4567-e89b-12d3-a456-000000000002",
				"name":         "a",
				"metadata":     "{\"test\":\"test\"}",
				"group_id":     gr.ID,
				"operation":    mfevents.ProfileCreate,
				"evt_org_id":   orgID,
				"evt_group_id": gr.ID,
			},
		},
		{
			desc:    "create profiles with invalid credentials",
			prs:     []things.Profile{{Name: "a", Metadata: map[string]any{"test": "test"}}},
			token:   "",
			groupID: gr.ID,
			err:     errors.ErrAuthentication,
			event:   nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, err := svc.CreateProfiles(context.Background(), tc.token, tc.groupID, tc.prs...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &r.XReadArgs{
			Streams: []string{mfevents.ThingsStream, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]any
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			lastID = msg.ID

			assert.NotEmpty(t, event["occurred_at"], tc.desc)
			delete(event, "occurred_at")
		}

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}

func TestUpdateProfile(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	sprs, err := svc.CreateProfiles(context.Background(), token, gr.ID, things.Profile{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	spr := sprs[0]

	svc = withEventStore(t, svc)

	cases := []struct {
		desc    string
		profile things.Profile
		key     string
		err     error
		event   map[string]any
	}{
		{
			desc: "update profile successfully",
			profile: things.Profile{
				ID:       spr.ID,
				Name:     "b",
				Metadata: map[string]any{"test": "test"},
			},
			key: token,
			err: nil,
			event: map[string]any{
				"id":           spr.ID,
				"name":         "b",
				"metadata":     "{\"test\":\"test\"}",
				"operation":    mfevents.ProfileUpdate,
				"evt_org_id":   orgID,
				"evt_group_id": gr.ID,
			},
		},
		{
			desc: "update non-existent profile",
			profile: things.Profile{
				ID:   strconv.FormatUint(math.MaxUint64, 10),
				Name: "c",
			},
			key:   token,
			err:   dbutil.ErrNotFound,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.UpdateProfile(context.Background(), tc.key, tc.profile)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &r.XReadArgs{
			Streams: []string{mfevents.ThingsStream, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]any
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			lastID = msg.ID

			assert.NotEmpty(t, event["occurred_at"], tc.desc)
			delete(event, "occurred_at")
		}

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}

func TestViewProfile(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]
	// Create profile without sending event.
	sprs, err := svc.CreateProfiles(context.Background(), token, gr.ID, things.Profile{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	spr := sprs[0]

	essvc := withEventStore(t, svc)
	espr, eserr := essvc.ViewProfile(context.Background(), token, spr.ID)
	pr, err := svc.ViewProfile(context.Background(), token, spr.ID)
	assert.Equal(t, pr, espr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", pr, espr))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", err, eserr))
}

func TestListProfiles(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]
	// Create thing without sending event.
	_, err = svc.CreateProfiles(context.Background(), token, gr.ID, things.Profile{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	essvc := withEventStore(t, svc)
	esprs, eserr := essvc.ListProfiles(context.Background(), token, things.PageMetadata{Offset: 0, Limit: 10})
	prs, err := svc.ListProfiles(context.Background(), token, things.PageMetadata{Offset: 0, Limit: 10})
	assert.Equal(t, prs, esprs, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", prs, esprs))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", err, eserr))
}

func TestListProfilesByThing(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	// Create thing without sending event.
	sths, err := svc.CreateThings(context.Background(), token, prID, things.Thing{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sth := sths[0]

	essvc := withEventStore(t, svc)
	esprs, eserr := essvc.ViewProfileByThing(context.Background(), token, sth.ID)
	prps, err := svc.ViewProfileByThing(context.Background(), token, sth.ID)
	assert.Equal(t, prps, esprs, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", prps, esprs))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", err, eserr))
}

func TestRemoveProfile(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]
	// Create profile without sending event.
	sprs, err := svc.CreateProfiles(context.Background(), token, gr.ID, things.Profile{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	spr := sprs[0]

	svc = withEventStore(t, svc)

	cases := []struct {
		desc  string
		id    string
		key   string
		err   error
		event map[string]any
	}{
		{
			desc: "remove profile successfully",
			id:   spr.ID,
			key:  token,
			err:  nil,
			event: map[string]any{
				"id":           spr.ID,
				"operation":    mfevents.ProfileRemove,
				"evt_org_id":   orgID,
				"evt_group_id": gr.ID,
			},
		},
		{
			desc:  "remove non-existent profile",
			id:    strconv.FormatUint(math.MaxUint64, 10),
			key:   "",
			err:   dbutil.ErrNotFound,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.RemoveProfiles(context.Background(), tc.key, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &r.XReadArgs{
			Streams: []string{mfevents.ThingsStream, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]any
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			lastID = msg.ID

			assert.NotEmpty(t, event["occurred_at"], tc.desc)
			delete(event, "occurred_at")
		}

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}

// nextEvent reads the event after lastID from the things stream, without its occurred_at field.
func nextEvent(t *testing.T, desc, lastID string) (map[string]any, string) {
	t.Helper()
	streams := redisClient.XRead(context.Background(), &r.XReadArgs{
		Streams: []string{mfevents.ThingsStream, lastID},
		Count:   1,
		Block:   time.Second,
	}).Val()

	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, lastID
	}

	msg := streams[0].Messages[0]
	assert.NotEmpty(t, msg.Values["occurred_at"], desc)
	delete(msg.Values, "occurred_at")

	return msg.Values, msg.ID
}

// setupThing creates a group, a profile and a thing without sending events.
func setupThing(t *testing.T, svc things.Service) (string, things.Thing) {
	t.Helper()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	th := things.Thing{Name: "a", Metadata: map[string]any{"test": "test"}}
	ths, err := svc.CreateThings(context.Background(), token, prs[0].ID, th)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	return grID, ths[0]
}

func TestUpdateThingsMetadata(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	grID, th := setupThing(t, svc)
	svc = withEventStore(t, svc)

	update := things.Thing{
		ID:       th.ID,
		Metadata: map[string]any{"new": "value"},
	}

	cases := []struct {
		desc  string
		thing things.Thing
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "update things metadata successfully",
			thing: update,
			token: token,
			err:   nil,
			event: map[string]any{
				"id":           th.ID,
				"metadata":     "{\"new\":\"value\"}",
				"operation":    mfevents.ThingUpdateMetadata,
				"evt_org_id":   orgID,
				"evt_group_id": grID,
			},
		},
		{
			desc:  "update things metadata with invalid credentials",
			thing: update,
			token: "",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.UpdateThingsMetadata(context.Background(), tc.token, tc.thing)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestUpdateExternalKey(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	grID, th := setupThing(t, svc)
	svc = withEventStore(t, svc)

	cases := []struct {
		desc  string
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "update external key successfully",
			token: token,
			err:   nil,
			event: map[string]any{
				"id":           th.ID,
				"operation":    mfevents.ThingUpdateExternalKey,
				"evt_org_id":   orgID,
				"evt_group_id": grID,
			},
		},
		{
			desc:  "update external key with invalid credentials",
			token: "",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.UpdateExternalKey(context.Background(), tc.token, "external-key", th.ID)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestRemoveExternalKey(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	grID, th := setupThing(t, svc)
	err := svc.UpdateExternalKey(context.Background(), token, "external-key", th.ID)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	svc = withEventStore(t, svc)

	cases := []struct {
		desc  string
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "remove external key successfully",
			token: token,
			err:   nil,
			event: map[string]any{
				"id":           th.ID,
				"operation":    mfevents.ThingRemoveExternalKey,
				"evt_org_id":   orgID,
				"evt_group_id": grID,
			},
		},
		{
			desc:  "remove external key with invalid credentials",
			token: "",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.RemoveExternalKey(context.Background(), tc.token, th.ID)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestCreateGroups(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := withEventStore(t, newService())

	gr := things.Group{
		Name:        "test-group",
		Description: "test-group-desc",
		Metadata:    map[string]any{"test": "test"},
	}

	cases := []struct {
		desc  string
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "create groups successfully",
			token: token,
			err:   nil,
			event: map[string]any{
				"name":        gr.Name,
				"description": gr.Description,
				"metadata":    "{\"test\":\"test\"}",
				"operation":   mfevents.GroupCreate,
				"evt_org_id":  orgID,
			},
		},
		{
			desc:  "create groups with invalid credentials",
			token: "",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		grs, err := svc.CreateGroups(context.Background(), tc.token, orgID, gr)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)
		if len(grs) > 0 {
			tc.event["id"] = grs[0].ID
			tc.event["evt_group_id"] = grs[0].ID
		}

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestUpdateGroup(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID
	svc = withEventStore(t, svc)

	update := things.Group{
		ID:          grID,
		Name:        "updated-group",
		Description: "updated-group-desc",
		Metadata:    map[string]any{"test": "updated"},
	}

	cases := []struct {
		desc  string
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "update group successfully",
			token: token,
			err:   nil,
			event: map[string]any{
				"id":           grID,
				"name":         update.Name,
				"description":  update.Description,
				"metadata":     "{\"test\":\"updated\"}",
				"operation":    mfevents.GroupUpdate,
				"evt_org_id":   orgID,
				"evt_group_id": grID,
			},
		},
		{
			desc:  "update group with invalid credentials",
			token: "",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, err := svc.UpdateGroup(context.Background(), tc.token, update)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestCreateGroupMemberships(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID
	svc = withEventStore(t, svc)

	gm := things.GroupMembership{GroupID: grID, MemberID: otherUser.ID, Role: things.Viewer}

	cases := []struct {
		desc  string
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "create group memberships with invalid credentials",
			token: "",
			err:   errors.ErrAuthentication,
			event: nil,
		},
		{
			desc:  "create group memberships successfully",
			token: token,
			err:   nil,
			event: map[string]any{
				"group_id":     grID,
				"member_id":    otherUser.ID,
				"role":         things.Viewer,
				"operation":    mfevents.GroupMembershipCreate,
				"evt_org_id":   orgID,
				"evt_group_id": grID,
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.CreateGroupMemberships(context.Background(), tc.token, "", gm)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestUpdateGroupMemberships(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	gm := things.GroupMembership{GroupID: grID, MemberID: otherUser.ID, Role: things.Viewer}
	err = svc.CreateGroupMemberships(context.Background(), token, "", gm)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	svc = withEventStore(t, svc)

	gm.Role = things.Editor

	cases := []struct {
		desc  string
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "update group memberships successfully",
			token: token,
			err:   nil,
			event: map[string]any{
				"group_id":     grID,
				"member_id":    otherUser.ID,
				"role":         things.Editor,
				"operation":    mfevents.GroupMembershipUpdate,
				"evt_org_id":   orgID,
				"evt_group_id": grID,
			},
		},
		{
			desc:  "update group memberships with invalid credentials",
			token: "",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.UpdateGroupMemberships(context.Background(), tc.token, gm)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestRemoveGroupMemberships(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	gm := things.GroupMembership{GroupID: grID, MemberID: otherUser.ID, Role: things.Viewer}
	err = svc.CreateGroupMemberships(context.Background(), token, "", gm)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	svc = withEventStore(t, svc)

	cases := []struct {
		desc  string
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "remove group memberships with invalid credentials",
			token: "",
			err:   errors.ErrAuthentication,
			event: nil,
		},
		{
			desc:  "remove group memberships successfully",
			token: token,
			err:   nil,
			event: map[string]any{
				"group_id":     grID,
				"member_id":    otherUser.ID,
				"operation":    mfevents.GroupMembershipRemove,
				"evt_org_id":   orgID,
				"evt_group_id": grID,
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.RemoveGroupMemberships(context.Background(), tc.token, grID, otherUser.ID)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}
