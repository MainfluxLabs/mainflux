// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package ws_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
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

func newService(tc protomfx.ThingsServiceClient) (ws.Service, mocks.MockPubSub) {
	pubsub := mocks.NewPubSub()
	return ws.New(tc, pubsub), pubsub
}

func TestPublish(t *testing.T) {
	tc := pkgmock.NewThingsServiceClient(map[string]things.Profile{thingKey: {ID: profileID}}, nil, nil)
	svc, _ := newService(tc)

	cases := []struct {
		desc     string
		thingKey things.ThingKey
		msg      protomfx.Message
		err      error
	}{
		{
			desc:     "publish a valid message with valid thing key",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: thingKey},
			msg:      msg,
			err:      nil,
		},
		{
			desc:     "publish a valid message with empty thing key",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: ""},
			msg:      msg,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "publish a valid message with invalid thing key",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: "invalid"},
			msg:      msg,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "publish an empty message with valid thing key",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: thingKey},
			msg:      protomfx.Message{},
			err:      messaging.ErrPublishMessage,
		},
		{
			desc:     "publish an empty message with empty thing key",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: ""},
			msg:      protomfx.Message{},
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "publish an empty message with invalid thing key",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: "invalid"},
			msg:      protomfx.Message{},
			err:      errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		err := svc.Publish(context.Background(), tc.thingKey, tc.msg)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSubscribe(t *testing.T) {
	tc := pkgmock.NewThingsServiceClient(nil, map[string]things.Thing{thingKey: {ID: id}}, nil)
	svc, pubsub := newService(tc)

	c := ws.NewClient(nil)

	cases := []struct {
		desc     string
		thingKey things.ThingKey
		subtopic string
		fail     bool
		err      error
	}{
		{
			desc:     "subscribe with valid thing key and subtopic",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: thingKey},
			subtopic: subTopic,
			fail:     false,
			err:      nil,
		},
		{
			desc:     "subscribe again with valid thing key and subtopic",
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
			err:      messaging.ErrFailedSubscribe,
		},
		{
			desc:     "subscribe with invalid thing key",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: "invalid"},
			subtopic: subTopic,
			fail:     false,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "subscribe with empty thing key",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: ""},
			subtopic: subTopic,
			fail:     false,
			err:      errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		pubsub.SetFail(tc.fail)
		err := svc.Subscribe(context.Background(), tc.thingKey, tc.subtopic, c)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUnsubscribe(t *testing.T) {
	tc := pkgmock.NewThingsServiceClient(nil, map[string]things.Thing{thingKey: {ID: id}}, nil)
	svc, pubsub := newService(tc)

	cases := []struct {
		desc     string
		thingKey things.ThingKey
		subtopic string
		fail     bool
		err      error
	}{
		{
			desc:     "unsubscribe with valid thing key and subtopic",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: thingKey},
			subtopic: subTopic,
			fail:     false,
			err:      nil,
		},
		{
			desc:     "unsubscribe with valid thing key and empty subtopic",
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
			err:      messaging.ErrFailedUnsubscribe,
		},
		{
			desc:     "unsubscribe with empty thing key",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: ""},
			subtopic: subTopic,
			fail:     false,
			err:      errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		pubsub.SetFail(tc.fail)
		err := svc.Unsubscribe(context.Background(), tc.thingKey, tc.subtopic)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
