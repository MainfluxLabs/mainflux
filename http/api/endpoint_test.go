// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adapter "github.com/MainfluxLabs/mainflux/http"
	"github.com/MainfluxLabs/mainflux/http/api"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

const ServiceErrToken = "unavailable"

func newService(tc protomfx.ThingsServiceClient) adapter.Service {
	pub := mocks.NewPublisher()
	return adapter.New(pub, tc)
}

func newHTTPServer(svc adapter.Service) *httptest.Server {
	lm := logger.NewMock()
	mux := api.MakeHandler(svc, mocktracer.New(), lm)
	return httptest.NewServer(mux)
}

type testRequest struct {
	client       *http.Client
	method       string
	url          string
	contentType  string
	token        string
	body         io.Reader
	basicAuth    bool
	externalAuth bool
	bearerToken  bool
}

func (tr testRequest) make() (*http.Response, error) {
	req, err := http.NewRequest(tr.method, tr.url, tr.body)
	if err != nil {
		return nil, err
	}

	switch {
	case tr.basicAuth && tr.token != "":
		req.SetBasicAuth("", tr.token)
	case tr.externalAuth && tr.token != "":
		req.Header.Set("Authorization", apiutil.ThingKeyPrefixExternal+tr.token)
	case tr.bearerToken && tr.token != "":
		req.Header.Set("Authorization", apiutil.BearerPrefix+tr.token)
	case tr.token != "":
		req.Header.Set("Authorization", apiutil.ThingKeyPrefixInternal+tr.token)
	}

	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}
	return tr.client.Do(req)
}

func TestPublish(t *testing.T) {
	profileID := "1"
	ctSenmlJSON := "application/senml+json"
	ctSenmlCBOR := "application/senml+cbor"
	ctJSON := "application/json"
	thingKey := "thing_key"
	invalidKey := "invalid"
	msg := `[{"n":"current","t":-1,"v":1.6}]`
	msgJSON := `{"field1":"val1","field2":"val2"}`
	msgCBOR := `81A3616E6763757272656E746174206176FB3FF999999999999A`
	thingsClient := mocks.NewThingsServiceClient(map[string]things.Profile{thingKey: {ID: profileID}}, nil, nil)
	svc := newService(thingsClient)
	ts := newHTTPServer(svc)
	defer ts.Close()

	cases := map[string]struct {
		msg         string
		contentType string
		key         string
		status      int
		basicAuth   bool
	}{
		"publish message": {
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         thingKey,
			status:      http.StatusAccepted,
		},
		"publish message with application/senml+cbor content-type": {
			msg:         msgCBOR,
			contentType: ctSenmlCBOR,
			key:         thingKey,
			status:      http.StatusAccepted,
		},
		"publish message with application/json content-type": {
			msg:         msgJSON,
			contentType: ctJSON,
			key:         thingKey,
			status:      http.StatusAccepted,
		},
		"publish message with empty key": {
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         "",
			status:      http.StatusUnauthorized,
		},
		"publish message with basic auth": {
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         thingKey,
			basicAuth:   true,
			status:      http.StatusAccepted,
		},
		"publish message with invalid key": {
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         invalidKey,
			status:      http.StatusUnauthorized,
		},
		"publish message with invalid basic auth": {
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         invalidKey,
			basicAuth:   true,
			status:      http.StatusUnauthorized,
		},
		"publish message without content type": {
			msg:         msg,
			contentType: "",
			key:         thingKey,
			status:      http.StatusUnsupportedMediaType,
		},
		"publish message unable to authorize": {
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         ServiceErrToken,
			status:      http.StatusInternalServerError,
		},
	}

	for desc, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/messages", ts.URL),
			contentType: tc.contentType,
			token:       tc.key,
			body:        strings.NewReader(tc.msg),
			basicAuth:   tc.basicAuth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", desc, tc.status, res.StatusCode))
	}
}

