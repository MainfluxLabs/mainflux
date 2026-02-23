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
	"github.com/MainfluxLabs/mainflux/downlinks"
	httpapi "github.com/MainfluxLabs/mainflux/downlinks/api/http"
	dlmocks "github.com/MainfluxLabs/mainflux/downlinks/mocks"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/cron"
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	adminEmail  = "admin@example.com"
	token       = adminEmail
	wrongToken  = "wrong-token"
	emptyValue  = ""
	contentType = "application/json"
	thingID     = "5384fb1c-d0ae-4cbe-be52-c54223150fe0"
	groupID     = "574106f7-030e-4881-8ab0-151195c29f94"
	wrongID     = "wrong-id"
	testURL     = "https://example.com"
	invalidURL  = "not-a-url"
)

var (
	adminUser = users.User{
		ID:    "874106f7-030e-4881-8ab0-151195c29f97",
		Email: adminEmail,
		Role:  auth.RootSub,
	}
	usersList = []users.User{adminUser}

	validScheduler   = `"scheduler":{"frequency":"minutely","minute":5,"time_zone":"UTC"}`
	invalidScheduler = `"scheduler":{"frequency":"invalid"}`

	validCreateBody = fmt.Sprintf(`[{"name":"test-downlink","url":"%s/data","method":"GET",%s}]`, testURL, validScheduler)
	validUpdateBody = fmt.Sprintf(`{"name":"updated-downlink","url":"%s/updated","method":"GET",%s}`, testURL, validScheduler)

	testDownlink = downlinks.Downlink{
		Name:   "test-downlink",
		Url:    testURL + "/data",
		Method: "GET",
		Scheduler: cron.Scheduler{
			Frequency: cron.MinutelyFreq,
			Minute:    5,
			TimeZone:  "UTC",
		},
	}
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
		req.Header.Set("Authorization", "Bearer "+tr.token)
	}

	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}

	return tr.client.Do(req)
}

type downlinkRes struct {
	ID         string            `json:"id"`
	GroupID    string            `json:"group_id"`
	ThingID    string            `json:"thing_id"`
	Name       string            `json:"name"`
	Url        string            `json:"url"`
	Method     string            `json:"method"`
	Scheduler  cron.Scheduler    `json:"scheduler"`
	ResHeaders map[string]string `json:"headers"`
}

type downlinksPageRes struct {
	Downlinks []downlinkRes `json:"downlinks"`
	Total     uint64        `json:"total"`
	Offset    uint64        `json:"offset"`
	Limit     uint64        `json:"limit"`
}

func toJSON(data any) string {
	b, _ := json.Marshal(data)
	return string(b)
}

func newService() downlinks.Service {
	authSvc := pkgmocks.NewAuthService(adminUser.ID, usersList, nil)
	thingsSvc := pkgmocks.NewThingsServiceClient(
		nil,
		map[string]things.Thing{
			token:   {ID: thingID, GroupID: groupID},
			thingID: {ID: thingID, GroupID: groupID},
		},
		map[string]things.Group{token: {ID: groupID}},
	)
	repo := dlmocks.NewDownlinkRepository()
	pub := pkgmocks.NewPublisher()
	idp := uuid.NewMock()
	log := logger.NewMock()

	return downlinks.New(thingsSvc, authSvc, pub, repo, idp, log)
}

func newHTTPServer(svc downlinks.Service) *httptest.Server {
	mux := httpapi.MakeHandler(mocktracer.New(), svc, logger.NewMock())
	return httptest.NewServer(mux)
}

