// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	httpapi "github.com/MainfluxLabs/mainflux/consumers/alarms/api/http"
	alarmmocks "github.com/MainfluxLabs/mainflux/consumers/alarms/mocks"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token       = "admin@example.com"
	wrongValue  = "wrong-value"
	emptyValue  = ""
	contentType = "application/json"
	thingID     = "5384fb1c-d0ae-4cbe-be52-c54223150fe0"
	groupID     = "574106f7-030e-4881-8ab0-151195c29f94"
	orgID       = "7e3d5e48-b0b4-4d7b-9d6a-c81f40e30e2c"
	ruleID      = "5384fb1c-d0ae-4cbe-be52-c54223150fe1"
	subtopic    = "sensors"
	protocol    = "mqtt"
)

type alarmRes struct {
	ID       string         `json:"id"`
	ThingID  string         `json:"thing_id"`
	GroupID  string         `json:"group_id"`
	RuleID   string         `json:"rule_id"`
	Subtopic string         `json:"subtopic"`
	Protocol string         `json:"protocol"`
	Payload  map[string]any `json:"payload"`
	Created  int64          `json:"created"`
}

type alarmsPageRes struct {
	Total  uint64      `json:"total"`
	Offset uint64      `json:"offset"`
	Limit  uint64      `json:"limit"`
	Alarms []alarmRes  `json:"alarms"`
}

func newService() alarms.Service {
	ths := pkgmocks.NewThingsServiceClient(
		nil,
		map[string]things.Thing{
			token:   {ID: thingID, GroupID: groupID},
			thingID: {ID: thingID, GroupID: groupID},
		},
		map[string]things.Group{
			token: {ID: groupID, OrgID: orgID},
		},
	)
	alarmRepo := alarmmocks.NewAlarmRepository()
	idp := uuid.NewMock()

	return alarms.New(ths, alarmRepo, idp)
}

func newHTTPServer(svc alarms.Service) *httptest.Server {
	log := logger.NewMock()
	mux := httpapi.MakeHandler(mocktracer.New(), svc, log)
	return httptest.NewServer(mux)
}

func toJSON(data any) string {
	b, _ := json.Marshal(data)
	return string(b)
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
	if tr.token != emptyValue {
		req.Header.Set("Authorization", apiutil.BearerPrefix+tr.token)
	}
	if tr.contentType != emptyValue {
		req.Header.Set("Content-Type", tr.contentType)
	}
	return tr.client.Do(req)
}

func saveAlarms(t *testing.T, svc alarms.Service, n int) {
	t.Helper()

	pyd, err := json.Marshal(map[string]any{"temperature": float64(30)})
	require.Nil(t, err)

	for i := 0; i < n; i++ {
		msg := protomfx.Message{
			Publisher: thingID,
			Subject:   fmt.Sprintf("alarms.%s", ruleID),
			Subtopic:  subtopic,
			Protocol:  protocol,
			Payload:   pyd,
			Created:   int64(1000000 + i),
		}
		err := svc.Consume(msg)
		require.Nil(t, err, fmt.Sprintf("unexpected error saving alarm %d: %s", i+1, err))
	}
}

func TestListAlarmsByGroup(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	n := 10
	saveAlarms(t, svc, n)

	cases := []struct {
		desc   string
		auth   string
		url    string
		status int
		size   int
	}{
		{
			desc:   "list alarms by group",
			auth:   token,
			url:    fmt.Sprintf("%s/groups/%s/alarms?limit=%d&offset=0", ts.URL, groupID, n),
			status: http.StatusOK,
			size:   n,
		},
		{
			desc:   "list alarms by group with default limit",
			auth:   token,
			url:    fmt.Sprintf("%s/groups/%s/alarms", ts.URL, groupID),
			status: http.StatusOK,
			size:   n,
		},
		{
			desc:   "list alarms by group with limit",
			auth:   token,
			url:    fmt.Sprintf("%s/groups/%s/alarms?limit=5&offset=0", ts.URL, groupID),
			status: http.StatusOK,
			size:   5,
		},
		{
			desc:   "list alarms by group with offset",
			auth:   token,
			url:    fmt.Sprintf("%s/groups/%s/alarms?limit=%d&offset=%d", ts.URL, groupID, n, n-1),
			status: http.StatusOK,
			size:   1,
		},
		{
			desc:   "list alarms by group with limit exceeding max",
			auth:   token,
			url:    fmt.Sprintf("%s/groups/%s/alarms?limit=201", ts.URL, groupID),
			status: http.StatusBadRequest,
			size:   0,
		},
		{
			desc:   "list alarms by group with invalid limit",
			auth:   token,
			url:    fmt.Sprintf("%s/groups/%s/alarms?limit=invalid", ts.URL, groupID),
			status: http.StatusBadRequest,
			size:   0,
		},
		{
			desc:   "list alarms by group with invalid offset",
			auth:   token,
			url:    fmt.Sprintf("%s/groups/%s/alarms?offset=invalid", ts.URL, groupID),
			status: http.StatusBadRequest,
			size:   0,
		},
		{
			desc:   "list alarms by group with wrong group ID",
			auth:   token,
			url:    fmt.Sprintf("%s/groups/%s/alarms?limit=%d", ts.URL, wrongValue, n),
			status: http.StatusForbidden,
			size:   0,
		},
		{
			desc:   "list alarms by group with empty token",
			auth:   emptyValue,
			url:    fmt.Sprintf("%s/groups/%s/alarms?limit=%d", ts.URL, groupID, n),
			status: http.StatusUnauthorized,
			size:   0,
		},
		{
			desc:   "list alarms by group with wrong token",
			auth:   wrongValue,
			url:    fmt.Sprintf("%s/groups/%s/alarms?limit=%d", ts.URL, groupID, n),
			status: http.StatusUnauthorized,
			size:   0,
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
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))

		var page alarmsPageRes
		json.NewDecoder(res.Body).Decode(&page)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.size, len(page.Alarms), fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, len(page.Alarms)))
	}
}

