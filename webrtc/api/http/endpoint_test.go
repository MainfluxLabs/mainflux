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

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/MainfluxLabs/mainflux/webrtc"
	httpapi "github.com/MainfluxLabs/mainflux/webrtc/api/http"
	wmocks "github.com/MainfluxLabs/mainflux/webrtc/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token        = "user@example.com"
	wrongValue   = "wrong-value"
	emptyValue   = ""
	contentType  = "application/json"
	userID       = "574106f7-030e-4881-8ab0-151195c29f94"
	thingID      = "5384fb1c-d0ae-4cbe-be52-c54223150fe0"
	otherThingID = "c3a1a6f0-3d1a-4a39-9f7b-5d3c5b5f0e22"
	publisherSID = "publisher-session"
	offer        = "v=0 offer"
	stunURL      = "stun:stun.example.com:3478"

	validView      = `{"sdp":"v=0 offer"}`
	invalidSDPView = `{"sdp":"invalid"}`
	emptySDPView   = `{"sdp":""}`
	validCandidate = `{"candidate":{"candidate":"candidate:1 1 UDP 1 127.0.0.1 5000 typ host"}}`
	emptyCandidate = `{"candidate":{"candidate":""}}`
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
		req.Header.Set("Authorization", apiutil.BearerPrefix+tr.token)
	}

	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}

	return tr.client.Do(req)
}

