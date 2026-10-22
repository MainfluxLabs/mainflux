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
	"strings"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/ota"
	httpapi "github.com/MainfluxLabs/mainflux/ota/api/http"
	otamocks "github.com/MainfluxLabs/mainflux/ota/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token       = "admin@example.com"
	wrongValue  = "wrong-value"
	contentType = "application/json"
	groupID     = "50e6b371-60ff-45cf-bb52-8200e7cde536"
	thingID     = "5384fb1c-d0ae-4cbe-be52-c54223150fe0"
	thingKey    = "8384fb1c-d0ae-4cbe-be52-c54223150fe2"
	fwVersion   = "1.4.0"
	firmwareURL = "https://localhost/fs/groupfiles"
)

var firmware = ota.Update{
	FileName: "fw-sensor-1.4.0.bin",
	Version:  fwVersion,
	FileSize: 412,
	Checksum: "9f2ca71e",
	Timeout:  time.Hour,
}

func newService() ota.Service {
	things := map[string]domain.Thing{
		token:    {ID: thingID, GroupID: groupID},
		thingKey: {ID: thingID, GroupID: groupID},
	}
	groups := map[string]domain.Group{token: {ID: groupID}}
	tc := mocks.NewThingsServiceClient(nil, things, groups)

	return ota.New(tc, otamocks.NewUpdateRepository(), otamocks.NewCommandPublisher(), uuid.NewMock(), firmwareURL, logger.NewMock())
}

func newHTTPServer(svc ota.Service) *httptest.Server {
	ac := mocks.NewAuthService("", nil, nil)
	mux := httpapi.MakeHandler(mocktracer.New(), svc, ac, logger.NewMock())
	return httptest.NewServer(mux)
}

func createUpdate(t *testing.T, svc ota.Service) ota.Update {
	t.Helper()

	u, err := svc.CreateUpdate(context.Background(), token, groupID, firmware, thingID)
	require.Nil(t, err, fmt.Sprintf("creating update failed: %s", err))

	return u
}

type testRequest struct {
	client      *http.Client
	method      string
	url         string
	contentType string
	token       string
	thingKey    string
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

	if tr.thingKey != "" {
		req.Header.Set("Authorization", apiutil.ThingKeyPrefixInternal+tr.thingKey)
	}

	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}

	return tr.client.Do(req)
}

