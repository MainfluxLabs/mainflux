// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	sdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	authapi "github.com/MainfluxLabs/mainflux/things/api/http"
	httpapi "github.com/MainfluxLabs/mainflux/things/api/http"
	thmocks "github.com/MainfluxLabs/mainflux/things/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	contentType = "application/senml+json"
	email       = "user@example.com"
	adminEmail  = "admin@example.com"
	otherEmail  = "other_user@example.com"
	token       = email
	otherToken  = otherEmail
	wrongValue  = "wrong_value"
	orgID       = "1"
	wrongID     = "999"
	badKey      = "999"
	emptyValue  = ""
)

var (
	metadata   = map[string]interface{}{"meta": "data"}
	metadata2  = map[string]interface{}{"meta": "data2"}
	th1        = sdk.Thing{GroupID: "1", ID: "fe6b4e92-cc98-425e-b0aa-000000000001", Name: "test1", Metadata: metadata}
	th2        = sdk.Thing{GroupID: "1", ID: "fe6b4e92-cc98-425e-b0aa-000000000002", Name: "test2", Metadata: metadata}
	emptyThing = sdk.Thing{GroupID: "1"}
	group      = sdk.Group{OrgID: "1", Name: "test_group", Metadata: metadata}
)

func newThingsService() things.Service {
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

func newThingsServer(svc things.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(mocktracer.New(), svc, logger)
	return httptest.NewServer(mux)
}

func newAuthServer(svc things.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := authapi.MakeHandler(mocktracer.New(), svc, logger)
	return httptest.NewServer(mux)
}

func TestCreateThing(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()

	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc     string
		thing    sdk.Thing
		groupID  string
		token    string
		err      error
		location string
	}{
		{
			desc:     "create new thing",
			thing:    th1,
			groupID:  groupID,
			token:    token,
			err:      nil,
			location: th1.ID,
		},
		{
			desc:     "create new empty thing",
			thing:    emptyThing,
			groupID:  groupID,
			token:    token,
			err:      nil,
			location: fmt.Sprintf("%s%012d", uuid.Prefix, 2),
		},
		{
			desc:     "create new thing with empty token",
			thing:    th1,
			groupID:  groupID,
			token:    "",
			err:      createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
			location: "",
		},
		{
			desc:     "create new thing with invalid token",
			thing:    th1,
			groupID:  groupID,
			token:    wrongValue,
			err:      createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
			location: "",
		},
	}
	for _, tc := range cases {
		loc, err := mainfluxSDK.CreateThing(tc.thing, tc.groupID, tc.token)

		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.location, loc, fmt.Sprintf("%s: expected location %s got %s", tc.desc, tc.location, loc))
	}
}

func TestCreateThings(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()

	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	things := []sdk.Thing{
		th1,
		th2,
	}
	thsExtID := []sdk.Thing{
		{GroupID: "1", ID: th1.ID, Name: "1", Key: "1", Metadata: metadata},
		{GroupID: "1", ID: th2.ID, Name: "2", Key: "2", Metadata: metadata},
	}
	thsWrongExtID := []sdk.Thing{
		{ID: "b0aa-000000000001", Name: "1", Key: "1", Metadata: metadata},
		{ID: "b0aa-000000000002", Name: "2", Key: "2", Metadata: metadata2},
	}

	cases := []struct {
		desc   string
		things []sdk.Thing
		token  string
		err    error
		res    []sdk.Thing
	}{
		{
			desc:   "create new things",
			things: things,
			token:  token,
			err:    nil,
			res:    things,
		},
		{
			desc:   "create new things with empty things",
			things: []sdk.Thing{},
			token:  token,
			err:    createError(sdk.ErrFailedCreation, http.StatusBadRequest),
			res:    []sdk.Thing{},
		},
		{
			desc:   "create new thing with empty token",
			things: things,
			token:  "",
			err:    createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
			res:    []sdk.Thing{},
		},
		{
			desc:   "create new thing with invalid token",
			things: things,
			token:  wrongValue,
			err:    createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
			res:    []sdk.Thing{},
		},
		{
			desc:   "create new things with external UUID",
			things: thsExtID,
			token:  token,
			err:    nil,
			res:    things,
		},
		{
			desc:   "create new things with wrong external UUID",
			things: thsWrongExtID,
			token:  token,
			err:    createError(sdk.ErrFailedCreation, http.StatusBadRequest),
			res:    []sdk.Thing{},
		},
	}
	for _, tc := range cases {
		res, err := mainfluxSDK.CreateThings(tc.things, groupID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))

		for idx := range tc.res {
			assert.Equal(t, tc.res[idx].ID, res[idx].ID, fmt.Sprintf("%s: expected response ID %s got %s", tc.desc, tc.res[idx].ID, res[idx].ID))
		}
	}
}

