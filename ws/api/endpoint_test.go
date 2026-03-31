// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/ws"
	"github.com/MainfluxLabs/mainflux/ws/api"
	"github.com/MainfluxLabs/mainflux/ws/mocks"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

const (
	protocol  = "ws"
	thingKey  = "c02ff576-ccd5-40f6-ba5f-c85377aad529"
	thingID   = "513d02d2-16c1-4f23-98be-9e12f8fee898"
	groupID   = "56d9df5e-6d36-11ef-b979-0242ac120002"
	userToken = "user-auth-token"
)

var (
	msg        = []byte(`[{"n":"current","t":-1,"v":1.6}]`)
	thingAuth  = apiutil.ThingKeyPrefixInternal + thingKey
	bearerAuth = apiutil.BearerPrefix + userToken
)

func newService(tc protomfx.ThingsServiceClient) (ws.Service, mocks.MockPubSub) {
	pubsub := mocks.NewPubSub()
	return ws.New(tc, pubsub), pubsub
}

func newHTTPServer(svc ws.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := api.MakeHandler(svc, logger)
	return httptest.NewServer(mux)
}

func messagesHandshake(tsURL, subtopic, auth string) (*websocket.Conn, *http.Response, error) {
	u, err := url.Parse(tsURL)
	if err != nil {
		return nil, nil, err
	}
	u.Scheme = protocol
	u.Path = "/messages"
	if subtopic != "" {
		u.Path += "/" + subtopic
	}

	header := http.Header{}
	if auth != "" {
		header.Set("Authorization", auth)
	}

	return websocket.DefaultDialer.Dial(u.String(), header)
}

func commandsHandshake(tsURL, commandType, id, subtopic, auth string) (*websocket.Conn, *http.Response, error) {
	u, err := url.Parse(tsURL)
	if err != nil {
		return nil, nil, err
	}
	u.Scheme = protocol
	u.Path = fmt.Sprintf("/%s/%s/commands", commandType, id)
	if subtopic != "" {
		u.Path += "/" + subtopic
	}

	header := http.Header{}
	if auth != "" {
		header.Set("Authorization", auth)
	}

	return websocket.DefaultDialer.Dial(u.String(), header)
}

func TestHandshake(t *testing.T) {
	thingsClient := pkgmocks.NewThingsServiceClient(nil, nil, nil)
	svc, _ := newService(thingsClient)
	ts := newHTTPServer(svc)
	defer ts.Close()

	cases := []struct {
		desc     string
		subtopic string
		auth     string
		status   int
		msg      []byte
	}{
		{
			desc:     "connect and send message",
			subtopic: "",
			auth:     thingAuth,
			status:   http.StatusSwitchingProtocols,
			msg:      msg,
		},
		{
			desc:     "connect and send message with thingKey as query parameter",
			subtopic: "",
			auth:     thingAuth,
			status:   http.StatusSwitchingProtocols,
			msg:      msg,
		},
		{
			desc:     "connect and send message that cannot be published",
			subtopic: "",
			auth:     thingAuth,
			status:   http.StatusSwitchingProtocols,
			msg:      []byte{},
		},
		{
			desc:     "connect and send message to subtopic",
			subtopic: "subtopic",
			auth:     thingAuth,
			status:   http.StatusSwitchingProtocols,
			msg:      msg,
		},
		{
			desc:     "connect and send message to nested subtopic",
			subtopic: "subtopic/nested",
			auth:     thingAuth,
			status:   http.StatusSwitchingProtocols,
			msg:      msg,
		},
		{
			desc:     "connect with empty thingKey",
			subtopic: "",
			auth:     "",
			status:   http.StatusUnauthorized,
			msg:      []byte{},
		},
		{
			desc:     "connect with invalid subtopic",
			subtopic: "sub/a*b/topic",
			auth:     thingAuth,
			status:   http.StatusBadRequest,
			msg:      msg,
		},
	}

	for _, tc := range cases {
		conn, res, _ := messagesHandshake(ts.URL, tc.subtopic, tc.auth)
		if res != nil {
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
		}
		if conn != nil {
			err := conn.WriteMessage(websocket.TextMessage, tc.msg)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
			conn.Close()
		}
	}
}

func TestCommandsHandshake(t *testing.T) {
	thingsClient := pkgmocks.NewThingsServiceClient(nil, map[string]things.Thing{
		thingID: {ID: thingID, GroupID: groupID, Type: things.ThingTypeController},
	}, map[string]things.Group{
		groupID: {ID: groupID},
	})
	svc, _ := newService(thingsClient)
	ts := newHTTPServer(svc)
	defer ts.Close()

	cases := []struct {
		desc        string
		commandType string
		id          string
		subtopic    string
		auth        string
		status      int
	}{
		{
			desc:        "connect to thing with thing key",
			commandType: "things",
			id:          thingID,
			auth:        thingAuth,
			status:      http.StatusSwitchingProtocols,
		},
		{
			desc:        "connect to thing with bearer token",
			commandType: "things",
			id:          thingID,
			auth:        bearerAuth,
			status:      http.StatusSwitchingProtocols,
		},
		{
			desc:        "connect to thing with thing key and subtopic",
			commandType: "things",
			id:          thingID,
			subtopic:    "sub/topic",
			auth:        thingAuth,
			status:      http.StatusSwitchingProtocols,
		},
		{
			desc:        "connect to thing without auth",
			commandType: "things",
			id:          thingID,
			auth:        "",
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "connect to thing with invalid subtopic",
			commandType: "things",
			id:          thingID,
			subtopic:    "sub/a*b/topic",
			auth:        thingAuth,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "connect to group with thing key",
			commandType: "groups",
			id:          groupID,
			auth:        thingAuth,
			status:      http.StatusSwitchingProtocols,
		},
		{
			desc:        "connect to group with bearer token",
			commandType: "groups",
			id:          groupID,
			auth:        bearerAuth,
			status:      http.StatusSwitchingProtocols,
		},
		{
			desc:        "connect to group with thing key and subtopic",
			commandType: "groups",
			id:          groupID,
			subtopic:    "sub/topic",
			auth:        thingAuth,
			status:      http.StatusSwitchingProtocols,
		},
		{
			desc:        "connect to group without auth",
			commandType: "groups",
			id:          groupID,
			auth:        "",
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "connect to group with invalid subtopic",
			commandType: "groups",
			id:          groupID,
			subtopic:    "sub/a*b/topic",
			auth:        thingAuth,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		conn, res, _ := commandsHandshake(ts.URL, tc.commandType, tc.id, tc.subtopic, tc.auth)
		if res != nil {
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
		}
		if conn != nil {
			err := conn.WriteMessage(websocket.TextMessage, msg)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
			conn.Close()
		}
	}
}
