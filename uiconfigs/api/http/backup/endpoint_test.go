// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup_test

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
	octetStream = "application/octet-stream"
	wrongValue  = "wrong-value"

	adminID    = "admin-id"
	adminEmail = "admin@example.com"
	adminToken = adminEmail

	editorID    = "editor-id"
	editorEmail = "editor@example.com"
	editorToken = editorEmail

	unauthID    = "unauth-id"
	unauthEmail = "unauth@example.com"
	unauthToken = unauthEmail

	orgID   = "9325aef3-5a2b-448c-bae1-5d45f86ba2aa"
	thingID = "fe6b4e92-cc98-425e-b0aa-000000000001"
	groupID = "574106f7-030e-4881-8ab0-151195c29f94"
)

var usersList = []domain.User{
	{ID: adminID, Email: adminEmail},
	{ID: editorID, Email: editorEmail, Role: domain.OrgEditor},
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

	things := map[string]domain.Thing{
		editorToken: {ID: thingID, GroupID: groupID},
		thingID:     {ID: thingID, GroupID: groupID},
	}
	groupsByToken := map[string]domain.Group{
		editorToken: {ID: groupID},
	}
	thingsClient := pkgmocks.NewThingsServiceClient(map[string]domain.Profile{}, things, groupsByToken)

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

type thingConfigRes struct {
	ThingID string         `json:"thing_id,omitempty"`
	GroupID string         `json:"group_id,omitempty"`
	Config  map[string]any `json:"config,omitempty"`
}

type groupConfigRes struct {
	GroupID string         `json:"group_id,omitempty"`
	Config  map[string]any `json:"config,omitempty"`
}

type backupRes struct {
	OrgsConfigs   []orgConfigRes   `json:"orgs_configs"`
	ThingsConfigs []thingConfigRes `json:"things_configs"`
	GroupsConfigs []groupConfigRes `json:"groups_configs"`
}

func TestBackup(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	err := svc.UpdateOrgConfig(context.Background(), editorToken, uiconfigs.OrgConfig{OrgID: orgID, Config: uiconfigs.Config{"theme": "dark"}})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	err = svc.UpdateThingConfig(context.Background(), editorToken, uiconfigs.ThingConfig{ThingID: thingID, Config: uiconfigs.Config{"chart": "line"}})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	err = svc.UpdateGroupConfig(context.Background(), editorToken, uiconfigs.GroupConfig{GroupID: groupID, Config: uiconfigs.Config{"dashboard_layout": "grid"}})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc   string
		token  string
		status int
		size   int
	}{
		{
			desc:   "backup with admin token",
			token:  adminToken,
			status: http.StatusOK,
			size:   1,
		},
		{
			desc:   "backup as unauthorized user",
			token:  unauthToken,
			status: http.StatusOK,
			size:   0,
		},
		{
			desc:   "backup with invalid token",
			token:  wrongValue,
			status: http.StatusUnauthorized,
			size:   0,
		},
		{
			desc:   "backup with empty token",
			token:  "",
			status: http.StatusUnauthorized,
			size:   0,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/backup", ts.URL),
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

		if tc.status == http.StatusOK {
			var body backupRes
			err = json.NewDecoder(res.Body).Decode(&body)
			require.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Len(t, body.OrgsConfigs, tc.size, tc.desc)
			assert.Len(t, body.ThingsConfigs, tc.size, tc.desc)
			assert.Len(t, body.GroupsConfigs, tc.size, tc.desc)
		}
	}
}

func TestRestore(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	data := toJSON(backupRes{
		OrgsConfigs:   []orgConfigRes{{OrgID: orgID, Config: map[string]any{"theme": "dark"}}},
		ThingsConfigs: []thingConfigRes{{ThingID: thingID, GroupID: groupID, Config: map[string]any{"chart": "line"}}},
		GroupsConfigs: []groupConfigRes{{GroupID: groupID, Config: map[string]any{"dashboard_layout": "grid"}}},
	})

	cases := []struct {
		desc   string
		token  string
		req    string
		ct     string
		status int
	}{
		{
			desc:   "restore with invalid token",
			token:  wrongValue,
			req:    data,
			ct:     octetStream,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "restore with empty token",
			token:  "",
			req:    data,
			ct:     octetStream,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "restore with non-admin token",
			token:  editorToken,
			req:    data,
			ct:     octetStream,
			status: http.StatusForbidden,
		},
		{
			desc:   "restore without content type",
			token:  adminToken,
			req:    data,
			ct:     "",
			status: http.StatusUnsupportedMediaType,
		},
		{
			desc:   "restore with malformed body",
			token:  adminToken,
			req:    "{",
			ct:     octetStream,
			status: http.StatusBadRequest,
		},
		{
			desc:   "restore with empty body",
			token:  adminToken,
			req:    toJSON(backupRes{}),
			ct:     octetStream,
			status: http.StatusBadRequest,
		},
		{
			desc:   "restore with admin token",
			token:  adminToken,
			req:    data,
			ct:     octetStream,
			status: http.StatusCreated,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/restore", ts.URL),
			token:       tc.token,
			contentType: tc.ct,
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
