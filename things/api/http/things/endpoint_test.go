// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things_test

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
	contentTypeJSON        = "application/json"
	contentTypeOctetStream = "application/octet-stream"
	email                  = "user@example.com"
	adminEmail             = "admin@example.com"
	otherUserEmail         = "other_user@example.com"
	token                  = email
	otherToken             = otherUserEmail
	adminToken             = adminEmail
	wrongValue             = "wrong_value"
	emptyValue             = ""
	emptyJson              = "{}"
	wrongID                = 0
	password               = "password"
	maxNameSize            = 1024
	nameKey                = "name"
	ascKey                 = "asc"
	descKey                = "desc"
	orgID                  = "374106f7-030e-4881-8ab0-151195c29f92"
	orgID2                 = "374106f7-030e-4881-8ab0-151195c29f93"
	prefix                 = "fe6b4e92-cc98-425e-b0aa-"
	n                      = 101
	noLimit                = -1
	validData              = `{"limit":5,"offset":0}`
	descData               = `{"limit":5,"offset":0,"dir":"desc","order":"name"}`
	ascData                = `{"limit":5,"offset":0,"dir":"asc","order":"name"}`
	invalidOrderData       = `{"limit":5,"offset":0,"dir":"asc","order":"wrong"}`
	zeroLimitData          = `{"limit":0,"offset":0}`
	invalidDirData         = `{"limit":5,"offset":0,"dir":"wrong"}`
	invalidLimitData       = `{"limit":210,"offset":0}`
	invalidData            = `{"limit": "invalid"}`
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

	user            = users.User{ID: "574106f7-030e-4881-8ab0-151195c29f94", Email: email, Password: password, Role: auth.Owner}
	otherUser       = users.User{ID: "ecf9e48b-ba3b-41c4-82a9-72e063b17868", Email: otherUserEmail, Password: password, Role: auth.Editor}
	admin           = users.User{ID: "2e248e36-2d26-46ea-97b0-1e38d674cbe4", Email: adminEmail, Password: password, Role: auth.RootSub}
	usersList       = []users.User{admin, user, otherUser}
	group           = things.Group{Name: "test-group", Description: "test-group-desc", OrgID: orgID}
	orgsList        = []auth.Org{{ID: orgID, OwnerID: user.ID}, {ID: orgID2, OwnerID: user.ID}}
	metadata        = map[string]any{"test": "data"}
	invalidName     = strings.Repeat("m", maxNameSize+1)
	invalidNameData = fmt.Sprintf(`{"limit":5,"offset":0,"name":"%s"}`, invalidName)
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
		req.Header.Set("Authorization", apiutil.ThingKeyPrefixInternal+tr.key)
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
	groupMembershipsRepo := thmocks.NewGroupMembershipsRepository()
	groupsRepo := thmocks.NewGroupRepository(groupMembershipsRepo)
	profileCache := thmocks.NewProfileCache()
	thingCache := thmocks.NewThingCache()
	groupCache := thmocks.NewGroupCache()
	idProvider := uuid.NewMock()
	emailerMock := thmocks.NewEmailer()

	return things.New(auth, nil, thingsRepo, profilesRepo, groupsRepo, groupMembershipsRepo, profileCache, thingCache, groupCache, idProvider, emailerMock)
}

func newServer(svc things.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(svc, mocktracer.New(), logger)
	return httptest.NewServer(mux)
}

