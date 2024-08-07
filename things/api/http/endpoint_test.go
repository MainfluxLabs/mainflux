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
	wrongID        = 0
	password       = "password"
	maxNameSize    = 1024
	nameKey        = "name"
	ascKey         = "asc"
	descKey        = "desc"
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
	channel = things.Channel{
		Name:     "test",
		Metadata: map[string]interface{}{"test": "data"},
	}
	invalidName    = strings.Repeat("m", maxNameSize+1)
	searchThingReq = things.PageMetadata{
		Limit:  5,
		Offset: 0,
	}
	user      = users.User{ID: "574106f7-030e-4881-8ab0-151195c29f94", Email: email, Password: password}
	otherUser = users.User{ID: "ecf9e48b-ba3b-41c4-82a9-72e063b17868", Email: otherUserEmail, Password: password}
	admin     = users.User{ID: "2e248e36-2d26-46ea-97b0-1e38d674cbe4", Email: adminEmail, Password: password}
	usersList = []users.User{admin, user, otherUser}
	group     = things.Group{Name: "test-group", Description: "test-group-desc"}
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
	channelsRepo := thmocks.NewChannelRepository(thingsRepo, conns)
	groupsRepo := thmocks.NewGroupRepository()
	rolesRepo := thmocks.NewRolesRepository()
	chanCache := thmocks.NewChannelCache()
	thingCache := thmocks.NewThingCache()
	idProvider := uuid.NewMock()

	return things.New(auth, nil, thingsRepo, channelsRepo, groupsRepo, rolesRepo, chanCache, thingCache, idProvider)
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
			response:    "",
		},
		{
			desc:        "create things with empty request",
			data:        "",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    "",
		},
		{
			desc:        "create thing with invalid request format",
			data:        "}",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    "",
		},
		{
			desc:        "create thing with invalid name",
			data:        invalidData,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    "",
		},
		{
			desc:        "create things with empty JSON array",
			data:        "[]",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    "",
		},
		{
			desc:        "create thing with existing key",
			data:        data,
			contentType: contentType,
			auth:        token,
			status:      http.StatusConflict,
			response:    "",
		},
		{
			desc:        "create thing with invalid auth token",
			data:        data,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
			response:    "",
		},
		{
			desc:        "create thing with empty auth token",
			data:        data,
			contentType: contentType,
			auth:        "",
			status:      http.StatusUnauthorized,
			response:    "",
		},
		{
			desc:        "create thing without content type",
			data:        data,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
			response:    "",
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
			status:      http.StatusOK,
		},
		{
			desc:        "update non-existent thing",
			req:         data,
			id:          strconv.FormatUint(wrongID, 10),
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update thing with invalid id",
			req:         data,
			id:          "invalid",
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
			auth:        "",
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
			req:         "",
			id:          th1.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing without content type",
			req:         data,
			id:          th1.ID,
			contentType: "",
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

	th := thing
	th.Key = "key"
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
			id:          "invalid",
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
			auth:        "",
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
			req:         "",
			id:          th.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing without content type",
			req:         data,
			id:          th.ID,
			contentType: "",
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
			auth:   "",
			status: http.StatusUnauthorized,
			res:    thingRes{},
		},
		{
			desc:   "view thing by passing invalid id",
			id:     "invalid",
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

		thing.GroupID = gr.ID
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
			desc:   "get a list of things with no limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?limit=%d", thingURL, noLimit),
			res:    data,
		},
		{
			desc:   "get a list of things ordered by name descendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=name&dir=desc", thingURL, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things ordered by name ascendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=name&dir=asc", thingURL, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=wrong", thingURL, 0, 5),
			res:    nil,
		},
		{
			desc:   "get a list of things with invalid dir",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=name&dir=wrong", thingURL, 0, 5),
			res:    nil,
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
			auth:   "",
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
			url:    fmt.Sprintf("%s%s", thingURL, ""),
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
			desc:   "get a list of things sorted by name ascendent",
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

	th := searchThingReq
	validData := toJSON(th)

	th.Dir = "desc"
	th.Order = "name"
	descData := toJSON(th)

	th.Dir = "asc"
	ascData := toJSON(th)

	th.Order = "wrong"
	invalidOrderData := toJSON(th)

	th.Limit = 0
	zeroLimitData := toJSON(th)

	th = searchThingReq
	th.Dir = "wrong"
	invalidDirData := toJSON(th)

	th = searchThingReq
	th.Limit = 110
	limitMaxData := toJSON(th)

	th = searchThingReq
	th.Name = invalidName
	invalidNameData := toJSON(th)

	th.Name = invalidName
	invalidData := toJSON(th)

	data := []thingRes{}
	for i := 0; i < 100; i++ {
		name := "name_" + fmt.Sprintf("%03d", i+1)
		id := fmt.Sprintf("%s%012d", prefix, i+1)
		ths, err := svc.CreateThings(context.Background(), token, things.Thing{ID: id, Name: name, Metadata: map[string]interface{}{"test": name}})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th := ths[0]
		data = append(data, thingRes{
			ID:       th.ID,
			Name:     th.Name,
			Key:      th.Key,
			Metadata: th.Metadata,
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
			desc:   "search things ordered by name descendent",
			auth:   token,
			status: http.StatusOK,
			req:    descData,
			res:    data[0:5],
		},
		{
			desc:   "search things ordered by name ascendent",
			auth:   token,
			status: http.StatusOK,
			req:    ascData,
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
			auth:   "",
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
			desc:   "search things sorted by name ascendent",
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

func TestListThingsByChannel(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	channel.GroupID = gr.ID
	chs, err := svc.CreateChannels(context.Background(), token, channel)
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

	thingURL := fmt.Sprintf("%s/channels", ts.URL)

	// Wait for things and channels to connect.
	time.Sleep(time.Second)

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []thingRes
	}{
		{
			desc:   "get a list of things by channel",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by channel with no limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?limit=%d", thingURL, ch.ID, noLimit),
			res:    data,
		},
		{
			desc:   "get a list of things by channel with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of things by channel with empty token",
			auth:   "",
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of things by channel with negative offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, -2, 5),
			res:    nil,
		},
		{
			desc:   "get a list of things by channel with negative limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, 1, -5),
			res:    nil,
		},
		{
			desc:   "get a list of things by channel with zero limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, 1, 0),
			res:    nil,
		},
		{
			desc:   "get a list of things by channel without offset",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?limit=%d", thingURL, ch.ID, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by channel without limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d", thingURL, ch.ID, 1),
			res:    data[1:11],
		},
		{
			desc:   "get a list of things by channel with redundant query params",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&value=something", thingURL, ch.ID, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by channel with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, 0, 110),
			res:    nil,
		},
		{
			desc:   "get a list of things by channel with default URL",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things", thingURL, ch.ID),
			res:    data[0:10],
		},
		{
			desc:   "get a list of things by channel with invalid number of params",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things%s", thingURL, ch.ID, "?offset=4&limit=4&limit=5&offset=5"),
			res:    nil,
		},
		{
			desc:   "get a list of things by channel with invalid offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things%s", thingURL, ch.ID, "?offset=e&limit=5"),
			res:    nil,
		},
		{
			desc:   "get a list of things by channel with invalid limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things%s", thingURL, ch.ID, "?offset=5&limit=e"),
			res:    nil,
		},
		{
			desc:   "get a list of things by channel sorted by name ascendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&order=%s&dir=%s", thingURL, ch.ID, 0, 5, nameKey, ascKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by channel sorted by name descendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&order=%s&dir=%s", thingURL, ch.ID, 0, 5, nameKey, descKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by channel sorted with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&order=%s&dir=%s", thingURL, ch.ID, 0, 5, "wrong", ascKey),
			res:    nil,
		},
		{
			desc:   "get a list of things by channel sorted by name with invalid direction",
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
		{
			desc:   "remove thing with invalid token",
			id:     th.ID,
			auth:   wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove thing with empty token",
			id:     th.ID,
			auth:   "",
			status: http.StatusUnauthorized,
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

	ths := []things.Thing{thing, thing1}
	usrThs, err := svc.CreateThings(context.Background(), token, ths...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	var thingIDs []string
	for _, th := range usrThs {
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
			desc:        "remove things with invalid token",
			data:        thingIDs,
			auth:        wrongValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove things with empty token",
			data:        thingIDs,
			auth:        "",
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

func TestCreateChannels(t *testing.T) {
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
			desc:        "create valid channels",
			data:        data,
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			response:    "",
		},
		{
			desc:        "create channel with empty request",
			data:        "",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    "",
		},
		{
			desc:        "create channels with empty JSON",
			data:        "[]",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    "",
		},
		{
			desc:        "create channel with invalid auth token",
			data:        data,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
			response:    "",
		},
		{
			desc:        "create channel with empty auth token",
			data:        data,
			contentType: contentType,
			auth:        "",
			status:      http.StatusUnauthorized,
			response:    "",
		},
		{
			desc:     "create channel with invalid request format",
			data:     "}",
			auth:     token,
			status:   http.StatusUnsupportedMediaType,
			response: "",
		},
		{
			desc:        "create channel without content type",
			data:        data,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
			response:    "",
		},
		{
			desc:        "create channel with invalid name",
			data:        invalidData,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    "",
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/groups/%s/channels", ts.URL, gr.ID),
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

func TestUpdateChannel(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	channel.GroupID = gr.ID
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]

	c := channel
	c.Name = "updated_channel"
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
			desc:        "update existing channel",
			req:         updateData,
			id:          ch.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update non-existing channel",
			req:         updateData,
			id:          strconv.FormatUint(wrongID, 10),
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update channel with invalid id",
			req:         updateData,
			id:          "invalid",
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update channel with invalid token",
			req:         updateData,
			id:          ch.ID,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update channel with empty token",
			req:         updateData,
			id:          ch.ID,
			contentType: contentType,
			auth:        "",
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update channel with invalid data format",
			req:         "}",
			id:          ch.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update channel with empty JSON object",
			req:         "{}",
			id:          ch.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update channel with empty request",
			req:         "",
			id:          ch.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update channel with missing content type",
			req:         updateData,
			id:          ch.ID,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "update channel with invalid name",
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
			url:         fmt.Sprintf("%s/channels/%s", ts.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestViewChannel(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	channel.GroupID = gr.ID
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]

	data := channelRes{
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
		res    channelRes
	}{
		{
			desc:   "view existing channel",
			id:     ch.ID,
			auth:   token,
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "view non-existent channel",
			id:     strconv.FormatUint(wrongID, 10),
			auth:   token,
			status: http.StatusNotFound,
			res:    channelRes{},
		},
		{
			desc:   "view channel with invalid token",
			id:     ch.ID,
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			res:    channelRes{},
		},
		{
			desc:   "view channel with empty token",
			id:     ch.ID,
			auth:   "",
			status: http.StatusUnauthorized,
			res:    channelRes{},
		},
		{
			desc:   "view channel with invalid id",
			id:     "invalid",
			auth:   token,
			status: http.StatusNotFound,
			res:    channelRes{},
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/channels/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var body channelRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, body, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body))
	}
}

func TestListChannels(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	channels := []channelRes{}
	for i := 0; i < n; i++ {
		name := "name_" + fmt.Sprintf("%03d", i+1)
		chs, err := svc.CreateChannels(context.Background(), token,
			things.Channel{
				Name:     name,
				Metadata: map[string]interface{}{"test": "data"},
			})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		ch := chs[0]
		ths, err := svc.CreateThings(context.Background(), token, thing)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th := ths[0]
		svc.Connect(context.Background(), token, ch.ID, []string{th.ID})

		channels = append(channels, channelRes{
			ID:       ch.ID,
			Name:     ch.Name,
			Metadata: ch.Metadata,
		})
	}
	channelURL := fmt.Sprintf("%s/channels", ts.URL)

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []channelRes
	}{
		{
			desc:   "get a list of channels",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 0, 6),
			res:    channels[0:6],
		},
		{
			desc:   "get a list of all channels with no limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?limit=%d", channelURL, noLimit),
			res:    channels,
		},
		{
			desc:   "get a list of channels ordered by id descendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=id&dir=desc", channelURL, 0, 6),
			res:    channels[len(channels)-6:],
		},
		{
			desc:   "get a list of channels ordered by id ascendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=id&dir=asc", channelURL, 0, 6),
			res:    channels[0:6],
		},
		{
			desc:   "get a list of channels with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=wrong", channelURL, 0, 6),
			res:    nil,
		},
		{
			desc:   "get a list of channels with invalid dir",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=name&dir=wrong", channelURL, 0, 6),
			res:    nil,
		},
		{
			desc:   "get a list of channels with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of channels with empty token",
			auth:   "",
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of channels with negative offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, -1, 5),
			res:    nil,
		},
		{
			desc:   "get a list of channels with negative limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 5, -2),
			res:    nil,
		},
		{
			desc:   "get a list of channels with zero limit and offset 1",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 1, 0),
			res:    nil,
		},
		{
			desc:   "get a list of channels with no offset provided",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?limit=%d", channelURL, 5),
			res:    channels[0:5],
		},
		{
			desc:   "get a list of channels with no limit provided",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d", channelURL, 1),
			res:    channels[1:11],
		},
		{
			desc:   "get a list of channels with redundant query params",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&value=something", channelURL, 0, 5),
			res:    channels[0:5],
		},
		{
			desc:   "get a list of channels with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 0, 110),
			res:    nil,
		},
		{
			desc:   "get a list of channels with default URL",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s%s", channelURL, ""),
			res:    channels[0:10],
		},
		{
			desc:   "get a list of channels with invalid number of params",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", channelURL, "?offset=4&limit=4&limit=5&offset=5"),
			res:    nil,
		},
		{
			desc:   "get a list of channels with invalid offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", channelURL, "?offset=e&limit=5"),
			res:    nil,
		},
		{
			desc:   "get a list of channels with invalid limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", channelURL, "?offset=5&limit=e"),
			res:    nil,
		},
		{
			desc:   "get a list of channels with invalid name",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&name=%s", channelURL, 0, 10, invalidName),
			res:    nil,
		},
		{
			desc:   "get a list of channels sorted by name ascendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", channelURL, 0, 6, nameKey, ascKey),
			res:    channels[0:6],
		},
		{
			desc:   "get a list of channels sorted by name descendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", channelURL, 0, 6, nameKey, descKey),
			res:    channels[len(channels)-6:],
		},
		{
			desc:   "get a list of channels sorted with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", channelURL, 0, 6, "wrong", ascKey),
			res:    nil,
		},
		{
			desc:   "get a list of channels sorted by name with invalid direction",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", channelURL, 0, 6, nameKey, "wrong"),
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
		var body channelsPageRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, body.Channels, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body.Channels))
	}
}

