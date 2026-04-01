// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package scripts_test

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
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/rules"
	httpapi "github.com/MainfluxLabs/mainflux/rules/api/http"
	rulesmocks "github.com/MainfluxLabs/mainflux/rules/mocks"
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
	scriptName  = "test-script"
	scriptBody  = "return 1 + 1"
)

type scriptRes struct {
	ID          string `json:"id"`
	GroupID     string `json:"group_id"`
	Name        string `json:"name"`
	Script      string `json:"script,omitempty"`
	Description string `json:"description,omitempty"`
}

type scriptsPageRes struct {
	Total   uint64      `json:"total"`
	Offset  uint64      `json:"offset"`
	Limit   uint64      `json:"limit"`
	Scripts []scriptRes `json:"scripts"`
}

type thingIDsRes struct {
	ThingIDs []string `json:"thing_ids"`
}

type scriptRunRes struct {
	ID         string    `json:"id"`
	ScriptID   string    `json:"script_id"`
	ThingID    string    `json:"thing_id"`
	Logs       []string  `json:"logs"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Status     string    `json:"status"`
	Error      string    `json:"error,omitempty"`
}

type scriptRunsPageRes struct {
	Total  uint64         `json:"total"`
	Offset uint64         `json:"offset"`
	Limit  uint64         `json:"limit"`
	Runs   []scriptRunRes `json:"runs"`
}

func newService() rules.Service {
	ths := pkgmocks.NewThingsServiceClient(
		nil,
		map[string]things.Thing{
			token:   {ID: thingID, GroupID: groupID},
			thingID: {ID: thingID, GroupID: groupID},
		},
		map[string]things.Group{
			token: {ID: groupID},
		},
	)
	rulesRepo := rulesmocks.NewRuleRepository()
	pubsub := rulesmocks.NewPubSub()
	idp := uuid.NewMock()
	log := logger.NewMock()

	return rules.New(rulesRepo, ths, pkgmocks.NewReadersClient(), pubsub, idp, log, true)
}

func newHTTPServer(svc rules.Service) *httptest.Server {
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

func saveScripts(t *testing.T, svc rules.Service, n int) []rules.LuaScript {
	t.Helper()
	var saved []rules.LuaScript
	for i := range n {
		script := rules.LuaScript{
			Name:   fmt.Sprintf("%s-%d", scriptName, i+1),
			Script: fmt.Sprintf("%s -- variant %d", scriptBody, i+1),
		}
		scripts, err := svc.CreateScripts(context.Background(), token, groupID, script)
		require.Nil(t, err, fmt.Sprintf("unexpected error saving script %d: %s", i+1, err))
		saved = append(saved, scripts...)
	}
	return saved
}

func TestCreateScripts(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	validBody := toJSON(map[string]any{
		"scripts": []any{
			map[string]any{"name": scriptName, "script": scriptBody},
		},
	})
	multipleScriptsBody := toJSON(map[string]any{
		"scripts": []any{
			map[string]any{"name": "script-1", "script": "return 1"},
			map[string]any{"name": "script-2", "script": "return 2"},
		},
	})

	cases := []struct {
		desc        string
		token       string
		groupID     string
		contentType string
		body        string
		status      int
		size        int
	}{
		{
			desc:        "create valid script",
			token:       token,
			groupID:     groupID,
			contentType: contentType,
			body:        validBody,
			status:      http.StatusCreated,
			size:        1,
		},
		{
			desc:        "create multiple scripts",
			token:       token,
			groupID:     groupID,
			contentType: contentType,
			body:        multipleScriptsBody,
			status:      http.StatusCreated,
			size:        2,
		},
		{
			desc:        "create script with empty token",
			token:       emptyValue,
			groupID:     groupID,
			contentType: contentType,
			body:        validBody,
			status:      http.StatusUnauthorized,
			size:        0,
		},
		{
			desc:        "create script with wrong token",
			token:       wrongValue,
			groupID:     groupID,
			contentType: contentType,
			body:        validBody,
			status:      http.StatusUnauthorized,
			size:        0,
		},
		{
			desc:        "create script with wrong group ID",
			token:       token,
			groupID:     wrongValue,
			contentType: contentType,
			body:        validBody,
			status:      http.StatusForbidden,
			size:        0,
		},
		{
			desc:        "create script without content type",
			token:       token,
			groupID:     groupID,
			contentType: emptyValue,
			body:        validBody,
			status:      http.StatusUnsupportedMediaType,
			size:        0,
		},
		{
			desc:        "create script with malformed JSON",
			token:       token,
			groupID:     groupID,
			contentType: contentType,
			body:        "}{",
			status:      http.StatusBadRequest,
			size:        0,
		},
		{
			desc:        "create script with empty body",
			token:       token,
			groupID:     groupID,
			contentType: contentType,
			body:        emptyValue,
			status:      http.StatusBadRequest,
			size:        0,
		},
		{
			desc:        "create script with empty scripts array",
			token:       token,
			groupID:     groupID,
			contentType: contentType,
			body:        toJSON(map[string]any{"scripts": []any{}}),
			status:      http.StatusBadRequest,
			size:        0,
		},
		{
			desc:        "create script with missing name",
			token:       token,
			groupID:     groupID,
			contentType: contentType,
			body: toJSON(map[string]any{
				"scripts": []any{map[string]any{"script": scriptBody}},
			}),
			status: http.StatusBadRequest,
			size:   0,
		},
		{
			desc:        "create script with missing script content",
			token:       token,
			groupID:     groupID,
			contentType: contentType,
			body: toJSON(map[string]any{
				"scripts": []any{map[string]any{"name": scriptName}},
			}),
			status: http.StatusBadRequest,
			size:   0,
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
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))

		var body struct {
			Scripts []scriptRes `json:"scripts"`
		}
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.size, len(body.Scripts), fmt.Sprintf("%s: expected %d scripts got %d\n", tc.desc, tc.size, len(body.Scripts)))
	}
}

func TestViewScript(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	saved := saveScripts(t, svc, 1)
	scriptID := saved[0].ID

	cases := []struct {
		desc   string
		token  string
		id     string
		status int
	}{
		{
			desc:   "view existing script",
			token:  token,
			id:     scriptID,
			status: http.StatusOK,
		},
		{
			desc:   "view script with empty token",
			token:  emptyValue,
			id:     scriptID,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "view script with wrong token",
			token:  wrongValue,
			id:     scriptID,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "view script with empty ID",
			token:  token,
			id:     emptyValue,
			status: http.StatusBadRequest,
		},
		{
			desc:   "view non-existing script",
			token:  token,
			id:     wrongValue,
			status: http.StatusNotFound,
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
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
	}
}

func TestListScriptsByGroup(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	n := 10
	saved := saveScripts(t, svc, n)
	_ = saved

	cases := []struct {
		desc   string
		token  string
		url    string
		status int
		size   int
	}{
		{
			desc:   "list scripts by group",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/scripts?limit=%d&offset=0", ts.URL, groupID, n),
			status: http.StatusOK,
			size:   n,
		},
		{
			desc:   "list scripts by group with limit",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/scripts?limit=5&offset=0", ts.URL, groupID),
			status: http.StatusOK,
			size:   5,
		},
		{
			desc:   "list scripts by group with offset",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/scripts?limit=%d&offset=%d", ts.URL, groupID, n, n-1),
			status: http.StatusOK,
			size:   1,
		},
		{
			desc:   "list scripts by group with limit exceeding max",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/scripts?limit=201", ts.URL, groupID),
			status: http.StatusBadRequest,
			size:   0,
		},
		{
			desc:   "list scripts by group with wrong group ID",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/scripts?limit=%d", ts.URL, wrongValue, n),
			status: http.StatusForbidden,
			size:   0,
		},
		{
			desc:   "list scripts by group with empty token",
			token:  emptyValue,
			url:    fmt.Sprintf("%s/groups/%s/scripts?limit=%d", ts.URL, groupID, n),
			status: http.StatusUnauthorized,
			size:   0,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))

		var page scriptsPageRes
		json.NewDecoder(res.Body).Decode(&page)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.size, len(page.Scripts), fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, len(page.Scripts)))
	}
}

func TestListScriptsByThing(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	n := 5
	saved := saveScripts(t, svc, n)
	var scriptIDs []string
	for _, s := range saved {
		scriptIDs = append(scriptIDs, s.ID)
	}
	err := svc.AssignScripts(context.Background(), token, thingID, scriptIDs...)
	require.Nil(t, err, fmt.Sprintf("unexpected error assigning scripts: %s", err))

	cases := []struct {
		desc   string
		token  string
		url    string
		status int
		size   int
	}{
		{
			desc:   "list scripts by thing",
			token:  token,
			url:    fmt.Sprintf("%s/things/%s/scripts?limit=%d&offset=0", ts.URL, thingID, n),
			status: http.StatusOK,
			size:   n,
		},
		{
			desc:   "list scripts by thing with limit",
			token:  token,
			url:    fmt.Sprintf("%s/things/%s/scripts?limit=3&offset=0", ts.URL, thingID),
			status: http.StatusOK,
			size:   3,
		},
		{
			desc:   "list scripts by thing with wrong thing ID",
			token:  token,
			url:    fmt.Sprintf("%s/things/%s/scripts?limit=%d", ts.URL, wrongValue, n),
			status: http.StatusForbidden,
			size:   0,
		},
		{
			desc:   "list scripts by thing with empty token",
			token:  emptyValue,
			url:    fmt.Sprintf("%s/things/%s/scripts?limit=%d", ts.URL, thingID, n),
			status: http.StatusUnauthorized,
			size:   0,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))

		var page scriptsPageRes
		json.NewDecoder(res.Body).Decode(&page)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.size, len(page.Scripts), fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, len(page.Scripts)))
	}
}

func TestListThingIDsByScript(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	saved := saveScripts(t, svc, 1)
	scriptID := saved[0].ID

	err := svc.AssignScripts(context.Background(), token, thingID, scriptID)
	require.Nil(t, err, fmt.Sprintf("unexpected error assigning script: %s", err))

	cases := []struct {
		desc   string
		token  string
		id     string
		status int
		size   int
	}{
		{
			desc:   "list thing IDs by script",
			token:  token,
			id:     scriptID,
			status: http.StatusOK,
			size:   1,
		},
		{
			desc:   "list thing IDs by non-existing script",
			token:  token,
			id:     wrongValue,
			status: http.StatusNotFound,
			size:   0,
		},
		{
			desc:   "list thing IDs by script with empty token",
			token:  emptyValue,
			id:     scriptID,
			status: http.StatusUnauthorized,
			size:   0,
		},
		{
			desc:   "list thing IDs by script with empty script ID",
			token:  token,
			id:     emptyValue,
			status: http.StatusBadRequest,
			size:   0,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/scripts/%s/things", ts.URL, tc.id),
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))

		var body thingIDsRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.size, len(body.ThingIDs), fmt.Sprintf("%s: expected %d thing IDs got %d\n", tc.desc, tc.size, len(body.ThingIDs)))
	}
}

func TestUpdateScript(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	saved := saveScripts(t, svc, 1)
	scriptID := saved[0].ID

	updatedBody := toJSON(map[string]any{
		"name":   "updated-script",
		"script": "return 2 + 2",
	})

	cases := []struct {
		desc        string
		token       string
		id          string
		contentType string
		body        string
		status      int
	}{
		{
			desc:        "update existing script",
			token:       token,
			id:          scriptID,
			contentType: contentType,
			body:        updatedBody,
			status:      http.StatusOK,
		},
		{
			desc:        "update script with empty token",
			token:       emptyValue,
			id:          scriptID,
			contentType: contentType,
			body:        updatedBody,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update script with wrong token",
			token:       wrongValue,
			id:          scriptID,
			contentType: contentType,
			body:        updatedBody,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update non-existing script",
			token:       token,
			id:          wrongValue,
			contentType: contentType,
			body:        updatedBody,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update script without content type",
			token:       token,
			id:          scriptID,
			contentType: emptyValue,
			body:        updatedBody,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "update script with malformed JSON",
			token:       token,
			id:          scriptID,
			contentType: contentType,
			body:        "}{",
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update script with missing name",
			token:       token,
			id:          scriptID,
			contentType: contentType,
			body:        toJSON(map[string]any{"script": "return 1"}),
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update script with missing script content",
			token:       token,
			id:          scriptID,
			contentType: contentType,
			body:        toJSON(map[string]any{"name": "updated"}),
			status:      http.StatusBadRequest,
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
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
	}
}

func TestRemoveScripts(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	saved := saveScripts(t, svc, 1)
	scriptID := saved[0].ID

	cases := []struct {
		desc        string
		token       string
		contentType string
		ids         []string
		status      int
	}{
		{
			desc:        "remove scripts with empty token",
			token:       emptyValue,
			contentType: contentType,
			ids:         []string{scriptID},
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove existing scripts",
			token:       token,
			contentType: contentType,
			ids:         []string{scriptID},
			status:      http.StatusNoContent,
		},
		{
			desc:        "remove non-existing scripts",
			token:       token,
			contentType: contentType,
			ids:         []string{wrongValue},
			status:      http.StatusNotFound,
		},
		{
			desc:        "remove scripts with empty list",
			token:       token,
			contentType: contentType,
			ids:         []string{},
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove scripts without content type",
			token:       token,
			contentType: emptyValue,
			ids:         []string{scriptID},
			status:      http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		body := toJSON(struct {
			ScriptIDs []string `json:"script_ids"`
		}{tc.ids})

		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/scripts", ts.URL),
			token:       tc.token,
			contentType: tc.contentType,
			body:        strings.NewReader(body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
	}
}

func TestAssignScripts(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	saved := saveScripts(t, svc, 2)
	scriptID1 := saved[0].ID
	scriptID2 := saved[1].ID

	cases := []struct {
		desc        string
		token       string
		thingID     string
		contentType string
		ids         []string
		status      int
	}{
		{
			desc:        "assign scripts to thing",
			token:       token,
			thingID:     thingID,
			contentType: contentType,
			ids:         []string{scriptID1, scriptID2},
			status:      http.StatusOK,
		},
		{
			desc:        "assign scripts with empty token",
			token:       emptyValue,
			thingID:     thingID,
			contentType: contentType,
			ids:         []string{scriptID1},
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "assign scripts with wrong token",
			token:       wrongValue,
			thingID:     thingID,
			contentType: contentType,
			ids:         []string{scriptID1},
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "assign scripts to wrong thing ID",
			token:       token,
			thingID:     wrongValue,
			contentType: contentType,
			ids:         []string{scriptID1},
			status:      http.StatusForbidden,
		},
		{
			desc:        "assign scripts without content type",
			token:       token,
			thingID:     thingID,
			contentType: emptyValue,
			ids:         []string{scriptID1},
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "assign scripts with empty list",
			token:       token,
			thingID:     thingID,
			contentType: contentType,
			ids:         []string{},
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		body := toJSON(struct {
			ScriptIDs []string `json:"script_ids"`
		}{tc.ids})

		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/%s/scripts", ts.URL, tc.thingID),
			token:       tc.token,
			contentType: tc.contentType,
			body:        strings.NewReader(body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
	}
}

func TestUnassignScripts(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	saved := saveScripts(t, svc, 2)
	scriptID1 := saved[0].ID
	scriptID2 := saved[1].ID

	err := svc.AssignScripts(context.Background(), token, thingID, scriptID1, scriptID2)
	require.Nil(t, err, fmt.Sprintf("unexpected error assigning scripts: %s", err))

	cases := []struct {
		desc        string
		token       string
		thingID     string
		contentType string
		ids         []string
		status      int
	}{
		{
			desc:        "unassign scripts from thing",
			token:       token,
			thingID:     thingID,
			contentType: contentType,
			ids:         []string{scriptID1, scriptID2},
			status:      http.StatusOK,
		},
		{
			desc:        "unassign scripts with empty token",
			token:       emptyValue,
			thingID:     thingID,
			contentType: contentType,
			ids:         []string{scriptID1},
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "unassign scripts from wrong thing ID",
			token:       token,
			thingID:     wrongValue,
			contentType: contentType,
			ids:         []string{scriptID1},
			status:      http.StatusForbidden,
		},
		{
			desc:        "unassign scripts without content type",
			token:       token,
			thingID:     thingID,
			contentType: emptyValue,
			ids:         []string{scriptID1},
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "unassign scripts with empty list",
			token:       token,
			thingID:     thingID,
			contentType: contentType,
			ids:         []string{},
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		body := toJSON(struct {
			ScriptIDs []string `json:"script_ids"`
		}{tc.ids})

		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/things/%s/scripts", ts.URL, tc.thingID),
			token:       tc.token,
			contentType: tc.contentType,
			body:        strings.NewReader(body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
	}
}

func TestListScriptRunsByThing(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	cases := []struct {
		desc   string
		token  string
		url    string
		status int
		size   int
	}{
		{
			desc:   "list script runs by thing",
			token:  token,
			url:    fmt.Sprintf("%s/things/%s/runs?limit=10&offset=0", ts.URL, thingID),
			status: http.StatusOK,
			size:   0,
		},
		{
			desc:   "list script runs with wrong thing ID",
			token:  token,
			url:    fmt.Sprintf("%s/things/%s/runs?limit=10", ts.URL, wrongValue),
			status: http.StatusForbidden,
			size:   0,
		},
		{
			desc:   "list script runs with empty token",
			token:  emptyValue,
			url:    fmt.Sprintf("%s/things/%s/runs?limit=10", ts.URL, thingID),
			status: http.StatusUnauthorized,
			size:   0,
		},
		{
			desc:   "list script runs with limit exceeding max",
			token:  token,
			url:    fmt.Sprintf("%s/things/%s/runs?limit=201", ts.URL, thingID),
			status: http.StatusBadRequest,
			size:   0,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))

		var page scriptRunsPageRes
		json.NewDecoder(res.Body).Decode(&page)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
		if tc.status == http.StatusOK {
			assert.Equal(t, tc.size, len(page.Runs), fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, len(page.Runs)))
		}
	}
}

func TestRemoveScriptRuns(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	cases := []struct {
		desc        string
		token       string
		contentType string
		ids         []string
		status      int
	}{
		{
			desc:        "remove script runs with empty token",
			token:       emptyValue,
			contentType: contentType,
			ids:         []string{"non-existent-run-id"},
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove non-existing script runs",
			token:       token,
			contentType: contentType,
			ids:         []string{wrongValue},
			status:      http.StatusNotFound,
		},
		{
			desc:        "remove script runs with empty list",
			token:       token,
			contentType: contentType,
			ids:         []string{},
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove script runs without content type",
			token:       token,
			contentType: emptyValue,
			ids:         []string{"run-id"},
			status:      http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		body := toJSON(struct {
			ScriptRunIDs []string `json:"script_run_ids"`
		}{tc.ids})

		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/runs", ts.URL),
			token:       tc.token,
			contentType: tc.contentType,
			body:        strings.NewReader(body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
	}
}
