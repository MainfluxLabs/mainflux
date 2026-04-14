// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package ws_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
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
	thingID      = "513d02d2-16c1-4f23-98be-9e12f8fee898"
	thingKey     = "c02ff576-ccd5-40f6-ba5f-c85377aad529"
	subtopic     = "subtopic"
	protocol     = "ws"
	userToken    = "user-auth-token"
	invalidValue = "invalid"
	otherThingID = "f1a2b3c4-6789-abcd-ef01-234567890123"
	controllerID = "a6b7c8d9-1234-5678-abcd-ef0123456789"
	actuatorID   = "b7c8d9e0-2345-6789-bcde-f01234567890"
	sensorID     = "c8d9e0f1-3456-789a-cdef-012345678901"
	groupID      = "d9e0f1a2-4567-89ab-def0-123456789012"
	otherGroupID = "e0f1a2b3-5678-9abc-ef01-234567890123"
)

var msg = protomfx.Message{
	Publisher: thingID,
	Protocol:  protocol,
	Payload:   []byte(`[{"n":"current","t":-5,"v":1.2}]`),
}

func newService(tc domain.ThingsClient) (ws.Service, mocks.MockPubSub) {
	pubsub := mocks.NewPubSub()
	return ws.New(tc, pubsub), pubsub
}

func TestPublish(t *testing.T) {
	tc := pkgmock.NewThingsServiceClient(nil, map[string]things.Thing{thingKey: {ID: thingID}}, nil)
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
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: invalidValue},
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
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: invalidValue},
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
	tc := pkgmock.NewThingsServiceClient(nil, map[string]things.Thing{thingKey: {ID: thingID}}, nil)
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
			subtopic: subtopic,
			fail:     false,
			err:      nil,
		},
		{
			desc:     "subscribe again with valid thing key and subtopic",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: thingKey},
			subtopic: subtopic,
			fail:     false,
			err:      nil,
		},
		{
			desc:     "subscribe with subscribe set to fail",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: thingKey},
			subtopic: subtopic,
			fail:     true,
			err:      messaging.ErrFailedSubscribe,
		},
		{
			desc:     "subscribe with invalid thing key",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: invalidValue},
			subtopic: subtopic,
			fail:     false,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "subscribe with empty thing key",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: ""},
			subtopic: subtopic,
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
	tc := pkgmock.NewThingsServiceClient(nil, map[string]things.Thing{thingKey: {ID: thingID}}, nil)
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
			subtopic: subtopic,
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
			subtopic: subtopic,
			fail:     true,
			err:      messaging.ErrFailedUnsubscribe,
		},
		{
			desc:     "unsubscribe with empty thing key",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: ""},
			subtopic: subtopic,
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

func TestSendCommandToThing(t *testing.T) {
	tc := pkgmock.NewThingsServiceClient(nil, map[string]things.Thing{
		userToken: {ID: thingID},
	}, nil)
	svc, _ := newService(tc)

	cases := []struct {
		desc    string
		token   string
		thingID string
		err     error
	}{
		{
			desc:    "send command with valid token and matching thing ID",
			token:   userToken,
			thingID: thingID,
			err:     nil,
		},
		{
			desc:    "send command with invalid token",
			token:   invalidValue,
			thingID: thingID,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "send command with valid token to a different thing ID",
			token:   userToken,
			thingID: otherThingID,
			err:     errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		err := svc.SendCommandToThing(context.Background(), tc.token, tc.thingID, msg)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSendCommandToThingByKey(t *testing.T) {
	tc := pkgmock.NewThingsServiceClient(nil, map[string]things.Thing{
		controllerID: {ID: controllerID, GroupID: groupID, Type: things.ThingTypeController},
		actuatorID:   {ID: actuatorID, GroupID: groupID, Type: things.ThingTypeActuator},
		sensorID:     {ID: sensorID, GroupID: groupID, Type: things.ThingTypeSensor},
		otherThingID: {ID: otherThingID, GroupID: otherGroupID, Type: things.ThingTypeActuator},
	}, nil)
	svc, _ := newService(tc)

	cases := []struct {
		desc        string
		thingKey    things.ThingKey
		recipientID string
		err         error
	}{
		{
			desc:        "send command with controller key to actuator in same group",
			thingKey:    things.ThingKey{Type: things.KeyTypeInternal, Value: controllerID},
			recipientID: actuatorID,
			err:         nil,
		},
		{
			desc:        "send command with invalid key",
			thingKey:    things.ThingKey{Type: things.KeyTypeInternal, Value: invalidValue},
			recipientID: actuatorID,
			err:         errors.ErrAuthentication,
		},
		{
			desc:        "send command with sensor key (no command rights)",
			thingKey:    things.ThingKey{Type: things.KeyTypeInternal, Value: sensorID},
			recipientID: actuatorID,
			err:         errors.ErrAuthorization,
		},
		{
			desc:        "send command to thing in a different group",
			thingKey:    things.ThingKey{Type: things.KeyTypeInternal, Value: controllerID},
			recipientID: otherThingID,
			err:         errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		err := svc.SendCommandToThingByKey(context.Background(), tc.thingKey, tc.recipientID, msg)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSendCommandToGroup(t *testing.T) {
	tc := pkgmock.NewThingsServiceClient(nil, nil, map[string]things.Group{
		userToken: {ID: groupID},
	})
	svc, _ := newService(tc)

	cases := []struct {
		desc    string
		token   string
		groupID string
		err     error
	}{
		{
			desc:    "send group command with valid token and matching group ID",
			token:   userToken,
			groupID: groupID,
			err:     nil,
		},
		{
			desc:    "send group command with invalid token",
			token:   invalidValue,
			groupID: groupID,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "send group command with valid token to a different group ID",
			token:   userToken,
			groupID: otherGroupID,
			err:     errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		err := svc.SendCommandToGroup(context.Background(), tc.token, tc.groupID, msg)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSendCommandToGroupByKey(t *testing.T) {
	tc := pkgmock.NewThingsServiceClient(nil, map[string]things.Thing{
		controllerID: {ID: controllerID, GroupID: groupID, Type: things.ThingTypeController},
		sensorID:     {ID: sensorID, GroupID: groupID, Type: things.ThingTypeSensor},
	}, nil)
	svc, _ := newService(tc)

	cases := []struct {
		desc     string
		thingKey things.ThingKey
		groupID  string
		err      error
	}{
		{
			desc:     "send group command with controller key and matching group",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: controllerID},
			groupID:  groupID,
			err:      nil,
		},
		{
			desc:     "send group command with invalid key",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: invalidValue},
			groupID:  groupID,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "send group command with sensor key (no command rights)",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: sensorID},
			groupID:  groupID,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "send group command to a different group",
			thingKey: things.ThingKey{Type: things.KeyTypeInternal, Value: controllerID},
			groupID:  otherGroupID,
			err:      errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		err := svc.SendCommandToGroupByKey(context.Background(), tc.thingKey, tc.groupID, msg)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
