// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package ws_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	pkgmock "github.com/MainfluxLabs/mainflux/pkg/mocks"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
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
	Publisher: id,
	Subtopic:  "",
	Protocol:  protocol,
	Payload:   []byte(`[{"n":"current","t":-5,"v":1.2}]`),
}

func newService(tc protomfx.ThingsServiceClient, logger logger.Logger) (ws.Service, mocks.MockPubSub) {
	pubsub := mocks.NewPubSub()
	return ws.New(tc, pubsub, logger), pubsub
}

func TestPublish(t *testing.T) {
	tc := pkgmock.NewThingsServiceClient(map[string]things.Profile{thingKey: {ID: profileID}}, nil, nil)
	lm := logger.NewMock()
	svc, _ := newService(tc, lm)

	cases := []struct {
		desc     string
		thingKey things.ThingKey
		msg      protomfx.Message
		err      error
	}{
		{
			desc:     "publish a valid message with valid thingKey",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: thingKey},
			msg:      msg,
			err:      nil,
		},
		{
			desc:     "publish a valid message with empty thingKey",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: ""},
			msg:      msg,
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "publish a valid message with invalid thingKey",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: "invalid"},
			msg:      msg,
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "publish an empty message with valid thingKey",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: thingKey},
			msg:      protomfx.Message{},
			err:      messaging.ErrPublishMessage,
		},
		{
			desc:     "publish an empty message with empty thingKey",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: ""},
			msg:      protomfx.Message{},
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "publish an empty message with invalid thingKey",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: "invalid"},
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
	tc := pkgmock.NewThingsServiceClient(map[string]things.Profile{thingKey: {ID: profileID}}, nil, nil)
	lm := logger.NewMock()
	svc, pubsub := newService(tc, lm)

	c := ws.NewClient(nil)

	cases := []struct {
		desc     string
		thingKey things.ThingKey
		subtopic string
		fail     bool
		err      error
	}{
		{
			desc:     "subscribe with valid thingKey and subtopic",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: thingKey},
			subtopic: subTopic,
			fail:     false,
			err:      nil,
		},
		{
			desc:     "subscribe again with valid thingKey and subtopic",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: thingKey},
			subtopic: subTopic,
			fail:     false,
			err:      nil,
		},
		{
			desc:     "subscribe with subscribe set to fail",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: thingKey},
			subtopic: subTopic,
			fail:     true,
			err:      ws.ErrFailedSubscription,
		},
		{
			desc:     "subscribe with invalid thingKey",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: "invalid"},
			subtopic: subTopic,
			fail:     false,
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "subscribe with empty thingKey",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: ""},
			subtopic: subTopic,
			fail:     false,
			err:      ws.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		pubsub.SetFail(tc.fail)
		err := svc.Subscribe(context.Background(), tc.thingKey, tc.subtopic, c)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUnsubscribe(t *testing.T) {
	tc := pkgmock.NewThingsServiceClient(map[string]things.Profile{thingKey: {ID: profileID}}, nil, nil)
	lm := logger.NewMock()
	svc, pubsub := newService(tc, lm)

	cases := []struct {
		desc     string
		thingKey things.ThingKey
		subtopic string
		fail     bool
		err      error
	}{
		{
			desc:     "unsubscribe with valid thingKey and subtopic",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: thingKey},
			subtopic: subTopic,
			fail:     false,
			err:      nil,
		},
		{
			desc:     "unsubscribe with valid thingKey and empty subtopic",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: thingKey},
			subtopic: "",
			fail:     false,
			err:      nil,
		},
		{
			desc:     "unsubscribe with unsubscribe set to fail",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: thingKey},
			subtopic: subTopic,
			fail:     true,
			err:      ws.ErrFailedUnsubscribe,
		},
		{
			desc:     "unsubscribe with empty thingKey",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: ""},
			subtopic: subTopic,
			fail:     false,
			err:      ws.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		pubsub.SetFail(tc.fail)
		err := svc.Unsubscribe(context.Background(), tc.thingKey, tc.subtopic)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