func TestThing(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()

	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	id, err := mainfluxSDK.CreateThing(th1, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th1.Key = fmt.Sprintf("%s%012d", uuid.Prefix, 2)
	th1.GroupID = grID

	cases := []struct {
		desc     string
		thID     string
		token    string
		err      error
		response sdk.Thing
	}{
		{
			desc:     "get existing thing",
			thID:     id,
			token:    token,
			err:      nil,
			response: th1,
		},
		{
			desc:     "get non-existent thing",
			thID:     "43",
			token:    token,
			err:      createError(sdk.ErrFailedFetch, http.StatusNotFound),
			response: sdk.Thing{},
		},
		{
			desc:     "get thing with invalid token",
			thID:     id,
			token:    wrongValue,
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: sdk.Thing{},
		},
	}

	for _, tc := range cases {
		respTh, err := mainfluxSDK.Thing(tc.thID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, respTh, fmt.Sprintf("%s: expected response thing %s, got %s", tc.desc, tc.response, respTh))
	}
}

func TestThings(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}
	var things []sdk.Thing

	mainfluxSDK := sdk.NewSDK(sdkConf)
	for i := 1; i < 101; i++ {
		id := fmt.Sprintf("%s%012d", chPrefix, i)
		name := fmt.Sprintf("test-%d", i)
		th := sdk.Thing{GroupID: groupID, ID: id, Name: name, Metadata: metadata}
		_, err := mainfluxSDK.CreateThing(th, th.GroupID, token)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th.Key = fmt.Sprintf("%s%012d", uuid.Prefix, i)
		things = append(things, th)
	}

	cases := []struct {
		desc     string
		token    string
		offset   uint64
		limit    uint64
		err      error
		response []sdk.Thing
		name     string
		metadata map[string]interface{}
	}{
		{
			desc:     "get a list of things",
			token:    token,
			offset:   offset,
			limit:    limit,
			err:      nil,
			response: things[0:limit],
			metadata: make(map[string]interface{}),
		},
		{
			desc:     "get a list of things with invalid token",
			token:    wrongValue,
			offset:   offset,
			limit:    limit,
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: nil,
			metadata: make(map[string]interface{}),
		},
		{
			desc:     "get a list of things with empty token",
			token:    "",
			offset:   offset,
			limit:    limit,
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: nil,
			metadata: make(map[string]interface{}),
		},
		{
			desc:     "get a list of things with zero limit",
			token:    token,
			offset:   0,
			limit:    0,
			err:      createError(sdk.ErrFailedFetch, http.StatusBadRequest),
			response: nil,
			metadata: make(map[string]interface{}),
		},
		{
			desc:     "get a list of things with limit greater than max",
			token:    token,
			offset:   offset,
			limit:    110,
			err:      createError(sdk.ErrFailedFetch, http.StatusBadRequest),
			response: nil,
			metadata: make(map[string]interface{}),
		},
		{
			desc:     "get a list of things with offset greater than max",
			token:    token,
			offset:   110,
			limit:    limit,
			err:      nil,
			response: []sdk.Thing{},
			metadata: make(map[string]interface{}),
		},
	}
	for _, tc := range cases {
		filter := sdk.PageMetadata{
			Name:     tc.name,
			Total:    total,
			Offset:   uint64(tc.offset),
			Limit:    uint64(tc.limit),
			Metadata: tc.metadata,
		}
		page, err := mainfluxSDK.Things(tc.token, filter)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page.Things, fmt.Sprintf("%s: expected response channel %s, got %s", tc.desc, tc.response, page.Things))
	}
}

