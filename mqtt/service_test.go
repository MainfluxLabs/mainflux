package mqtt_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/mqtt"
	"github.com/MainfluxLabs/mainflux/mqtt/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	total        = 100
	noLimit      = 0
	exampleUser1 = "email1@example.com"
	adminUser    = "admin@example.com"
	invalidUser  = "invalid@example.com"
)

var idProvider = uuid.NewMock()

func newService() mqtt.Service {
	repo := mocks.NewRepo(make(map[string][]mqtt.Subscription))
	mockAuthzDB := map[string][]mocks.SubjectSet{}
	mockAuthzDB[adminUser] = []mocks.SubjectSet{{Object: "authorities", Relation: "member"}}
	mockAuthzDB["*"] = []mocks.SubjectSet{{Object: "user", Relation: "create"}}
	auth := mocks.NewAuth(map[string]string{exampleUser1: exampleUser1, adminUser: adminUser}, mockAuthzDB)
	return mqtt.NewMqttService(auth, nil, repo, idProvider)
}

func TestCreateSubscription(t *testing.T) {
	svc := newService()

	chanID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	thingID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	sub := mqtt.Subscription{
		Subtopic: subtopic,
		ChanID:   chanID,
		ThingID:  thingID,
	}

	cases := []struct {
		desc string
		sub  mqtt.Subscription
		err  error
	}{
		{
			desc: "create new subscription",
			sub:  sub,
			err:  nil,
		},
		{
			desc: "create with existing subscription",
			sub:  sub,
			err:  errors.ErrConflict,
		},
	}

	for _, tc := range cases {
		err := svc.CreateSubscription(context.Background(), tc.sub)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveSubscription(t *testing.T) {
	svc := newService()

	chanID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	thingID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	sub := mqtt.Subscription{
		Subtopic: subtopic,
		ChanID:   chanID,
		ThingID:  thingID,
	}

	err = svc.CreateSubscription(context.Background(), sub)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc string
		sub  mqtt.Subscription
		err  error
	}{
		{
			desc: "remove subscription successfully",
			sub:  sub,
			err:  nil,
		},
		{
			desc: "subscription does not exist",
			sub:  mqtt.Subscription{},
			err:  errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveSubscription(context.Background(), tc.sub)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveByChannelID(t *testing.T) {
	svc := newService()

	chanID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	var subs []mqtt.Subscription
	for i := 0; i < total; i++ {
		thID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		sub := mqtt.Subscription{
			Subtopic: subtopic,
			ThingID:  thID,
			ChanID:   chanID,
		}

		err = svc.CreateSubscription(context.Background(), sub)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		subs = append(subs, sub)
	}

	cases := []struct {
		desc      string
		channelID string
		token     string
		pageMeta  mqtt.PageMetadata
		page      mqtt.Page
		err       error
	}{
		{
			desc:      "retrieve subscriptions by channel",
			channelID: chanID,
			token:     exampleUser1,
			pageMeta: mqtt.PageMetadata{
				Total:  total,
				Offset: 0,
				Limit:  10,
			},
			page: mqtt.Page{
				PageMetadata: mqtt.PageMetadata{
					Total:  total,
					Offset: 0,
					Limit:  10,
				},
				Subscriptions: subs[:10],
			},
			err: nil,
		},
		{
			desc:      "retrieve subscriptions by channel with no limit",
			channelID: chanID,
			token:     exampleUser1,
			pageMeta: mqtt.PageMetadata{
				Total:  total,
				Offset: 0,
				Limit:  noLimit,
			},
			page: mqtt.Page{
				PageMetadata: mqtt.PageMetadata{
					Total:  total,
					Offset: 0,
					Limit:  noLimit,
				},
				Subscriptions: subs,
			},
			err: nil,
		},
		{
			desc:      "retrieve subscriptions with invalid user",
			channelID: chanID,
			token:     invalidUser,
			pageMeta: mqtt.PageMetadata{
				Total: 0,
			},
			page: mqtt.Page{
				PageMetadata: mqtt.PageMetadata{
					Total: 0,
				},
				Subscriptions: nil,
			},
			err: errors.ErrAuthentication,
		},
		{
			desc:      "retrieve subscriptions with empty token",
			channelID: chanID,
			token:     "",
			pageMeta: mqtt.PageMetadata{
				Total: 0,
			},
			page: mqtt.Page{
				PageMetadata: mqtt.PageMetadata{
					Total: 0,
				},
				Subscriptions: nil,
			},
			err: errors.ErrAuthentication,
		},
	}
	for _, tc := range cases {
		page, err := svc.ListSubscriptions(context.Background(), tc.channelID, tc.token, "", tc.pageMeta)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.page, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.page, page))
	}
}
