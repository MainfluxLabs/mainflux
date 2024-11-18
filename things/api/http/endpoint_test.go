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
	adminToken     = adminEmail
	otherToken     = otherUserEmail
	wrongValue     = "wrong_value"
	emptyValue     = ""
	invalidValue   = "invalid"
	wrongID        = 0
	password       = "password"
	maxNameSize    = 1024
	nameKey        = "name"
	ascKey         = "asc"
	descKey        = "desc"
	orgID          = "374106f7-030e-4881-8ab0-151195c29f92"
	prefix         = "fe6b4e92-cc98-425e-b0aa-"
	n              = 101
	noLimit        = -1
)

var (
	thing = things.Thing{
		Name:     "test_app",
		Metadata: map[string]interface{}{"test": "data"},
	}
	thing1 = things.Thing{
		Name:     "test_app1",
		Metadata: map[string]interface{}{"test": "data"},
	}
	profile = things.Profile{
		Name:     "test",
		Metadata: map[string]interface{}{"test": "data"},
	}
	profile1 = things.Profile{
		Name:     "test1",
		Metadata: map[string]interface{}{"test": "data"},
	}
	invalidName    = strings.Repeat("m", maxNameSize+1)
	searchThingReq = things.PageMetadata{
		Limit:  5,
		Offset: 0,
	}
	user      = users.User{ID: "574106f7-030e-4881-8ab0-151195c29f94", Email: email, Password: password, Role: auth.Editor}
	otherUser = users.User{ID: "ecf9e48b-ba3b-41c4-82a9-72e063b17868", Email: otherUserEmail, Password: password, Role: auth.Owner}
	admin     = users.User{ID: "2e248e36-2d26-46ea-97b0-1e38d674cbe4", Email: adminEmail, Password: password, Role: auth.RootSub}
	usersList = []users.User{admin, user, otherUser}
	group     = things.Group{Name: "test-group", Description: "test-group-desc", OrgID: orgID}
)

type testRequest struct {
	client      *http.Client
	method      string
	url         string
	contentType string
	token       string
	body        io.Reader
}

