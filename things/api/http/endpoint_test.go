// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	httpapi "github.com/MainfluxLabs/mainflux/things/api/http"
	thmocks "github.com/MainfluxLabs/mainflux/things/mocks"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	contentType    = "application/json"
	email          = "user@example.com"
	adminEmail     = "admin@example.com"
	otherUserEmail = "other_user@example.com"
	token          = email
	otherToken     = otherUserEmail
	adminToken     = adminEmail
	wrongValue     = "wrong_value"
	emptyValue     = ""
	wrongID        = 0
	password       = "password"
	maxNameSize    = 1024
	nameKey        = "name"
	ascKey         = "asc"
	descKey        = "desc"
	orgID          = "374106f7-030e-4881-8ab0-151195c29f92"
	orgID2         = "374106f7-030e-4881-8ab0-151195c29f93"
	prefix         = "fe6b4e92-cc98-425e-b0aa-"
	n              = 101
	noLimit        = -1
)

var (
	thing = things.Thing{
		Name:     "test_app",
		Metadata: metadata,
	}
	profile = things.Profile{
		Name:     "test",
		Metadata: metadata,
	}
	profile1 = things.Profile{
		Name:     "test1",
		Metadata: metadata,
	}
	invalidName    = strings.Repeat("m", maxNameSize+1)
	searchThingReq = things.PageMetadata{
		Limit:  5,
		Offset: 0,
	}
	user      = users.User{ID: "574106f7-030e-4881-8ab0-151195c29f94", Email: email, Password: password, Role: auth.Owner}
	otherUser = users.User{ID: "ecf9e48b-ba3b-41c4-82a9-72e063b17868", Email: otherUserEmail, Password: password, Role: auth.Editor}
	admin     = users.User{ID: "2e248e36-2d26-46ea-97b0-1e38d674cbe4", Email: adminEmail, Password: password, Role: auth.RootSub}
	usersList = []users.User{admin, user, otherUser}
	group     = things.Group{Name: "test-group", Description: "test-group-desc", OrgID: orgID}
	orgsList  = []auth.Org{{ID: orgID, OwnerID: user.ID}, {ID: orgID2, OwnerID: user.ID}}
	metadata  = map[string]interface{}{"test": "data"}
)

type testRequest struct {
	client      *http.Client
	method      string
	url         string
	contentType string
	key         string
	token       string
	body        io.Reader
}

func (tr testRequest) make() (*http.Response, error) {
	req, err := http.NewRequest(tr.method, tr.url, tr.body)
	if err != nil {
		return nil, err
	}
	if tr.key != "" {
		req.Header.Set("Authorization", apiutil.ThingPrefix+tr.key)
	}
	if tr.token != "" {
		req.Header.Set("Authorization", apiutil.BearerPrefix+tr.token)
	}
	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}
	return tr.client.Do(req)
}

