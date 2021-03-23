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

	"github.com/mainflux/mainflux/rules"
	httpapi "github.com/mainflux/mainflux/rules/api/http"
	"github.com/mainflux/mainflux/rules/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

const (
	url         = "localhost"
	contentType = "application/json"
	token       = "token"
	token2      = "token2"
	wrong       = "wrong"
	email       = "angry_albattani@email.com"
	email2      = "xenodochial_goldwasser@email.com"
	channel     = "103ec2f2-2034-4d9e-8039-13f4efd36b04"
	channel2    = "243fec72-7cf7-4bca-ac87-44a53b318510"
	ruleAction  = "start"
	name        = "name"
	name2       = "name2"
	row         = "v float, n string"
	sql         = "select * from stream where v > 1.2;"
)

var (
	stream = rules.Stream{
		Name:    name,
		Channel: channel,
		Row:     row,
		Host:    url,
	}
	stream2 = rules.Stream{
		Name:    name2,
		Channel: channel2,
		Row:     row,
		Host:    url,
	}
	rule  = mocks.CreateRule("rule", channel)
	rule2 = mocks.CreateRule("rule2", channel2)
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
		req.Header.Set("Authorization", tr.token)
	}
	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}
	return tr.client.Do(req)
}

func newServer(svc rules.Service) *httptest.Server {
	mux := httpapi.MakeHandler(mocktracer.New(), svc)
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func TestCreateStream(t *testing.T) {
	svc := mocks.NewService(map[string]string{token: email}, map[string]string{channel: email}, url)

	ts := newServer(svc)
	defer ts.Close()

	valid := toJSON(stream)

	invalidStream := stream
	invalidStream.Name = ""
	invalidName := toJSON(invalidStream)

	invalidStream = stream
	invalidStream.Row = ""
	invalidRow := toJSON(invalidStream)

	invalidStream = stream
	invalidStream.Channel = ""
	invalidChannel := toJSON(invalidStream)

	invalidStream = stream
	invalidStream.Host = ""
	invalidHost := toJSON(invalidStream)

	cases := []struct {
		desc        string
		req         string
		contentType string
		auth        string
		status      int
	}{
		{
			desc:        "add valid stream",
			req:         valid,
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
		},
		{
			desc:        "add stream with invalid name",
			req:         invalidName,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "add stream with invalid row",
			req:         invalidRow,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "add stream with invalid channel",
			req:         invalidChannel,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "add stream with invalid host",
			req:         invalidHost,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "add stream invalid token",
			req:         valid,
			contentType: contentType,
			auth:        wrong,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "add stream with empty token",
			req:         valid,
			contentType: contentType,
			auth:        "",
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "add stream with invalid request format",
			req:         "}",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "add stream with empty request",
			req:         "",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "add stream without content type",
			req:         valid,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/streams", ts.URL),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}
