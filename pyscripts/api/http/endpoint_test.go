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

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/pyscripts"
	httpapi "github.com/MainfluxLabs/mainflux/pyscripts/api/http"
	psmocks "github.com/MainfluxLabs/mainflux/pyscripts/mocks"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	email         = "admin@example.com"
	token         = email
	emptyValue    = ""
	contentType   = "application/json"
	groupID       = "574106f7-030e-4881-8ab0-151195c29f94"
	wrongID       = "wrong-id"
	validSource   = "1 + 1"
	maxScriptSize = 1024
)

var (
	admin = users.User{
		ID:    "874106f7-030e-4881-8ab0-151195c29f97",
		Email: email,
		Role:  auth.RootSub,
	}
	usersList = []users.User{admin}
)

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

	if tr.token != "" {
		req.Header.Set("Authorization", apiutil.BearerPrefix+tr.token)
	}

	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}

	return tr.client.Do(req)
}

func newService() pyscripts.Service {
	things := pkgmocks.NewThingsServiceClient(
		nil,
		map[string]domain.Thing{token: {ID: "thing", GroupID: groupID}},
		map[string]domain.Group{token: {ID: groupID}},
	)
	repo := psmocks.NewScriptRepository()
	run := psmocks.NewRunner()
	idp := uuid.NewMock()

	return pyscripts.New(repo, run, things, idp, logger.NewMock(), maxScriptSize)
}

func newHTTPServer(svc pyscripts.Service) *httptest.Server {
	ac := pkgmocks.NewAuthService(admin.ID, usersList, nil)
	mux := httpapi.MakeHandler(mocktracer.New(), svc, ac, logger.NewMock())
	return httptest.NewServer(mux)
}

func createScript(t *testing.T, svc pyscripts.Service, name string) pyscripts.Script {
	saved, err := svc.CreateScripts(context.Background(), token, groupID, pyscripts.Script{Name: name, Source: validSource})
	require.Nil(t, err, fmt.Sprintf("unexpected error creating fixture script: %s", err))
	return saved[0]
}

func TestCreateScripts(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	valid := toJSON(map[string]any{"scripts": []map[string]any{{"name": "a", "source": validSource}}})

	cases := []struct {
		desc        string
		body        string
		groupID     string
		contentType string
		token       string
		status      int
	}{
		{
			desc:        "create scripts with valid request",
			body:        valid,
			groupID:     groupID,
			contentType: contentType,
			token:       token,
			status:      http.StatusCreated,
		},
		{
			desc:        "create scripts without content type",
			body:        valid,
			groupID:     groupID,
			contentType: emptyValue,
			token:       token,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "create scripts with invalid JSON",
			body:        `}{`,
			groupID:     groupID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create scripts with empty list",
			body:        toJSON(map[string]any{"scripts": []map[string]any{}}),
			groupID:     groupID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create scripts with a missing name",
			body:        toJSON(map[string]any{"scripts": []map[string]any{{"source": validSource}}}),
			groupID:     groupID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create scripts with empty token",
			body:        valid,
			groupID:     groupID,
			contentType: contentType,
			token:       emptyValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "create scripts for group the user cannot access",
			body:        valid,
			groupID:     wrongID,
			contentType: contentType,
			token:       token,
			status:      http.StatusForbidden,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/groups/%s/scripts", ts.URL, tc.groupID),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.body),
		}
		res, err := req.make()
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))
		res.Body.Close()
	}
}

func TestListScriptsByGroup(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	createScript(t, svc, "a")

	cases := []struct {
		desc    string
		groupID string
		token   string
		status  int
	}{
		{
			desc:    "list scripts with valid request",
			groupID: groupID,
			token:   token,
			status:  http.StatusOK,
		},
		{
			desc:    "list scripts with empty token",
			groupID: groupID,
			token:   emptyValue,
			status:  http.StatusUnauthorized,
		},
		{
			desc:    "list scripts for group the user cannot access",
			groupID: wrongID,
			token:   token,
			status:  http.StatusForbidden,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/groups/%s/scripts", ts.URL, tc.groupID),
			token:  tc.token,
		}
		res, err := req.make()
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))
		res.Body.Close()
	}
}

func TestViewScript(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	s := createScript(t, svc, "a")

	cases := []struct {
		desc   string
		id     string
		token  string
		status int
	}{
		{
			desc:   "view script with valid request",
			id:     s.ID,
			token:  token,
			status: http.StatusOK,
		},
		{
			desc:   "view non-existent script",
			id:     wrongID,
			token:  token,
			status: http.StatusNotFound,
		},
		{
			desc:   "view script with empty token",
			id:     s.ID,
			token:  emptyValue,
			status: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/scripts/%s", ts.URL, tc.id),
			token:  tc.token,
		}
		res, err := req.make()
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))
		res.Body.Close()
	}
}