func newService() things.Service {
	auth := mocks.NewAuthService(admin.ID, usersList, orgsList)
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

func newServer(svc things.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(mocktracer.New(), svc, logger)
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func TestCreateThings(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID, grID1 := grs[0].ID, grs[1].ID

	profile.GroupID = grID
	profile1 := profile
	profile1.GroupID = grID1
	prs, err := svc.CreateProfiles(context.Background(), token, profile, profile1)
	prID, prID1 := prs[0].ID, prs[1].ID

	data := fmt.Sprintf(`[{"name": "1", "key": "1","profile_id":"%s"}, {"name": "2", "key": "2","profile_id":"%s"}]`, prID, prID)
	invalidNameData := fmt.Sprintf(`[{"name": "%s", "key": "10","profile_id":"%s"}]`, invalidName, prID)
	invalidProfileData := `[{"name": "test", "key": "1"}]`
	invalidGroupData := fmt.Sprintf(`[{"name": "test", "key": "10","profile_id":"%s"}]`, prID1)

	cases := []struct {
		desc        string
		data        string
		contentType string
		auth        string
		status      int
		response    string
	}{
		{
			desc:        "create valid things",
			data:        data,
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			response:    emptyValue,
		},
		{
			desc:        "create things with empty request",
			data:        emptyValue,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create thing with invalid request format",
			data:        "}",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create thing with invalid name",
			data:        invalidNameData,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create thing without profile id",
			data:        invalidProfileData,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create thing with profile from different group",
			data:        invalidGroupData,
			contentType: contentType,
			auth:        token,
			status:      http.StatusForbidden,
			response:    emptyValue,
		},
		{
			desc:        "create things with empty JSON array",
			data:        "[]",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create thing with existing key",
			data:        data,
			contentType: contentType,
			auth:        token,
			status:      http.StatusConflict,
			response:    emptyValue,
		},
		{
			desc:        "create thing with invalid auth token",
			data:        data,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
			response:    emptyValue,
		},
		{
			desc:        "create thing with empty auth token",
			data:        data,
			contentType: contentType,
			auth:        emptyValue,
			status:      http.StatusUnauthorized,
			response:    emptyValue,
		},
		{
			desc:        "create thing without content type",
			data:        data,
			contentType: emptyValue,
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
			response:    emptyValue,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/groups/%s/things", ts.URL, grID),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.data),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		location := res.Header.Get("Location")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.response, location, fmt.Sprintf("%s: expected response %s got %s", tc.desc, tc.response, location))
	}
}

func TestUpdateThing(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID, grID1 := grs[0].ID, grs[1].ID

	profile1 := profile
	profile1.GroupID = grID1
	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile, profile1)
	prID, prID1 := prs[0].ID, prs[1].ID

	thing.GroupID = grID
	thing.ProfileID = prID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	data := fmt.Sprintf(`{"name":"test","profile_id":"%s"}`, prID)
	invalidNameData := fmt.Sprintf(`{"name": "%s","profile_id":"%s"}`, invalidName, prID)
	invalidProfileData := `{"name": "test"}`
	invalidGroupData := fmt.Sprintf(`{"name":"test","profile_id":"%s"}`, prID1)

	cases := []struct {
		desc        string
		req         string
		id          string
		contentType string
		auth        string
		status      int
	}{
		{
			desc:        "update existing thing",
			req:         data,
			id:          th.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update thing with empty JSON request",
			req:         "{}",
			id:          th.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update non-existent thing",
			req:         data,
			id:          wrongValue,
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update thing with invalid id",
			req:         data,
			id:          wrongValue,
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update thing with invalid user token",
			req:         data,
			id:          th.ID,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update thing with empty user token",
			req:         data,
			id:          th.ID,
			contentType: contentType,
			auth:        emptyValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update thing with invalid data format",
			req:         "{",
			id:          th.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing with empty request",
			req:         emptyValue,
			id:          th.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing without content type",
			req:         data,
			id:          th.ID,
			contentType: emptyValue,
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "update thing with invalid name",
			req:         invalidNameData,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing without profile id",
			req:         invalidProfileData,
			id:          th.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing with profile from different group",
			req:         invalidGroupData,
			id:          th.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusForbidden,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/things/%s", ts.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestUpdateKey(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	prID := prs[0].ID

	th := thing
	th.Key = "key"
	th.GroupID = grID
	th.ProfileID = prID
	ths, err := svc.CreateThings(context.Background(), token, th)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th = ths[0]

	th.Key = "new-key"
	data := toJSON(th)

	th.Key = "key"
	dummyData := toJSON(th)

	cases := []struct {
		desc        string
		req         string
		id          string
		contentType string
		auth        string
		status      int
	}{
		{
			desc:        "update key for an existing thing",
			req:         data,
			id:          th.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update thing with conflicting key",
			req:         data,
			id:          th.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusConflict,
		},
		{
			desc:        "update key with empty JSON request",
			req:         "{}",
			id:          th.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update key of non-existent thing",
			req:         dummyData,
			id:          strconv.FormatUint(wrongID, 10),
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update thing with invalid id",
			req:         dummyData,
			id:          wrongValue,
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update thing with invalid user token",
			req:         data,
			id:          th.ID,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update thing with empty user token",
			req:         data,
			id:          th.ID,
			contentType: contentType,
			auth:        emptyValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update thing with invalid data format",
			req:         "{",
			id:          th.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing with empty request",
			req:         emptyValue,
			id:          th.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing without content type",
			req:         data,
			id:          th.ID,
			contentType: emptyValue,
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/things/%s/key", ts.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestViewThing(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	prID := prs[0].ID

	thing.GroupID = grID
	thing.ProfileID = prID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th := ths[0]

	data := thingRes{
		ID:        th.ID,
		GroupID:   th.GroupID,
		ProfileID: th.ProfileID,
		Name:      th.Name,
		Key:       th.Key,
		Metadata:  th.Metadata,
	}

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
		res    thingRes
	}{
		{
			desc:   "view existing thing",
			id:     th.ID,
			auth:   token,
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "view non-existent thing",
			id:     strconv.FormatUint(wrongID, 10),
			auth:   token,
			status: http.StatusNotFound,
			res:    thingRes{},
		},
		{
			desc:   "view thing by passing invalid token",
			id:     th.ID,
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			res:    thingRes{},
		},
		{
			desc:   "view thing by passing empty token",
			id:     th.ID,
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			res:    thingRes{},
		},
		{
			desc:   "view thing by passing invalid id",
			id:     wrongValue,
			auth:   token,
			status: http.StatusNotFound,
			res:    thingRes{},
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/things/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var body thingRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, body, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body))
	}
}

func TestViewMetadataByKey(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	idProvider := uuid.New()

	key, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	prID := prs[0].ID

	thing := things.Thing{
		GroupID:   grID,
		ProfileID: prID,
		Name:      "test-meta",
		Key:       key,
		Metadata:  metadata,
	}
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th := ths[0]

	metaRes := viewMetadataRes{
		Metadata: th.Metadata,
	}
	otherKey, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc   string
		auth   string
		status int
		res    viewMetadataRes
	}{
		{
			desc:   "view thing metadata",
			auth:   key,
			status: http.StatusOK,
			res:    metaRes,
		},
		{
			desc:   "view metadata from a non-existing thing",
			auth:   otherKey,
			status: http.StatusNotFound,
			res:    viewMetadataRes{},
		},
		{
			desc:   "view thing metadata with empty key",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			res:    viewMetadataRes{},
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/metadata", ts.URL),
			key:    tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var body viewMetadataRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, body, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body))
	}
}

func TestListThings(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	prID := prs[0].ID

	data := []thingRes{}
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%s%012d", prefix, i+1)
		thing1 := thing
		thing1.ID = id
		thing1.GroupID = grID
		thing1.ProfileID = prID

		ths, err := svc.CreateThings(context.Background(), token, thing1)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th := ths[0]

		data = append(data, thingRes{
			ID:        th.ID,
			GroupID:   th.GroupID,
			ProfileID: th.ProfileID,
			Name:      th.Name,
			Key:       th.Key,
			Metadata:  th.Metadata,
		})
	}

	thingURL := fmt.Sprintf("%s/things", ts.URL)
	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []thingRes
	}{
		{
			desc:   "get a list of things",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of things with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of things with negative offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, -1, 5),
			res:    nil,
		},
		{
			desc:   "get a list of things with negative limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 1, -5),
			res:    nil,
		},
		{
			desc:   "get a list of things with zero limit and offset 1",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 1, 0),
			res:    nil,
		},
		{
			desc:   "get a list of things without offset",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?limit=%d", thingURL, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things without limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d", thingURL, 1),
			res:    data[1:11],
		},
		{
			desc:   "get a list of things with redundant query params",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&value=something", thingURL, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 0, 110),
			res:    nil,
		},
		{
			desc:   "get a list of things with default URL",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s%s", thingURL, emptyValue),
			res:    data[0:10],
		},
		{
			desc:   "get a list of things with invalid number of params",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", thingURL, "?offset=4&limit=4&limit=5&offset=5"),
			res:    nil,
		},
		{
			desc:   "get a list of things with invalid offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", thingURL, "?offset=e&limit=5"),
			res:    nil,
		},
		{
			desc:   "get a list of things with invalid limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", thingURL, "?offset=5&limit=e"),
			res:    nil,
		},
		{
			desc:   "get a list of things filtering with invalid name",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&name=%s", thingURL, 0, 5, invalidName),
			res:    nil,
		},
		{
			desc:   "get a list of things sorted by name ascendant",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", thingURL, 0, 5, nameKey, ascKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things sorted by name descendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", thingURL, 0, 5, nameKey, descKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things sorted with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", thingURL, 0, 5, "wrong", descKey),
			res:    nil,
		},
		{
			desc:   "get a list of things sorted by name with invalid direction",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", thingURL, 0, 5, nameKey, "wrong"),
			res:    nil,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var data thingsPageRes
		json.NewDecoder(res.Body).Decode(&data)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, data.Things, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, data.Things))
	}
}

