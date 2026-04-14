// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	adapter "github.com/MainfluxLabs/mainflux/http"
	"github.com/MainfluxLabs/mainflux/http/api"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	sdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

var (
	thingID      = "513d02d2-16c1-4f23-98be-9e12f8fee898"
	atoken       = "auth_token"
	invalidValue = "invalid"
	msg          = `[{"n":"current","t":-1,"v":1.6}]`
)

func newMessageService(tc domain.ThingsClient) adapter.Service {
	pub := mocks.NewPublisher()
	return adapter.New(pub, tc)
}

func newMessageServer(svc adapter.Service) *httptest.Server {
	lm := logger.NewMock()
	mux := api.MakeHandler(svc, mocktracer.New(), lm)
	return httptest.NewServer(mux)
}

func TestSendMessage(t *testing.T) {
	tc := mocks.NewThingsServiceClient(nil, map[string]things.Thing{atoken: {ID: thingID}}, nil)
	pub := newMessageService(tc)
	ts := newMessageServer(pub)
	defer ts.Close()
	sdkConf := sdk.Config{
		HTTPAdapterURL:  ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	cases := map[string]struct {
		msg  string
		auth string
		err  error
	}{
		"publish message": {
			msg:  msg,
			auth: atoken,
			err:  nil,
		},
		"publish message without authorization token": {
			msg:  msg,
			auth: "",
			err:  createError(sdk.ErrFailedPublish, http.StatusUnauthorized),
		},
		"publish message with invalid authorization token": {
			msg:  msg,
			auth: invalidValue,
			err:  createError(sdk.ErrFailedPublish, http.StatusUnauthorized),
		},
		"publish message with wrong content type": {
			msg:  "text",
			auth: atoken,
			err:  nil,
		},
		"publish message unable to authorize": {
			msg:  msg,
			auth: invalidValue,
			err:  createError(sdk.ErrFailedPublish, http.StatusUnauthorized),
		},
	}
	for desc, tc := range cases {
		err := mainfluxSDK.SendMessage("/messages", tc.msg, things.ThingKey{Type: things.KeyTypeInternal, Value: tc.auth})
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", desc, tc.err, err))
	}
}

func TestValidateContentType(t *testing.T) {
	tc := mocks.NewThingsServiceClient(nil, map[string]things.Thing{atoken: {ID: thingID}}, nil)
	pub := newMessageService(tc)
	ts := newMessageServer(pub)
	defer ts.Close()

	sdkConf := sdk.Config{
		HTTPAdapterURL:  ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}
	mainfluxSDK := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc  string
		cType sdk.ContentType
		err   error
	}{
		{
			desc:  "set senml+json content type",
			cType: "application/senml+json",
			err:   nil,
		},
		{
			desc:  "set invalid content type",
			cType: "invalid",
			err:   sdk.ErrInvalidContentType,
		},
	}
	for _, tc := range cases {
		err := mainfluxSDK.ValidateContentType(tc.cType)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}
