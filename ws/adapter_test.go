// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package ws_test

import (
	"context"
	"fmt"
	"testing"

	thmock "github.com/MainfluxLabs/mainflux/pkg/mocks"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/ws"
	"github.com/MainfluxLabs/mainflux/ws/mocks"
	"github.com/stretchr/testify/assert"
)

const (
	profileID = "1"
	id        = "1"
	thingKey  = "thing_key"
	subTopic  = "subtopic"
	protocol  = "ws"
)

var msg = protomfx.Message{
	Profile:   profileID,
	Publisher: id,
	Subtopic:  "",
	Protocol:  protocol,
	Payload:   []byte(`[{"n":"current","t":-5,"v":1.2}]`),
}

func newService(tc protomfx.ThingsServiceClient) (ws.Service, mocks.MockPubSub) {
	pubsub := mocks.NewPubSub()
	return ws.New(tc, pubsub), pubsub
}

func TestPublish(t *testing.T) {
	thingsClient := thmock.NewThingsServiceClient(map[string]string{thingKey: profileID}, nil, nil)
	svc, _ := newService(thingsClient)

	cases := []struct {
		desc     string
		thingKey string
		msg      protomfx.Message
		err      error
	}{
		{
			desc:     "publish a valid message with valid thingKey",
			thingKey: thingKey,
			msg:      msg,
			err:      nil,
		},
		{
			desc:     "publish a valid message with empty thingKey",
			thingKey: "",
			msg:      msg,
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "publish a valid message with invalid thingKey",
			thingKey: "invalid",
			msg:      msg,
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "publish an empty message with valid thingKey",
			thingKey: thingKey,
			msg:      protomfx.Message{},
			err:      ws.ErrFailedMessagePublish,
		},
		{
			desc:     "publish an empty message with empty thingKey",
			thingKey: "",
			msg:      protomfx.Message{},
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "publish an empty message with invalid thingKey",
			thingKey: "invalid",
			msg:      protomfx.Message{},
			err:      ws.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		err := svc.Publish(context.Background(), tc.thingKey, tc.msg)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSubscribe(t *testing.T) {
	thingsClient := thmock.NewThingsServiceClient(map[string]string{thingKey: profileID}, nil, nil)
	svc, pubsub := newService(thingsClient)

	c := ws.NewClient(nil)

	cases := []struct {
		desc      string
		thingKey  string
		profileID string
		subtopic  string
		fail      bool
		err       error
	}{
		{
			desc:      "subscribe with valid thingKey, profileID, subtopic",
			thingKey:  thingKey,
			profileID: profileID,
			subtopic:  subTopic,
			fail:      false,
			err:       nil,
		},
		{
			desc:      "subscribe again with valid thingKey, profileID, subtopic",
			thingKey:  thingKey,
			profileID: profileID,
			subtopic:  subTopic,
			fail:      false,
			err:       nil,
		},
		{
			desc:      "subscribe with subscribe set to fail",
			thingKey:  thingKey,
			profileID: profileID,
			subtopic:  subTopic,
			fail:      true,
			err:       ws.ErrFailedSubscription,
		},
		{
			desc:      "subscribe with invalid profileID and invalid thingKey",
			thingKey:  "invalid",
			profileID: "0",
			subtopic:  subTopic,
			fail:      false,
			err:       ws.ErrUnauthorizedAccess,
		},
		{
			desc:      "subscribe with empty profile",
			thingKey:  thingKey,
			profileID: "",
			subtopic:  subTopic,
			fail:      false,
			err:       ws.ErrUnauthorizedAccess,
		},
		{
			desc:      "subscribe with empty thingKey",
			thingKey:  "",
			profileID: profileID,
			subtopic:  subTopic,
			fail:      false,
			err:       ws.ErrUnauthorizedAccess,
		},
		{
			desc:      "subscribe with empty thingKey and empty profile",
			thingKey:  "",
			profileID: "",
			subtopic:  subTopic,
			fail:      false,
			err:       ws.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		pubsub.SetFail(tc.fail)
		err := svc.Subscribe(context.Background(), tc.thingKey, tc.profileID, tc.subtopic, c)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUnsubscribe(t *testing.T) {
	thingsClient := thmock.NewThingsServiceClient(map[string]string{thingKey: profileID}, nil, nil)
	svc, pubsub := newService(thingsClient)

	cases := []struct {
		desc      string
		thingKey  string
		profileID string
		subtopic  string
		fail      bool
		err       error
	}{
		{
			desc:      "unsubscribe with valid thingKey, profileID, subtopic",
			thingKey:  thingKey,
			profileID: profileID,
			subtopic:  subTopic,
			fail:      false,
			err:       nil,
		},
		{
			desc:      "unsubscribe with valid thingKey, profileID, and empty subtopic",
			thingKey:  thingKey,
			profileID: profileID,
			subtopic:  "",
			fail:      false,
			err:       nil,
		},
		{
			desc:      "unsubscribe  with unsubscribe set to fail",
			thingKey:  thingKey,
			profileID: profileID,
			subtopic:  subTopic,
			fail:      true,
			err:       ws.ErrFailedUnsubscribe,
		},
		{
			desc:      "unsubscribe with empty profile",
			thingKey:  thingKey,
			profileID: "",
			subtopic:  subTopic,
			fail:      false,
			err:       ws.ErrUnauthorizedAccess,
		},
		{
			desc:      "unsubscribe with empty thingKey",
			thingKey:  "",
			profileID: profileID,
			subtopic:  subTopic,
			fail:      false,
			err:       ws.ErrUnauthorizedAccess,
		},
		{
			desc:      "unsubscribe with empty thingKey and empty profile",
			thingKey:  "",
			profileID: "",
			subtopic:  subTopic,
			fail:      false,
			err:       ws.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		pubsub.SetFail(tc.fail)
		err := svc.Unsubscribe(context.Background(), tc.thingKey, tc.profileID, tc.subtopic)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