func TestPublishWithExternalKey(t *testing.T) {
	profileID := "1"
	ctSenmlJSON := "application/senml+json"
	thingKey := "external_thing_key"
	invalidKey := "invalid"
	msg := `[{"n":"current","t":-1,"v":1.6}]`
	thingsClient := mocks.NewThingsServiceClient(map[string]things.Profile{thingKey: {ID: profileID}}, nil, nil)
	svc := newService(thingsClient)
	ts := newHTTPServer(svc)
	defer ts.Close()

	cases := map[string]struct {
		msg         string
		contentType string
		key         string
		status      int
	}{
		"publish message with external key": {
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         thingKey,
			status:      http.StatusAccepted,
		},
		"publish message with invalid external key": {
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         invalidKey,
			status:      http.StatusUnauthorized,
		},
		"publish message with empty external key": {
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         "",
			status:      http.StatusUnauthorized,
		},
	}

	for desc, tc := range cases {
		req := testRequest{
			client:       ts.Client(),
			method:       http.MethodPost,
			url:          fmt.Sprintf("%s/messages", ts.URL),
			contentType:  tc.contentType,
			token:        tc.key,
			body:         strings.NewReader(tc.msg),
			externalAuth: true,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", desc, tc.status, res.StatusCode))
	}
}

func TestPublishSubtopic(t *testing.T) {
	profileID := "1"
	ctSenmlJSON := "application/senml+json"
	thingKey := "thing_key"
	msg := `[{"n":"current","t":-1,"v":1.6}]`
	thingsClient := mocks.NewThingsServiceClient(map[string]things.Profile{thingKey: {ID: profileID}}, nil, nil)
	svc := newService(thingsClient)
	ts := newHTTPServer(svc)
	defer ts.Close()

	cases := map[string]struct {
		msg         string
		contentType string
		key         string
		subtopic    string
		status      int
	}{
		"publish message to a subtopic": {
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         thingKey,
			subtopic:    "temperature",
			status:      http.StatusAccepted,
		},
		"publish message to a nested subtopic": {
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         thingKey,
			subtopic:    "temperature/humidity",
			status:      http.StatusAccepted,
		},
		"publish message to a subtopic with invalid key": {
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         "invalid",
			subtopic:    "temperature",
			status:      http.StatusUnauthorized,
		},
		"publish message to a subtopic without content type": {
			msg:         msg,
			contentType: "",
			key:         thingKey,
			subtopic:    "temperature",
			status:      http.StatusUnsupportedMediaType,
		},
		"publish message to a subtopic with malformed subtopic": {
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         thingKey,
			subtopic:    "subtopic.ab*",
			status:      http.StatusBadRequest,
		},
	}

	for desc, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/messages/%s", ts.URL, tc.subtopic),
			contentType: tc.contentType,
			token:       tc.key,
			body:        strings.NewReader(tc.msg),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", desc, tc.status, res.StatusCode))
	}
}

func TestSendCommandToThing(t *testing.T) {
	ctJSON := "application/json"
	userToken := "user_token"
	thingID := "thing-id-1"
	cmd := `{"command":"reboot","params":{"delay":5}}`

	// CanUserAccessThing uses req.Token as key into the things map.
	thingsClient := mocks.NewThingsServiceClient(
		nil,
		map[string]things.Thing{userToken: {ID: thingID}},
		nil,
	)
	svc := newService(thingsClient)
	ts := newHTTPServer(svc)
	defer ts.Close()

	cases := map[string]struct {
		token       string
		thingID     string
		msg         string
		contentType string
		status      int
	}{
		"send command to thing": {
			token:       userToken,
			thingID:     thingID,
			msg:         cmd,
			contentType: ctJSON,
			status:      http.StatusAccepted,
		},
		"send command to thing without token": {
			token:       "",
			thingID:     thingID,
			msg:         cmd,
			contentType: ctJSON,
			status:      http.StatusUnauthorized,
		},
		"send command to thing with invalid token": {
			token:       "invalid_token",
			thingID:     thingID,
			msg:         cmd,
			contentType: ctJSON,
			status:      http.StatusUnauthorized,
		},
		"send command to thing with wrong thing id": {
			token:       userToken,
			thingID:     "wrong-thing-id",
			msg:         cmd,
			contentType: ctJSON,
			status:      http.StatusForbidden,
		},
		"send command to thing without content type": {
			token:       userToken,
			thingID:     thingID,
			msg:         cmd,
			contentType: "",
			status:      http.StatusUnsupportedMediaType,
		},
	}

	for desc, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/%s/commands", ts.URL, tc.thingID),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.msg),
			bearerToken: true,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", desc, tc.status, res.StatusCode))
	}
}

