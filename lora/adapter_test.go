// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package lora_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/lora"
	"github.com/MainfluxLabs/mainflux/lora/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	pubmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	thingID  = "thingID-1"
	chanID   = "chanID-1"
	devEUI   = "devEUI-1"
	appID    = "appID-1"
	thingID2 = "thingID-2"
	chanID2  = "chanID-2"
	devEUI2  = "devEUI-2"
	appID2   = "appID-2"
	msg      = `[{"bn":"msg-base-name","n":"temperature","v": 17},{"n":"humidity","v": 56}]`
)

func newService() lora.Service {
	pub := pubmocks.NewPublisher()
	thingsRM := mocks.NewRouteMap()
	channelsRM := mocks.NewRouteMap()
	connsRM := mocks.NewRouteMap()

	return lora.New(pub, thingsRM, channelsRM, connsRM)
}

func TestPublish(t *testing.T) {
	svc := newService()

	err := svc.CreateChannel(nil, chanID, appID)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.CreateThing(nil, thingID, devEUI)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.ConnectThing(nil, chanID, thingID)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.CreateChannel(nil, chanID2, appID2)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.CreateThing(nil, thingID2, devEUI2)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	msgBase64 := base64.StdEncoding.EncodeToString([]byte(msg))

	cases := []struct {
		desc string
		err  error
		msg  lora.Message
	}{
		{
			desc: "publish message with existing route-map and valid Data",
			err:  nil,
			msg: lora.Message{
				ApplicationID: appID,
				DevEUI:        devEUI,
				Data:          msgBase64,
			},
		},
		{
			desc: "publish message with existing route-map and invalid Data",
			err:  lora.ErrMalformedMessage,
			msg: lora.Message{
				ApplicationID: appID,
				DevEUI:        devEUI,
				Data:          "wrong",
			},
		},
		{
			desc: "publish message with non existing appID route-map",
			err:  lora.ErrNotFoundApp,
			msg: lora.Message{
				ApplicationID: "wrong",
				DevEUI:        devEUI,
			},
		},
		{
			desc: "publish message with non existing devEUI route-map",
			err:  lora.ErrNotFoundDev,
			msg: lora.Message{
				ApplicationID: appID,
				DevEUI:        "wrong",
			},
		},
		{
			desc: "publish message with non existing connection route-map",
			err:  lora.ErrNotConnected,
			msg: lora.Message{
				ApplicationID: appID2,
				DevEUI:        devEUI2,
			},
		},
	}

	for _, tc := range cases {
		err := svc.Publish(nil, tc.msg)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
