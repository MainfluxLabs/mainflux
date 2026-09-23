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
	"strings"
	"testing"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/uiconfigs"
	httpapi "github.com/MainfluxLabs/mainflux/uiconfigs/api/http"
	uimocks "github.com/MainfluxLabs/mainflux/uiconfigs/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	contentType = "application/json"
	wrongValue  = "wrong-value"

	adminID    = "admin-id"
	adminEmail = "admin@example.com"
	adminToken = adminEmail

	editorID    = "editor-id"
	editorEmail = "editor@example.com"
	editorToken = editorEmail

	viewerID    = "viewer-id"
	viewerEmail = "viewer@example.com"
	viewerToken = viewerEmail

	thingID      = "fe6b4e92-cc98-425e-b0aa-000000000001"
	otherThingID = "fe6b4e92-cc98-425e-b0aa-000000000002"

	groupID      = "574106f7-030e-4881-8ab0-151195c29f94"
	otherGroupID = "574106f7-030e-4881-8ab0-151195c29f95"
)

var usersList = []domain.User{
	{ID: adminID, Email: adminEmail},
	{ID: editorID, Email: editorEmail},
	{ID: viewerID, Email: viewerEmail},
}

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

func newService() uiconfigs.Service {
	orgConfigs := uimocks.NewOrgConfigRepository()
	thingConfigs := uimocks.NewThingConfigRepository()
	groupConfigs := uimocks.NewGroupConfigRepository()

	authClient := pkgmocks.NewAuthService(adminID, usersList, nil)

	things := map[string]domain.Thing{
		editorToken:  {ID: thingID, GroupID: groupID},
		thingID:      {ID: thingID, GroupID: groupID},
		viewerToken:  {ID: otherThingID, GroupID: otherGroupID},
		otherThingID: {ID: otherThingID, GroupID: otherGroupID},
	}
	thingsClient := pkgmocks.NewThingsServiceClient(map[string]domain.Profile{}, things, nil)

	return uiconfigs.New(orgConfigs, thingConfigs, groupConfigs, thingsClient, authClient, uuid.NewMock(), logger.NewMock())
}

func newServer(svc uiconfigs.Service) *httptest.Server {
	log := logger.NewMock()
	ac := pkgmocks.NewAuthService(adminID, usersList, nil)
	mux := httpapi.MakeHandler(mocktracer.New(), svc, ac, log)
	return httptest.NewServer(mux)
}

func toJSON(data any) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

type thingConfigRes struct {
	ThingID string         `json:"thing_id,omitempty"`
	GroupID string         `json:"group_id,omitempty"`
	Config  map[string]any `json:"config,omitempty"`
}

type thingsConfigsRes struct {
	Total         uint64           `json:"total"`
	Offset        uint64           `json:"offset"`
	Limit         uint64           `json:"limit"`
	ThingsConfigs []thingConfigRes `json:"things_configs"`
}

func TestViewThingConfig(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	err := svc.UpdateThingConfig(context.Background(), editorToken, uiconfigs.ThingConfig{ThingID: thingID, Config: uiconfigs.Config{"chart": "line"}})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc   string
		id     string
		token  string
		status int
	}{
		{
			desc:   "view thing config with valid token",
			id:     thingID,
			token:  editorToken,
			status: http.StatusOK,
		},
		{
			desc:   "view thing config with invalid token",
			id:     thingID,
			token:  wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "view thing config with empty token",
			id:     thingID,
			token:  "",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "view thing config without access to the thing",
			id:     thingID,
			token:  viewerToken,
			status: http.StatusForbidden,
		},
		{
			desc:   "view thing config without thing id",
			id:     "",
			token:  editorToken,
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/things/%s/configs", ts.URL, tc.id),
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestUpdateThingConfig(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	data := toJSON(map[string]any{"config": map[string]any{"chart": "line"}})

	cases := []struct {
		desc   string
		id     string
		req    string
		ct     string
		token  string
		status int
	}{
		{
			desc:   "update thing config with valid token",
			id:     thingID,
			req:    data,
			ct:     contentType,
			token:  editorToken,
			status: http.StatusOK,
		},
		{
			desc:   "update thing config with invalid token",
			id:     thingID,
			req:    data,
			ct:     contentType,
			token:  wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "update thing config without access to the thing",
			id:     thingID,
			req:    data,
			ct:     contentType,
			token:  viewerToken,
			status: http.StatusForbidden,
		},
		{
			desc:   "update thing config without thing id",
			id:     "",
			req:    data,
			ct:     contentType,
			token:  editorToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "update thing config with malformed body",
			id:     thingID,
			req:    "{",
			ct:     contentType,
			token:  editorToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "update thing config without content type",
			id:     thingID,
			req:    data,
			ct:     "",
			token:  editorToken,
			status: http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/things/%s/configs", ts.URL, tc.id),
			contentType: tc.ct,
			token:       tc.token,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}

	req := testRequest{
		client: client,
		method: http.MethodGet,
		url:    fmt.Sprintf("%s/things/%s/configs", ts.URL, thingID),
		token:  editorToken,
	}
	res, err := req.make()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var body thingConfigRes
	err = json.NewDecoder(res.Body).Decode(&body)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	assert.Equal(t, "line", body.Config["chart"])
}

func TestListThingsConfigs(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	err := svc.UpdateThingConfig(context.Background(), editorToken, uiconfigs.ThingConfig{ThingID: thingID, Config: uiconfigs.Config{"chart": "line"}})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc   string
		token  string
		url    string
		status int
	}{
		{
			desc:   "list thing configs with valid token",
			token:  adminToken,
			url:    fmt.Sprintf("%s/things/configs", ts.URL),
			status: http.StatusOK,
		},
		{
			desc:   "list thing configs with invalid token",
			token:  wrongValue,
			url:    fmt.Sprintf("%s/things/configs", ts.URL),
			status: http.StatusUnauthorized,
		},
		{
			desc:   "list thing configs with invalid limit",
			token:  adminToken,
			url:    fmt.Sprintf("%s/things/configs?limit=%s", ts.URL, "abc"),
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}

	req := testRequest{
		client: client,
		method: http.MethodGet,
		url:    fmt.Sprintf("%s/things/configs", ts.URL),
		token:  adminToken,
	}
	res, err := req.make()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var body thingsConfigsRes
	err = json.NewDecoder(res.Body).Decode(&body)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	assert.Equal(t, uint64(1), body.Total)
}
