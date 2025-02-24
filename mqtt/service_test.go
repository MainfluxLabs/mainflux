package mqtt_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/mqtt"
	"github.com/MainfluxLabs/mainflux/mqtt/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	thmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	total        = 100
	noLimit      = 0
	exampleUser1 = "email1@example.com"
	adminUser    = "admin@example.com"
	invalidUser  = "invalid@example.com"
	key          = "thing-key"
)

var idProvider = uuid.NewMock()

func newService() mqtt.Service {
	repo := mocks.NewRepo(make(map[string][]mqtt.Subscription))
	mockAuthzDB := map[string][]mocks.SubjectSet{}
	mockAuthzDB[adminUser] = []mocks.SubjectSet{{Object: "authorities", Relation: "member"}}
	mockAuthzDB["*"] = []mocks.SubjectSet{{Object: "user", Relation: "create"}}
	tc := thmocks.NewThingsServiceClient(nil, map[string]things.Thing{exampleUser1: {GroupID: groupID}}, map[string]things.Group{exampleUser1: {ID: groupID}})
	ac := mocks.NewAuth(map[string]string{exampleUser1: exampleUser1, adminUser: adminUser}, mockAuthzDB)
	return mqtt.NewMqttService(ac, tc, repo, idProvider)
}

func TestCreateSubscription(t *testing.T) {
	svc := newService()

	gID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	thID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	sub := mqtt.Subscription{
		Subtopic: subtopic,
		GroupID:  gID,
		ThingID:  thID,
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

	gID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	thID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	sub := mqtt.Subscription{
		Subtopic: subtopic,
		GroupID:  gID,
		ThingID:  thID,
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

func TestRetrieveByGroupID(t *testing.T) {
	svc := newService()

	var subs []mqtt.Subscription
	for i := 0; i < total; i++ {
		thID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		sub := mqtt.Subscription{
			Subtopic: subtopic,
			ThingID:  thID,
			GroupID:  groupID,
		}

		err = svc.CreateSubscription(context.Background(), sub)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		subs = append(subs, sub)
	}

	cases := []struct {
		desc     string
		groupID  string
		token    string
		key      string
		pageMeta mqtt.PageMetadata
		page     mqtt.Page
		err      error
	}{
		{
			desc:    "retrieve subscriptions by group as user",
			groupID: groupID,
			token:   exampleUser1,
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
			desc:    "retrieve subscriptions by group as user with no limit",
			groupID: groupID,
			token:   exampleUser1,
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
			desc:    "retrieve subscriptions with invalid user",
			groupID: groupID,
			token:   invalidUser,
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
			desc:    "retrieve subscriptions as user with empty token",
			groupID: groupID,
			token:   "",
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
			desc:    "retrieve subscriptions by group as thing",
			groupID: groupID,
			key:     key,
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
			desc:    "retrieve subscriptions by group as thing with no limit",
			groupID: groupID,
			key:     key,
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
			desc:    "retrieve subscriptions as thing with invalid group",
			groupID: invalidID,
			key:     key,
			pageMeta: mqtt.PageMetadata{
				Total: 0,
			},
			page: mqtt.Page{
				PageMetadata: mqtt.PageMetadata{
					Total: 0,
				},
				Subscriptions: nil,
			},
			err: errors.ErrNotFound,
		},
		{
			desc:    "retrieve subscriptions by group without thing key",
			groupID: groupID,
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
		page, err := svc.ListSubscriptions(context.Background(), tc.groupID, tc.token, tc.key, tc.pageMeta)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.page, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.page, page))
	}
}
