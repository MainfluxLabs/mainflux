// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package coap_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/coap"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/stretchr/testify/assert"
)

const (
	thingKey    = "thing-key"
	thingID     = "5dc43781-ad79-4143-a083-24167ba69351"
	otherID     = "0d6ed7ba-c4b6-4b73-a0a6-4fd5a3d2d5a3"
	groupID     = "aecc6a4a-a860-4816-81b5-74d8a309c1bf"
	otherGroup  = "7b7b7d6e-2c1e-4b3a-9f0e-1e7c5d3a2b10"
	clientToken = "token"
)

type pubsubMock struct {
	subscribed []string
}

func (ps *pubsubMock) PublishCommand(string, protomfx.Command) error {
	return nil
}

func (ps *pubsubMock) Dispatch(protomfx.Message, *domain.ProfileConfig) error {
	return nil
}

func (ps *pubsubMock) Subscribe(_, topic string, _ messaging.MessageHandler) error {
	ps.subscribed = append(ps.subscribed, topic)
	return nil
}

func (ps *pubsubMock) Unsubscribe(string, string) error {
	return nil
}

func (ps *pubsubMock) Close() error {
	return nil
}

type clientMock struct{}

func (clientMock) Done() <-chan struct{} {
	return nil
}

func (clientMock) Cancel() error {
	return nil
}

func (clientMock) Token() string {
	return clientToken
}

func (clientMock) Handle(string, protomfx.Message) error {
	return nil
}

func TestSubscribe(t *testing.T) {
	thing := domain.Thing{ID: thingID, Key: thingKey, GroupID: groupID}
	things := mocks.NewThingsServiceClient(nil, map[string]domain.Thing{thingKey: thing, thingID: thing}, nil)

	key := domain.ThingKey{Value: thingKey, Type: domain.KeyTypeInternal}

	cases := []struct {
		desc    string
		key     domain.ThingKey
		subject string
		err     error
	}{
		{
			desc:    "observe own thing commands",
			key:     key,
			subject: fmt.Sprintf("things.%s.commands.shadow", thingID),
			err:     nil,
		},
		{
			desc:    "observe own thing commands with wildcard subtopic",
			key:     key,
			subject: fmt.Sprintf("things.%s.commands.>", thingID),
			err:     nil,
		},
		{
			desc:    "observe own group commands",
			key:     key,
			subject: fmt.Sprintf("groups.%s.commands", groupID),
			err:     nil,
		},
		{
			desc:    "observe custom subtopic",
			key:     key,
			subject: "home.room.temperature",
			err:     nil,
		},
		{
			desc:    "observe another thing's commands",
			key:     key,
			subject: fmt.Sprintf("things.%s.commands.shadow", otherID),
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "observe another group's commands",
			key:     key,
			subject: fmt.Sprintf("groups.%s.commands", otherGroup),
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "observe all things",
			key:     key,
			subject: "things.*.commands",
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "observe everything",
			key:     key,
			subject: ">",
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "observe with invalid key",
			key:     domain.ThingKey{Value: "invalid", Type: domain.KeyTypeInternal},
			subject: fmt.Sprintf("things.%s.commands.shadow", thingID),
			err:     errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		ps := &pubsubMock{}
		svc := coap.New(things, ps)

		err := svc.Subscribe(context.Background(), tc.key, tc.subject, clientMock{})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		if tc.err == nil {
			assert.Equal(t, []string{tc.subject}, ps.subscribed, fmt.Sprintf("%s: expected subscription to %s\n", tc.desc, tc.subject))
		} else {
			assert.Empty(t, ps.subscribed, fmt.Sprintf("%s: expected no subscription\n", tc.desc))
		}
	}
}