func TestCreateDownlinks(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	longName := strings.Repeat("a", 255)
	longParam := strings.Repeat("p", 65)

	withTimeFilter := func(filter string) string {
		return fmt.Sprintf(`[{"name":"test","url":"%s",%s,"time_filter":%s}]`, testURL, validScheduler, filter)
	}

	cases := []struct {
		desc        string
		body        string
		thingID     string
		contentType string
		token       string
		status      int
	}{
		{
			desc:        "create downlinks with valid request",
			body:        validCreateBody,
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusCreated,
		},
		{
			desc:        "create downlinks without content type",
			body:        validCreateBody,
			thingID:     thingID,
			contentType: emptyValue,
			token:       token,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "create downlinks with invalid JSON",
			body:        `}{`,
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create downlinks with empty JSON array",
			body:        `[]`,
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create downlinks with empty name",
			body:        fmt.Sprintf(`[{"name":"","url":"%s",%s}]`, testURL, validScheduler),
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create downlinks with name too long",
			body:        fmt.Sprintf(`[{"name":"%s","url":"%s",%s}]`, longName, testURL, validScheduler),
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create downlinks with invalid URL",
			body:        fmt.Sprintf(`[{"name":"test","url":"%s",%s}]`, invalidURL, validScheduler),
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create downlinks with missing URL",
			body:        fmt.Sprintf(`[{"name":"test","url":"",%s}]`, validScheduler),
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create downlinks with invalid scheduler frequency",
			body:        fmt.Sprintf(`[{"name":"test","url":"%s",%s}]`, testURL, invalidScheduler),
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create downlinks with time filter invalid interval",
			body:        withTimeFilter(`{"start_param":"from","end_param":"to","interval":"invalid","value":5,"format":"unix"}`),
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create downlinks with time filter zero value",
			body:        withTimeFilter(`{"start_param":"from","end_param":"to","interval":"minute","value":0,"format":"unix"}`),
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create downlinks with time filter missing format",
			body:        withTimeFilter(`{"start_param":"from","end_param":"to","interval":"minute","value":5}`),
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create downlinks with time filter param too long",
			body:        withTimeFilter(fmt.Sprintf(`{"start_param":"%s","end_param":"to","interval":"minute","value":5,"format":"unix"}`, longParam)),
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create downlinks with wrong token",
			body:        validCreateBody,
			thingID:     thingID,
			contentType: contentType,
			token:       wrongToken,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "create downlinks with empty token",
			body:        validCreateBody,
			thingID:     thingID,
			contentType: contentType,
			token:       emptyValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "create downlinks with wrong thing ID",
			body:        validCreateBody,
			thingID:     wrongID,
			contentType: contentType,
			token:       token,
			status:      http.StatusForbidden,
		},
		{
			desc:        "create downlinks with empty thing ID",
			body:        validCreateBody,
			thingID:     emptyValue,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/%s/downlinks", ts.URL, tc.thingID),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestListDownlinksByThing(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	dls, err := svc.CreateDownlinks(context.Background(), token, thingID, testDownlink)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	require.Equal(t, 1, len(dls))

	cases := []struct {
		desc    string
		token   string
		thingID string
		url     string
		status  int
		size    int
	}{
		{
			desc:    "list downlinks by thing",
			token:   token,
			thingID: thingID,
			url:     fmt.Sprintf("%s/things/%s/downlinks", ts.URL, thingID),
			status:  http.StatusOK,
			size:    1,
		},
		{
			desc:    "list downlinks by thing with wrong token",
			token:   wrongToken,
			thingID: thingID,
			url:     fmt.Sprintf("%s/things/%s/downlinks", ts.URL, thingID),
			status:  http.StatusUnauthorized,
			size:    0,
		},
		{
			desc:    "list downlinks by thing with empty token",
			token:   emptyValue,
			thingID: thingID,
			url:     fmt.Sprintf("%s/things/%s/downlinks", ts.URL, thingID),
			status:  http.StatusUnauthorized,
			size:    0,
		},
		{
			desc:    "list downlinks by thing with wrong thing ID",
			token:   token,
			thingID: wrongID,
			url:     fmt.Sprintf("%s/things/%s/downlinks", ts.URL, wrongID),
			status:  http.StatusForbidden,
			size:    0,
		},
		{
			desc:    "list downlinks by thing with negative offset",
			token:   token,
			thingID: thingID,
			url:     fmt.Sprintf("%s/things/%s/downlinks?offset=-1&limit=5", ts.URL, thingID),
			status:  http.StatusBadRequest,
			size:    0,
		},
		{
			desc:    "list downlinks by thing with invalid limit",
			token:   token,
			thingID: thingID,
			url:     fmt.Sprintf("%s/things/%s/downlinks?offset=0&limit=abc", ts.URL, thingID),
			status:  http.StatusBadRequest,
			size:    0,
		},
		{
			desc:    "list downlinks by thing with limit exceeding maximum",
			token:   token,
			thingID: thingID,
			url:     fmt.Sprintf("%s/things/%s/downlinks?offset=0&limit=201", ts.URL, thingID),
			status:  http.StatusBadRequest,
			size:    0,
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
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))

		var body downlinksPageRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.size, len(body.Downlinks), fmt.Sprintf("%s: expected %d downlinks got %d", tc.desc, tc.size, len(body.Downlinks)))
	}
}