func TestThingsByChannel(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	ch := sdk.Channel{Name: "test_channel"}
	cid, err := mainfluxSDK.CreateChannel(ch, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var n = 10
	var thsDiscoNum = 1
	var ths []sdk.Thing
	for i := 1; i < n+1; i++ {
		id := fmt.Sprintf("%s%012d", chPrefix, i)
		name := fmt.Sprintf("test-%d", i)
		th := sdk.Thing{
			ID:       id,
			Name:     name,
			GroupID:  grID,
			Metadata: metadata,
			Key:      fmt.Sprintf("%s%012d", uuid.Prefix, 2*i+1),
		}
		tid, err := mainfluxSDK.CreateThing(th, th.GroupID, token)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		ths = append(ths, th)

		// Don't connect last Thing
		if i == n+1-thsDiscoNum {
			break
		}

		// Don't connect last 2 Channels
		connIDs := sdk.ConnectionIDs{
			ChannelID: cid,
			ThingIDs:  []string{tid},
		}

		err = mainfluxSDK.Connect(connIDs, token)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	cases := []struct {
		desc     string
		channel  string
		token    string
		offset   uint64
		limit    uint64
		err      error
		response []sdk.Thing
	}{
		{
			desc:     "get a list of things by channel",
			channel:  cid,
			token:    token,
			offset:   offset,
			limit:    limit,
			err:      nil,
			response: ths[0:limit],
		},
		{
			desc:     "get a list of things by channel with invalid token",
			channel:  cid,
			token:    wrongValue,
			offset:   offset,
			limit:    limit,
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of things by channel with empty token",
			channel:  cid,
			token:    "",
			offset:   offset,
			limit:    limit,
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of things by channel with zero limit",
			channel:  cid,
			token:    token,
			offset:   offset,
			limit:    0,
			err:      createError(sdk.ErrFailedFetch, http.StatusBadRequest),
			response: nil,
		},
		{
			desc:     "get a list of things by channel with limit greater than max",
			channel:  cid,
			token:    token,
			offset:   offset,
			limit:    110,
			err:      createError(sdk.ErrFailedFetch, http.StatusBadRequest),
			response: nil,
		},
		{
			desc:     "get a list of things by channel with offset greater than max",
			channel:  cid,
			token:    token,
			offset:   110,
			limit:    limit,
			err:      nil,
			response: []sdk.Thing{},
		},
		{
			desc:     "get a list of things by channel with invalid args (zero limit) and invalid token",
			channel:  cid,
			token:    wrongValue,
			offset:   offset,
			limit:    0,
			err:      createError(sdk.ErrFailedFetch, http.StatusBadRequest),
			response: nil,
		},
	}
	for _, tc := range cases {
		page, err := mainfluxSDK.ThingsByChannel(tc.token, tc.channel, tc.offset, tc.limit)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page.Things, fmt.Sprintf("%s: expected response channel %s, got %s", tc.desc, tc.response, page.Things))
	}
}

func TestUpdateThing(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	id, err := mainfluxSDK.CreateThing(th1, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th1.Name = "test2"

	cases := []struct {
		desc  string
		thing sdk.Thing
		token string
		err   error
	}{
		{
			desc: "update existing thing",
			thing: sdk.Thing{
				ID:       id,
				Name:     "test_app",
				Metadata: metadata2,
			},
			token: token,
			err:   nil,
		},
		{
			desc: "update non-existing thing",
			thing: sdk.Thing{
				ID:       "0",
				Name:     "test_device",
				Metadata: metadata,
			},
			token: token,
			err:   createError(sdk.ErrFailedUpdate, http.StatusNotFound),
		},
		{
			desc: "update channel with invalid id",
			thing: sdk.Thing{
				ID:       "",
				Name:     "test_device",
				Metadata: metadata,
			},
			token: token,
			err:   createError(sdk.ErrFailedUpdate, http.StatusBadRequest),
		},
		{
			desc: "update channel with invalid token",
			thing: sdk.Thing{
				ID:       id,
				Name:     "test_app",
				Metadata: metadata2,
			},
			token: wrongValue,
			err:   createError(sdk.ErrFailedUpdate, http.StatusUnauthorized),
		},
		{
			desc: "update channel with empty token",
			thing: sdk.Thing{
				ID:       id,
				Name:     "test_app",
				Metadata: metadata2,
			},
			token: "",
			err:   createError(sdk.ErrFailedUpdate, http.StatusUnauthorized),
		},
	}

	for _, tc := range cases {
		err := mainfluxSDK.UpdateThing(tc.thing, tc.thing.ID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestDeleteThing(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	id, err := mainfluxSDK.CreateThing(th1, groupID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc    string
		thingID string
		token   string
		err     error
	}{
		{
			desc:    "delete thing with invalid token",
			thingID: id,
			token:   wrongValue,
			err:     createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:    "delete non-existing thing",
			thingID: "2",
			token:   token,
			err:     createError(sdk.ErrFailedRemoval, http.StatusNotFound),
		},
		{
			desc:    "delete thing with invalid id",
			thingID: "",
			token:   token,
			err:     createError(sdk.ErrFailedRemoval, http.StatusBadRequest),
		},
		{
			desc:    "delete thing with empty token",
			thingID: id,
			token:   "",
			err:     createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:    "delete existing thing",
			thingID: id,
			token:   token,
			err:     nil,
		},
		{
			desc:    "delete deleted thing",
			thingID: id,
			token:   token,
			err:     createError(sdk.ErrFailedRemoval, http.StatusNotFound),
		},
	}

	for _, tc := range cases {
		err := mainfluxSDK.DeleteThing(tc.thingID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestDeleteThings(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	id1, err := mainfluxSDK.CreateThing(th1, groupID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	thIDs := []string{id1}

	cases := []struct {
		desc     string
		thingIDs []string
		token    string
		err      error
	}{
		{
			desc:     "delete things with invalid token",
			thingIDs: thIDs,
			token:    wrongValue,
			err:      createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:     "delete non-existing things",
			thingIDs: []string{wrongID},
			token:    token,
			err:      createError(sdk.ErrFailedRemoval, http.StatusNotFound),
		},
		{
			desc:     "delete things with invalid id",
			thingIDs: []string{""},
			token:    token,
			err:      createError(sdk.ErrFailedRemoval, http.StatusBadRequest),
		},
		{
			desc:     "delete things with empty token",
			thingIDs: thIDs,
			token:    "",
			err:      createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:     "delete existing things",
			thingIDs: thIDs,
			token:    token,
			err:      nil,
		},
		{
			desc:     "delete deleted things",
			thingIDs: thIDs,
			token:    token,
			err:      createError(sdk.ErrFailedRemoval, http.StatusNotFound),
		},
	}

	for _, tc := range cases {
		err := mainfluxSDK.DeleteThings(tc.thingIDs, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestIdentifyThing(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	as := newAuthServer(svc)
	defer ts.Close()
	defer as.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}
	authSdkConf := sdk.Config{
		ThingsURL:       as.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	mainfluxAuthSDK := sdk.NewSDK(authSdkConf)
	th := sdk.Thing{ID: "fe6b4e92-cc98-425e-b0aa-000000007891", Name: "identify"}

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	id, err := mainfluxSDK.CreateThing(th, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	thing, err := mainfluxSDK.Thing(th.ID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc     string
		thingKey string
		err      error
		response string
	}{
		{
			desc:     "identify thing with valid key",
			thingKey: thing.Key,
			err:      nil,
			response: id,
		},
		{
			desc:     "identify thing with invalid key",
			thingKey: badKey,
			err:      createError(sdk.ErrFailedFetch, http.StatusNotFound),
			response: "",
		},
		{
			desc:     "identify thing with empty key",
			thingKey: "",
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: "",
		},
	}

	for _, tc := range cases {
		thingID, err := mainfluxAuthSDK.IdentifyThing(tc.thingKey)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, thingID, fmt.Sprintf("%s: expected response id %s, got %s", tc.desc, tc.response, thingID))
	}
}

func TestConnectThing(t *testing.T) {
	svc := newThingsService()

	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	thingID, err := mainfluxSDK.CreateThing(th1, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	chanID1, err := mainfluxSDK.CreateChannel(ch2, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	chanID2, err := mainfluxSDK.CreateChannel(ch3, grID, otherToken)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc    string
		thingID string
		chanID  string
		token   string
		err     error
	}{
		{
			desc:    "connect existing thing to existing channel",
			thingID: thingID,
			chanID:  chanID1,
			token:   token,
			err:     nil,
		},

		{
			desc:    "connect existing thing to non-existing channel",
			thingID: thingID,
			chanID:  "9",
			token:   token,
			err:     createError(sdk.ErrFailedConnect, http.StatusNotFound),
		},
		{
			desc:    "connect non-existing thing to existing channel",
			thingID: "9",
			chanID:  chanID1,
			token:   token,
			err:     createError(sdk.ErrFailedConnect, http.StatusNotFound),
		},
		{
			desc:    "connect existing thing to channel with invalid ID",
			thingID: thingID,
			chanID:  "",
			token:   token,
			err:     createError(sdk.ErrFailedConnect, http.StatusBadRequest),
		},
		{
			desc:    "connect thing with invalid ID to existing channel",
			thingID: "",
			chanID:  chanID1,
			token:   token,
			err:     createError(sdk.ErrFailedConnect, http.StatusBadRequest),
		},

		{
			desc:    "connect existing thing to existing channel with invalid token",
			thingID: thingID,
			chanID:  chanID1,
			token:   wrongValue,
			err:     createError(sdk.ErrFailedConnect, http.StatusUnauthorized),
		},
		{
			desc:    "connect existing thing to existing channel with empty token",
			thingID: thingID,
			chanID:  chanID1,
			token:   "",
			err:     createError(sdk.ErrFailedConnect, http.StatusUnauthorized),
		},
		{
			desc:    "connect thing from owner to channel of other user",
			thingID: thingID,
			chanID:  chanID2,
			token:   token,
			err:     createError(sdk.ErrFailedConnect, http.StatusConflict),
		},
	}

	for _, tc := range cases {
		connIDs := sdk.ConnectionIDs{
			ChannelID: tc.chanID,
			ThingIDs:  []string{tc.thingID},
		}
		err := mainfluxSDK.Connect(connIDs, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestConnect(t *testing.T) {
	svc := newThingsService()

	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	thingID, err := mainfluxSDK.CreateThing(th1, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	chanID1, err := mainfluxSDK.CreateChannel(ch2, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	chanID2, err := mainfluxSDK.CreateChannel(ch3, grID, otherToken)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc    string
		thingID string
		chanID  string
		token   string
		err     error
	}{
		{
			desc:    "connect existing things to existing channels",
			thingID: thingID,
			chanID:  chanID1,
			token:   token,
			err:     nil,
		},

		{
			desc:    "connect existing things to non-existing channels",
			thingID: thingID,
			chanID:  wrongID,
			token:   token,
			err:     createError(sdk.ErrFailedConnect, http.StatusNotFound),
		},
		{
			desc:    "connect non-existing things to existing channels",
			thingID: wrongID,
			chanID:  chanID1,
			token:   token,
			err:     createError(sdk.ErrFailedConnect, http.StatusNotFound),
		},
		{
			desc:    "connect existing things to channels with invalid ID",
			thingID: thingID,
			chanID:  emptyValue,
			token:   token,
			err:     createError(sdk.ErrFailedConnect, http.StatusBadRequest),
		},
		{
			desc:    "connect things with invalid ID to existing channels",
			thingID: emptyValue,
			chanID:  chanID1,
			token:   token,
			err:     createError(sdk.ErrFailedConnect, http.StatusBadRequest),
		},

		{
			desc:    "connect existing things to existing channels with invalid token",
			thingID: thingID,
			chanID:  chanID1,
			token:   wrongValue,
			err:     createError(sdk.ErrFailedConnect, http.StatusUnauthorized),
		},
		{
			desc:    "connect existing things to existing channels with empty token",
			thingID: thingID,
			chanID:  chanID1,
			token:   emptyValue,
			err:     createError(sdk.ErrFailedConnect, http.StatusUnauthorized),
		},
		{
			desc:    "connect things from owner to channels of other user",
			thingID: thingID,
			chanID:  chanID2,
			token:   token,
			err:     createError(sdk.ErrFailedConnect, http.StatusNotFound),
		},
	}

	for _, tc := range cases {
		connIDs := sdk.ConnectionIDs{
			tc.thingID,
			[]string{tc.chanID},
		}

		err := mainfluxSDK.Connect(connIDs, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestDisconnect(t *testing.T) {
	svc := newThingsService()

	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	thingID, err := mainfluxSDK.CreateThing(th1, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	chanID1, err := mainfluxSDK.CreateChannel(ch2, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	connIDs := sdk.ConnectionIDs{
		ChannelID: chanID1,
		ThingIDs:  []string{thingID},
	}
	err = mainfluxSDK.Connect(connIDs, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	chanID2, err := mainfluxSDK.CreateChannel(ch2, grID, otherToken)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc       string
		disconnIDs sdk.ConnectionIDs
		token      string
		err        error
	}{
		{
			desc:       "disconnect connected thing from channel",
			disconnIDs: sdk.ConnectionIDs{ChannelID: chanID1, ThingIDs: []string{thingID}},
			token:      token,
			err:        nil,
		},
		{
			desc:       "disconnect existing thing from non-existing channel",
			disconnIDs: sdk.ConnectionIDs{ChannelID: wrongID, ThingIDs: []string{thingID}},
			token:      token,
			err:        createError(sdk.ErrFailedDisconnect, http.StatusNotFound),
		},
		{
			desc:       "disconnect non-existing thing from existing channel",
			disconnIDs: sdk.ConnectionIDs{ChannelID: chanID1, ThingIDs: []string{wrongID}},
			token:      token,
			err:        createError(sdk.ErrFailedDisconnect, http.StatusNotFound),
		},
		{
			desc:       "disconnect existing thing from channel with invalid ID",
			disconnIDs: sdk.ConnectionIDs{ChannelID: "", ThingIDs: []string{thingID}},
			token:      token,
			err:        createError(sdk.ErrFailedDisconnect, http.StatusBadRequest),
		},
		{
			desc:       "disconnect thing with invalid ID from existing channel",
			disconnIDs: sdk.ConnectionIDs{ChannelID: chanID1, ThingIDs: []string{""}},
			token:      token,
			err:        createError(sdk.ErrFailedDisconnect, http.StatusBadRequest),
		},
		{
			desc:       "disconnect existing thing from existing channel with invalid token",
			disconnIDs: sdk.ConnectionIDs{ChannelID: chanID1, ThingIDs: []string{thingID}},
			token:      wrongValue,
			err:        createError(sdk.ErrFailedDisconnect, http.StatusUnauthorized),
		},
		{
			desc:       "disconnect existing thing from existing channel with empty token",
			disconnIDs: sdk.ConnectionIDs{ChannelID: chanID1, ThingIDs: []string{thingID}},
			token:      "",
			err:        createError(sdk.ErrFailedDisconnect, http.StatusUnauthorized),
		},
		{
			desc:       "disconnect owner's thing from someone elses channel",
			disconnIDs: sdk.ConnectionIDs{ChannelID: chanID2, ThingIDs: []string{thingID}},
			token:      token,
			err:        createError(sdk.ErrFailedDisconnect, http.StatusNotFound),
		},
	}

	for _, tc := range cases {
		err := mainfluxSDK.Disconnect(tc.disconnIDs, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}