func TestCreateUpdate(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	validData := fmt.Sprintf(`{"file_name":"fw.bin","version":"%s","file_size":412,"checksum":"9f2ca71e","timeout":3600,"thing_ids":["%s"]}`, fwVersion, thingID)

	cases := []struct {
		desc        string
		data        string
		contentType string
		token       string
		status      int
	}{
		{
			desc:        "create update",
			data:        validData,
			contentType: contentType,
			token:       token,
			status:      http.StatusCreated,
		},
		{
			desc:        "create update with invalid token",
			data:        validData,
			contentType: contentType,
			token:       wrongValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "create update with empty token",
			data:        validData,
			contentType: contentType,
			token:       "",
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "create update without file name",
			data:        fmt.Sprintf(`{"version":"%s","thing_ids":["%s"]}`, fwVersion, thingID),
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create update without version",
			data:        fmt.Sprintf(`{"file_name":"fw.bin","thing_ids":["%s"]}`, thingID),
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create update with negative timeout",
			data:        fmt.Sprintf(`{"file_name":"fw.bin","version":"%s","timeout":-1,"thing_ids":["%s"]}`, fwVersion, thingID),
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create update without things",
			data:        fmt.Sprintf(`{"file_name":"fw.bin","version":"%s","thing_ids":[]}`, fwVersion),
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create update with malformed data",
			data:        `{"file_name":`,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create update without content type",
			data:        validData,
			contentType: "",
			token:       token,
			status:      http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/groups/%s/updates", ts.URL, groupID),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.data),
		}
		res, err := req.make()
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		res.Body.Close()
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestViewUpdate(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()
	u := createUpdate(t, svc)

	cases := []struct {
		desc   string
		id     string
		token  string
		status int
	}{
		{
			desc:   "view update",
			id:     u.ID,
			token:  token,
			status: http.StatusOK,
		},
		{
			desc:   "view update with invalid token",
			id:     u.ID,
			token:  wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "view non-existing update",
			id:     wrongValue,
			token:  token,
			status: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/updates/%s", ts.URL, tc.id),
			token:  tc.token,
		}
		res, err := req.make()
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		if tc.status == http.StatusOK {
			var body map[string]any
			require.Nil(t, json.NewDecoder(res.Body).Decode(&body), fmt.Sprintf("%s: invalid response body", tc.desc))
			assert.Equal(t, fwVersion, body["version"], fmt.Sprintf("%s: version mismatch", tc.desc))
			assert.Equal(t, float64(3600), body["timeout"], fmt.Sprintf("%s: timeout must be in seconds", tc.desc))
		}
		res.Body.Close()
	}
}

func TestListUpdates(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()
	for range 3 {
		createUpdate(t, svc)
	}

	cases := []struct {
		desc   string
		query  string
		token  string
		status int
		size   int
	}{
		{
			desc:   "list updates",
			query:  "",
			token:  token,
			status: http.StatusOK,
			size:   3,
		},
		{
			desc:   "list updates with limit",
			query:  "?limit=2",
			token:  token,
			status: http.StatusOK,
			size:   2,
		},
		{
			desc:   "list updates with invalid order",
			query:  "?order=wrong",
			token:  token,
			status: http.StatusBadRequest,
		},
		{
			desc:   "list updates with invalid token",
			query:  "",
			token:  wrongValue,
			status: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/groups/%s/updates%s", ts.URL, groupID, tc.query),
			token:  tc.token,
		}
		res, err := req.make()
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		if tc.status == http.StatusOK {
			var body struct {
				Updates []map[string]any `json:"updates"`
			}
			require.Nil(t, json.NewDecoder(res.Body).Decode(&body), fmt.Sprintf("%s: invalid response body", tc.desc))
			assert.Len(t, body.Updates, tc.size, fmt.Sprintf("%s: size mismatch", tc.desc))
		}
		res.Body.Close()
	}
}

func TestListThingUpdates(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()
	u := createUpdate(t, svc)

	cases := []struct {
		desc   string
		query  string
		status int
		size   int
	}{
		{
			desc:   "list thing updates",
			query:  "",
			status: http.StatusOK,
			size:   1,
		},
		{
			desc:   "list pending thing updates",
			query:  "?status=pending",
			status: http.StatusOK,
			size:   1,
		},
		{
			desc:   "list failed thing updates",
			query:  "?status=failed",
			status: http.StatusOK,
			size:   0,
		},
		{
			desc:   "list thing updates with invalid status",
			query:  "?status=wrong",
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/updates/%s/things%s", ts.URL, u.ID, tc.query),
			token:  token,
		}
		res, err := req.make()
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		if tc.status == http.StatusOK {
			var body struct {
				ThingUpdates []map[string]any `json:"thing_updates"`
			}
			require.Nil(t, json.NewDecoder(res.Body).Decode(&body), fmt.Sprintf("%s: invalid response body", tc.desc))
			assert.Len(t, body.ThingUpdates, tc.size, fmt.Sprintf("%s: size mismatch", tc.desc))
		}
		res.Body.Close()
	}
}

func TestRetryFailed(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()
	u := createUpdate(t, svc)

	cases := []struct {
		desc   string
		id     string
		token  string
		status int
	}{
		{
			desc:   "retry failed things",
			id:     u.ID,
			token:  token,
			status: http.StatusOK,
		},
		{
			desc:   "retry with invalid token",
			id:     u.ID,
			token:  wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "retry non-existing update",
			id:     wrongValue,
			token:  token,
			status: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodPost,
			url:    fmt.Sprintf("%s/updates/%s/retry", ts.URL, tc.id),
			token:  tc.token,
		}
		res, err := req.make()
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		res.Body.Close()
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestRemoveUpdate(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()
	u := createUpdate(t, svc)

	cases := []struct {
		desc   string
		id     string
		token  string
		status int
	}{
		{
			desc:   "remove update with invalid token",
			id:     u.ID,
			token:  wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove update",
			id:     u.ID,
			token:  token,
			status: http.StatusNoContent,
		},
		{
			desc:   "remove removed update",
			id:     u.ID,
			token:  token,
			status: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/updates/%s", ts.URL, tc.id),
			token:  tc.token,
		}
		res, err := req.make()
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		res.Body.Close()
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestViewPending(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	pending := func(key string) *http.Response {
		req := testRequest{
			client:   ts.Client(),
			method:   http.MethodGet,
			url:      fmt.Sprintf("%s/ota/pending", ts.URL),
			thingKey: key,
		}
		res, err := req.make()
		require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
		return res
	}

	res := pending(thingKey)
	res.Body.Close()
	assert.Equal(t, http.StatusNoContent, res.StatusCode, "thing without update must get no content")

	u := createUpdate(t, svc)

	res = pending(thingKey)
	assert.Equal(t, http.StatusOK, res.StatusCode, "thing with update must get it")
	var body map[string]any
	require.Nil(t, json.NewDecoder(res.Body).Decode(&body), "invalid response body")
	res.Body.Close()
	assert.Equal(t, u.ID, body["update_id"], "update id mismatch")
	assert.Equal(t, firmware.FileName, body["file_name"], "file name mismatch")
	assert.Equal(t, firmwareURL+"/"+firmware.FileName, body["url"], "url mismatch")

	res = pending(wrongValue)
	res.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "invalid key must be rejected")
}

func TestReportStatus(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()
	u := createUpdate(t, svc)

	cases := []struct {
		desc        string
		data        string
		contentType string
		key         string
		status      int
	}{
		{
			desc:        "report downloading",
			data:        fmt.Sprintf(`{"update_id":"%s","status":"downloading"}`, u.ID),
			contentType: contentType,
			key:         thingKey,
			status:      http.StatusOK,
		},
		{
			desc:        "report invalid status",
			data:        fmt.Sprintf(`{"update_id":"%s","status":"wrong"}`, u.ID),
			contentType: contentType,
			key:         thingKey,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "report without update id",
			data:        `{"status":"downloading"}`,
			contentType: contentType,
			key:         thingKey,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "report with invalid key",
			data:        fmt.Sprintf(`{"update_id":"%s","status":"downloading"}`, u.ID),
			contentType: contentType,
			key:         wrongValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "report for unknown update",
			data:        fmt.Sprintf(`{"update_id":"%s","status":"downloading"}`, wrongValue),
			contentType: contentType,
			key:         thingKey,
			status:      http.StatusNotFound,
		},
		{
			desc:        "report without content type",
			data:        fmt.Sprintf(`{"update_id":"%s","status":"downloading"}`, u.ID),
			contentType: "",
			key:         thingKey,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "report succeeded",
			data:        fmt.Sprintf(`{"update_id":"%s","status":"succeeded","version":"%s"}`, u.ID, fwVersion),
			contentType: contentType,
			key:         thingKey,
			status:      http.StatusOK,
		},
		{
			desc:        "report after update finished",
			data:        fmt.Sprintf(`{"update_id":"%s","status":"failed"}`, u.ID),
			contentType: contentType,
			key:         thingKey,
			status:      http.StatusConflict,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/ota/status", ts.URL),
			contentType: tc.contentType,
			thingKey:    tc.key,
			body:        strings.NewReader(tc.data),
		}
		res, err := req.make()
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		res.Body.Close()
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}