func TestSearchThings(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	str := searchThingReq
	validData := toJSON(str)

	str.Dir = "desc"
	str.Order = "name"
	descData := toJSON(str)

	str.Dir = "asc"
	ascData := toJSON(str)

	str.Order = "wrong"
	invalidOrderData := toJSON(str)

	str.Limit = 0
	zeroLimitData := toJSON(str)

	str = searchThingReq
	str.Dir = "wrong"
	invalidDirData := toJSON(str)

	str = searchThingReq
	str.Limit = 110
	limitMaxData := toJSON(str)

	str = searchThingReq
	str.Name = invalidName
	invalidNameData := toJSON(str)

	str.Name = invalidName
	invalidData := toJSON(str)

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	prID := prs[0].ID

	data := []thingRes{}
	for i := 0; i < 100; i++ {
		name := "name_" + fmt.Sprintf("%03d", i+1)
		id := fmt.Sprintf("%s%012d", prefix, i+1)

		th := things.Thing{ID: id, GroupID: grID, ProfileID: prID, Name: name, Metadata: map[string]interface{}{"test": name}}
		ths, err := svc.CreateThings(context.Background(), token, th)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		thing := ths[0]

		data = append(data, thingRes{
			ID:        thing.ID,
			GroupID:   thing.GroupID,
			ProfileID: thing.ProfileID,
			Name:      thing.Name,
			Key:       thing.Key,
			Metadata:  thing.Metadata,
		})
	}

	cases := []struct {
		desc   string
		auth   string
		status int
		req    string
		res    []thingRes
	}{
		{
			desc:   "search things",
			auth:   token,
			status: http.StatusOK,
			req:    validData,
			res:    data[0:5],
		},
		{
			desc:   "search things ordered by name ascendant",
			auth:   token,
			status: http.StatusOK,
			req:    ascData,
			res:    data[0:5],
		},
		{
			desc:   "search things ordered by name descendent",
			auth:   token,
			status: http.StatusOK,
			req:    descData,
			res:    data[0:5],
		},
		{
			desc:   "search things with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidOrderData,
			res:    nil,
		},
		{
			desc:   "search things with invalid dir",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidDirData,
			res:    nil,
		},
		{
			desc:   "search things with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search things with invalid data",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidData,
			res:    nil,
		},
		{
			desc:   "search things with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search things with zero limit",
			auth:   token,
			status: http.StatusBadRequest,
			req:    zeroLimitData,
			res:    nil,
		},
		{
			desc:   "search things without offset",
			auth:   token,
			status: http.StatusOK,
			req:    validData,
			res:    data[0:5],
		},
		{
			desc:   "search things with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			req:    limitMaxData,
			res:    nil,
		},
		{
			desc:   "search things with default URL",
			auth:   token,
			status: http.StatusOK,
			req:    validData,
			res:    data[0:5],
		},
		{
			desc:   "search things filtering with invalid name",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidNameData,
			res:    nil,
		},
		{
			desc:   "search things sorted by name ascendant",
			auth:   token,
			status: http.StatusOK,
			req:    validData,
			res:    data[0:5],
		},
		{
			desc:   "search things sorted by name descendent",
			auth:   token,
			status: http.StatusOK,
			req:    validData,
			res:    data[0:5],
		},
		{
			desc:   "search things sorted with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidOrderData,
			res:    nil,
		},
		{
			desc:   "search things sorted by name with invalid direction",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidDirData,
			res:    nil,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodPost,
			url:    fmt.Sprintf("%s/things/search", ts.URL),
			token:  tc.auth,
			body:   strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var data thingsPageRes
		json.NewDecoder(res.Body).Decode(&data)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, data.Things, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, data.Things))
	}
}

func TestListThingsByProfile(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profile.GroupID = gr.ID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	data := []thingRes{}

	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%s%012d", prefix, i+1)
		thing1 := thing
		thing1.ID = id
		thing1.GroupID = gr.ID
		thing1.ProfileID = pr.ID

		ths, err := svc.CreateThings(context.Background(), token, thing1)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th := ths[0]

		data = append(data, thingRes{
			ID:        th.ID,
			GroupID:   th.GroupID,
			ProfileID: th.ProfileID,
			Name:      th.Name,
			Key:       th.Key,
			Metadata:  th.Metadata,
		})
	}

	thingURL := fmt.Sprintf("%s/profiles", ts.URL)

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []thingRes
	}{
		{
			desc:   "get a list of things by profile",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, pr.ID, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by profile with no limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?limit=%d", thingURL, pr.ID, noLimit),
			res:    data,
		},
		{
			desc:   "get a list of things by profile with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, pr.ID, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of things by profile with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, pr.ID, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of things by profile with negative offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, pr.ID, -2, 5),
			res:    nil,
		},
		{
			desc:   "get a list of things by profile with negative limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, pr.ID, 1, -5),
			res:    nil,
		},
		{
			desc:   "get a list of things by profile with zero limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, pr.ID, 1, 0),
			res:    nil,
		},
		{
			desc:   "get a list of things by profile without offset",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?limit=%d", thingURL, pr.ID, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by profile without limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d", thingURL, pr.ID, 1),
			res:    data[1:11],
		},
		{
			desc:   "get a list of things by profile with redundant query params",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&value=something", thingURL, pr.ID, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by profile with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, pr.ID, 0, 110),
			res:    nil,
		},
		{
			desc:   "get a list of things by profile with default URL",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things", thingURL, pr.ID),
			res:    data[0:10],
		},
		{
			desc:   "get a list of things by profile with invalid number of params",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things%s", thingURL, pr.ID, "?offset=4&limit=4&limit=5&offset=5"),
			res:    nil,
		},
		{
			desc:   "get a list of things by profile with invalid offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things%s", thingURL, pr.ID, "?offset=e&limit=5"),
			res:    nil,
		},
		{
			desc:   "get a list of things by profile with invalid limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things%s", thingURL, pr.ID, "?offset=5&limit=e"),
			res:    nil,
		},
		{
			desc:   "get a list of things by profile sorted by name ascendant",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&order=%s&dir=%s", thingURL, pr.ID, 0, 5, nameKey, ascKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by profile sorted by name descendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&order=%s&dir=%s", thingURL, pr.ID, 0, 5, nameKey, descKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by profile sorted with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&order=%s&dir=%s", thingURL, pr.ID, 0, 5, "wrong", ascKey),
			res:    nil,
		},
		{
			desc:   "get a list of things by profile sorted by name with invalid direction",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&order=%s&dir=%s", thingURL, pr.ID, 0, 5, nameKey, "wrong"),
			res:    nil,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var data thingsPageRes
		json.NewDecoder(res.Body).Decode(&data)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, data.Things, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, data.Things))
	}
}

