// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	sdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
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
	wrongValue  = "wrong_value"
	orgID       = "374106f7-030e-4881-8ab0-151195c29f92"
	wrongID     = "999"
	badKey      = "999"
	emptyValue  = ""
)

var (
	metadata  = map[string]interface{}{"meta": "data"}
	metadata2 = map[string]interface{}{"meta": "data2"}
	th1       = sdk.Thing{GroupID: groupID, ID: "fe6b4e92-cc98-425e-b0aa-000000000001", Name: "test1", Metadata: metadata}
	th2       = sdk.Thing{GroupID: groupID, ID: "fe6b4e92-cc98-425e-b0aa-000000000002", Name: "test2", Metadata: metadata}
	profile   = sdk.Profile{ID: "fe6b4e92-cc98-425e-b0aa-000000000003", Name: "test1"}
	group     = sdk.Group{OrgID: orgID, Name: "test_group", Metadata: metadata}
	orgs      = []auth.Org{{ID: orgID, OwnerID: user.ID}}
)

func newThingsService() things.Service {
	auth := mocks.NewAuthService("", usersList, orgs)
	thingsRepo := thmocks.NewThingRepository()
	profilesRepo := thmocks.NewProfileRepository(thingsRepo)
	groupMembersRepo := thmocks.NewGroupMembersRepository()
	groupsRepo := thmocks.NewGroupRepository(groupMembersRepo)
	profileCache := thmocks.NewProfileCache()
	thingCache := thmocks.NewThingCache()
	groupCache := thmocks.NewGroupCache()
	idProvider := uuid.NewMock()

	return things.New(auth, nil, thingsRepo, profilesRepo, groupsRepo, groupMembersRepo, profileCache, thingCache, groupCache, idProvider)
}

func newThingsServer(svc things.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(svc, mocktracer.New(), logger)
	return httptest.NewServer(mux)
}