func TestUpdateScript(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	s := createScript(t, svc, "a")
	valid := toJSON(map[string]any{"name": "renamed", "source": "2 + 2"})

	cases := []struct {
		desc        string
		id          string
		body        string
		contentType string
		token       string
		status      int
	}{
		{
			desc:        "update script with valid request",
			id:          s.ID,
			body:        valid,
			contentType: contentType,
			token:       token,
			status:      http.StatusOK,
		},
		{
			desc:        "update script without content type",
			id:          s.ID,
			body:        valid,
			contentType: emptyValue,
			token:       token,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "update non-existent script",
			id:          wrongID,
			body:        valid,
			contentType: contentType,
			token:       token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update script with empty token",
			id:          s.ID,
			body:        valid,
			contentType: contentType,
			token:       emptyValue,
			status:      http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/scripts/%s", ts.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.body),
		}
		res, err := req.make()
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))
		res.Body.Close()
	}
}

func TestRemoveScripts(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	s := createScript(t, svc, "a")

	cases := []struct {
		desc        string
		body        string
		contentType string
		token       string
		status      int
	}{
		{
			desc:        "remove scripts with empty token",
			body:        toJSON(map[string]any{"script_ids": []string{s.ID}}),
			contentType: contentType,
			token:       emptyValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove scripts with empty list",
			body:        toJSON(map[string]any{"script_ids": []string{}}),
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove scripts with valid request",
			body:        toJSON(map[string]any{"script_ids": []string{s.ID}}),
			contentType: contentType,
			token:       token,
			status:      http.StatusNoContent,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/scripts", ts.URL),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.body),
		}
		res, err := req.make()
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))
		res.Body.Close()
	}
}

func TestRunScript(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	s := createScript(t, svc, "a")
	valid := toJSON(map[string]any{"payload": map[string]any{"x": 1}, "thing_id": "thing-1"})

	cases := []struct {
		desc        string
		id          string
		body        string
		contentType string
		token       string
		status      int
	}{
		{
			desc:        "run script with valid request",
			id:          s.ID,
			body:        valid,
			contentType: contentType,
			token:       token,
			status:      http.StatusOK,
		},
		{
			desc:        "run script without content type",
			id:          s.ID,
			body:        valid,
			contentType: emptyValue,
			token:       token,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "run script with oversized payload",
			id:          s.ID,
			body:        toJSON(map[string]any{"payload": map[string]any{"blob": strings.Repeat("a", (1<<20)+1)}}),
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "run non-existent script",
			id:          wrongID,
			body:        valid,
			contentType: contentType,
			token:       token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "run script with empty token",
			id:          s.ID,
			body:        valid,
			contentType: contentType,
			token:       emptyValue,
			status:      http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/scripts/%s/run", ts.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.body),
		}
		res, err := req.make()
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))
		res.Body.Close()
	}
}

func TestListRunsByScript(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	s := createScript(t, svc, "a")
	_, err := svc.RunScript(context.Background(), token, s.ID, pyscripts.Input{})
	require.Nil(t, err, fmt.Sprintf("unexpected run error: %s", err))

	cases := []struct {
		desc   string
		id     string
		token  string
		status int
	}{
		{
			desc:   "list runs with valid request",
			id:     s.ID,
			token:  token,
			status: http.StatusOK,
		},
		{
			desc:   "list runs for non-existent script",
			id:     wrongID,
			token:  token,
			status: http.StatusNotFound,
		},
		{
			desc:   "list runs with empty token",
			id:     s.ID,
			token:  emptyValue,
			status: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/scripts/%s/runs", ts.URL, tc.id),
			token:  tc.token,
		}
		res, err := req.make()
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))
		res.Body.Close()
	}
}

func TestRemoveRuns(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	s := createScript(t, svc, "a")
	run, err := svc.RunScript(context.Background(), token, s.ID, pyscripts.Input{})
	require.Nil(t, err, fmt.Sprintf("unexpected run error: %s", err))

	cases := []struct {
		desc        string
		body        string
		contentType string
		token       string
		status      int
	}{
		{
			desc:        "remove runs with empty token",
			body:        toJSON(map[string]any{"run_ids": []string{run.ID}}),
			contentType: contentType,
			token:       emptyValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove runs with empty list",
			body:        toJSON(map[string]any{"run_ids": []string{}}),
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove runs with valid request",
			body:        toJSON(map[string]any{"run_ids": []string{run.ID}}),
			contentType: contentType,
			token:       token,
			status:      http.StatusNoContent,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/runs", ts.URL),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.body),
		}
		res, err := req.make()
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))
		res.Body.Close()
	}
}
