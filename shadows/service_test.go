// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package shadows_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/shadows"
	shmocks "github.com/MainfluxLabs/mainflux/shadows/mocks"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	email   = "admin@example.com"
	token   = email
	thingID = "5384fb1c-d0ae-4cbe-be52-c54223150fe0"
	groupID = "574106f7-030e-4881-8ab0-151195c29f94"
	wrongID = "wrong-id"
)

var desiredState = shadows.State{"led": "on"}

func newService() shadows.Service {
	thingsSvc := pkgmocks.NewThingsServiceClient(
		nil,
		map[string]things.Thing{token: {ID: thingID, GroupID: groupID}},
		map[string]things.Group{token: {ID: groupID}},
	)
	repo := shmocks.NewShadowRepository()
	pub := shmocks.NewCommandPublisher()
	log := logger.NewMock()

	return shadows.New(thingsSvc, repo, pub, log)
}

func TestUpdateDesiredState(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc    string
		token   string
		thingID string
		desired shadows.State
		delta   shadows.State
		err     error
	}{
		{
			desc:    "update desired state with valid token",
			token:   token,
			thingID: thingID,
			desired: desiredState,
			delta:   desiredState,
			err:     nil,
		},
		{
			desc:    "update desired state with empty token",
			token:   "",
			thingID: thingID,
			desired: desiredState,
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "update desired state for wrong thing ID",
			token:   token,
			thingID: wrongID,
			desired: desiredState,
			err:     errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		sh, err := svc.UpdateDesiredState(context.Background(), tc.token, tc.thingID, tc.desired)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Equal(t, tc.desired, sh.Desired, fmt.Sprintf("%s: expected desired %v got %v", tc.desc, tc.desired, sh.Desired))
			// With nothing reported yet, every desired key is part of the delta.
			assert.Equal(t, tc.delta, sh.Delta, fmt.Sprintf("%s: expected delta %v got %v", tc.desc, tc.delta, sh.Delta))
		}
	}
}

func TestViewShadow(t *testing.T) {
	svc := newService()

	_, err := svc.UpdateDesiredState(context.Background(), token, thingID, shadows.State{"led": "on", "temp": "20"})
	require.Nil(t, err, fmt.Sprintf("unexpected error setting desired state: %s", err))

	// The thing reports one of the two desired values, the other stays pending in the delta
	err = svc.ConsumeMessage("", protomfx.Message{Publisher: thingID, Payload: toPayload(shadows.State{"led": "on"})})
	require.Nil(t, err, fmt.Sprintf("unexpected error reporting state: %s", err))

	cases := []struct {
		desc     string
		token    string
		thingID  string
		desired  shadows.State
		reported shadows.State
		delta    shadows.State
		err      error
	}{
		{
			desc:     "view shadow with valid token",
			token:    token,
			thingID:  thingID,
			desired:  shadows.State{"led": "on", "temp": "20"},
			reported: shadows.State{"led": "on"},
			delta:    shadows.State{"temp": "20"},
			err:      nil,
		},
		{
			desc:    "view shadow with empty token",
			token:   "",
			thingID: thingID,
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "view shadow for wrong thing ID",
			token:   token,
			thingID: wrongID,
			err:     errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		sh, err := svc.ViewShadow(context.Background(), tc.token, tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Equal(t, tc.desired, sh.Desired, fmt.Sprintf("%s: expected desired %v got %v", tc.desc, tc.desired, sh.Desired))
			assert.Equal(t, tc.reported, sh.Reported, fmt.Sprintf("%s: expected reported %v got %v", tc.desc, tc.reported, sh.Reported))
			assert.Equal(t, tc.delta, sh.Delta, fmt.Sprintf("%s: expected delta %v got %v", tc.desc, tc.delta, sh.Delta))
		}
	}
}

func TestRemoveShadow(t *testing.T) {
	svc := newService()

	_, err := svc.UpdateDesiredState(context.Background(), token, thingID, desiredState)
	require.Nil(t, err, fmt.Sprintf("unexpected error setting desired state: %s", err))

	cases := []struct {
		desc    string
		token   string
		thingID string
		err     error
	}{
		{
			desc:    "remove shadow with empty token",
			token:   "",
			thingID: thingID,
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "remove shadow for wrong thing ID",
			token:   token,
			thingID: wrongID,
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "remove shadow with valid token",
			token:   token,
			thingID: thingID,
			err:     nil,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveShadow(context.Background(), tc.token, tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestRemoveByThing(t *testing.T) {
	svc := newService()

	_, err := svc.UpdateDesiredState(context.Background(), token, thingID, desiredState)
	require.Nil(t, err, fmt.Sprintf("unexpected error setting desired state: %s", err))

	cases := []struct {
		desc    string
		thingID string
		err     error
	}{
		{
			desc:    "remove by thing with valid thing ID",
			thingID: thingID,
			err:     nil,
		},
		{
			desc:    "remove by thing with unknown thing ID",
			thingID: wrongID,
			err:     nil,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveByThing(context.Background(), tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func toPayload(s shadows.State) []byte {
	b, _ := json.Marshal(s)
	return b
}

func TestConsumeMessage(t *testing.T) {
	svc := newService()

	_, err := svc.UpdateDesiredState(context.Background(), token, thingID, desiredState)
	require.Nil(t, err, fmt.Sprintf("unexpected error setting desired state: %s", err))

	cases := []struct {
		desc     string
		payload  []byte
		reported shadows.State
		delta    shadows.State
	}{
		{
			desc:     "consume message with empty payload is ignored",
			payload:  []byte{},
			reported: nil,
			delta:    shadows.State{"led": "on"},
		},
		{
			desc:     "consume message matching desired state clears the delta",
			payload:  toPayload(shadows.State{"led": "on"}),
			reported: shadows.State{"led": "on"},
			delta:    nil,
		},
		{
			desc:     "consume message diverging from desired state repopulates the delta",
			payload:  toPayload(shadows.State{"led": "off"}),
			reported: shadows.State{"led": "off"},
			delta:    shadows.State{"led": "on"},
		},
		{
			desc:     "consume message with extra key merges into reported state",
			payload:  toPayload(shadows.State{"temp": "20"}),
			reported: shadows.State{"led": "off", "temp": "20"},
			delta:    shadows.State{"led": "on"},
		},
	}

	for _, tc := range cases {
		msg := protomfx.Message{
			Publisher: thingID,
			Payload:   tc.payload,
		}

		err := svc.ConsumeMessage("", msg)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))

		sh, err := svc.ViewShadow(context.Background(), token, thingID)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error viewing shadow: %s", tc.desc, err))
		assert.Equal(t, tc.reported, sh.Reported, fmt.Sprintf("%s: expected reported %v got %v", tc.desc, tc.reported, sh.Reported))
		assert.Equal(t, tc.delta, sh.Delta, fmt.Sprintf("%s: expected delta %v got %v", tc.desc, tc.delta, sh.Delta))
	}
}
