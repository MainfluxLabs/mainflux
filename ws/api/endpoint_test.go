// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	log "github.com/MainfluxLabs/mainflux/logger"
	thmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/ws"
	"github.com/MainfluxLabs/mainflux/ws/api"
	"github.com/MainfluxLabs/mainflux/ws/mocks"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

const (
	profileID = "30315311-56ba-484d-b500-c1e08305511f"
	id        = "1"
	thingKey  = "c02ff576-ccd5-40f6-ba5f-c85377aad529"
	protocol  = "ws"
)

var msg = []byte(`[{"n":"current","t":-1,"v":1.6}]`)

func newService(tc protomfx.ThingsServiceClient) (ws.Service, mocks.MockPubSub) {
	pubsub := mocks.NewPubSub()
	return ws.New(tc, pubsub), pubsub
}

func newHTTPServer(svc ws.Service) *httptest.Server {
	logger := log.NewMock()
	mux := api.MakeHandler(svc, logger)
	return httptest.NewServer(mux)
}

func handshake(tsURL, subtopic, thingKey string, addHeader bool) (*websocket.Conn, *http.Response, error) {
	header := http.Header{}
	if addHeader && thingKey != "" {
		header.Set("Authorization", thingKey)
	}

	// Construct URL properly
	u, err := url.Parse(tsURL)
	if err != nil {
		return nil, nil, err
	}
	u.Scheme = "ws"
	u.Path = "/messages"
	if subtopic != "" {
		u.Path += "/" + subtopic
	}

	if !addHeader && thingKey != "" {
		q := u.Query()
		q.Set("authorization", thingKey)
		u.RawQuery = q.Encode()
	}

	dialer := websocket.DefaultDialer
	conn, res, err := dialer.Dial(u.String(), header)

	// Return response even on error for inspection
	if err != nil {
		return nil, res, err
	}

	return conn, res, nil
}

func TestHandshake(t *testing.T) {
	thingsClient := thmocks.NewThingsServiceClient(map[string]things.Profile{thingKey: {ID: profileID}}, nil, nil)
	svc, _ := newService(thingsClient)
	ts := newHTTPServer(svc)
	defer ts.Close()

	cases := []struct {
		desc     string
		subtopic string
		header   bool
		thingKey string
		status   int
		err      error
		msg      []byte
	}{
		{
			desc:     "connect and send message",
			subtopic: "",
			header:   true,
			thingKey: thingKey,
			status:   http.StatusSwitchingProtocols,
			msg:      msg,
		},
		{
			desc:     "connect and send message with thingKey as query parameter",
			subtopic: "",
			header:   false,
			thingKey: thingKey,
			status:   http.StatusSwitchingProtocols,
			msg:      msg,
		},
		{
			desc:     "connect and send message that cannot be published",
			subtopic: "",
			header:   true,
			thingKey: thingKey,
			status:   http.StatusSwitchingProtocols,
			msg:      []byte{},
		},
		{
			desc:     "connect and send message to subtopic",
			subtopic: "subtopic",
			header:   true,
			thingKey: thingKey,
			status:   http.StatusSwitchingProtocols,
			msg:      msg,
		},
		{
			desc:     "connect and send message to nested subtopic",
			subtopic: "subtopic/nested",
			header:   true,
			thingKey: thingKey,
			status:   http.StatusSwitchingProtocols,
			msg:      msg,
		},
		{
			desc:     "connect and send message to all subtopics",
			subtopic: ">",
			header:   true,
			thingKey: thingKey,
			status:   http.StatusSwitchingProtocols,
			msg:      msg,
		},
		{
			desc:     "connect with empty thingKey",
			subtopic: "",
			header:   true,
			thingKey: "",
			status:   http.StatusForbidden,
			msg:      []byte{},
		},
		{
			desc:     "connect and send message to subtopic with invalid name",
			subtopic: "sub/a*b/topic",
			header:   true,
			thingKey: thingKey,
			status:   http.StatusBadRequest,
			msg:      msg,
		},
	}

	for _, tc := range cases {
		conn, res, err := handshake(ts.URL, tc.subtopic, tc.thingKey, tc.header)
		if err != nil {
			return
		}
		defer conn.Close()

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code '%d' got '%d'\n", tc.desc, tc.status, res.StatusCode))

		if tc.status == http.StatusSwitchingProtocols {
			assert.Nil(t, err, fmt.Sprintf("%s: got unexpected error %s\n", tc.desc, err))

			err = conn.WriteMessage(websocket.TextMessage, tc.msg)
			assert.Nil(t, err, fmt.Sprintf("%s: got unexpected error %s\n", tc.desc, err))
		}
	}
}
