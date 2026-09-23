// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package groups_test

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

	groupsByToken := map[string]domain.Group{
		editorToken: {ID: groupID},
		viewerToken: {ID: otherGroupID},
	}
	thingsClient := pkgmocks.NewThingsServiceClient(map[string]domain.Profile{}, nil, groupsByToken)

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

type groupConfigRes struct {
	GroupID string         `json:"group_id,omitempty"`
	Config  map[string]any `json:"config,omitempty"`
}

type groupsConfigsRes struct {
	Total         uint64           `json:"total"`
	Offset        uint64           `json:"offset"`
	Limit         uint64           `json:"limit"`
	GroupsConfigs []groupConfigRes `json:"groups_configs"`
}

func TestViewGroupConfig(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	err := svc.UpdateGroupConfig(context.Background(), editorToken, uiconfigs.GroupConfig{GroupID: groupID, Config: uiconfigs.Config{"dashboard_layout": "grid"}})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc   string
		id     string
		token  string
		status int
	}{
		{
			desc:   "view group config with valid token",
			id:     groupID,
			token:  editorToken,
			status: http.StatusOK,
		},
		{
			desc:   "view group config with invalid token",
			id:     groupID,
			token:  wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "view group config with empty token",
			id:     groupID,
			token:  "",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "view group config without access to the group",
			id:     groupID,
			token:  viewerToken,
			status: http.StatusForbidden,
		},
		{
			desc:   "view group config without group id",
			id:     "",
			token:  editorToken,
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/groups/%s/configs", ts.URL, tc.id),
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestUpdateGroupConfig(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	data := toJSON(map[string]any{"config": map[string]any{"dashboard_layout": "grid"}})

	cases := []struct {
		desc   string
		id     string
		req    string
		ct     string
		token  string
		status int
	}{
		{
			desc:   "update group config with valid token",
			id:     groupID,
			req:    data,
			ct:     contentType,
			token:  editorToken,
			status: http.StatusOK,
		},
		{
			desc:   "update group config with invalid token",
			id:     groupID,
			req:    data,
			ct:     contentType,
			token:  wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "update group config without access to the group",
			id:     groupID,
			req:    data,
			ct:     contentType,
			token:  viewerToken,
			status: http.StatusForbidden,
		},
		{
			desc:   "update group config without group id",
			id:     "",
			req:    data,
			ct:     contentType,
			token:  editorToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "update group config with malformed body",
			id:     groupID,
			req:    "{",
			ct:     contentType,
			token:  editorToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "update group config without content type",
			id:     groupID,
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
			url:         fmt.Sprintf("%s/groups/%s/configs", ts.URL, tc.id),
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
		url:    fmt.Sprintf("%s/groups/%s/configs", ts.URL, groupID),
		token:  editorToken,
	}
	res, err := req.make()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var body groupConfigRes
	err = json.NewDecoder(res.Body).Decode(&body)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	assert.Equal(t, "grid", body.Config["dashboard_layout"])
}

func TestListGroupsConfigs(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	err := svc.UpdateGroupConfig(context.Background(), editorToken, uiconfigs.GroupConfig{GroupID: groupID, Config: uiconfigs.Config{"dashboard_layout": "grid"}})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc   string
		token  string
		url    string
		status int
	}{
		{
			desc:   "list group configs with valid token",
			token:  adminToken,
			url:    fmt.Sprintf("%s/groups/configs", ts.URL),
			status: http.StatusOK,
		},
		{
			desc:   "list group configs with invalid token",
			token:  wrongValue,
			url:    fmt.Sprintf("%s/groups/configs", ts.URL),
			status: http.StatusUnauthorized,
		},
		{
			desc:   "list group configs with invalid limit",
			token:  adminToken,
			url:    fmt.Sprintf("%s/groups/configs?limit=%s", ts.URL, "abc"),
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
		url:    fmt.Sprintf("%s/groups/configs", ts.URL),
		token:  adminToken,
	}
	res, err := req.make()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var body groupsConfigsRes
	err = json.NewDecoder(res.Body).Decode(&body)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	assert.Equal(t, uint64(1), body.Total)
}