func newAuthServer(svc things.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(svc, mocktracer.New(), logger)
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

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID, err := mainfluxSDK.CreateProfile(profile, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th1.ProfileID = prID

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
			groupID:  grID,
			token:    token,
			err:      nil,
			location: th1.ID,
		},
		{
			desc:     "create new thing with empty token",
			thing:    th1,
			groupID:  grID,
			token:    emptyValue,
			err:      createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
			location: emptyValue,
		},
		{
			desc:     "create new thing with invalid token",
			thing:    th1,
			groupID:  grID,
			token:    wrongValue,
			err:      createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
			location: emptyValue,
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

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	prs, err := mainfluxSDK.CreateProfiles([]sdk.Profile{profile}, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	th1.ProfileID = prID
	th2.ProfileID = prID
	things := []sdk.Thing{
		th1,
		th2,
	}
	thsExtID := []sdk.Thing{
		{GroupID: grID, ID: th1.ID, ProfileID: prID, Name: "1", Key: "1", Metadata: metadata},
		{GroupID: grID, ID: th2.ID, ProfileID: prID, Name: "2", Key: "2", Metadata: metadata},
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
			token:  emptyValue,
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
		res, err := mainfluxSDK.CreateThings(tc.things, grID, tc.token)
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

	prID, err := mainfluxSDK.CreateProfile(profile, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	th1.ProfileID = prID
	th1.Key = fmt.Sprintf("%s%012d", uuid.Prefix, 2)
	th1.GroupID = grID
	id, err := mainfluxSDK.CreateThing(th1, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

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

func TestMetadataByKey(t *testing.T) {
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

	prID, err := mainfluxSDK.CreateProfile(profile, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	th1.ProfileID = prID
	th1.GroupID = grID
	th1.Key = fmt.Sprintf("%s%012d", uuid.Prefix, 1)
	_, err = mainfluxSDK.CreateThing(th1, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	otherKey := fmt.Sprintf("%s%012d", uuid.Prefix, 2)

	res := sdk.Metadata{
		"metadata": th1.Metadata,
	}

	cases := []struct {
		desc     string
		key      string
		err      error
		response sdk.Metadata
	}{
		{
			desc:     "get thing metadata",
			key:      th1.Key,
			err:      nil,
			response: res,
		},
		{
			desc:     "get metadata from a non-existing thing",
			key:      otherKey,
			err:      createError(sdk.ErrFailedFetch, http.StatusNotFound),
			response: sdk.Metadata{},
		},
		{
			desc:     "get thing metadata with empty key",
			key:      "",
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: sdk.Metadata{},
		},
	}

	for _, tc := range cases {
		resMeta, err := mainfluxSDK.MetadataByKey(tc.key)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, resMeta, fmt.Sprintf("%s: expected response thing %s, got %s", tc.desc, tc.response, resMeta))
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
	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	prID, err := mainfluxSDK.CreateProfile(profile, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	for i := 1; i < 101; i++ {
		id := fmt.Sprintf("%s%012d", prPrefix, i)
		name := fmt.Sprintf("test-%d", i)
		key := fmt.Sprintf("%s%012d", uuid.Prefix, i)
		th := sdk.Thing{GroupID: grID, ID: id, ProfileID: prID, Name: name, Key: key, Metadata: metadata}
		_, err := mainfluxSDK.CreateThing(th, th.GroupID, token)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		things = append(things, th)
	}

	cases := []struct {
		desc     string
		token    string
		offset   uint64
		limit    uint64
		dir      string
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
			dir:      ascDir,
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
			token:    emptyValue,
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
			Offset:   tc.offset,
			Limit:    tc.limit,
			Dir:      tc.dir,
			Metadata: tc.metadata,
		}
		page, err := mainfluxSDK.Things(tc.token, filter)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page.Things, fmt.Sprintf("%s: expected response profile %s, got %s", tc.desc, tc.response, page.Things))
	}
}

func TestThingsByProfile(t *testing.T) {
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

	pr := sdk.Profile{Name: "test_profile"}
	prID, err := mainfluxSDK.CreateProfile(pr, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var n = 10
	var ths []sdk.Thing
	for i := 1; i < n+1; i++ {
		id := fmt.Sprintf("%s%012d", prPrefix, i)
		name := fmt.Sprintf("test-%d", i)
		th := sdk.Thing{
			ID:        id,
			Name:      name,
			GroupID:   grID,
			ProfileID: prID,
			Metadata:  metadata,
			Key:       fmt.Sprintf("%s%012d", uuid.Prefix, 2*i+1),
		}
		_, err := mainfluxSDK.CreateThing(th, th.GroupID, token)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		ths = append(ths, th)
	}

	cases := []struct {
		desc     string
		profile  string
		token    string
		offset   uint64
		limit    uint64
		dir      string
		err      error
		response []sdk.Thing
	}{
		{
			desc:     "get a list of things by profile",
			profile:  prID,
			token:    token,
			offset:   offset,
			limit:    limit,
			dir:      ascDir,
			err:      nil,
			response: ths[0:limit],
		},
		{
			desc:     "get a list of things by profile with invalid token",
			profile:  prID,
			token:    wrongValue,
			offset:   offset,
			limit:    limit,
			dir:      ascDir,
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of things by profile with empty token",
			profile:  prID,
			token:    emptyValue,
			offset:   offset,
			limit:    limit,
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of things by profile with zero limit",
			profile:  prID,
			token:    token,
			offset:   offset,
			limit:    0,
			err:      createError(sdk.ErrFailedFetch, http.StatusBadRequest),
			response: nil,
		},
		{
			desc:     "get a list of things by profile with limit greater than max",
			profile:  prID,
			token:    token,
			offset:   offset,
			limit:    110,
			err:      createError(sdk.ErrFailedFetch, http.StatusBadRequest),
			response: nil,
		},
		{
			desc:     "get a list of things by profile with offset greater than max",
			profile:  prID,
			token:    token,
			offset:   110,
			limit:    limit,
			dir:      ascDir,
			err:      nil,
			response: []sdk.Thing{},
		},
		{
			desc:     "get a list of things by profile with invalid args (zero limit) and invalid token",
			profile:  prID,
			token:    wrongValue,
			offset:   offset,
			limit:    0,
			err:      createError(sdk.ErrFailedFetch, http.StatusBadRequest),
			response: nil,
		},
	}
	for _, tc := range cases {
		filter := sdk.PageMetadata{
			Total:  total,
			Offset: tc.offset,
			Limit:  tc.limit,
			Dir:    tc.dir,
		}
		page, err := mainfluxSDK.ThingsByProfile(tc.profile, tc.token, filter)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page.Things, fmt.Sprintf("%s: expected response profile %s, got %s", tc.desc, tc.response, page.Things))
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

	prID, err := mainfluxSDK.CreateProfile(profile, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	th1.ProfileID = prID
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
				ID:        id,
				ProfileID: prID,
				Name:      "test_app",
				Metadata:  metadata2,
			},
			token: token,
			err:   nil,
		},
		{
			desc: "update non-existing thing",
			thing: sdk.Thing{
				ID:        "0",
				ProfileID: prID,
				Name:      "test_device",
				Metadata:  metadata,
			},
			token: token,
			err:   createError(sdk.ErrFailedUpdate, http.StatusNotFound),
		},
		{
			desc: "update profile with invalid id",
			thing: sdk.Thing{
				ID:        emptyValue,
				ProfileID: prID,
				Name:      "test_device",
				Metadata:  metadata,
			},
			token: token,
			err:   createError(sdk.ErrFailedUpdate, http.StatusBadRequest),
		},
		{
			desc: "update profile with invalid token",
			thing: sdk.Thing{
				ID:        id,
				ProfileID: prID,
				Name:      "test_app",
				Metadata:  metadata2,
			},
			token: wrongValue,
			err:   createError(sdk.ErrFailedUpdate, http.StatusUnauthorized),
		},
		{
			desc: "update profile with empty token",
			thing: sdk.Thing{
				ID:        id,
				ProfileID: prID,
				Name:      "test_app",
				Metadata:  metadata2,
			},
			token: emptyValue,
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

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	prID, err := mainfluxSDK.CreateProfile(profile, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	th1.ProfileID = prID
	id, err := mainfluxSDK.CreateThing(th1, grID, token)
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
			thingID: emptyValue,
			token:   token,
			err:     createError(sdk.ErrFailedRemoval, http.StatusBadRequest),
		},
		{
			desc:    "delete thing with empty token",
			thingID: id,
			token:   emptyValue,
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

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	prID, err := mainfluxSDK.CreateProfile(profile, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	th1.ProfileID = prID
	id1, err := mainfluxSDK.CreateThing(th1, grID, token)
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
			thingIDs: []string{emptyValue},
			token:    token,
			err:      createError(sdk.ErrFailedRemoval, http.StatusBadRequest),
		},
		{
			desc:     "delete things with empty token",
			thingIDs: thIDs,
			token:    emptyValue,
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

	prID, err := mainfluxSDK.CreateProfile(profile, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	th.ProfileID = prID
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
			response: emptyValue,
		},
		{
			desc:     "identify thing with empty key",
			thingKey: emptyValue,
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: emptyValue,
		},
	}

	for _, tc := range cases {
		thingID, err := mainfluxAuthSDK.IdentifyThing(tc.thingKey)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, thingID, fmt.Sprintf("%s: expected response id %s, got %s", tc.desc, tc.response, thingID))
	}
}