func TestViewChannelByThing(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	channel.GroupID = gr.ID
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	ch := chs[0]

	thing.GroupID = gr.ID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th := ths[0]

	chRes := channelRes{
		ID:       ch.ID,
		Name:     ch.Name,
		GroupID:  ch.GroupID,
		Metadata: ch.Metadata,
	}

	err = svc.Connect(context.Background(), token, ch.ID, []string{th.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	channelURL := fmt.Sprintf("%s/things", ts.URL)

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    channelRes
	}{
		{
			desc:   "view channel by thing",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/channels", channelURL, th.ID),
			res:    chRes,
		},
		{
			desc:   "view channel by thing with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/channels", channelURL, th.ID),
			res:    channelRes{},
		},
		{
			desc:   "view channel by thing with empty token",
			auth:   "",
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/channels", channelURL, th.ID),
			res:    channelRes{},
		},
		{
			desc:   "view channel by thing without thing id",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/channels", channelURL, ""),
			res:    channelRes{},
		},
		{
			desc:   "view channel by thing with wrong thing id",
			auth:   token,
			status: http.StatusNotFound,
			url:    fmt.Sprintf("%s/%s/channels", channelURL, wrongValue),
			res:    channelRes{},
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
		var body channelRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, body, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body))
	}
}

func TestRemoveChannel(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	chs, _ := svc.CreateChannels(context.Background(), token, channel)
	ch := chs[0]

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
	}{
		{
			desc:   "remove channel with invalid token",
			id:     ch.ID,
			auth:   wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove existing channel",
			id:     ch.ID,
			auth:   token,
			status: http.StatusNoContent,
		},
		{
			desc:   "remove removed channel",
			id:     ch.ID,
			auth:   token,
			status: http.StatusNotFound,
		},
		{
			desc:   "remove channel with invalid token",
			id:     ch.ID,
			auth:   wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove channel with empty token",
			id:     ch.ID,
			auth:   "",
			status: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/channels/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestRemoveChannels(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	channel1 := things.Channel{Name: "test1"}

	c := []things.Channel{channel, channel1}
	chs, _ := svc.CreateChannels(context.Background(), token, c...)

	var chIDs []string
	for _, ch := range chs {
		chIDs = append(chIDs, ch.ID)
	}

	cases := []struct {
		desc        string
		ids         []string
		auth        string
		contentType string
		status      int
	}{
		{
			desc:        "remove channels with invalid token",
			ids:         chIDs,
			auth:        wrongValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove existing channels",
			ids:         chIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNoContent,
		},
		{
			desc:        "remove removed channels",
			ids:         chIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "remove channels with invalid token",
			ids:         chIDs,
			auth:        wrongValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove channels with empty channel ids",
			ids:         []string{""},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove channels without channel ids",
			ids:         []string{},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove channels with empty token",
			ids:         chIDs,
			auth:        "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove channels with invalid content type",
			ids:         chIDs,
			auth:        token,
			contentType: wrongValue,
			status:      http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		data := struct {
			ChannelIDs []string `json:"channel_ids"`
		}{
			tc.ids,
		}

		body := toJSON(data)

		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/channels", ts.URL),
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

	group2 := group
	group2.Name = "group-2"
	grs, err := svc.CreateGroups(context.Background(), token, group, group2)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]
	gr2 := grs[1]

	thing.GroupID = gr.ID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	thIDs := []string{}
	for _, th := range ths {
		thIDs = append(thIDs, th.ID)
	}

	channel.GroupID = gr.ID
	chs1, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch1 := chs1[0]

	channel.GroupID = gr2.ID
	chs2, err := svc.CreateChannels(context.Background(), otherToken, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch2 := chs2[0]

	cases := []struct {
		desc        string
		channelID   string
		thingIDs    []string
		auth        string
		contentType string
		body        string
		status      int
	}{
		{
			desc:        "connect existing things to existing channel",
			channelID:   ch1.ID,
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusOK,
		},
		{
			desc:        "connect existing things to non-existent channel",
			channelID:   strconv.FormatUint(wrongID, 10),
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "connect non-existing things to existing channel",
			channelID:   ch1.ID,
			thingIDs:    []string{strconv.FormatUint(wrongID, 10)},
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "connect existing things to channel with invalid id",
			channelID:   "invalid",
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "connect things with invalid id to existing channel",
			channelID:   ch1.ID,
			thingIDs:    []string{"invalid"},
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "connect existing things to empty channel id",
			channelID:   "",
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "connect empty things id to existing channel",
			channelID:   ch1.ID,
			thingIDs:    []string{""},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "connect existing things to existing channel with invalid token",
			channelID:   ch1.ID,
			thingIDs:    thIDs,
			auth:        wrongValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "connect existing things to existing channel with empty token",
			channelID:   ch1.ID,
			thingIDs:    thIDs,
			auth:        "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "connect things from owner to channel of other user",
			channelID:   ch2.ID,
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusForbidden,
		},
		{
			desc:        "connect with invalid content type",
			channelID:   ch2.ID,
			thingIDs:    thIDs,
			auth:        token,
			contentType: "invalid",
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
			desc:        "connect valid thing ids with empty channel id",
			channelID:   "",
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "connect valid channel id with empty thing ids",
			channelID:   ch2.ID,
			thingIDs:    []string{},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "connect empty channel id and empty thing ids",
			channelID:   "",
			thingIDs:    []string{},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		data := struct {
			ChannelID string   `json:"channel_id"`
			ThingIDs  []string `json:"thing_ids"`
		}{
			tc.channelID,
			tc.thingIDs,
		}
		body := toJSON(data)

		if tc.body != "" {
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

	group2 := group
	group2.Name = "group-2"
	grs, err := svc.CreateGroups(context.Background(), token, group, group2)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]
	gr2 := grs[1]

	thing.GroupID = gr.ID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	thIDs := []string{}
	for _, th := range ths {
		thIDs = append(thIDs, th.ID)
	}

	channel.GroupID = gr.ID
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]

	channel.GroupID = gr2.ID
	uCh, err := svc.CreateChannels(context.Background(), otherToken, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	usrCh := uCh[0]

	err = svc.Connect(context.Background(), token, ch.ID, thIDs)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc        string
		channelID   string
		thingIDs    []string
		auth        string
		contentType string
		body        string
		status      int
	}{
		{
			desc:        "disconnect existing things from existing channels",
			channelID:   ch.ID,
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusOK,
		},
		{
			desc:        "disconnect existing things from non-existent channels",
			channelID:   strconv.FormatUint(wrongID, 10),
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "disconnect non-existing things from existing channels",
			channelID:   ch.ID,
			thingIDs:    []string{strconv.FormatUint(wrongID, 10)},
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "disconnect existing things from channel with invalid id",
			channelID:   "invalid",
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "disconnect things with invalid id from existing channels",
			channelID:   ch.ID,
			thingIDs:    []string{"invalid"},
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "disconnect existing things from empty channel ids",
			channelID:   "",
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "disconnect empty things id from existing channels",
			channelID:   ch.ID,
			thingIDs:    []string{""},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "disconnect existing things from existing channels with invalid token",
			channelID:   ch.ID,
			thingIDs:    thIDs,
			auth:        wrongValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "disconnect existing things from existing channels with empty token",
			channelID:   ch.ID,
			thingIDs:    thIDs,
			auth:        "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "disconnect things from channels of other user",
			channelID:   usrCh.ID,
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "disconnect with invalid content type",
			channelID:   ch.ID,
			thingIDs:    thIDs,
			auth:        token,
			contentType: "invalid",
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
			desc:        "disconnect valid thing ids from empty channel ids",
			channelID:   "",
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "disconnect empty thing ids from valid channel ids",
			channelID:   ch.ID,
			thingIDs:    []string{},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "disconnect empty thing ids from empty channel ids",
			channelID:   "",
			thingIDs:    []string{},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		data := struct {
			ChannelID string   `json:"channel_id"`
			ThingIDs  []string `json:"thing_ids"`
		}{
			tc.channelID,
			tc.thingIDs,
		}
		body := toJSON(data)

		if tc.body != "" {
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
			data:        groupIDs,
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
			data:        groupIDs,
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
			data:        []string{""},
			auth:        wrongValue,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove groups with empty token",
			data:        groupIDs,
			auth:        "",
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

	channels := []things.Channel{}
	for i := 0; i < 10; i++ {
		name := "name_" + fmt.Sprintf("%03d", i+1)
		chs, err := svc.CreateChannels(context.Background(), token,
			things.Channel{
				Name:     name,
				GroupID:  gr.ID,
				Metadata: map[string]interface{}{"test": "data"},
			})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		ch := chs[0]

		channels = append(channels, ch)
	}
	ch := channels[0]

	err = svc.Connect(context.Background(), token, ch.ID, []string{th.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	connections := []things.Connection{}
	connections = append(connections, things.Connection{
		ChannelID: ch.ID,
		ThingID:   th.ID,
	})

	var thingsRes []backupThingRes
	for _, th := range ths {
		thingsRes = append(thingsRes, backupThingRes{
			ID:       th.ID,
			OwnerID:  th.OwnerID,
			Name:     th.Name,
			Key:      th.Key,
			Metadata: th.Metadata,
		})
	}

	var channelsRes []backupChannelRes
	for _, ch := range channels {
		channelsRes = append(channelsRes, backupChannelRes{
			ID:       ch.ID,
			OwnerID:  ch.OwnerID,
			Name:     ch.Name,
			Metadata: ch.Metadata,
		})
	}

	var groupsRes []viewGroupRes
	for _, gr := range groups {
		groupsRes = append(groupsRes, viewGroupRes{
			ID:          gr.ID,
			Name:        gr.Name,
			OwnerID:     gr.OwnerID,
			Description: gr.Description,
			Metadata:    gr.Metadata,
		})
	}

	var connectionsRes []backupConnectionRes
	for _, conn := range connections {
		connectionsRes = append(connectionsRes, backupConnectionRes{
			ChannelID: conn.ChannelID,
			ThingID:   conn.ThingID,
		})
	}

	backup := backupRes{
		Groups:      groupsRes,
		Things:      thingsRes,
		Channels:    channelsRes,
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
			desc:   "backup all things channels and connections",
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
			auth:   "",
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
		assert.ElementsMatch(t, tc.res.Channels, body.Channels, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res.Channels, body.Channels))
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
		OwnerID:  adminEmail,
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

	channels := []things.Channel{}
	for i := 0; i < n; i++ {
		chID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		name := "name_" + fmt.Sprintf("%03d", i+1)
		channels = append(channels, things.Channel{
			ID:       chID,
			OwnerID:  adminEmail,
			GroupID:  "",
			Name:     name,
			Metadata: map[string]interface{}{"test": "data"},
		})
	}
	ch := channels[0]

	connections := things.Connection{
		ChannelID: ch.ID,
		ThingID:   testThing.ID,
	}

	thr := []restoreThingReq{
		{
			ID:       testThing.ID,
			OwnerID:  testThing.OwnerID,
			Name:     testThing.Name,
			Key:      testThing.Key,
			Metadata: testThing.Metadata,
		},
	}

	var chr []restoreChannelReq
	for _, ch := range channels {
		chr = append(chr, restoreChannelReq{
			ID:       ch.ID,
			OwnerID:  ch.OwnerID,
			Name:     ch.Name,
			Metadata: ch.Metadata,
		})
	}

	var cr []restoreConnectionReq

	cr = append(cr, restoreConnectionReq{
		ChannelID: connections.ChannelID,
		ThingID:   connections.ThingID,
	})

	var gr []restoreGroupReq
	for _, group := range groups {
		gr = append(gr, restoreGroupReq{
			ID:          group.ID,
			Name:        group.Name,
			OwnerID:     group.OwnerID,
			Description: group.Description,
			Metadata:    group.Metadata,
		})
	}

	resReq := restoreReq{
		Things:      thr,
		Channels:    chr,
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
			desc:        "restore all things channels and connections",
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
			auth:        "",
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
			req:         "",
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

	channel.GroupID = gr.ID
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("failed to create channel: %s", err))
	ch := chs[0]

	err = svc.Connect(context.Background(), token, ch.ID, []string{th.ID})
	require.Nil(t, err, fmt.Sprintf("failed to connect thing and channel: %s", err))

	data := toJSON(getConnByKeyReq{
		Key: th.Key,
	})

	cases := map[string]struct {
		contentType string
		req         string
		status      int
	}{
		"check access for connected thing and channel": {
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
			req:         "",
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

type channelRes struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name,omitempty"`
	GroupID  string                 `json:"group_id,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Profile  map[string]interface{} `json:"profile,omitempty"`
}

type thingsPageRes struct {
	Things []thingRes `json:"things"`
	Total  uint64     `json:"total"`
	Offset uint64     `json:"offset"`
	Limit  uint64     `json:"limit"`
}

type channelsPageRes struct {
	Channels []channelRes `json:"channels"`
	Total    uint64       `json:"total"`
	Offset   uint64       `json:"offset"`
	Limit    uint64       `json:"limit"`
}

type backupThingRes struct {
	ID       string                 `json:"id"`
	OwnerID  string                 `json:"owner_id,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Key      string                 `json:"key"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type backupChannelRes struct {
	ID       string                 `json:"id"`
	OwnerID  string                 `json:"owner_id,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type backupConnectionRes struct {
	ChannelID    string `json:"channel_id"`
	ChannelOwner string `json:"channel_owner"`
	ThingID      string `json:"thing_id"`
	ThingOwner   string `json:"thing_owner"`
}

type viewGroupRes struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	OwnerID     string                 `json:"owner_id"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type backupRes struct {
	Things      []backupThingRes      `json:"things"`
	Channels    []backupChannelRes    `json:"channels"`
	Connections []backupConnectionRes `json:"connections"`
	Groups      []viewGroupRes        `json:"groups"`
}

type restoreThingReq struct {
	ID       string                 `json:"id"`
	OwnerID  string                 `json:"owner_id"`
	Name     string                 `json:"name"`
	Key      string                 `json:"key"`
	Metadata map[string]interface{} `json:"metadata"`
}

type restoreChannelReq struct {
	ID       string                 `json:"id"`
	OwnerID  string                 `json:"owner_id"`
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata"`
}

type restoreConnectionReq struct {
	ChannelID    string `json:"channel_id"`
	ChannelOwner string `json:"channel_owner"`
	ThingID      string `json:"thing_id"`
	ThingOwner   string `json:"thing_owner"`
}

type restoreGroupReq struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	OwnerID     string                 `json:"owner_id"`
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

type restoreGroupChannelRelationReq struct {
	ChannelID string    `json:"channel_id"`
	GroupID   string    `json:"group_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type restoreReq struct {
	Things                []restoreThingReq                `json:"things"`
	Channels              []restoreChannelReq              `json:"channels"`
	Connections           []restoreConnectionReq           `json:"connections"`
	Groups                []restoreGroupReq                `json:"groups"`
	GroupThingRelations   []restoreGroupThingRelationReq   `json:"group_thing_relations"`
	GroupChannelRelations []restoreGroupChannelRelationReq `json:"group_channel_relations"`
}