func TestListThingsByOrg(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), adminToken, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grs2, err := svc.CreateGroups(context.Background(), otherToken, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr, gr2 := grs[0], grs2[0]

	profile.GroupID = gr.ID
	prs, err := svc.CreateProfiles(context.Background(), adminToken, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	profile.GroupID = gr2.ID
	prs2, err := svc.CreateProfiles(context.Background(), otherToken, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr, pr2 := prs[0], prs2[0]

	data := []thingRes{}
	for i := 0; i < n; i++ {
		suffix := i + 1
		thing1 := thing
		thing1.Name = fmt.Sprintf("%s%012d", prefix, suffix)
		thing1.ID = fmt.Sprintf("%s%012d", prefix, suffix)
		thing1.GroupID = gr.ID
		thing1.ProfileID = pr.ID

		ths, err := svc.CreateThings(context.Background(), adminToken, thing1)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th := ths[0]

		data = append(data, thingRes{
			ID:        th.ID,
			GroupID:   th.GroupID,
			ProfileID: th.ProfileID,
			Name:      th.Name,
			Key:       th.Key,
			Metadata:  th.Metadata,
		})
	}

	data2 := []thingRes{}
	for i := 0; i < n; i++ {
		suffix := n + i + 1
		thing2 := thing
		thing2.Name = fmt.Sprintf("%s%012d", prefix, suffix)
		thing2.ID = fmt.Sprintf("%s%012d", prefix, suffix)
		thing2.GroupID = gr2.ID
		thing2.ProfileID = pr2.ID

		ths2, err := svc.CreateThings(context.Background(), otherToken, thing2)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th2 := ths2[0]

		data2 = append(data2, thingRes{
			ID:        th2.ID,
			GroupID:   th2.GroupID,
			ProfileID: th2.ProfileID,
			Name:      th2.Name,
			Key:       th2.Key,
			Metadata:  th2.Metadata,
		})
	}

	thingURL := fmt.Sprintf("%s/orgs", ts.URL)

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []thingRes
	}{
		{
			desc:   "get a list of things by org",
			auth:   otherToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, orgID, n, 5),
			res:    data2[0:5],
		},
		{
			desc:   "get a list of things by org as org owner",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, orgID, 0, 100),
			res:    data[0:100],
		},
		{
			desc:   "get a list of things by org with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, orgID, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of things by org with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, orgID, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of things by org with negative offset",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, orgID, -2, 5),
			res:    nil,
		},
		{
			desc:   "get a list of things by org with negative limit",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, orgID, 1, -5),
			res:    nil,
		},
		{
			desc:   "get a list of things by org with no limit",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, orgID, 1, 0),
			res:    nil,
		},
		{
			desc:   "get a list of things by org without offset",
			auth:   adminToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?limit=%d", thingURL, orgID, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by org without limit",
			auth:   adminToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d", thingURL, orgID, 1),
			res:    data[1:11],
		},
		{
			desc:   "get a list of things by org with redundant query params",
			auth:   adminToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&value=something", thingURL, orgID, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by org with limit greater than max",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, orgID, 0, 110),
			res:    nil,
		},
		{
			desc:   "get a list of things by org with default URL",
			auth:   adminToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things", thingURL, orgID),
			res:    data[0:10],
		},
		{
			desc:   "get a list of things by org with invalid number of params",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things%s", thingURL, orgID, "?offset=4&limit=4&limit=5&offset=5"),
			res:    nil,
		},
		{
			desc:   "get a list of things by org with invalid offset",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things%s", thingURL, orgID, "?offset=e&limit=5"),
			res:    nil,
		},
		{
			desc:   "get a list of things by org with invalid limit",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things%s", thingURL, orgID, "?offset=5&limit=e"),
			res:    nil,
		},
		{
			desc:   "get a list of things by org sorted by name ascendant",
			auth:   adminToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&order=%s&dir=%s", thingURL, orgID, 0, 5, nameKey, ascKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by org sorted by name descendent",
			auth:   adminToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&order=%s&dir=%s", thingURL, orgID, 0, 5, nameKey, descKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by org sorted with invalid order",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&order=%s&dir=%s", thingURL, orgID, 0, 5, "wrong", ascKey),
			res:    nil,
		},
		{
			desc:   "get a list of things by org sorted by name with invalid direction",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&order=%s&dir=%s", thingURL, orgID, 0, 5, nameKey, "wrong"),
			res:    nil,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var data thingsPageRes
		json.NewDecoder(res.Body).Decode(&data)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, data.Things, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, data.Things))
	}
}

