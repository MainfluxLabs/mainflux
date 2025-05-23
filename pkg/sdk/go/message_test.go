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
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	sdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

func newMessageService(tc protomfx.ThingsServiceClient, rc protomfx.RulesServiceClient, logger logger.Logger) adapter.Service {
	pub := mocks.NewPublisher()
	return adapter.New(pub, tc, rc, logger)
}

func newMessageServer(svc adapter.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := api.MakeHandler(svc, mocktracer.New(), logger)
	return httptest.NewServer(mux)
}

func TestSendMessage(t *testing.T) {
	profileID := "1"
	atoken := "auth_token"
	invalidToken := "invalid"
	msg := `[{"n":"current","t":-1,"v":1.6}]`
	thingsClient := mocks.NewThingsServiceClient(map[string]things.Profile{atoken: {ID: profileID}}, nil, nil)
	rulesClient := mocks.NewRulesServiceClient()
	logger := logger.NewMock()
	pub := newMessageService(thingsClient, rulesClient, logger)
	ts := newMessageServer(pub)
	defer ts.Close()
	sdkConf := sdk.Config{
		HTTPAdapterURL:  ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	cases := map[string]struct {
		profileID string
		msg       string
		auth      string
		err       error
	}{
		"publish message": {
			profileID: profileID,
			msg:       msg,
			auth:      atoken,
			err:       nil,
		},
		"publish message without authorization token": {
			profileID: profileID,
			msg:       msg,
			auth:      "",
			err:       createError(sdk.ErrFailedPublish, http.StatusUnauthorized),
		},
		"publish message with invalid authorization token": {
			profileID: profileID,
			msg:       msg,
			auth:      invalidToken,
			err:       createError(sdk.ErrFailedPublish, http.StatusUnauthorized),
		},
		"publish message with wrong content type": {
			profileID: profileID,
			msg:       "text",
			auth:      atoken,
			err:       nil,
		},
		"publish message unable to authorize": {
			profileID: profileID,
			msg:       msg,
			auth:      invalidToken,
			err:       createError(sdk.ErrFailedPublish, http.StatusUnauthorized),
		},
	}
	for desc, tc := range cases {
		err := mainfluxSDK.SendMessage(tc.profileID, tc.msg, tc.auth)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", desc, tc.err, err))
	}
}

func TestValidateContentType(t *testing.T) {
	profileID := "1"
	atoken := "auth_token"
	thingsClient := mocks.NewThingsServiceClient(map[string]things.Profile{atoken: {ID: profileID}}, nil, nil)
	rulesClient := mocks.NewRulesServiceClient()
	logger := logger.NewMock()
	pub := newMessageService(thingsClient, rulesClient, logger)
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
