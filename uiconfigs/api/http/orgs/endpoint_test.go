// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package orgs_test

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

	unauthID    = "unauth-id"
	unauthEmail = "unauth@example.com"
	unauthToken = unauthEmail

	orgID = "9325aef3-5a2b-448c-bae1-5d45f86ba2aa"
)

var usersList = []domain.User{
	{ID: adminID, Email: adminEmail},
	{ID: editorID, Email: editorEmail, Role: domain.OrgEditor},
	{ID: viewerID, Email: viewerEmail, Role: domain.OrgViewer},
	{ID: unauthID, Email: unauthEmail},
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
	thingsClient := pkgmocks.NewThingsServiceClient(map[string]domain.Profile{}, nil, nil)

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

type orgConfigRes struct {
	OrgID  string         `json:"org_id,omitempty"`
	Config map[string]any `json:"config,omitempty"`
}

type orgsConfigsRes struct {
	Total       uint64         `json:"total"`
	Offset      uint64         `json:"offset"`
	Limit       uint64         `json:"limit"`
	OrgsConfigs []orgConfigRes `json:"orgs_configs"`
}

func TestViewOrgConfig(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	err := svc.UpdateOrgConfig(context.Background(), editorToken, uiconfigs.OrgConfig{OrgID: orgID, Config: uiconfigs.Config{"theme": "dark"}})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc   string
		id     string
		token  string
		status int
	}{
		{
			desc:   "view org config with valid token",
			id:     orgID,
			token:  editorToken,
			status: http.StatusOK,
		},
		{
			desc:   "view org config with invalid token",
			id:     orgID,
			token:  wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "view org config with empty token",
			id:     orgID,
			token:  "",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "view org config as unauthorized user",
			id:     orgID,
			token:  unauthToken,
			status: http.StatusForbidden,
		},
		{
			desc:   "view org config without org id",
			id:     "",
			token:  editorToken,
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/orgs/%s/configs", ts.URL, tc.id),
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestUpdateOrgConfig(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	data := toJSON(map[string]any{"config": map[string]any{"theme": "dark"}})

	cases := []struct {
		desc   string
		id     string
		req    string
		ct     string
		token  string
		status int
	}{
		{
			desc:   "update org config with valid token",
			id:     orgID,
			req:    data,
			ct:     contentType,
			token:  editorToken,
			status: http.StatusOK,
		},
		{
			desc:   "update org config with invalid token",
			id:     orgID,
			req:    data,
			ct:     contentType,
			token:  wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "update org config with viewer-only role",
			id:     orgID,
			req:    data,
			ct:     contentType,
			token:  viewerToken,
			status: http.StatusForbidden,
		},
		{
			desc:   "update org config without org id",
			id:     "",
			req:    data,
			ct:     contentType,
			token:  editorToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "update org config with malformed body",
			id:     orgID,
			req:    "{",
			ct:     contentType,
			token:  editorToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "update org config without content type",
			id:     orgID,
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
			url:         fmt.Sprintf("%s/orgs/%s/configs", ts.URL, tc.id),
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
		url:    fmt.Sprintf("%s/orgs/%s/configs", ts.URL, orgID),
		token:  editorToken,
	}
	res, err := req.make()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var body orgConfigRes
	err = json.NewDecoder(res.Body).Decode(&body)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	assert.Equal(t, "dark", body.Config["theme"])
}

func TestListOrgsConfigs(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	err := svc.UpdateOrgConfig(context.Background(), editorToken, uiconfigs.OrgConfig{OrgID: orgID, Config: uiconfigs.Config{"theme": "dark"}})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc   string
		token  string
		url    string
		status int
	}{
		{
			desc:   "list org configs with valid token",
			token:  adminToken,
			url:    fmt.Sprintf("%s/orgs/configs", ts.URL),
			status: http.StatusOK,
		},
		{
			desc:   "list org configs with invalid token",
			token:  wrongValue,
			url:    fmt.Sprintf("%s/orgs/configs", ts.URL),
			status: http.StatusUnauthorized,
		},
		{
			desc:   "list org configs with invalid limit",
			token:  adminToken,
			url:    fmt.Sprintf("%s/orgs/configs?limit=%s", ts.URL, "abc"),
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
		url:    fmt.Sprintf("%s/orgs/configs", ts.URL),
		token:  adminToken,
	}
	res, err := req.make()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var body orgsConfigsRes
	err = json.NewDecoder(res.Body).Decode(&body)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	assert.Equal(t, uint64(1), body.Total)
}
