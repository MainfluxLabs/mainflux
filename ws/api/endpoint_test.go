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

func makeURL(tsURL, profileID, subtopic, thingKey string, header bool) (string, error) {
	u, _ := url.Parse(tsURL)
	u.Scheme = protocol

	if profileID == "0" || profileID == "" {
		if header {
			return fmt.Sprintf("%s/profiles/%s/messages", u, profileID), fmt.Errorf("invalid profile id")
		}
		return fmt.Sprintf("%s/profiles/%s/messages?authorization=%s", u, profileID, thingKey), fmt.Errorf("invalid profile id")
	}

	subtopicPart := ""
	if subtopic != "" {
		subtopicPart = fmt.Sprintf("/%s", subtopic)
	}
	if header {
		return fmt.Sprintf("%s/profiles/%s/messages%s", u, profileID, subtopicPart), nil
	}

	return fmt.Sprintf("%s/profiles/%s/messages%s?authorization=%s", u, profileID, subtopicPart, thingKey), nil
}

func handshake(tsURL, profileID, subtopic, thingKey string, addHeader bool) (*websocket.Conn, *http.Response, error) {
	header := http.Header{}
	if addHeader {
		header.Add("Authorization", thingKey)
	}

	url, _ := makeURL(tsURL, profileID, subtopic, thingKey, addHeader)
	conn, res, errRet := websocket.DefaultDialer.Dial(url, header)

	return conn, res, errRet
}

func TestHandshake(t *testing.T) {
	thingsClient := thmocks.NewThingsServiceClient(map[string]things.Profile{thingKey: {ID: profileID}}, nil, nil)
	svc, _ := newService(thingsClient)
	ts := newHTTPServer(svc)
	defer ts.Close()

	cases := []struct {
		desc      string
		profileID string
		subtopic  string
		header    bool
		thingKey  string
		status    int
		err       error
		msg       []byte
	}{
		{
			desc:      "connect and send message",
			profileID: id,
			subtopic:  "",
			header:    true,
			thingKey:  thingKey,
			status:    http.StatusSwitchingProtocols,
			msg:       msg,
		},
		{
			desc:      "connect and send message with thingKey as query parameter",
			profileID: id,
			subtopic:  "",
			header:    false,
			thingKey:  thingKey,
			status:    http.StatusSwitchingProtocols,
			msg:       msg,
		},
		{
			desc:      "connect and send message that cannot be published",
			profileID: id,
			subtopic:  "",
			header:    true,
			thingKey:  thingKey,
			status:    http.StatusSwitchingProtocols,
			msg:       []byte{},
		},
		{
			desc:      "connect and send message to subtopic",
			profileID: id,
			subtopic:  "subtopic",
			header:    true,
			thingKey:  thingKey,
			status:    http.StatusSwitchingProtocols,
			msg:       msg,
		},
		{
			desc:      "connect and send message to nested subtopic",
			profileID: id,
			subtopic:  "subtopic/nested",
			header:    true,
			thingKey:  thingKey,
			status:    http.StatusSwitchingProtocols,
			msg:       msg,
		},
		{
			desc:      "connect and send message to all subtopics",
			profileID: id,
			subtopic:  ">",
			header:    true,
			thingKey:  thingKey,
			status:    http.StatusSwitchingProtocols,
			msg:       msg,
		},
		{
			desc:      "connect and send message to subtopic without profile",
			profileID: "",
			subtopic:  "subtopic",
			header:    true,
			thingKey:  thingKey,
			status:    http.StatusSwitchingProtocols,
			msg:       msg,
		},
		{
			desc:      "connect with empty thingKey",
			profileID: id,
			subtopic:  "",
			header:    true,
			thingKey:  "",
			status:    http.StatusForbidden,
			msg:       []byte{},
		},
		{
			desc:      "connect and send message to subtopic with invalid name",
			profileID: id,
			subtopic:  "sub/a*b/topic",
			header:    true,
			thingKey:  thingKey,
			status:    http.StatusBadRequest,
			msg:       msg,
		},
	}

	for _, tc := range cases {
		conn, res, err := handshake(ts.URL, tc.profileID, tc.subtopic, tc.thingKey, tc.header)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code '%d' got '%d'\n", tc.desc, tc.status, res.StatusCode))

		if tc.status == http.StatusSwitchingProtocols {
			assert.Nil(t, err, fmt.Sprintf("%s: got unexpected error %s\n", tc.desc, err))

			err = conn.WriteMessage(websocket.TextMessage, tc.msg)
			assert.Nil(t, err, fmt.Sprintf("%s: got unexpected error %s\n", tc.desc, err))
		}
	}
}