func TestRemoveThing(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	thing.GroupID = grID
	thing.ProfileID = prID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
	}{
		{
			desc:   "remove thing with invalid token",
			id:     th.ID,
			auth:   wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove thing with empty token",
			id:     th.ID,
			auth:   emptyValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove existing thing",
			id:     th.ID,
			auth:   token,
			status: http.StatusNoContent,
		},
		{
			desc:   "remove non-existent thing",
			id:     strconv.FormatUint(wrongID, 10),
			auth:   token,
			status: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/things/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestRemoveThings(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	thing.GroupID = grID
	thing.ProfileID = prID
	ths, err := svc.CreateThings(context.Background(), token, thing, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	var thingIDs []string
	for _, th := range ths {
		thingIDs = append(thingIDs, th.ID)
	}

	cases := []struct {
		desc        string
		data        []string
		auth        string
		contentType string
		status      int
	}{
		{
			desc:        "remove things with invalid token",
			data:        thingIDs,
			auth:        wrongValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove things with empty token",
			data:        thingIDs,
			auth:        emptyValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove things with invalid content type",
			data:        thingIDs,
			auth:        token,
			contentType: wrongValue,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "remove existing things",
			data:        thingIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNoContent,
		},
		{
			desc:        "remove non-existent things",
			data:        []string{wrongValue},
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "remove things with empty thing ids",
			data:        []string{emptyValue},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove profiles without profile ids",
			data:        []string{},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		data := struct {
			ThingIDs []string `json:"thing_ids"`
		}{
			tc.data,
		}

		body := toJSON(data)

		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/things", ts.URL),
			token:       tc.auth,
			contentType: tc.contentType,
			body:        strings.NewReader(body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestCreateProfiles(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	data := `[{"name": "1"}, {"name": "2"}]`
	invalidData := fmt.Sprintf(`[{"name": "%s"}]`, invalidName)

	cases := []struct {
		desc        string
		data        string
		contentType string
		auth        string
		status      int
		response    string
	}{
		{
			desc:        "create valid profiles",
			data:        data,
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			response:    emptyValue,
		},
		{
			desc:        "create profile with empty request",
			data:        emptyValue,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create profiles with empty JSON",
			data:        "[]",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create profile with invalid auth token",
			data:        data,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
			response:    emptyValue,
		},
		{
			desc:        "create profile with empty auth token",
			data:        data,
			contentType: contentType,
			auth:        emptyValue,
			status:      http.StatusUnauthorized,
			response:    emptyValue,
		},
		{
			desc:     "create profile with invalid request format",
			data:     "}",
			auth:     token,
			status:   http.StatusUnsupportedMediaType,
			response: emptyValue,
		},
		{
			desc:        "create profile without content type",
			data:        data,
			contentType: emptyValue,
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
			response:    emptyValue,
		},
		{
			desc:        "create profile with invalid name",
			data:        invalidData,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/groups/%s/profiles", ts.URL, gr.ID),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.data),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		location := res.Header.Get("Location")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.response, location, fmt.Sprintf("%s: expected response %s got %s", tc.desc, tc.response, location))
	}
}

func TestUpdateProfile(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profile.GroupID = gr.ID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	pr := prs[0]

	c := profile
	c.Name = "updated_profile"
	updateData := toJSON(c)

	c.Name = invalidName
	invalidData := toJSON(c)

	cases := []struct {
		desc        string
		req         string
		id          string
		contentType string
		auth        string
		status      int
	}{
		{
			desc:        "update existing profile",
			req:         updateData,
			id:          pr.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update non-existing profile",
			req:         updateData,
			id:          strconv.FormatUint(wrongID, 10),
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update profile with invalid id",
			req:         updateData,
			id:          wrongValue,
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update profile with invalid token",
			req:         updateData,
			id:          pr.ID,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update profile with empty token",
			req:         updateData,
			id:          pr.ID,
			contentType: contentType,
			auth:        emptyValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update profile with invalid data format",
			req:         "}",
			id:          pr.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update profile with empty JSON object",
			req:         "{}",
			id:          pr.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update profile with empty request",
			req:         emptyValue,
			id:          pr.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update profile with missing content type",
			req:         updateData,
			id:          pr.ID,
			contentType: emptyValue,
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "update profile with invalid name",
			req:         invalidData,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/profiles/%s", ts.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestViewProfile(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profile.GroupID = gr.ID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	pr := prs[0]

	data := profileRes{
		ID:       pr.ID,
		Name:     pr.Name,
		GroupID:  pr.GroupID,
		Metadata: pr.Metadata,
	}

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
		res    profileRes
	}{
		{
			desc:   "view existing profile",
			id:     pr.ID,
			auth:   token,
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "view non-existent profile",
			id:     strconv.FormatUint(wrongID, 10),
			auth:   token,
			status: http.StatusNotFound,
			res:    profileRes{},
		},
		{
			desc:   "view profile with invalid token",
			id:     pr.ID,
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			res:    profileRes{},
		},
		{
			desc:   "view profile with empty token",
			id:     pr.ID,
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			res:    profileRes{},
		},
		{
			desc:   "view profile with invalid id",
			id:     wrongValue,
			auth:   token,
			status: http.StatusNotFound,
			res:    profileRes{},
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/profiles/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var body profileRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, body, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body))
	}
}

func TestListProfiles(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profiles := []profileRes{}
	for i := 0; i < n; i++ {
		name := "name_" + fmt.Sprintf("%03d", i+1)
		id := fmt.Sprintf("%s%012d", prefix, i+1)
		pr := things.Profile{ID: id, GroupID: gr.ID, Name: name, Metadata: metadata}

		prs, err := svc.CreateProfiles(context.Background(), token, pr)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		profile := prs[0]

		profiles = append(profiles, profileRes{
			ID:       profile.ID,
			Name:     profile.Name,
			Metadata: profile.Metadata,
			GroupID:  profile.GroupID,
			Config:   profile.Config,
		})
	}
	profileURL := fmt.Sprintf("%s/profiles", ts.URL)

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []profileRes
	}{
		{
			desc:   "get a list of profiles",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", profileURL, 0, 6),
			res:    profiles[0:6],
		},
		{
			desc:   "get a list of all profiles without limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?limit=%d", profileURL, noLimit),
			res:    []profileRes{},
		},
		{
			desc:   "get a list of profiles with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", profileURL, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of profiles with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", profileURL, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of profiles with negative offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", profileURL, -1, 5),
			res:    nil,
		},
		{
			desc:   "get a list of profiles with negative limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", profileURL, 5, -2),
			res:    nil,
		},
		{
			desc:   "get a list of profiles with zero limit and offset 1",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", profileURL, 1, 0),
			res:    nil,
		},
		{
			desc:   "get a list of profiles without offset",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?limit=%d", profileURL, 5),
			res:    profiles[0:5],
		},
		{
			desc:   "get a list of profiles without limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d", profileURL, 1),
			res:    profiles[1:11],
		},
		{
			desc:   "get a list of profiles with redundant query params",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&value=something", profileURL, 0, 5),
			res:    profiles[0:5],
		},
		{
			desc:   "get a list of profiles with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", profileURL, 0, 110),
			res:    nil,
		},
		{
			desc:   "get a list of profiles with default URL",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s%s", profileURL, emptyValue),
			res:    profiles[0:10],
		},
		{
			desc:   "get a list of profiles with invalid number of params",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", profileURL, "?offset=4&limit=4&limit=5&offset=5"),
			res:    nil,
		},
		{
			desc:   "get a list of profiles with invalid offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", profileURL, "?offset=e&limit=5"),
			res:    nil,
		},
		{
			desc:   "get a list of profiles with invalid limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", profileURL, "?offset=5&limit=e"),
			res:    nil,
		},
		{
			desc:   "get a list of profiles with invalid name",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&name=%s", profileURL, 0, 10, invalidName),
			res:    nil,
		},
		{
			desc:   "get a list of profiles sorted by name ascendant",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", profileURL, 0, 6, nameKey, ascKey),
			res:    profiles[0:6],
		},
		{
			desc:   "get a list of profiles sorted by name descendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", profileURL, 0, 6, nameKey, descKey),
			res:    profiles[0:6],
		},
		{
			desc:   "get a list of profiles sorted with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", profileURL, 0, 6, "wrong", ascKey),
			res:    nil,
		},
		{
			desc:   "get a list of profiles sorted by name with invalid direction",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", profileURL, 0, 6, nameKey, "wrong"),
			res:    nil,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var body profilesPageRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, body.Profiles, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body.Profiles))
	}
}

