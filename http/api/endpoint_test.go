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
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

const ServiceErrToken = "unavailable"

func newService(tc protomfx.ThingsServiceClient) adapter.Service {
	pub := mocks.NewPublisher()
	return adapter.New(pub, tc)
}

func newHTTPServer(svc adapter.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := api.MakeHandler(svc, mocktracer.New(), logger)
	return httptest.NewServer(mux)
}

type testRequest struct {
	client      *http.Client
	method      string
	url         string
	contentType string
	token       string
	body        io.Reader
	basicAuth   bool
}

func (tr testRequest) make() (*http.Response, error) {
	req, err := http.NewRequest(tr.method, tr.url, tr.body)
	if err != nil {
		return nil, err
	}

	if tr.token != "" {
		req.Header.Set("Authorization", apiutil.ThingPrefix+tr.token)
	}
	if tr.basicAuth && tr.token != "" {
		req.SetBasicAuth("", tr.token)
	}
	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}
	return tr.client.Do(req)
}

func TestPublish(t *testing.T) {
	chanID := "1"
	ctSenmlJSON := "application/senml+json"
	ctSenmlCBOR := "application/senml+cbor"
	ctJSON := "application/json"
	thingKey := "thing_key"
	invalidKey := "invalid"
	msg := `[{"n":"current","t":-1,"v":1.6}]`
	msgJSON := `{"field1":"val1","field2":"val2"}`
	msgCBOR := `81A3616E6763757272656E746174206176FB3FF999999999999A`
	thingsClient := mocks.NewThingsServiceClient(map[string]string{thingKey: chanID}, nil, nil)
	svc := newService(thingsClient)
	ts := newHTTPServer(svc)
	defer ts.Close()

	cases := map[string]struct {
		chanID      string
		msg         string
		contentType string
		key         string
		status      int
		basicAuth   bool
	}{
		"publish message": {
			chanID:      chanID,
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         thingKey,
			status:      http.StatusAccepted,
		},
		"publish message with application/senml+cbor content-type": {
			chanID:      chanID,
			msg:         msgCBOR,
			contentType: ctSenmlCBOR,
			key:         thingKey,
			status:      http.StatusAccepted,
		},
		"publish message with application/json content-type": {
			chanID:      chanID,
			msg:         msgJSON,
			contentType: ctJSON,
			key:         thingKey,
			status:      http.StatusAccepted,
		},
		"publish message with empty key": {
			chanID:      chanID,
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         "",
			status:      http.StatusUnauthorized,
		},
		"publish message with basic auth": {
			chanID:      chanID,
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         thingKey,
			basicAuth:   true,
			status:      http.StatusAccepted,
		},
		"publish message with invalid key": {
			chanID:      chanID,
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         invalidKey,
			status:      http.StatusUnauthorized,
		},
		"publish message with invalid basic auth": {
			chanID:      chanID,
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         invalidKey,
			basicAuth:   true,
			status:      http.StatusUnauthorized,
		},
		"publish message without content type": {
			chanID:      chanID,
			msg:         msg,
			contentType: "",
			key:         thingKey,
			status:      http.StatusUnsupportedMediaType,
		},
		"publish message without channel": {
			chanID:      "",
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         thingKey,
			status:      http.StatusAccepted,
		},
		"publish message unable to authorize": {
			chanID:      chanID,
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
			url:         fmt.Sprintf("%s/channels/%s/messages", ts.URL, tc.chanID),
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