func TestListDownlinksByGroup(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	_, err := svc.CreateDownlinks(context.Background(), token, thingID, testDownlink)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc    string
		token   string
		groupID string
		url     string
		status  int
		size    int
	}{
		{
			desc:    "list downlinks by group",
			token:   token,
			groupID: groupID,
			url:     fmt.Sprintf("%s/groups/%s/downlinks", ts.URL, groupID),
			status:  http.StatusOK,
			size:    1,
		},
		{
			desc:    "list downlinks by group with wrong token",
			token:   wrongToken,
			groupID: groupID,
			url:     fmt.Sprintf("%s/groups/%s/downlinks", ts.URL, groupID),
			status:  http.StatusUnauthorized,
			size:    0,
		},
		{
			desc:    "list downlinks by group with empty token",
			token:   emptyValue,
			groupID: groupID,
			url:     fmt.Sprintf("%s/groups/%s/downlinks", ts.URL, groupID),
			status:  http.StatusUnauthorized,
			size:    0,
		},
		{
			desc:    "list downlinks by group with wrong group ID",
			token:   token,
			groupID: wrongID,
			url:     fmt.Sprintf("%s/groups/%s/downlinks", ts.URL, wrongID),
			status:  http.StatusForbidden,
			size:    0,
		},
		{
			desc:    "list downlinks by group with negative offset",
			token:   token,
			groupID: groupID,
			url:     fmt.Sprintf("%s/groups/%s/downlinks?offset=-1&limit=5", ts.URL, groupID),
			status:  http.StatusBadRequest,
			size:    0,
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
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))

		var body downlinksPageRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.size, len(body.Downlinks), fmt.Sprintf("%s: expected %d downlinks got %d", tc.desc, tc.size, len(body.Downlinks)))
	}
}

func TestViewDownlink(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	dls, err := svc.CreateDownlinks(context.Background(), token, thingID, testDownlink)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	dlID := dls[0].ID

	cases := []struct {
		desc   string
		token  string
		id     string
		status int
	}{
		{
			desc:   "view downlink",
			token:  token,
			id:     dlID,
			status: http.StatusOK,
		},
		{
			desc:   "view downlink with non-existent ID",
			token:  token,
			id:     wrongID,
			status: http.StatusNotFound,
		},
		{
			desc:   "view downlink with wrong token",
			token:  wrongToken,
			id:     dlID,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "view downlink with empty token",
			token:  emptyValue,
			id:     dlID,
			status: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/downlinks/%s", ts.URL, tc.id),
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))

		if tc.status == http.StatusOK {
			var body downlinkRes
			json.NewDecoder(res.Body).Decode(&body)
			assert.Equal(t, dlID, body.ID, fmt.Sprintf("%s: expected ID %s got %s", tc.desc, dlID, body.ID))
		}
	}
}

func TestUpdateDownlink(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	dls, err := svc.CreateDownlinks(context.Background(), token, thingID, testDownlink)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	dlID := dls[0].ID

	cases := []struct {
		desc        string
		token       string
		id          string
		body        string
		contentType string
		status      int
	}{
		{
			desc:        "update downlink with valid request",
			token:       token,
			id:          dlID,
			body:        validUpdateBody,
			contentType: contentType,
			status:      http.StatusOK,
		},
		{
			desc:        "update downlink without content type",
			token:       token,
			id:          dlID,
			body:        validUpdateBody,
			contentType: emptyValue,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "update downlink with invalid JSON",
			token:       token,
			id:          dlID,
			body:        `}{`,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update downlink with non-existent ID",
			token:       token,
			id:          wrongID,
			body:        validUpdateBody,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update downlink with wrong token",
			token:       wrongToken,
			id:          dlID,
			body:        validUpdateBody,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update downlink with empty token",
			token:       emptyValue,
			id:          dlID,
			body:        validUpdateBody,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update downlink with invalid URL",
			token:       token,
			id:          dlID,
			body:        fmt.Sprintf(`{"name":"test","url":"%s",%s}`, invalidURL, validScheduler),
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update downlink with invalid scheduler",
			token:       token,
			id:          dlID,
			body:        fmt.Sprintf(`{"name":"test","url":"%s",%s}`, testURL, invalidScheduler),
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/downlinks/%s", ts.URL, tc.id),
			token:       tc.token,
			body:        strings.NewReader(tc.body),
			contentType: tc.contentType,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestRemoveDownlinks(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	dls, err := svc.CreateDownlinks(context.Background(), token, thingID, testDownlink)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	dlID := dls[0].ID

	removeBody := toJSON(map[string]any{"downlink_ids": []string{dlID}})

	cases := []struct {
		desc        string
		token       string
		body        string
		contentType string
		status      int
	}{
		{
			desc:        "remove downlinks with non-existent ID",
			token:       token,
			body:        toJSON(map[string]any{"downlink_ids": []string{wrongID}}),
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "remove downlinks without content type",
			token:       token,
			body:        removeBody,
			contentType: emptyValue,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "remove downlinks with invalid JSON",
			token:       token,
			body:        `}{`,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove downlinks with empty ID list",
			token:       token,
			body:        toJSON(map[string]any{"downlink_ids": []string{}}),
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove downlinks with wrong token",
			token:       wrongToken,
			body:        removeBody,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove downlinks with empty token",
			token:       emptyValue,
			body:        removeBody,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove downlinks with valid request",
			token:       token,
			body:        removeBody,
			contentType: contentType,
			status:      http.StatusNoContent,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/downlinks", ts.URL),
			token:       tc.token,
			body:        strings.NewReader(tc.body),
			contentType: tc.contentType,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}