func TestListProfilesByOrg(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), adminToken, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grs2, err := svc.CreateGroups(context.Background(), otherToken, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr, gr2 := grs[0], grs2[0]

	data := []profileRes{}
	for i := 0; i < n; i++ {
		suffix := i + 1
		name := "name_" + fmt.Sprintf("%03d", suffix)
		id := fmt.Sprintf("%s%012d", prefix, suffix)
		pr := things.Profile{ID: id, GroupID: gr.ID, Name: name, Metadata: metadata}

		prs, err := svc.CreateProfiles(context.Background(), adminToken, pr)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		profile := prs[0]

		data = append(data, profileRes{
			ID:       profile.ID,
			Name:     profile.Name,
			Metadata: profile.Metadata,
			GroupID:  profile.GroupID,
			Config:   profile.Config,
		})
	}

	data2 := []profileRes{}
	for i := 0; i < n; i++ {
		suffix := n + i + 1
		name := "name_" + fmt.Sprintf("%03d", suffix)
		id := fmt.Sprintf("%s%012d", prefix, suffix)
		pr := things.Profile{ID: id, GroupID: gr2.ID, Name: name, Metadata: metadata}

		prs, err := svc.CreateProfiles(context.Background(), otherToken, pr)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		profile := prs[0]

		data2 = append(data2, profileRes{
			ID:       profile.ID,
			Name:     profile.Name,
			Metadata: profile.Metadata,
			GroupID:  profile.GroupID,
			Config:   profile.Config,
		})
	}
	profileURL := fmt.Sprintf("%s/orgs", ts.URL)

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []profileRes
	}{
		{
			desc:   "get a list of profiles by org the user belongs to",
			auth:   otherToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d", profileURL, orgID, n, 5),
			res:    data2[0:5],
		},
		{
			desc:   "get a list of profiles by org as org owner",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d", profileURL, orgID, 0, 100),
			res:    data[0:100],
		},
		{
			desc:   "get a list of profiles by org with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d", profileURL, orgID, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of profiles by org with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d", profileURL, orgID, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of profiles by org with negative offset",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d", profileURL, orgID, -2, 5),
			res:    nil,
		},
		{
			desc:   "get a list of profiles by org with negative limit",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d", profileURL, orgID, 1, -5),
			res:    nil,
		},
		{
			desc:   "get a list of profiles by org with no limit",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d", profileURL, orgID, 1, 0),
			res:    nil,
		},
		{
			desc:   "get a list of profiles by org without offset",
			auth:   adminToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/profiles?limit=%d", profileURL, orgID, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of profiles by org without limit",
			auth:   adminToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d", profileURL, orgID, 1),
			res:    data[1:11],
		},
		{
			desc:   "get a list of profiles by org with redundant query params",
			auth:   adminToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d&value=something", profileURL, orgID, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of profiles by org with limit greater than max",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d", profileURL, orgID, 0, 110),
			res:    nil,
		},
		{
			desc:   "get a list of things by org with default URL",
			auth:   adminToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/profiles", profileURL, orgID),
			res:    data[0:10],
		},
		{
			desc:   "get a list of profiles by org with invalid number of params",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/profiles%s", profileURL, orgID, "?offset=4&limit=4&limit=5&offset=5"),
			res:    nil,
		},
		{
			desc:   "get a list of profiles by org with invalid offset",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/profiles%s", profileURL, orgID, "?offset=e&limit=5"),
			res:    nil,
		},
		{
			desc:   "get a list of profiles by org with invalid limit",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/profiles%s", profileURL, orgID, "?offset=5&limit=e"),
			res:    nil,
		},
		{
			desc:   "get a list of profiles by org sorted by name ascendant",
			auth:   adminToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d&order=%s&dir=%s", profileURL, orgID, 0, 5, nameKey, ascKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of profiles by org sorted by name descendent",
			auth:   adminToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d&order=%s&dir=%s", profileURL, orgID, 0, 5, nameKey, descKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of profiles by org sorted with invalid order",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d&order=%s&dir=%s", profileURL, orgID, 0, 5, "wrong", ascKey),
			res:    nil,
		},
		{
			desc:   "get a list of profiles by org sorted by name with invalid direction",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d&order=%s&dir=%s", profileURL, orgID, 0, 5, nameKey, "wrong"),
			res:    nil,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var body profilesPageRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, body.Profiles, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body.Profiles))
	}
}