func newServer(t *testing.T, publisher bool) (*httptest.Server, string) {
	things := mocks.NewThingsServiceClient(nil, map[string]domain.Thing{token: {ID: thingID}}, nil)
	auth := mocks.NewAuthService("", []users.User{{ID: userID, Email: token}}, nil)
	sfu := wmocks.NewSFU()

	if publisher {
		_, err := sfu.Publish(publisherSID, thingID, offer, func(webrtc.ICECandidate) {})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	svc := webrtc.New(things, auth, sfu, wmocks.NewPublisher(), uuid.NewMock(), webrtc.Config{STUNURLs: []string{stunURL}})

	ts := httptest.NewServer(httpapi.MakeHandler(mocktracer.New(), svc, logger.NewMock()))
	t.Cleanup(ts.Close)

	if !publisher {
		return ts, emptyValue
	}

	req := testRequest{client: ts.Client(), method: http.MethodPost, url: fmt.Sprintf("%s/webrtc/things/%s/view", ts.URL, thingID), contentType: contentType, token: token, body: strings.NewReader(validView)}
	res, err := req.make()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	defer res.Body.Close()

	var body struct {
		SessionID string `json:"session_id"`
	}
	require.Nil(t, json.NewDecoder(res.Body).Decode(&body), "unexpected decode error")

	return ts, body.SessionID
}

func TestViewICEServers(t *testing.T) {
	ts, _ := newServer(t, false)

	cases := []struct {
		desc   string
		token  string
		status int
	}{
		{desc: "view ice servers", token: token, status: http.StatusOK},
		{desc: "view ice servers with invalid token", token: wrongValue, status: http.StatusUnauthorized},
		{desc: "view ice servers with empty token", token: emptyValue, status: http.StatusUnauthorized},
	}

	for _, tc := range cases {
		req := testRequest{client: ts.Client(), method: http.MethodGet, url: ts.URL + "/webrtc/ice-servers", token: tc.token}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

		if tc.status == http.StatusOK {
			var body struct {
				ICEServers []webrtc.ICEServer `json:"ice_servers"`
			}
			assert.Nil(t, json.NewDecoder(res.Body).Decode(&body), fmt.Sprintf("%s: unexpected decode error", tc.desc))
			require.Len(t, body.ICEServers, 1, fmt.Sprintf("%s: expected one ice server", tc.desc))
			assert.Equal(t, []string{stunURL}, body.ICEServers[0].URLs, fmt.Sprintf("%s: unexpected urls", tc.desc))
		}
		res.Body.Close()
	}
}

func TestView(t *testing.T) {
	cases := []struct {
		desc        string
		publisher   bool
		thingID     string
		data        string
		contentType string
		token       string
		status      int
	}{
		{desc: "view stream", publisher: true, thingID: thingID, data: validView, contentType: contentType, token: token, status: http.StatusCreated},
		{desc: "view stream with malformed json", publisher: true, thingID: thingID, data: "}{", contentType: contentType, token: token, status: http.StatusBadRequest},
		{desc: "view stream with empty sdp", publisher: true, thingID: thingID, data: emptySDPView, contentType: contentType, token: token, status: http.StatusBadRequest},
		{desc: "view stream with malformed sdp", publisher: true, thingID: thingID, data: invalidSDPView, contentType: contentType, token: token, status: http.StatusBadRequest},
		{desc: "view stream with invalid token", publisher: true, thingID: thingID, data: validView, contentType: contentType, token: wrongValue, status: http.StatusUnauthorized},
		{desc: "view stream with empty token", publisher: true, thingID: thingID, data: validView, contentType: contentType, token: emptyValue, status: http.StatusUnauthorized},
		{desc: "view stream of thing without access", publisher: true, thingID: otherThingID, data: validView, contentType: contentType, token: token, status: http.StatusForbidden},
		{desc: "view stream of thing without publisher", publisher: false, thingID: thingID, data: validView, contentType: contentType, token: token, status: http.StatusNotFound},
		{desc: "view stream with invalid content type", publisher: true, thingID: thingID, data: validView, contentType: "text/plain", token: token, status: http.StatusUnsupportedMediaType},
	}

	for _, tc := range cases {
		ts, _ := newServer(t, tc.publisher)

		req := testRequest{client: ts.Client(), method: http.MethodPost, url: fmt.Sprintf("%s/webrtc/things/%s/view", ts.URL, tc.thingID), contentType: tc.contentType, token: tc.token, body: strings.NewReader(tc.data)}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

		if tc.status == http.StatusCreated {
			var body webrtc.Session
			assert.Nil(t, json.NewDecoder(res.Body).Decode(&body), fmt.Sprintf("%s: unexpected decode error", tc.desc))
			assert.NotEmpty(t, body.ID, fmt.Sprintf("%s: expected session id", tc.desc))
			assert.NotEmpty(t, body.SDP, fmt.Sprintf("%s: expected sdp answer", tc.desc))
		}
		res.Body.Close()
	}
}

func TestAddViewerCandidate(t *testing.T) {
	ts, sessionID := newServer(t, true)

	cases := []struct {
		desc        string
		sessionID   string
		data        string
		contentType string
		token       string
		status      int
	}{
		{desc: "add candidate", sessionID: sessionID, data: validCandidate, contentType: contentType, token: token, status: http.StatusNoContent},
		{desc: "add empty candidate", sessionID: sessionID, data: emptyCandidate, contentType: contentType, token: token, status: http.StatusBadRequest},
		{desc: "add candidate with malformed json", sessionID: sessionID, data: "}{", contentType: contentType, token: token, status: http.StatusBadRequest},
		{desc: "add candidate with invalid token", sessionID: sessionID, data: validCandidate, contentType: contentType, token: wrongValue, status: http.StatusUnauthorized},
		{desc: "add candidate to unknown session", sessionID: wrongValue, data: validCandidate, contentType: contentType, token: token, status: http.StatusNotFound},
		{desc: "add candidate with invalid content type", sessionID: sessionID, data: validCandidate, contentType: "text/plain", token: token, status: http.StatusUnsupportedMediaType},
	}

	for _, tc := range cases {
		req := testRequest{client: ts.Client(), method: http.MethodPatch, url: fmt.Sprintf("%s/webrtc/sessions/%s/candidates", ts.URL, tc.sessionID), contentType: tc.contentType, token: tc.token, body: strings.NewReader(tc.data)}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		res.Body.Close()
	}
}