func TestListAlarmsByThing(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	n := 10
	saveAlarms(t, svc, n)

	cases := []struct {
		desc   string
		auth   string
		url    string
		status int
		size   int
	}{
		{
			desc:   "list alarms by thing",
			auth:   token,
			url:    fmt.Sprintf("%s/things/%s/alarms?limit=%d&offset=0", ts.URL, thingID, n),
			status: http.StatusOK,
			size:   n,
		},
		{
			desc:   "list alarms by thing with default limit",
			auth:   token,
			url:    fmt.Sprintf("%s/things/%s/alarms", ts.URL, thingID),
			status: http.StatusOK,
			size:   n,
		},
		{
			desc:   "list alarms by thing with limit",
			auth:   token,
			url:    fmt.Sprintf("%s/things/%s/alarms?limit=5&offset=0", ts.URL, thingID),
			status: http.StatusOK,
			size:   5,
		},
		{
			desc:   "list alarms by thing with offset",
			auth:   token,
			url:    fmt.Sprintf("%s/things/%s/alarms?limit=%d&offset=%d", ts.URL, thingID, n, n-1),
			status: http.StatusOK,
			size:   1,
		},
		{
			desc:   "list alarms by thing with limit exceeding max",
			auth:   token,
			url:    fmt.Sprintf("%s/things/%s/alarms?limit=201", ts.URL, thingID),
			status: http.StatusBadRequest,
			size:   0,
		},
		{
			desc:   "list alarms by thing with invalid limit",
			auth:   token,
			url:    fmt.Sprintf("%s/things/%s/alarms?limit=invalid", ts.URL, thingID),
			status: http.StatusBadRequest,
			size:   0,
		},
		{
			desc:   "list alarms by thing with wrong thing ID",
			auth:   token,
			url:    fmt.Sprintf("%s/things/%s/alarms?limit=%d", ts.URL, wrongValue, n),
			status: http.StatusForbidden,
			size:   0,
		},
		{
			desc:   "list alarms by thing with empty token",
			auth:   emptyValue,
			url:    fmt.Sprintf("%s/things/%s/alarms?limit=%d", ts.URL, thingID, n),
			status: http.StatusUnauthorized,
			size:   0,
		},
		{
			desc:   "list alarms by thing with wrong token",
			auth:   wrongValue,
			url:    fmt.Sprintf("%s/things/%s/alarms?limit=%d", ts.URL, thingID, n),
			status: http.StatusUnauthorized,
			size:   0,
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
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))

		var page alarmsPageRes
		json.NewDecoder(res.Body).Decode(&page)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.size, len(page.Alarms), fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, len(page.Alarms)))
	}
}

