// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package keys_test

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

	"github.com/MainfluxLabs/mainflux/auth"
	httpapi "github.com/MainfluxLabs/mainflux/auth/api/http"
	"github.com/MainfluxLabs/mainflux/auth/jwt"
	"github.com/MainfluxLabs/mainflux/auth/mocks"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

const (
	secret         = "secret"
	contentType    = "application/json"
	id             = "123e4567-e89b-12d3-a456-000000000001"
	email          = "user@example.com"
	loginDuration  = 30 * time.Minute
	inviteDuration = 7 * 24 * time.Hour
)

type issueRequest struct {
	Duration time.Duration `json:"duration,omitempty"`
	Type     uint32        `json:"type,omitempty"`
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

	req.Header.Set("Referer", "http://localhost")
	return tr.client.Do(req)
}

func newService() auth.Service {
	repo := mocks.NewKeyRepository()
	idProvider := uuid.NewMock()
	t := jwt.New(secret)

	return auth.New(nil, nil, nil, repo, nil, nil, nil, nil, idProvider, t, loginDuration, inviteDuration)
}

func newServer(svc auth.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(svc, mocktracer.New(), logger)
	return httptest.NewServer(mux)
}

func toJSON(data any) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func TestIssue(t *testing.T) {
	svc := newService()
	_, loginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	lk := issueRequest{Type: auth.LoginKey}
	ak := issueRequest{Type: auth.APIKey, Duration: time.Hour}
	rk := issueRequest{Type: auth.RecoveryKey}

	cases := []struct {
		desc   string
		req    string
		ct     string
		token  string
		status int
	}{
		{
			desc:   "issue login key with empty token",
			req:    toJSON(lk),
			ct:     contentType,
			token:  "",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "issue API key",
			req:    toJSON(ak),
			ct:     contentType,
			token:  loginSecret,
			status: http.StatusCreated,
		},
		{
			desc:   "issue recovery key",
			req:    toJSON(rk),
			ct:     contentType,
			token:  loginSecret,
			status: http.StatusCreated,
		},
		{
			desc:   "issue login key wrong content type",
			req:    toJSON(lk),
			ct:     "",
			token:  loginSecret,
			status: http.StatusUnsupportedMediaType,
		},
		{
			desc:   "issue recovery key wrong content type",
			req:    toJSON(rk),
			ct:     "",
			token:  loginSecret,
			status: http.StatusUnsupportedMediaType,
		},
		{
			desc:   "issue key with an invalid token",
			req:    toJSON(ak),
			ct:     contentType,
			token:  "wrong",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "issue recovery key with empty token",
			req:    toJSON(rk),
			ct:     contentType,
			token:  "",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "issue key with invalid request",
			req:    "{",
			ct:     contentType,
			token:  loginSecret,
			status: http.StatusBadRequest,
		},
		{
			desc:   "issue key with invalid JSON",
			req:    "{invalid}",
			ct:     contentType,
			token:  loginSecret,
			status: http.StatusBadRequest,
		},
		{
			desc:   "issue key with invalid JSON content",
			req:    `{"Type":{"key":"value"}}`,
			ct:     contentType,
			token:  loginSecret,
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/keys", ts.URL),
			contentType: tc.ct,
			token:       tc.token,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestRetrieve(t *testing.T) {
	svc := newService()
	_, loginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	key := auth.Key{Type: auth.APIKey, IssuedAt: time.Now(), IssuerID: id, Subject: email}

	k, _, err := svc.Issue(context.Background(), loginSecret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	cases := []struct {
		desc   string
		id     string
		token  string
		status int
	}{
		{
			desc:   "retrieve an existing key",
			id:     k.ID,
			token:  loginSecret,
			status: http.StatusOK,
		},
		{
			desc:   "retrieve a non-existing key",
			id:     "non-existing",
			token:  loginSecret,
			status: http.StatusNotFound,
		},
		{
			desc:   "retrieve a key with an invalid token",
			id:     k.ID,
			token:  "wrong",
			status: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/keys/%s", ts.URL, tc.id),
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestRevoke(t *testing.T) {
	svc := newService()
	_, loginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	key := auth.Key{Type: auth.APIKey, IssuedAt: time.Now(), IssuerID: id, Subject: email}

	k, _, err := svc.Issue(context.Background(), loginSecret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	cases := []struct {
		desc   string
		id     string
		token  string
		status int
	}{
		{
			desc:   "revoke an existing key",
			id:     k.ID,
			token:  loginSecret,
			status: http.StatusNoContent,
		},
		{
			desc:   "revoke a non-existing key",
			id:     "non-existing",
			token:  loginSecret,
			status: http.StatusNoContent,
		},
		{
			desc:   "revoke key with invalid token",
			id:     k.ID,
			token:  "wrong",
			status: http.StatusUnauthorized},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/keys/%s", ts.URL, tc.id),
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestListAPIKeys(t *testing.T) {
	svc := newService()

	// Issue a login key to authenticate list requests.
	_, loginSecret, err := svc.Issue(context.Background(), "", auth.Key{
		Type:     auth.LoginKey,
		IssuedAt: time.Now(),
		IssuerID: id,
		Subject:  email,
	})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	// Issue a couple of API keys for the same issuer.
	apiKey := auth.Key{Type: auth.APIKey, IssuedAt: time.Now(), IssuerID: id, Subject: email}
	_, _, err = svc.Issue(context.Background(), loginSecret, apiKey)
	assert.Nil(t, err, "expected issuing first API key to succeed")
	_, _, err = svc.Issue(context.Background(), loginSecret, apiKey)
	assert.Nil(t, err, "expected issuing second API key to succeed")

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	type listKeysResponse struct {
		Total uint64 `json:"total"`
		Limit uint64 `json:"limit"`
		Keys  []struct {
			ID string `json:"id"`
		} `json:"keys"`
	}

	cases := []struct {
		desc       string
		url        string
		token      string
		status     int
		checkBody  bool
		minTotal   uint64
		expectKeys int
	}{
		{
			desc:       "list API keys with valid token",
			url:        fmt.Sprintf("%s/keys", ts.URL),
			token:      loginSecret,
			status:     http.StatusOK,
			checkBody:  true,
			minTotal:   2,
			expectKeys: 2,
		},
		{
			desc:   "list API keys with empty token",
			url:    fmt.Sprintf("%s/keys", ts.URL),
			token:  "",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "list API keys with invalid token",
			url:    fmt.Sprintf("%s/keys", ts.URL),
			token:  "wrong",
			status: http.StatusUnauthorized,
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

		if tc.checkBody {
			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error reading body %s", tc.desc, err))

			var lr listKeysResponse
			err = json.Unmarshal(body, &lr)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error unmarshaling body %s", tc.desc, err))

			assert.GreaterOrEqual(t, lr.Total, tc.minTotal, fmt.Sprintf("%s: expected total >= %d, got %d", tc.desc, tc.minTotal, lr.Total))
			assert.Equal(t, tc.expectKeys, len(lr.Keys), fmt.Sprintf("%s: expected %d keys, got %d", tc.desc, tc.expectKeys, len(lr.Keys)))
		}
	}
}