func toJSON(data any) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func TestCreateThings(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	data := fmt.Sprintf(`[{"name": "1", "key": "1","profile_id":"%s"}, {"name": "2", "key": "2","profile_id":"%s"}]`, prID, prID)
	invalidNameData := fmt.Sprintf(`[{"name": "%s", "key": "10","profile_id":"%s"}]`, invalidName, prID)

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
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusCreated,
			response:    emptyValue,
		},
		{
			desc:        "create things with empty request",
			data:        emptyValue,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create thing with invalid request format",
			data:        "}",
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create thing with invalid name",
			data:        invalidNameData,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create things with empty JSON array",
			data:        "[]",
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create thing with existing key",
			data:        data,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusConflict,
			response:    emptyValue,
		},
		{
			desc:        "create thing with invalid auth token",
			data:        data,
			contentType: contentTypeJSON,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
			response:    emptyValue,
		},
		{
			desc:        "create thing with empty auth token",
			data:        data,
			contentType: contentTypeJSON,
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
			url:         fmt.Sprintf("%s/profiles/%s/things", ts.URL, prID),
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

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	ths, err := svc.CreateThings(context.Background(), token, prID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	data := `{"name":"test", "key": "tk1"}`
	dataMissingKey := `{"name":"test"}`
	dataMissingName := `{"key":"tk1"}`
	invalidNameData := fmt.Sprintf(`{"name": "%s", "key": "tk1"}`, invalidName)

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
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update thing with empty JSON request",
			req:         "{}",
			id:          th.ID,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update non-existent thing",
			req:         data,
			id:          wrongValue,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update thing with invalid id",
			req:         data,
			id:          wrongValue,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update thing with invalid user token",
			req:         data,
			id:          th.ID,
			contentType: contentTypeJSON,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update thing with empty user token",
			req:         data,
			id:          th.ID,
			contentType: contentTypeJSON,
			auth:        emptyValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update thing with invalid data format",
			req:         "{",
			id:          th.ID,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing with empty request",
			req:         emptyValue,
			id:          th.ID,
			contentType: contentTypeJSON,
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
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing with missing name",
			req:         dataMissingName,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing with missing key",
			req:         dataMissingKey,
			contentType: contentTypeJSON,
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

func TestUpdateThingGroupAndProfile(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	ths, err := svc.CreateThings(context.Background(), token, prID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	data := fmt.Sprintf(`{"profile_id":"%s","group_id": "%s"}`, prID, grID)
	dataMissingGroupID := fmt.Sprintf(`{"profile_id": "%s"}`, prID)
	dataMissingProfileID := fmt.Sprintf(`{"group_id": "%s"}`, grID)

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
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update thing with empty JSON request",
			req:         "{}",
			id:          th.ID,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing with missing group id",
			req:         dataMissingGroupID,
			id:          th.ID,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing with missing profile id",
			req:         dataMissingProfileID,
			id:          th.ID,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update non-existent thing",
			req:         data,
			id:          wrongValue,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update thing with invalid id",
			req:         data,
			id:          wrongValue,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update thing with invalid user token",
			req:         data,
			id:          th.ID,
			contentType: contentTypeJSON,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update thing with empty user token",
			req:         data,
			id:          th.ID,
			contentType: contentTypeJSON,
			auth:        emptyValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update thing with invalid data format",
			req:         "{",
			id:          th.ID,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing with empty request",
			req:         emptyValue,
			id:          th.ID,
			contentType: contentTypeJSON,
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

func TestViewThing(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	prID := prs[0].ID

	ths, err := svc.CreateThings(context.Background(), token, prID, thing)
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

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	prID := prs[0].ID

	thing := things.Thing{
		Name:     "test-meta",
		Key:      key,
		Metadata: metadata,
	}
	ths, err := svc.CreateThings(context.Background(), token, prID, thing)
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

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	prID := prs[0].ID

	data := []thingRes{}
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%s%012d", prefix, i+1)
		thing1 := thing
		thing1.ID = id

		ths, err := svc.CreateThings(context.Background(), token, prID, thing1)
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
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 0, 210),
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

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	prID := prs[0].ID

	data := []thingRes{}
	for i := 0; i < 100; i++ {
		name := "name_" + fmt.Sprintf("%03d", i+1)
		id := fmt.Sprintf("%s%012d", prefix, i+1)

		th := things.Thing{ID: id, Name: name, Metadata: map[string]any{"test": name}}
		ths, err := svc.CreateThings(context.Background(), token, prID, th)
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
			status: http.StatusOK,
			req:    zeroLimitData,
			res:    data[0:10],
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
			req:    invalidLimitData,
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

func TestSearchThingsByProfile(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	prs, err := svc.CreateProfiles(context.Background(), token, gr.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	data := []thingRes{}

	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%s%012d", prefix, i+1)
		thing1 := thing
		thing1.ID = id

		ths, err := svc.CreateThings(context.Background(), token, pr.ID, thing1)
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

	cases := []struct {
		desc   string
		auth   string
		status int
		req    string
		res    []thingRes
	}{
		{
			desc:   "search things by profile",
			auth:   token,
			status: http.StatusOK,
			req:    validData,
			res:    data[0:5],
		},
		{
			desc:   "search things by profile ordered by name ascendant",
			auth:   token,
			status: http.StatusOK,
			req:    ascData,
			res:    data[0:5],
		},
		{
			desc:   "search things by profile ordered by name descendant",
			auth:   token,
			status: http.StatusOK,
			req:    descData,
			res:    data[0:5],
		},
		{
			desc:   "search things by profile with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidOrderData,
			res:    nil,
		},
		{
			desc:   "search things by profile with invalid dir",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidDirData,
			res:    nil,
		},
		{
			desc:   "search things by profile with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search things by profile with invalid data",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidData,
			res:    nil,
		},
		{
			desc:   "search things by profile with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search things by profile with zero limit",
			auth:   token,
			status: http.StatusOK,
			req:    zeroLimitData,
			res:    data[0:10],
		},
		{
			desc:   "search things by profile with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidLimitData,
			res:    nil,
		},
		{
			desc:   "search things by profile filtering with invalid name",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidNameData,
			res:    nil,
		},
		{
			desc:   "search things by profile with no body",
			auth:   token,
			status: http.StatusOK,
			req:    emptyValue,
			res:    data[0:10],
		},
		{
			desc:   "search things by profile with empty body",
			auth:   token,
			status: http.StatusOK,
			req:    emptyJson,
			res:    data[0:10],
		},
	}
	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodPost,
			url:    fmt.Sprintf("%s/profiles/%s/things/search", ts.URL, pr.ID),
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

func TestSearchThingsByGroup(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	prs, err := svc.CreateProfiles(context.Background(), token, gr.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	data := []thingRes{}

	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%s%012d", prefix, i+1)
		thing1 := thing
		thing1.ID = id

		ths, err := svc.CreateThings(context.Background(), token, pr.ID, thing1)
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

	cases := []struct {
		desc   string
		auth   string
		status int
		req    string
		res    []thingRes
	}{
		{
			desc:   "search things by group",
			auth:   token,
			status: http.StatusOK,
			req:    validData,
			res:    data[0:5],
		},
		{
			desc:   "search things by group ordered by name ascendant",
			auth:   token,
			status: http.StatusOK,
			req:    ascData,
			res:    data[0:5],
		},
		{
			desc:   "search things by group ordered by name descendant",
			auth:   token,
			status: http.StatusOK,
			req:    descData,
			res:    data[0:5],
		},
		{
			desc:   "search things by group with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidOrderData,
			res:    nil,
		},
		{
			desc:   "search things by group with invalid dir",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidDirData,
			res:    nil,
		},
		{
			desc:   "search things by group with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search things by group with invalid data",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidData,
			res:    nil,
		},
		{
			desc:   "search things by group with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search things by group with zero limit",
			auth:   token,
			status: http.StatusOK,
			req:    zeroLimitData,
			res:    data[0:10],
		},
		{
			desc:   "search things by group with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidLimitData,
			res:    nil,
		},
		{
			desc:   "search things by group filtering with invalid name",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidNameData,
			res:    nil,
		},
		{
			desc:   "search things by group with no body",
			auth:   token,
			status: http.StatusOK,
			req:    emptyValue,
			res:    data[0:10],
		},
		{
			desc:   "search things by group with empty body",
			auth:   token,
			status: http.StatusOK,
			req:    emptyJson,
			res:    data[0:10],
		},
	}
	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodPost,
			url:    fmt.Sprintf("%s/groups/%s/things/search", ts.URL, gr.ID),
			token:  tc.auth,
			body:   strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var data thingsPageRes
		err = json.NewDecoder(res.Body).Decode(&data)
		require.Nil(t, err, fmt.Sprintf("%s: failed to decode response: %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, data.Things, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, data.Things))
	}
}

func TestSearchThingsByOrg(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	data := []thingRes{}

	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%s%012d", prefix, i+1)
		thing1 := thing
		thing1.ID = id

		ths, err := svc.CreateThings(context.Background(), token, pr.ID, thing1)
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

	cases := []struct {
		desc   string
		auth   string
		status int
		req    string
		res    []thingRes
	}{
		{
			desc:   "search things by organization",
			auth:   token,
			status: http.StatusOK,
			req:    validData,
			res:    data[0:5],
		},
		{
			desc:   "search things by organization ordered by name ascendant",
			auth:   token,
			status: http.StatusOK,
			req:    ascData,
			res:    data[0:5],
		},
		{
			desc:   "search things by organization ordered by name descendant",
			auth:   token,
			status: http.StatusOK,
			req:    descData,
			res:    data[0:5],
		},
		{
			desc:   "search things by organization with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidOrderData,
			res:    nil,
		},
		{
			desc:   "search things by organization with invalid dir",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidDirData,
			res:    nil,
		},
		{
			desc:   "search things by organization with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search things by organization with invalid data",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidData,
			res:    nil,
		},
		{
			desc:   "search things by organization with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search things by organization with zero limit",
			auth:   token,
			status: http.StatusOK,
			req:    zeroLimitData,
			res:    data[0:10],
		},
		{
			desc:   "search things by organization with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidLimitData,
			res:    nil,
		},
		{
			desc:   "search things by organization filtering with invalid name",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidNameData,
			res:    nil,
		},
		{
			desc:   "search things by organization with no body",
			auth:   token,
			status: http.StatusOK,
			req:    emptyValue,
			res:    data[0:10],
		},
		{
			desc:   "search things by organization with empty body",
			auth:   token,
			status: http.StatusOK,
			req:    emptyJson,
			res:    data[0:10],
		},
	}
	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodPost,
			url:    fmt.Sprintf("%s/orgs/%s/things/search", ts.URL, orgID),
			token:  tc.auth,
			body:   strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var data thingsPageRes
		err = json.NewDecoder(res.Body).Decode(&data)
		require.Nil(t, err, fmt.Sprintf("%s: failed to decode response: %s", tc.desc, err))
		assert.ElementsMatch(t, tc.res, data.Things, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, data.Things))
	}
}

func TestListThingsByProfile(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	data := []thingRes{}

	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%s%012d", prefix, i+1)
		thing1 := thing
		thing1.ID = id

		ths, err := svc.CreateThings(context.Background(), token, pr.ID, thing1)
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
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, pr.ID, 0, 210),
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

	grs, err := svc.CreateGroups(context.Background(), adminToken, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grs2, err := svc.CreateGroups(context.Background(), otherToken, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr, gr2 := grs[0], grs2[0]

	prs, err := svc.CreateProfiles(context.Background(), adminToken, gr.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	prs2, err := svc.CreateProfiles(context.Background(), otherToken, gr2.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr, pr2 := prs[0], prs2[0]

	data := []thingRes{}
	for i := 0; i < n; i++ {
		suffix := i + 1
		thing1 := thing
		thing1.Name = fmt.Sprintf("%s%012d", prefix, suffix)
		thing1.ID = fmt.Sprintf("%s%012d", prefix, suffix)

		ths, err := svc.CreateThings(context.Background(), adminToken, pr.ID, thing1)
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

		ths2, err := svc.CreateThings(context.Background(), otherToken, pr2.ID, thing2)
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
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, orgID, 0, 210),
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

func TestUpdateExternalKey(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	createdGroups, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	createdGroup := createdGroups[0]

	createdProfiles, err := svc.CreateProfiles(context.Background(), token, createdGroup.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	createdProfile := createdProfiles[0]

	createdThings, err := svc.CreateThings(context.Background(), token, createdProfile.ID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	createdThing := createdThings[0]

	cases := []struct {
		desc        string
		data        string
		thingID     string
		contentType string
		auth        string
		status      int
	}{
		{
			"update external key",
			`{"key": "abc123"}`,
			createdThing.ID,
			contentTypeJSON,
			token,
			http.StatusCreated,
		},
		{
			"update external key with invalid auth token",
			`{"key": "abc123"}`,
			createdThing.ID,
			contentTypeJSON,
			wrongValue,
			http.StatusUnauthorized,
		},
		{
			"update external key with empty auth token",
			`{"key": "abc123"}`,
			createdThing.ID,
			contentTypeJSON,
			emptyValue,
			http.StatusUnauthorized,
		},
		{
			"update external key as unauthorized user",
			`{"key": "abcd123"}`,
			createdThing.ID,
			contentTypeJSON,
			otherToken,
			http.StatusForbidden,
		},
		{
			"update external key to nonexistent thing",
			`{"key": "abc123"}`,
			"nonexistent",
			contentTypeJSON,
			token,
			http.StatusForbidden,
		},
		{
			"update external key with empty request body",
			emptyValue,
			createdThing.ID,
			contentTypeJSON,
			token,
			http.StatusBadRequest,
		},
		{
			"update external key with empty JSON object in request body",
			emptyJson,
			createdThing.ID,
			contentTypeJSON,
			token,
			http.StatusUnauthorized,
		},
		{
			"update external key without content type",
			`{"key": "abc123"}`,
			createdThing.ID,
			emptyValue,
			token,
			http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/things/%s/external-key", ts.URL, tc.thingID),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.data),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestRemoveExternalKey(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	createdGroups, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	createdGroup := createdGroups[0]

	createdProfiles, err := svc.CreateProfiles(context.Background(), token, createdGroup.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	createdProfile := createdProfiles[0]

	createdThings, err := svc.CreateThings(context.Background(), token, createdProfile.ID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	createdThing := createdThings[0]

	externalKey := "abc123"
	err = svc.UpdateExternalKey(context.Background(), token, externalKey, createdThing.ID)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc   string
		auth   string
		status int
	}{
		{
			"remove external key",
			token,
			http.StatusNoContent,
		},
		{
			"remove external key as unauthorized user",
			"invalid",
			http.StatusUnauthorized,
		},
		{
			"remove external key with empty auth token",
			emptyValue,
			http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/things/%s/external-key", ts.URL, createdThing.ID),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestBackupThingsByGroup(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), otherToken, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	prs, err := svc.CreateProfiles(context.Background(), token, gr.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	data := []viewThingRes{}

	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%s%012d", prefix, i+1)
		thing1 := thing
		thing1.ID = id

		ths, err := svc.CreateThings(context.Background(), token, pr.ID, thing1)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th := ths[0]

		externalKey := fmt.Sprintf("external_key_%d", i+1)
		err = svc.UpdateExternalKey(context.Background(), token, externalKey, th.ID)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		data = append(data, viewThingRes{
			ID:          th.ID,
			GroupID:     th.GroupID,
			ProfileID:   th.ProfileID,
			Name:        th.Name,
			Key:         th.Key,
			ExternalKey: externalKey,
			Metadata:    th.Metadata,
		})
	}

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []viewThingRes
	}{
		{
			desc:   "backup things by group as group owner",
			auth:   token,
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "backup things by group the user belongs to",
			auth:   otherToken,
			status: http.StatusForbidden,
			res:    nil,
		},
		{
			desc:   "backup things by group as admin",
			auth:   adminToken,
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "backup things by group with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "backup things by group with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			res:    nil,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/groups/%s/things/backup", ts.URL, gr.ID),
			token:  tc.auth,
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		var body []viewThingRes

		json.NewDecoder(res.Body).Decode(&body)

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, body, fmt.Sprintf("%s: expected Things %v got %v", tc.desc, tc.res, body))
	}
}

func TestRestoreThingsByGroup(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	idProvider := uuid.New()

	data := []viewThingRes{}

	grs, err := svc.CreateGroups(context.Background(), otherToken, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	prID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	for i := 0; i < n; i++ {
		thId, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		thKey, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		externalKey, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		data = append(data, viewThingRes{
			ID:          thId,
			GroupID:     gr.ID,
			ProfileID:   prID,
			Name:        fmt.Sprintf("thing_%d", i),
			Key:         thKey,
			ExternalKey: externalKey,
			Metadata:    metadata,
		})
	}

	backupBytes, err := json.Marshal(data)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	backupString := string(backupBytes)

	cases := []struct {
		desc        string
		auth        string
		contentType string
		data        string
		status      int
		res         string
	}{
		{
			desc:        "restore things by group as group owner",
			auth:        token,
			data:        backupString,
			contentType: contentTypeOctetStream,
			status:      http.StatusCreated,
			res:         emptyValue,
		},
		{
			desc:        "restore things by group the user belongs to",
			auth:        otherToken,
			data:        backupString,
			contentType: contentTypeOctetStream,
			status:      http.StatusForbidden,
			res:         emptyValue,
		},
		{
			desc:        "restore things by group with invalid token",
			auth:        wrongValue,
			data:        backupString,
			contentType: contentTypeOctetStream,
			status:      http.StatusUnauthorized,
			res:         emptyValue,
		},
		{
			desc:        "restore things by group with empty token",
			auth:        emptyValue,
			data:        backupString,
			contentType: contentTypeOctetStream,
			status:      http.StatusUnauthorized,
			res:         emptyValue,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/groups/%s/things/restore", ts.URL, gr.ID),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.data),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestBackupThingsByOrg(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	prs, err := svc.CreateProfiles(context.Background(), token, gr.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	data := []viewThingRes{}

	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%s%012d", prefix, i+1)
		thing1 := thing
		thing1.ID = id

		ths, err := svc.CreateThings(context.Background(), token, pr.ID, thing1)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th := ths[0]

		externalKey := fmt.Sprintf("external_key_%d", i+1)
		err = svc.UpdateExternalKey(context.Background(), token, externalKey, th.ID)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		data = append(data, viewThingRes{
			ID:          th.ID,
			GroupID:     th.GroupID,
			ProfileID:   th.ProfileID,
			Name:        th.Name,
			Key:         th.Key,
			ExternalKey: externalKey,
			Metadata:    th.Metadata,
		})
	}

	cases := []struct {
		desc   string
		auth   string
		status int
		res    []viewThingRes
	}{
		{
			desc:   "backup things by org as org owner",
			auth:   token,
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "backup things by org the user belongs to",
			auth:   otherToken,
			status: http.StatusForbidden,
			res:    nil,
		},
		{
			desc:   "backup things by org as admin",
			auth:   adminToken,
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "backup things by org with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "backup things by org with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			res:    nil,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/orgs/%s/things/backup", ts.URL, orgID),
			token:  tc.auth,
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		defer res.Body.Close()

		var body []viewThingRes
		json.NewDecoder(res.Body).Decode(&body)

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, body, fmt.Sprintf("%s: expected Things %v got %v", tc.desc, tc.res, body))
	}
}

func TestRestoreThingsByOrg(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	idProvider := uuid.New()

	data := []viewThingRes{}

	grID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	prID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	for i := 0; i < n; i++ {
		thId, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		thKey, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		data = append(data, viewThingRes{
			ID:          thId,
			GroupID:     grID,
			ProfileID:   prID,
			Name:        fmt.Sprintf("thing_%d", i),
			Key:         thKey,
			ExternalKey: fmt.Sprintf("external_key_%d", i),
			Metadata:    metadata,
		})
	}

	dataBytes, err := json.Marshal(data)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	dataString := string(dataBytes)

	thingURL := fmt.Sprintf("%s/orgs", ts.URL)

	cases := []struct {
		desc        string
		auth        string
		contentType string
		data        string
		status      int
		url         string
		res         string
	}{
		{
			desc:        "restore things by org as org owner",
			auth:        token,
			data:        dataString,
			contentType: contentTypeOctetStream,
			status:      http.StatusCreated,
			url:         fmt.Sprintf("%s/%s/things/restore", thingURL, orgID),
			res:         emptyValue,
		},
		{
			desc:        "restore things by org the user belongs to",
			auth:        otherToken,
			data:        dataString,
			contentType: contentTypeOctetStream,
			status:      http.StatusForbidden,
			url:         fmt.Sprintf("%s/%s/things/restore", thingURL, orgID),
			res:         emptyValue,
		},
		{
			desc:        "restore things by org with invalid token",
			auth:        wrongValue,
			data:        dataString,
			contentType: contentTypeOctetStream,
			status:      http.StatusUnauthorized,
			url:         fmt.Sprintf("%s/%s/things/restore", thingURL, orgID),
			res:         emptyValue,
		},
		{
			desc:        "restore things by org with empty token",
			auth:        emptyValue,
			data:        dataString,
			contentType: contentTypeOctetStream,
			status:      http.StatusUnauthorized,
			url:         fmt.Sprintf("%s/%s/things/restore", thingURL, orgID),
			res:         emptyValue,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         tc.url,
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.data),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestRemoveThing(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	ths, err := svc.CreateThings(context.Background(), token, prID, thing)
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

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	ths, err := svc.CreateThings(context.Background(), token, prID, thing)
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
			contentType: contentTypeJSON,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove things with empty token",
			data:        thingIDs,
			auth:        emptyValue,
			contentType: contentTypeJSON,
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
			contentType: contentTypeJSON,
			status:      http.StatusNoContent,
		},
		{
			desc:        "remove non-existent things",
			data:        []string{wrongValue},
			auth:        token,
			contentType: contentTypeJSON,
			status:      http.StatusNotFound,
		},
		{
			desc:        "remove things with empty thing ids",
			data:        []string{emptyValue},
			auth:        token,
			contentType: contentTypeJSON,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove profiles without profile ids",
			data:        []string{},
			auth:        token,
			contentType: contentTypeJSON,
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
		grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		gr := grs[0]

		groups = append(groups, gr)
	}
	gr := groups[0]

	profiles := []things.Profile{}
	for i := 0; i < 10; i++ {
		name := "name_" + fmt.Sprintf("%03d", i+1)
		prs, err := svc.CreateProfiles(context.Background(), token, gr.ID,
			things.Profile{
				Name:     name,
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
		things, err := svc.CreateThings(context.Background(), token, pr.ID,
			things.Thing{
				Name:        name,
				Metadata:    metadata,
				ExternalKey: fmt.Sprintf("external_key_%d", i),
			})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th := things[0]

		ths = append(ths, th)
	}

	var thingsRes []viewThingRes
	for _, th := range ths {
		thingsRes = append(thingsRes, viewThingRes{
			ID:          th.ID,
			GroupID:     th.GroupID,
			ProfileID:   th.ProfileID,
			Name:        th.Name,
			Key:         th.Key,
			ExternalKey: th.ExternalKey,
			Metadata:    th.Metadata,
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
			res:    backup,
		},
		{
			desc:   "backup with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			res:    backupRes{},
		},
		{
			desc:   "backup with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			res:    backupRes{},
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/backup", ts.URL),
			token:  tc.auth,
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		var body backupRes
		err = json.NewDecoder(res.Body).Decode(&body)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

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
		ID:          thId,
		Name:        nameKey,
		Key:         thKey,
		ExternalKey: "abc123",
		Metadata:    metadata,
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
			ID:          testThing.ID,
			Name:        testThing.Name,
			Key:         testThing.Key,
			ExternalKey: testThing.ExternalKey,
			Metadata:    testThing.Metadata,
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

	cases := []struct {
		desc        string
		auth        string
		status      int
		req         string
		contentType string
	}{
		{
			desc:        "restore all things, profiles and groups",
			auth:        adminToken,
			status:      http.StatusCreated,
			req:         data,
			contentType: contentTypeJSON,
		},
		{
			desc:        "restore with invalid token",
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
			req:         data,
			contentType: contentTypeJSON,
		},
		{
			desc:        "restore with empty token",
			auth:        emptyValue,
			status:      http.StatusUnauthorized,
			req:         data,
			contentType: contentTypeJSON,
		},
		{
			desc:        "restore with invalid request",
			auth:        token,
			status:      http.StatusBadRequest,
			req:         invalidData,
			contentType: contentTypeJSON,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/restore", ts.URL),
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

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	ths, err := svc.CreateThings(context.Background(), token, prID, thing)
	require.Nil(t, err, fmt.Sprintf("failed to create thing: %s", err))
	th := ths[0]

	externalKey := "abc123"
	err = svc.UpdateExternalKey(context.Background(), token, externalKey, th.ID)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	irInternal := identifyReq{Key: th.Key, Type: things.KeyTypeInternal}
	dataInternal := toJSON(irInternal)

	irExternal := identifyReq{Key: externalKey, Type: things.KeyTypeExternal}
	dataExternal := toJSON(irExternal)

	nonexistentData := toJSON(identifyReq{Key: wrongValue, Type: "internal"})

	cases := map[string]struct {
		contentType string
		req         string
		status      int
	}{
		"identify thing using internal key": {
			contentType: contentTypeJSON,
			req:         dataInternal,
			status:      http.StatusOK,
		},
		"identify thing using invalid iline key": {
			contentType: contentTypeJSON,
			req:         nonexistentData,
			status:      http.StatusNotFound,
		},
		"identify with missing content type": {
			contentType: wrongValue,
			req:         dataInternal,
			status:      http.StatusUnsupportedMediaType,
		},
		"identify with empty JSON request": {
			contentType: contentTypeJSON,
			req:         "{}",
			status:      http.StatusUnauthorized,
		},
		"identify with invalid JSON request": {
			contentType: contentTypeJSON,
			req:         emptyValue,
			status:      http.StatusBadRequest,
		},
		"identify thing using external key": {
			contentType: contentTypeJSON,
			req:         dataExternal,
			status:      http.StatusOK,
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
	Key  string `json:"key"`
	Type string `json:"type"`
}

type viewMetadataRes struct {
	Metadata map[string]any `json:"metadata,omitempty"`
}

type thingRes struct {
	ID          string         `json:"id"`
	GroupID     string         `json:"group_id,omitempty"`
	ProfileID   string         `json:"profile_id,omitempty"`
	Name        string         `json:"name,omitempty"`
	Key         string         `json:"key"`
	ExternalKey string         `json:"external_key"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type thingsPageRes struct {
	Things []thingRes `json:"things"`
	Total  uint64     `json:"total"`
	Offset uint64     `json:"offset"`
	Limit  uint64     `json:"limit"`
}

type viewThingRes struct {
	ID          string         `json:"id"`
	GroupID     string         `json:"group_id,omitempty"`
	ProfileID   string         `json:"profile_id"`
	Name        string         `json:"name,omitempty"`
	Key         string         `json:"key"`
	ExternalKey string         `json:"external_key,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type backupProfileRes struct {
	ID       string         `json:"id"`
	GroupID  string         `json:"group_id"`
	Name     string         `json:"name,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type viewGroupRes struct {
	ID          string         `json:"id"`
	OrgID       string         `json:"org_id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type backupRes struct {
	Things   []viewThingRes     `json:"things"`
	Profiles []backupProfileRes `json:"profiles"`
	Groups   []viewGroupRes     `json:"groups"`
}

type restoreThingReq struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Key         string         `json:"key"`
	ExternalKey string         `json:"external_key"`
	Metadata    map[string]any `json:"metadata"`
}

type restoreProfileReq struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Metadata map[string]any `json:"metadata"`
}

type restoreGroupReq struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type restoreReq struct {
	Things   []restoreThingReq   `json:"things"`
	Profiles []restoreProfileReq `json:"profiles"`
	Groups   []restoreGroupReq   `json:"groups"`
}