func TestListAlarmsByOrg(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	n := 5
	saveAlarms(t, svc, n)

	cases := []struct {
		desc   string
		auth   string
		url    string
		status int
		size   int
	}{
		{
			desc:   "list alarms by org",
			auth:   token,
			url:    fmt.Sprintf("%s/orgs/%s/alarms?limit=%d&offset=0", ts.URL, orgID, n),
			status: http.StatusOK,
			size:   n,
		},
		{
			desc:   "list alarms by org with wrong org ID",
			auth:   token,
			url:    fmt.Sprintf("%s/orgs/%s/alarms?limit=%d", ts.URL, wrongValue, n),
			status: http.StatusOK,
			size:   0,
		},
		{
			desc:   "list alarms by org with limit exceeding max",
			auth:   token,
			url:    fmt.Sprintf("%s/orgs/%s/alarms?limit=201", ts.URL, orgID),
			status: http.StatusBadRequest,
			size:   0,
		},
		{
			desc:   "list alarms by org with empty token",
			auth:   emptyValue,
			url:    fmt.Sprintf("%s/orgs/%s/alarms?limit=%d", ts.URL, orgID, n),
			status: http.StatusUnauthorized,
			size:   0,
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
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))

		var page alarmsPageRes
		json.NewDecoder(res.Body).Decode(&page)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.size, len(page.Alarms), fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, len(page.Alarms)))
	}
}

func TestViewAlarm(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	saveAlarms(t, svc, 1)
	alarmID := fmt.Sprintf("%s%012d", uuid.Prefix, 1)

	cases := []struct {
		desc   string
		auth   string
		id     string
		status int
	}{
		{
			desc:   "view existing alarm",
			auth:   token,
			id:     alarmID,
			status: http.StatusOK,
		},
		{
			desc:   "view alarm with empty token",
			auth:   emptyValue,
			id:     alarmID,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "view alarm with wrong token",
			auth:   wrongValue,
			id:     alarmID,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "view alarm with empty ID",
			auth:   token,
			id:     emptyValue,
			status: http.StatusBadRequest,
		},
		{
			desc:   "view non-existing alarm",
			auth:   token,
			id:     wrongValue,
			status: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/alarms/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
	}
}

func TestRemoveAlarms(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	saveAlarms(t, svc, 1)
	alarmID := fmt.Sprintf("%s%012d", uuid.Prefix, 1)

	cases := []struct {
		desc        string
		auth        string
		contentType string
		ids         []string
		status      int
	}{
		{
			desc:        "remove alarms with empty token",
			auth:        emptyValue,
			contentType: contentType,
			ids:         []string{alarmID},
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove existing alarms",
			auth:        token,
			contentType: contentType,
			ids:         []string{alarmID},
			status:      http.StatusNoContent,
		},
		{
			desc:        "remove non-existing alarms",
			auth:        token,
			contentType: contentType,
			ids:         []string{wrongValue},
			status:      http.StatusNotFound,
		},
		{
			desc:        "remove alarms with empty list",
			auth:        token,
			contentType: contentType,
			ids:         []string{},
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove alarms without content type",
			auth:        token,
			contentType: emptyValue,
			ids:         []string{alarmID},
			status:      http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		body := toJSON(struct {
			AlarmIDs []string `json:"alarm_ids"`
		}{tc.ids})

		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/alarms", ts.URL),
			token:       tc.auth,
			contentType: tc.contentType,
			body:        strings.NewReader(body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
	}
}

func TestExportAlarmsByThing(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	saveAlarms(t, svc, 3)

	cases := []struct {
		desc   string
		auth   string
		url    string
		status int
	}{
		{
			desc:   "export alarms as JSON",
			auth:   token,
			url:    fmt.Sprintf("%s/things/%s/alarms/export?convert=json&limit=10", ts.URL, thingID),
			status: http.StatusOK,
		},
		{
			desc:   "export alarms as CSV",
			auth:   token,
			url:    fmt.Sprintf("%s/things/%s/alarms/export?convert=csv&limit=10", ts.URL, thingID),
			status: http.StatusOK,
		},
		{
			desc:   "export alarms with invalid format",
			auth:   token,
			url:    fmt.Sprintf("%s/things/%s/alarms/export?convert=xml&limit=10", ts.URL, thingID),
			status: http.StatusBadRequest,
		},
		{
			desc:   "export alarms with empty token",
			auth:   emptyValue,
			url:    fmt.Sprintf("%s/things/%s/alarms/export?convert=json&limit=10", ts.URL, thingID),
			status: http.StatusUnauthorized,
		},
		{
			desc:   "export alarms with wrong token",
			auth:   wrongValue,
			url:    fmt.Sprintf("%s/things/%s/alarms/export?convert=json&limit=10", ts.URL, thingID),
			status: http.StatusUnauthorized,
		},
		{
			desc:   "export alarms with wrong thing ID",
			auth:   token,
			url:    fmt.Sprintf("%s/things/%s/alarms/export?convert=json&limit=10", ts.URL, wrongValue),
			status: http.StatusForbidden,
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
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
	}
}