func (tr testRequest) make() (*http.Response, error) {
	req, err := http.NewRequest(tr.method, tr.url, tr.body)
	if err != nil {
		return nil, err
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
	auth := mocks.NewAuthService(admin.ID, usersList)
	conns := make(chan thmocks.Connection)
	thingsRepo := thmocks.NewThingRepository(conns)
	profilesRepo := thmocks.NewProfileRepository(thingsRepo, conns)
	groupsRepo := thmocks.NewGroupRepository()
	rolesRepo := thmocks.NewRolesRepository()
	profileCache := thmocks.NewProfileCache()
	thingCache := thmocks.NewThingCache()
	idProvider := uuid.NewMock()

	return things.New(auth, nil, thingsRepo, profilesRepo, groupsRepo, rolesRepo, profileCache, thingCache, idProvider)
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

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	data := `[{"name": "1", "key": "1"}, {"name": "2", "key": "2"}]`
	invalidData := fmt.Sprintf(`[{"name": "%s", "key": "10"}]`, invalidName)

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
			data:        invalidData,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
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
			url:         fmt.Sprintf("%s/groups/%s/things", ts.URL, gr.ID),
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

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	thing.GroupID = gr.ID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th1 := ths[0]

	data := toJSON(th1)

	th2 := thing
	th2.Name = invalidName
	invalidData := toJSON(th2)

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
			id:          th1.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update thing with empty JSON request",
			req:         "{}",
			id:          th1.ID,
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
			id:          invalidValue,
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update thing with invalid user token",
			req:         data,
			id:          th1.ID,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update thing with empty user token",
			req:         data,
			id:          th1.ID,
			contentType: contentType,
			auth:        emptyValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update thing with invalid data format",
			req:         "{",
			id:          th1.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing with empty request",
			req:         emptyValue,
			id:          th1.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing without content type",
			req:         data,
			id:          th1.ID,
			contentType: emptyValue,
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "update thing with invalid name",
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
	gr := grs[0]

	th := thing
	th.Key = "key"
	thing.GroupID = gr.ID
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
			id:          invalidValue,
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
	gr := grs[0]

	thing.GroupID = gr.ID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th := ths[0]

	data := thingRes{
		ID:       th.ID,
		Name:     th.Name,
		Key:      th.Key,
		Metadata: th.Metadata,
		GroupID:  th.GroupID,
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
			id:     invalidValue,
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

func TestListThings(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	data := []thingRes{}
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%s%012d", prefix, i+1)
		thing1 := thing
		thing1.ID = id
		thing1.GroupID = gr.ID

		ths, err := svc.CreateThings(context.Background(), token, thing1)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th := ths[0]

		data = append(data, thingRes{
			ID:       th.ID,
			Name:     th.Name,
			Key:      th.Key,
			Metadata: th.Metadata,
			GroupID:  th.GroupID,
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
	gr := grs[0]

	data := []thingRes{}
	for i := 0; i < 100; i++ {
		name := "name_" + fmt.Sprintf("%03d", i+1)
		id := fmt.Sprintf("%s%012d", prefix, i+1)

		th := things.Thing{ID: id, GroupID: gr.ID, Name: name, Metadata: map[string]interface{}{"test": name}}
		ths, err := svc.CreateThings(context.Background(), token, th)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		thing := ths[0]

		data = append(data, thingRes{
			ID:       thing.ID,
			GroupID:  thing.GroupID,
			Name:     thing.Name,
			Key:      thing.Key,
			Metadata: thing.Metadata,
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
	chs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	ch := chs[0]

	data := []thingRes{}
	thIDs := []string{}
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%s%012d", prefix, i+1)
		thing1 := thing
		thing1.ID = id
		thing1.GroupID = gr.ID

		ths, err := svc.CreateThings(context.Background(), token, thing1)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th := ths[0]

		data = append(data, thingRes{
			ID:       th.ID,
			Name:     th.Name,
			Key:      th.Key,
			GroupID:  th.GroupID,
			Metadata: th.Metadata,
		})
		thIDs = append(thIDs, th.ID)
	}

	err = svc.Connect(context.Background(), token, ch.ID, thIDs)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	thingURL := fmt.Sprintf("%s/profiles", ts.URL)

	// Wait for things and profiles to connect.
	time.Sleep(time.Second)

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
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by profile with no limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?limit=%d", thingURL, ch.ID, noLimit),
			res:    data,
		},
		{
			desc:   "get a list of things by profile with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of things by profile with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of things by profile with negative offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, -2, 5),
			res:    nil,
		},
		{
			desc:   "get a list of things by profile with negative limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, 1, -5),
			res:    nil,
		},
		{
			desc:   "get a list of things by profile with zero limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, 1, 0),
			res:    nil,
		},
		{
			desc:   "get a list of things by profile without offset",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?limit=%d", thingURL, ch.ID, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by profile without limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d", thingURL, ch.ID, 1),
			res:    data[1:11],
		},
		{
			desc:   "get a list of things by profile with redundant query params",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&value=something", thingURL, ch.ID, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by profile with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, 0, 110),
			res:    nil,
		},
		{
			desc:   "get a list of things by profile with default URL",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things", thingURL, ch.ID),
			res:    data[0:10],
		},
		{
			desc:   "get a list of things by profile with invalid number of params",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things%s", thingURL, ch.ID, "?offset=4&limit=4&limit=5&offset=5"),
			res:    nil,
		},
		{
			desc:   "get a list of things by profile with invalid offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things%s", thingURL, ch.ID, "?offset=e&limit=5"),
			res:    nil,
		},
		{
			desc:   "get a list of things by profile with invalid limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things%s", thingURL, ch.ID, "?offset=5&limit=e"),
			res:    nil,
		},
		{
			desc:   "get a list of things by profile sorted by name ascendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&order=%s&dir=%s", thingURL, ch.ID, 0, 5, nameKey, ascKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by profile sorted by name descendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&order=%s&dir=%s", thingURL, ch.ID, 0, 5, nameKey, descKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by profile sorted with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&order=%s&dir=%s", thingURL, ch.ID, 0, 5, "wrong", ascKey),
			res:    nil,
		},
		{
			desc:   "get a list of things by profile sorted by name with invalid direction",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&order=%s&dir=%s", thingURL, ch.ID, 0, 5, nameKey, "wrong"),
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
	gr := grs[0]

	thing.GroupID = gr.ID
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
	gr := grs[0]

	thing.GroupID = gr.ID
	thing1.GroupID = gr.ID
	tList := []things.Thing{thing, thing1}
	ths, err := svc.CreateThings(context.Background(), token, tList...)
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
	chs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]

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
			id:          ch.ID,
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
			id:          invalidValue,
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update profile with invalid token",
			req:         updateData,
			id:          ch.ID,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update profile with empty token",
			req:         updateData,
			id:          ch.ID,
			contentType: contentType,
			auth:        emptyValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update profile with invalid data format",
			req:         "}",
			id:          ch.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update profile with empty JSON object",
			req:         "{}",
			id:          ch.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update profile with empty request",
			req:         emptyValue,
			id:          ch.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update profile with missing content type",
			req:         updateData,
			id:          ch.ID,
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
	chs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]

	data := profileRes{
		ID:       ch.ID,
		Name:     ch.Name,
		GroupID:  ch.GroupID,
		Metadata: ch.Metadata,
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
			id:     ch.ID,
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
			id:     ch.ID,
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			res:    profileRes{},
		},
		{
			desc:   "view profile with empty token",
			id:     ch.ID,
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			res:    profileRes{},
		},
		{
			desc:   "view profile with invalid id",
			id:     invalidValue,
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
		ch := things.Profile{ID: id, GroupID: gr.ID, Name: name, Metadata: map[string]interface{}{"test": "data"}}

		chs, err := svc.CreateProfiles(context.Background(), token, ch)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		profile := chs[0]

		thing.GroupID = gr.ID
		ths, err := svc.CreateThings(context.Background(), token, thing)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th := ths[0]

		svc.Connect(context.Background(), token, ch.ID, []string{th.ID})

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

func TestViewProfileByThing(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profile.GroupID = gr.ID
	chs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	ch := chs[0]

	thing.GroupID = gr.ID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th := ths[0]

	chRes := profileRes{
		ID:       ch.ID,
		Name:     ch.Name,
		GroupID:  ch.GroupID,
		Metadata: ch.Metadata,
	}

	err = svc.Connect(context.Background(), token, ch.ID, []string{th.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

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
			res:    chRes,
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
	gr := grs[0]

	profile.GroupID = gr.ID
	chs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	ch := chs[0]

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
	}{
		{
			desc:   "remove profile with invalid token",
			id:     ch.ID,
			auth:   wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove profile with empty token",
			id:     ch.ID,
			auth:   emptyValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove profile with invalid token",
			id:     ch.ID,
			auth:   wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove existing profile",
			id:     ch.ID,
			auth:   token,
			status: http.StatusNoContent,
		},
		{
			desc:   "remove removed profile",
			id:     ch.ID,
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
	gr := grs[0]

	profile.GroupID = gr.ID
	profile1.GroupID = gr.ID
	cList := []things.Profile{profile, profile1}
	chs, err := svc.CreateProfiles(context.Background(), token, cList...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var profileIDs []string
	for _, ch := range chs {
		profileIDs = append(profileIDs, ch.ID)
	}

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

func TestConnect(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr1 := grs[0]

	group2 := group
	group2.Name = "group-2"
	grs2, err := svc.CreateGroups(context.Background(), otherToken, group2)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr2 := grs2[0]

	thing.GroupID = gr1.ID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th1 := ths[0]

	thing.GroupID = gr2.ID
	ths2, err := svc.CreateThings(context.Background(), otherToken, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th2 := ths2[0]

	thIDs1 := []string{th1.ID}
	thIDs2 := []string{th2.ID}

	profile.GroupID = gr1.ID
	chs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch1 := chs[0]

	cases := []struct {
		desc        string
		profileID   string
		thingIDs    []string
		auth        string
		contentType string
		body        string
		status      int
	}{
		{
			desc:        "connect existing things to existing profile",
			profileID:   ch1.ID,
			thingIDs:    thIDs1,
			auth:        token,
			contentType: contentType,
			status:      http.StatusOK,
		},
		{
			desc:        "connect existing things to non-existent profile",
			profileID:   strconv.FormatUint(wrongID, 10),
			thingIDs:    thIDs1,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "connect non-existing things to existing profile",
			profileID:   ch1.ID,
			thingIDs:    []string{strconv.FormatUint(wrongID, 10)},
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "connect existing things to profile with invalid id",
			profileID:   invalidValue,
			thingIDs:    thIDs1,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "connect things with invalid id to existing profile",
			profileID:   ch1.ID,
			thingIDs:    []string{invalidValue},
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "connect existing things to empty profile id",
			profileID:   emptyValue,
			thingIDs:    thIDs1,
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "connect empty things id to existing profile",
			profileID:   ch1.ID,
			thingIDs:    []string{emptyValue},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "connect existing things to existing profile with invalid token",
			profileID:   ch1.ID,
			thingIDs:    thIDs1,
			auth:        wrongValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "connect existing things to existing profile with empty token",
			profileID:   ch1.ID,
			thingIDs:    thIDs1,
			auth:        emptyValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "connect with invalid content type",
			profileID:   ch1.ID,
			thingIDs:    thIDs1,
			auth:        token,
			contentType: invalidValue,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "connect with invalid JSON",
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
			body:        "{",
		},
		{
			desc:        "connect valid thing ids with empty profile id",
			profileID:   emptyValue,
			thingIDs:    thIDs1,
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "connect valid profile id with empty thing ids",
			profileID:   ch1.ID,
			thingIDs:    []string{},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "connect empty profile id and empty thing ids",
			profileID:   emptyValue,
			thingIDs:    []string{},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "connect things from another group's profile",
			profileID:   ch1.ID,
			thingIDs:    thIDs2,
			auth:        token,
			contentType: contentType,
			status:      http.StatusForbidden,
		},
	}

	for _, tc := range cases {
		data := struct {
			ProfileID string   `json:"profile_id"`
			ThingIDs  []string `json:"thing_ids"`
		}{
			tc.profileID,
			tc.thingIDs,
		}
		body := toJSON(data)

		if tc.body != emptyValue {
			body = tc.body
		}

		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/connect", ts.URL),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(body),
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestDisconnect(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr1 := grs[0]

	group2 := group
	group2.Name = "group-2"
	grs2, err := svc.CreateGroups(context.Background(), otherToken, group2)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr2 := grs2[0]

	thing.GroupID = gr1.ID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th1 := ths[0]

	thing.GroupID = gr2.ID
	ths2, err := svc.CreateThings(context.Background(), otherToken, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th2 := ths2[0]

	thIDs1 := []string{th1.ID}
	thIDs2 := []string{th2.ID}

	profile.GroupID = gr1.ID
	chs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch1 := chs[0]

	err = svc.Connect(context.Background(), token, ch1.ID, thIDs1)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc        string
		profileID   string
		thingIDs    []string
		auth        string
		contentType string
		body        string
		status      int
	}{
		{
			desc:        "disconnect existing things from existing profiles",
			profileID:   ch1.ID,
			thingIDs:    thIDs1,
			auth:        token,
			contentType: contentType,
			status:      http.StatusOK,
		},
		{
			desc:        "disconnect existing things from non-existent profiles",
			profileID:   strconv.FormatUint(wrongID, 10),
			thingIDs:    thIDs1,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "disconnect non-existing things from existing profiles",
			profileID:   ch1.ID,
			thingIDs:    []string{strconv.FormatUint(wrongID, 10)},
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "disconnect existing things from profile with invalid id",
			profileID:   invalidValue,
			thingIDs:    thIDs1,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "disconnect things with invalid id from existing profiles",
			profileID:   ch1.ID,
			thingIDs:    []string{invalidValue},
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "disconnect existing things from empty profile ids",
			profileID:   emptyValue,
			thingIDs:    thIDs1,
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "disconnect empty things id from existing profiles",
			profileID:   ch1.ID,
			thingIDs:    []string{emptyValue},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "disconnect existing things from existing profiles with invalid token",
			profileID:   ch1.ID,
			thingIDs:    thIDs1,
			auth:        wrongValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "disconnect existing things from existing profiles with empty token",
			profileID:   ch1.ID,
			thingIDs:    thIDs1,
			auth:        emptyValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "disconnect things from another group's profile",
			profileID:   ch1.ID,
			thingIDs:    thIDs2,
			auth:        token,
			contentType: contentType,
			status:      http.StatusForbidden,
		},
		{
			desc:        "disconnect with invalid content type",
			profileID:   ch1.ID,
			thingIDs:    thIDs1,
			auth:        token,
			contentType: invalidValue,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "disconnect with invalid JSON",
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
			body:        "{",
		},
		{
			desc:        "disconnect valid thing ids from empty profile ids",
			profileID:   emptyValue,
			thingIDs:    thIDs1,
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "disconnect empty thing ids from valid profile ids",
			profileID:   ch1.ID,
			thingIDs:    []string{},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "disconnect empty thing ids from empty profile ids",
			profileID:   emptyValue,
			thingIDs:    []string{},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		data := struct {
			ProfileID string   `json:"profile_id"`
			ThingIDs  []string `json:"thing_ids"`
		}{
			tc.profileID,
			tc.thingIDs,
		}
		body := toJSON(data)

		if tc.body != emptyValue {
			body = tc.body
		}

		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/disconnect", ts.URL),
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
			auth:        wrongValue,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove groups with empty group ids",
			data:        []string{emptyValue},
			auth:        wrongValue,
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

	profiles := []things.Profile{}
	for i := 0; i < 10; i++ {
		name := "name_" + fmt.Sprintf("%03d", i+1)
		chs, err := svc.CreateProfiles(context.Background(), token,
			things.Profile{
				Name:     name,
				GroupID:  gr.ID,
				Metadata: map[string]interface{}{"test": "data"},
			})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		ch := chs[0]

		profiles = append(profiles, ch)
	}
	ch := profiles[0]

	err = svc.Connect(context.Background(), token, ch.ID, []string{th.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	connections := []things.Connection{}
	connections = append(connections, things.Connection{
		ProfileID: ch.ID,
		ThingID:   th.ID,
	})

	var thingsRes []backupThingRes
	for _, th := range ths {
		thingsRes = append(thingsRes, backupThingRes{
			ID:       th.ID,
			Name:     th.Name,
			Key:      th.Key,
			Metadata: th.Metadata,
		})
	}

	var profilesRes []backupProfileRes
	for _, pr := range profiles {
		profilesRes = append(profilesRes, backupProfileRes{
			ID:       pr.ID,
			Name:     pr.Name,
			Metadata: pr.Metadata,
		})
	}

	var groupsRes []viewGroupRes
	for _, gr := range groups {
		groupsRes = append(groupsRes, viewGroupRes{
			ID:          gr.ID,
			Name:        gr.Name,
			Description: gr.Description,
			Metadata:    gr.Metadata,
		})
	}

	var connectionsRes []backupConnectionRes
	for _, conn := range connections {
		connectionsRes = append(connectionsRes, backupConnectionRes{
			ProfileID: conn.ProfileID,
			ThingID:   conn.ThingID,
		})
	}

	backup := backupRes{
		Groups:      groupsRes,
		Things:      thingsRes,
		Profiles:    profilesRes,
		Connections: connectionsRes,
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
			desc:   "backup all things profiles and connections",
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
		assert.ElementsMatch(t, tc.res.Connections, body.Connections, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res.Connections, body.Connections))
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
		Metadata: map[string]interface{}{"test": "data"},
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
		chID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		name := "name_" + fmt.Sprintf("%03d", i+1)
		profiles = append(profiles, things.Profile{
			ID:       chID,
			GroupID:  emptyValue,
			Name:     name,
			Metadata: map[string]interface{}{"test": "data"},
		})
	}
	ch := profiles[0]

	connections := things.Connection{
		ProfileID: ch.ID,
		ThingID:   testThing.ID,
	}

	thr := []restoreThingReq{
		{
			ID:       testThing.ID,
			Name:     testThing.Name,
			Key:      testThing.Key,
			Metadata: testThing.Metadata,
		},
	}

	var chr []restoreProfileReq
	for _, ch := range profiles {
		chr = append(chr, restoreProfileReq{
			ID:       ch.ID,
			Name:     ch.Name,
			Metadata: ch.Metadata,
		})
	}

	var cr []restoreConnectionReq

	cr = append(cr, restoreConnectionReq{
		ProfileID: connections.ProfileID,
		ThingID:   connections.ThingID,
	})

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
		Things:      thr,
		Profiles:    chr,
		Connections: cr,
		Groups:      gr,
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
			desc:        "restore all things profiles and connections",
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
	gr := grs[0]

	thing.GroupID = gr.ID
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
func TestGetConnByThingKey(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	thing.GroupID = gr.ID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("failed to create thing: %s", err))
	th := ths[0]

	profile.GroupID = gr.ID
	chs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("failed to create profile: %s", err))
	ch := chs[0]

	err = svc.Connect(context.Background(), token, ch.ID, []string{th.ID})
	require.Nil(t, err, fmt.Sprintf("failed to connect thing and profile: %s", err))

	data := toJSON(getConnByKeyReq{
		Key: th.Key,
	})

	cases := map[string]struct {
		contentType string
		req         string
		status      int
	}{
		"check access for connected thing and profile": {
			contentType: contentType,
			req:         data,
			status:      http.StatusOK,
		},
		"check access with invalid content type": {
			contentType: wrongValue,
			req:         data,
			status:      http.StatusUnsupportedMediaType,
		},
		"check access with empty JSON request": {
			contentType: contentType,
			req:         "{}",
			status:      http.StatusUnauthorized,
		},
		"check access with invalid JSON request": {
			contentType: contentType,
			req:         "}",
			status:      http.StatusBadRequest,
		},
		"check access with empty request": {
			contentType: contentType,
			req:         emptyValue,
			status:      http.StatusBadRequest,
		},
	}

	for desc, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/connections", ts.URL),
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

type getConnByKeyReq struct {
	Key string `json:"key"`
}

type thingRes struct {
	ID       string                 `json:"id"`
	GroupID  string                 `json:"group_id,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Key      string                 `json:"key"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
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
	ID       string                 `json:"id"`
	Name     string                 `json:"name,omitempty"`
	Key      string                 `json:"key"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type backupProfileRes struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type backupConnectionRes struct {
	ProfileID string `json:"profile_id"`
	ThingID   string `json:"thing_id"`
}

type viewGroupRes struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type backupRes struct {
	Things      []backupThingRes      `json:"things"`
	Profiles    []backupProfileRes    `json:"profiles"`
	Connections []backupConnectionRes `json:"connections"`
	Groups      []viewGroupRes        `json:"groups"`
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

type restoreConnectionReq struct {
	ProfileID string `json:"profile_id"`
	ThingID   string `json:"thing_id"`
}

type restoreGroupReq struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type restoreGroupThingRelationReq struct {
	ThingID   string    `json:"thing_id"`
	GroupID   string    `json:"group_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type restoreGroupProfileRelationReq struct {
	ProfileID string    `json:"profile_id"`
	GroupID   string    `json:"group_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type restoreReq struct {
	Things                []restoreThingReq                `json:"things"`
	Profiles              []restoreProfileReq              `json:"profiles"`
	Connections           []restoreConnectionReq           `json:"connections"`
	Groups                []restoreGroupReq                `json:"groups"`
	GroupThingRelations   []restoreGroupThingRelationReq   `json:"group_thing_relations"`
	GroupProfileRelations []restoreGroupProfileRelationReq `json:"group_profile_relations"`
}
