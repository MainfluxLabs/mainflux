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
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/shadows"
	httpapi "github.com/MainfluxLabs/mainflux/shadows/api/http"
	shmocks "github.com/MainfluxLabs/mainflux/shadows/mocks"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	email       = "admin@example.com"
	token       = email
	emptyValue  = ""
	contentType = "application/json"
	thingID     = "5384fb1c-d0ae-4cbe-be52-c54223150fe0"
	groupID     = "574106f7-030e-4881-8ab0-151195c29f94"
	wrongID     = "wrong-id"
)

var (
	admin = users.User{
		ID:    "874106f7-030e-4881-8ab0-151195c29f97",
		Email: email,
		Role:  auth.RootSub,
	}
	usersList = []users.User{admin}

	desiredState = shadows.State{"led": "on"}
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

type stateRes struct {
	Desired  shadows.State `json:"desired"`
	Reported shadows.State `json:"reported"`
	Delta    shadows.State `json:"delta,omitempty"`
}

type shadowRes struct {
	ThingID    string   `json:"thing_id"`
	State      stateRes `json:"state"`
	ReportedAt int64    `json:"reported_at"`
	UpdatedAt  int64    `json:"updated_at"`
}

func newService() shadows.Service {
	thingsSvc := pkgmocks.NewThingsServiceClient(
		nil,
		map[string]things.Thing{
			token:   {ID: thingID, GroupID: groupID},
			thingID: {ID: thingID, GroupID: groupID},
		},
		map[string]things.Group{token: {ID: groupID}},
	)
	repo := shmocks.NewShadowRepository()
	pub := shmocks.NewCommandPublisher()
	log := logger.NewMock()

	return shadows.New(thingsSvc, repo, pub, log)
}

func newHTTPServer(svc shadows.Service) *httptest.Server {
	ac := pkgmocks.NewAuthService(admin.ID, usersList, nil)
	mux := httpapi.MakeHandler(mocktracer.New(), svc, ac, logger.NewMock())
	return httptest.NewServer(mux)
}

func TestUpdateDesiredState(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	validUpdateBody := toJSON(map[string]any{"desired": desiredState})

	cases := []struct {
		desc        string
		body        string
		thingID     string
		contentType string
		token       string
		status      int
	}{
		{
			desc:        "update desired state with valid request",
			body:        validUpdateBody,
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusOK,
		},
		{
			desc:        "update desired state without content type",
			body:        validUpdateBody,
			thingID:     thingID,
			contentType: emptyValue,
			token:       token,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "update desired state with invalid JSON",
			body:        `}{`,
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update desired state with missing desired state",
			body:        `{}`,
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update desired state with empty token",
			body:        validUpdateBody,
			thingID:     thingID,
			contentType: contentType,
			token:       emptyValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update desired state with wrong thing ID",
			body:        validUpdateBody,
			thingID:     wrongID,
			contentType: contentType,
			token:       token,
			status:      http.StatusForbidden,
		},
		{
			desc:        "update desired state with empty thing ID",
			body:        validUpdateBody,
			thingID:     emptyValue,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/things/%s/shadows", ts.URL, tc.thingID),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))

		if tc.status == http.StatusOK {
			var body shadowRes
			json.NewDecoder(res.Body).Decode(&body)
			assert.Equal(t, tc.thingID, body.ThingID, fmt.Sprintf("%s: expected thing ID %s got %s", tc.desc, tc.thingID, body.ThingID))
			// Nothing has been reported yet, so the desired state is the delta.
			assert.Equal(t, desiredState, body.State.Delta, fmt.Sprintf("%s: expected delta %v got %v", tc.desc, desiredState, body.State.Delta))
		}
	}
}

func TestViewShadow(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	_, err := svc.UpdateDesiredState(context.Background(), token, thingID, desiredState)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc    string
		token   string
		thingID string
		status  int
	}{
		{
			desc:    "view shadow",
			token:   token,
			thingID: thingID,
			status:  http.StatusOK,
		},
		{
			desc:    "view shadow with empty token",
			token:   emptyValue,
			thingID: thingID,
			status:  http.StatusUnauthorized,
		},
		{
			desc:    "view shadow with wrong thing ID",
			token:   token,
			thingID: wrongID,
			status:  http.StatusForbidden,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/things/%s/shadows", ts.URL, tc.thingID),
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))

		if tc.status == http.StatusOK {
			var body shadowRes
			json.NewDecoder(res.Body).Decode(&body)
			assert.Equal(t, tc.thingID, body.ThingID, fmt.Sprintf("%s: expected thing ID %s got %s", tc.desc, tc.thingID, body.ThingID))
			assert.Equal(t, desiredState, body.State.Desired, fmt.Sprintf("%s: expected desired %v got %v", tc.desc, desiredState, body.State.Desired))
		}
	}
}

func TestRemoveShadow(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	_, err := svc.UpdateDesiredState(context.Background(), token, thingID, desiredState)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc    string
		token   string
		thingID string
		status  int
	}{
		{
			desc:    "remove shadow with empty token",
			token:   emptyValue,
			thingID: thingID,
			status:  http.StatusUnauthorized,
		},
		{
			desc:    "remove shadow with wrong thing ID",
			token:   token,
			thingID: wrongID,
			status:  http.StatusForbidden,
		},
		{
			desc:    "remove shadow with valid request",
			token:   token,
			thingID: thingID,
			status:  http.StatusNoContent,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/things/%s/shadows", ts.URL, tc.thingID),
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}