func TestViewProfileByThing(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profile.GroupID = gr.ID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	thing.GroupID = gr.ID
	thing.ProfileID = pr.ID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th := ths[0]

	prRes := profileRes{
		ID:       pr.ID,
		Name:     pr.Name,
		GroupID:  pr.GroupID,
		Metadata: pr.Metadata,
	}

	profileURL := fmt.Sprintf("%s/things", ts.URL)

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    profileRes
	}{
		{
			desc:   "view profile by thing",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/profiles", profileURL, th.ID),
			res:    prRes,
		},
		{
			desc:   "view profile by thing with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/profiles", profileURL, th.ID),
			res:    profileRes{},
		},
		{
			desc:   "view profile by thing with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/profiles", profileURL, th.ID),
			res:    profileRes{},
		},
		{
			desc:   "view profile by thing without thing id",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/profiles", profileURL, emptyValue),
			res:    profileRes{},
		},
		{
			desc:   "view profile by thing with wrong thing id",
			auth:   token,
			status: http.StatusNotFound,
			url:    fmt.Sprintf("%s/%s/profiles", profileURL, wrongValue),
			res:    profileRes{},
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var body profileRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, body, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body))
	}
}

func TestRemoveProfile(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID, prID1 := prs[0].ID, prs[1].ID

	thing.GroupID = grID
	thing.ProfileID = prID1
	_, err = svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
	}{
		{
			desc:   "remove profile with invalid token",
			id:     prID,
			auth:   wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove profile with empty token",
			id:     prID,
			auth:   emptyValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove profile with invalid token",
			id:     prID,
			auth:   wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove existing profile",
			id:     prID,
			auth:   token,
			status: http.StatusNoContent,
		},
		{
			desc:   "remove removed profile",
			id:     prID,
			auth:   token,
			status: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/profiles/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestRemoveProfiles(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	profile1.GroupID = grID
	cList := []things.Profile{profile, profile, profile1}
	prs, err := svc.CreateProfiles(context.Background(), token, cList...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	thing.GroupID = grID
	thing.ProfileID = prs[2].ID
	_, err = svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	profileIDs := []string{prs[0].ID, prs[1].ID}

	cases := []struct {
		desc        string
		data        []string
		auth        string
		contentType string
		status      int
	}{
		{
			desc:        "remove profiles with invalid token",
			data:        profileIDs,
			auth:        wrongValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove profiles with empty token",
			data:        profileIDs,
			auth:        emptyValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove profiles with invalid content type",
			data:        profileIDs,
			auth:        token,
			contentType: wrongValue,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "remove existing profiles",
			data:        profileIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNoContent,
		},
		{
			desc:        "remove non-existent profiles",
			data:        profileIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "remove profiles with empty profile ids",
			data:        []string{emptyValue},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove profiles without profile ids",
			data:        []string{},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		data := struct {
			ProfileIDs []string `json:"profile_ids"`
		}{
			tc.data,
		}

		body := toJSON(data)

		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/profiles", ts.URL),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestRemoveGroups(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	var groups []things.Group
	var groupIDs []string
	for i := uint64(0); i < 10; i++ {
		num := strconv.FormatUint(i, 10)
		group := things.Group{
			OrgID:       orgID,
			Name:        "test-group-" + num,
			Description: "test group desc",
		}
		grs, err := svc.CreateGroups(context.Background(), token, group)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		gr := grs[0]

		groups = append(groups, gr)
		groupIDs = append(groupIDs, gr.ID)
	}

	cases := []struct {
		desc        string
		data        []string
		auth        string
		contentType string
		status      int
	}{
		{
			desc:        "remove existing groups",
			data:        groupIDs[:5],
			auth:        token,
			contentType: contentType,
			status:      http.StatusNoContent,
		},
		{
			desc:        "remove non-existent groups",
			data:        []string{wrongValue},
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "remove groups with invalid token",
			data:        groupIDs[len(groupIDs)-5:],
			auth:        wrongValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove groups without group ids",
			data:        []string{},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove groups with empty group ids",
			data:        []string{emptyValue},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove groups with empty token",
			data:        groupIDs,
			auth:        emptyValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove groups with invalid content type",
			data:        groupIDs,
			auth:        token,
			contentType: wrongValue,
			status:      http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		data := struct {
			GroupIDs []string `json:"group_ids"`
		}{
			tc.data,
		}

		body := toJSON(data)

		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/groups", ts.URL),
			token:       tc.auth,
			contentType: tc.contentType,
			body:        strings.NewReader(body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestBackup(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	var groups []things.Group
	for i := uint64(0); i < 10; i++ {
		num := strconv.FormatUint(i, 10)
		group := things.Group{
			OrgID:       orgID,
			Name:        "test-group-" + num,
			Description: "test group desc",
		}
		grs, err := svc.CreateGroups(context.Background(), token, group)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		gr := grs[0]

		groups = append(groups, gr)
	}
	gr := groups[0]

	profiles := []things.Profile{}
	for i := 0; i < 10; i++ {
		name := "name_" + fmt.Sprintf("%03d", i+1)
		prs, err := svc.CreateProfiles(context.Background(), token,
			things.Profile{
				Name:     name,
				GroupID:  gr.ID,
				Metadata: metadata,
			})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		pr := prs[0]

		profiles = append(profiles, pr)
	}
	pr := profiles[0]

	ths := []things.Thing{}
	for i := 0; i < 10; i++ {
		name := "name_" + fmt.Sprintf("%03d", i+1)
		things, err := svc.CreateThings(context.Background(), token,
			things.Thing{
				Name:      name,
				GroupID:   gr.ID,
				ProfileID: pr.ID,
				Metadata:  metadata,
			})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th := things[0]

		ths = append(ths, th)
	}

	var thingsRes []backupThingRes
	for _, th := range ths {
		thingsRes = append(thingsRes, backupThingRes{
			ID:        th.ID,
			GroupID:   th.GroupID,
			ProfileID: th.ProfileID,
			Name:      th.Name,
			Key:       th.Key,
			Metadata:  th.Metadata,
		})
	}

	var profilesRes []backupProfileRes
	for _, pr := range profiles {
		profilesRes = append(profilesRes, backupProfileRes{
			ID:       pr.ID,
			GroupID:  pr.GroupID,
			Name:     pr.Name,
			Metadata: pr.Metadata,
		})
	}

	var groupsRes []viewGroupRes
	for _, gr := range groups {
		groupsRes = append(groupsRes, viewGroupRes{
			ID:          gr.ID,
			OrgID:       gr.OrgID,
			Name:        gr.Name,
			Description: gr.Description,
			Metadata:    gr.Metadata,
		})
	}

	backup := backupRes{
		Groups:   groupsRes,
		Things:   thingsRes,
		Profiles: profilesRes,
	}

	backupURL := fmt.Sprintf("%s/backup", ts.URL)

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    backupRes
	}{
		{
			desc:   "backup all things, profiles and groups",
			auth:   adminToken,
			status: http.StatusOK,
			url:    backupURL,
			res:    backup,
		},
		{
			desc:   "backup with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    backupURL,
			res:    backupRes{},
		},
		{
			desc:   "backup with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			url:    backupURL,
			res:    backupRes{},
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var body backupRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res.Profiles, body.Profiles, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res.Profiles, body.Profiles))
		assert.ElementsMatch(t, tc.res.Things, body.Things, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res.Things, body.Things))
		assert.ElementsMatch(t, tc.res.Groups, body.Groups, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res.Groups, body.Groups))
	}
}

func TestRestore(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	idProvider := uuid.New()

	thId, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thKey, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	testThing := things.Thing{
		ID:       thId,
		Name:     nameKey,
		Key:      thKey,
		Metadata: metadata,
	}

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

	profiles := []things.Profile{}
	for i := 0; i < n; i++ {
		prID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		name := "name_" + fmt.Sprintf("%03d", i+1)
		profiles = append(profiles, things.Profile{
			ID:       prID,
			GroupID:  emptyValue,
			Name:     name,
			Metadata: metadata,
		})
	}

	thr := []restoreThingReq{
		{
			ID:       testThing.ID,
			Name:     testThing.Name,
			Key:      testThing.Key,
			Metadata: testThing.Metadata,
		},
	}

	var prr []restoreProfileReq
	for _, pr := range profiles {
		prr = append(prr, restoreProfileReq{
			ID:       pr.ID,
			Name:     pr.Name,
			Metadata: pr.Metadata,
		})
	}

	var gr []restoreGroupReq
	for _, group := range groups {
		gr = append(gr, restoreGroupReq{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Metadata:    group.Metadata,
		})
	}

	resReq := restoreReq{
		Things:   thr,
		Profiles: prr,
		Groups:   gr,
	}

	data := toJSON(resReq)
	invalidData := toJSON(restoreReq{})
	restoreURL := fmt.Sprintf("%s/restore", ts.URL)

	cases := []struct {
		desc        string
		auth        string
		status      int
		url         string
		req         string
		contentType string
	}{
		{
			desc:        "restore all things, profiles and groups",
			auth:        adminToken,
			status:      http.StatusCreated,
			url:         restoreURL,
			req:         data,
			contentType: contentType,
		},
		{
			desc:        "restore with invalid token",
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
			url:         restoreURL,
			req:         data,
			contentType: contentType,
		},
		{
			desc:        "restore with empty token",
			auth:        emptyValue,
			status:      http.StatusUnauthorized,
			url:         restoreURL,
			req:         data,
			contentType: contentType,
		},
		{
			desc:        "restore with invalid request",
			auth:        token,
			status:      http.StatusBadRequest,
			url:         restoreURL,
			req:         invalidData,
			contentType: contentType,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         tc.url,
			token:       tc.auth,
			contentType: tc.contentType,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

	}
}

func TestIdentify(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	thing.GroupID = grID
	thing.ProfileID = prID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("failed to create thing: %s", err))
	th := ths[0]

	ir := identifyReq{Token: th.Key}
	data := toJSON(ir)

	nonexistentData := toJSON(identifyReq{Token: wrongValue})

	cases := map[string]struct {
		contentType string
		req         string
		status      int
	}{
		"identify existing thing": {
			contentType: contentType,
			req:         data,
			status:      http.StatusOK,
		},
		"identify non-existent thing": {
			contentType: contentType,
			req:         nonexistentData,
			status:      http.StatusNotFound,
		},
		"identify with missing content type": {
			contentType: wrongValue,
			req:         data,
			status:      http.StatusUnsupportedMediaType,
		},
		"identify with empty JSON request": {
			contentType: contentType,
			req:         "{}",
			status:      http.StatusUnauthorized,
		},
		"identify with invalid JSON request": {
			contentType: contentType,
			req:         emptyValue,
			status:      http.StatusBadRequest,
		},
	}

	for desc, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/identify", ts.URL),
			contentType: tc.contentType,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", desc, tc.status, res.StatusCode))
	}
}

type identifyReq struct {
	Token string `json:"token"`
}

type viewMetadataRes struct {
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type thingRes struct {
	ID        string                 `json:"id"`
	GroupID   string                 `json:"group_id,omitempty"`
	ProfileID string                 `json:"profile_id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Key       string                 `json:"key"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type profileRes struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name,omitempty"`
	GroupID  string                 `json:"group_id,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
}

type thingsPageRes struct {
	Things []thingRes `json:"things"`
	Total  uint64     `json:"total"`
	Offset uint64     `json:"offset"`
	Limit  uint64     `json:"limit"`
}

type profilesPageRes struct {
	Profiles []profileRes `json:"profiles"`
	Total    uint64       `json:"total"`
	Offset   uint64       `json:"offset"`
	Limit    uint64       `json:"limit"`
}

type backupThingRes struct {
	ID        string                 `json:"id"`
	GroupID   string                 `json:"group_id"`
	ProfileID string                 `json:"profile_id"`
	Name      string                 `json:"name,omitempty"`
	Key       string                 `json:"key"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type backupProfileRes struct {
	ID       string                 `json:"id"`
	GroupID  string                 `json:"group_id"`
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type viewGroupRes struct {
	ID          string                 `json:"id"`
	OrgID       string                 `json:"org_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type backupRes struct {
	Things   []backupThingRes   `json:"things"`
	Profiles []backupProfileRes `json:"profiles"`
	Groups   []viewGroupRes     `json:"groups"`
}

type restoreThingReq struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Key      string                 `json:"key"`
	Metadata map[string]interface{} `json:"metadata"`
}

type restoreProfileReq struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata"`
}

type restoreGroupReq struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type restoreReq struct {
	Things   []restoreThingReq   `json:"things"`
	Profiles []restoreProfileReq `json:"profiles"`
	Groups   []restoreGroupReq   `json:"groups"`
}