func TestSendCommandToThingSubtopic(t *testing.T) {
	ctJSON := "application/json"
	userToken := "user_token"
	thingID := "thing-id-1"
	cmd := `{"command":"reboot","params":{"delay":5}}`

	thingsClient := mocks.NewThingsServiceClient(
		nil,
		map[string]things.Thing{userToken: {ID: thingID}},
		nil,
	)
	svc := newService(thingsClient)
	ts := newHTTPServer(svc)
	defer ts.Close()

	cases := map[string]struct {
		token       string
		thingID     string
		subtopic    string
		msg         string
		contentType string
		status      int
	}{
		"send command to thing subtopic": {
			token:       userToken,
			thingID:     thingID,
			subtopic:    "firmware",
			msg:         cmd,
			contentType: ctJSON,
			status:      http.StatusAccepted,
		},
		"send command to thing subtopic with invalid token": {
			token:       "invalid_token",
			thingID:     thingID,
			subtopic:    "firmware",
			msg:         cmd,
			contentType: ctJSON,
			status:      http.StatusUnauthorized,
		},
		"send command to thing subtopic with malformed subtopic": {
			token:       userToken,
			thingID:     thingID,
			subtopic:    "subtopic.ab*",
			msg:         cmd,
			contentType: ctJSON,
			status:      http.StatusBadRequest,
		},
	}

	for desc, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/%s/commands/%s", ts.URL, tc.thingID, tc.subtopic),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.msg),
			bearerToken: true,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", desc, tc.status, res.StatusCode))
	}
}

func TestSendCommandToGroup(t *testing.T) {
	ctJSON := "application/json"
	userToken := "user_token"
	groupID := "group-id-1"
	cmd := `{"command":"reboot","params":{"delay":5}}`

	// CanUserAccessGroup uses req.Token as key into the groups map.
	thingsClient := mocks.NewThingsServiceClient(
		nil,
		nil,
		map[string]things.Group{userToken: {ID: groupID}},
	)
	svc := newService(thingsClient)
	ts := newHTTPServer(svc)
	defer ts.Close()

	cases := map[string]struct {
		token       string
		groupID     string
		msg         string
		contentType string
		status      int
	}{
		"send command to group": {
			token:       userToken,
			groupID:     groupID,
			msg:         cmd,
			contentType: ctJSON,
			status:      http.StatusAccepted,
		},
		"send command to group without token": {
			token:       "",
			groupID:     groupID,
			msg:         cmd,
			contentType: ctJSON,
			status:      http.StatusUnauthorized,
		},
		"send command to group with invalid token": {
			token:       "invalid_token",
			groupID:     groupID,
			msg:         cmd,
			contentType: ctJSON,
			status:      http.StatusUnauthorized,
		},
		"send command to group with wrong group id": {
			token:       userToken,
			groupID:     "wrong-group-id",
			msg:         cmd,
			contentType: ctJSON,
			status:      http.StatusForbidden,
		},
		"send command to group without content type": {
			token:       userToken,
			groupID:     groupID,
			msg:         cmd,
			contentType: "",
			status:      http.StatusUnsupportedMediaType,
		},
	}

	for desc, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/groups/%s/commands", ts.URL, tc.groupID),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.msg),
			bearerToken: true,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", desc, tc.status, res.StatusCode))
	}
}

func TestSendCommandToGroupSubtopic(t *testing.T) {
	ctJSON := "application/json"
	userToken := "user_token"
	groupID := "group-id-1"
	cmd := `{"command":"reboot","params":{"delay":5}}`

	thingsClient := mocks.NewThingsServiceClient(
		nil,
		nil,
		map[string]things.Group{userToken: {ID: groupID}},
	)
	svc := newService(thingsClient)
	ts := newHTTPServer(svc)
	defer ts.Close()

	cases := map[string]struct {
		token       string
		groupID     string
		subtopic    string
		msg         string
		contentType string
		status      int
	}{
		"send command to group subtopic": {
			token:       userToken,
			groupID:     groupID,
			subtopic:    "firmware",
			msg:         cmd,
			contentType: ctJSON,
			status:      http.StatusAccepted,
		},
		"send command to group subtopic with invalid token": {
			token:       "invalid_token",
			groupID:     groupID,
			subtopic:    "firmware",
			msg:         cmd,
			contentType: ctJSON,
			status:      http.StatusUnauthorized,
		},
		"send command to group subtopic with malformed subtopic": {
			token:       userToken,
			groupID:     groupID,
			subtopic:    "subtopic.ab*",
			msg:         cmd,
			contentType: ctJSON,
			status:      http.StatusBadRequest,
		},
	}

	for desc, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/groups/%s/commands/%s", ts.URL, tc.groupID, tc.subtopic),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.msg),
			bearerToken: true,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", desc, tc.status, res.StatusCode))
	}
}
