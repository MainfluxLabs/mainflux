// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/authn"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/events"
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/MainfluxLabs/mainflux/webhooks"
	whevents "github.com/MainfluxLabs/mainflux/webhooks/events"
	whmocks "github.com/MainfluxLabs/mainflux/webhooks/mocks"
	r "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	thingID         = "5384fb1c-d0ae-4cbe-be52-c54223150fe0"
	groupID         = "c5a1e8f2-6b3d-4a9e-8f7c-2d1b0e9a8c7f"
	orgID           = "f8d2c1b4-7a6e-4d3c-9b8a-5e4f3d2c1b0a"
	orphanThingID   = "a6e1c2f4-3b5d-4f7e-9c8a-1d2e3f4a5b6c"
	orphanGroupID   = "b7f2d3e5-4c6e-4a8f-8d9b-2e3f4a5b6c7d"
	webhookName     = "webhook"
	webhookURL      = "https://example.com/hook"
	updatedName     = "updated-webhook"
	invalidToken    = "invalid"
	unknownWebhook  = "unknown-webhook"
	orphanUserToken = "orphan@example.com"
)

var (
	user = users.User{
		ID:    "574106f7-030e-4881-8ab0-151195c29f94",
		Email: "user@example.com",
	}
	// orphanUser edits webhooks in a group whose org can't be looked up.
	orphanUser = users.User{
		ID:    "e0d1c2b3-a495-4867-8f9a-0b1c2d3e4f5a",
		Email: orphanUserToken,
	}
	metadata = map[string]any{"key": "value"}
	headers  = map[string]string{"Authorization": "Bearer secret"}
)

type testEnv struct {
	svc  webhooks.Service
	auth domain.AuthClient
}

func newEnv(t *testing.T) testEnv {
	t.Helper()
	ac := pkgmocks.NewAuthService("", []users.User{user, orphanUser}, nil)

	// The things mock authorizes by token and looks things and groups up by ID.
	ths := map[string]domain.Thing{
		user.Email:      {ID: thingID, GroupID: groupID},
		thingID:         {ID: thingID, GroupID: groupID},
		orphanUserToken: {ID: orphanThingID, GroupID: orphanGroupID},
		orphanThingID:   {ID: orphanThingID, GroupID: orphanGroupID},
	}
	groups := map[string]domain.Group{
		user.Email:      {ID: groupID},
		groupID:         {ID: groupID, OrgID: orgID},
		orphanUserToken: {ID: orphanGroupID},
	}
	tc := pkgmocks.NewThingsServiceClient(nil, ths, groups)

	pub, err := events.NewPublisher(events.PublisherConfig{
		URL:                  redisURL,
		Stream:               events.WebhooksStream,
		BufferSize:           events.DefBufferSize,
		DrainInitialInterval: 10 * time.Millisecond,
		DrainMaxBackoff:      100 * time.Millisecond,
		ShutdownDrainTimeout: 2 * time.Second,
	}, logger.NewMock())
	require.NoError(t, err)
	t.Cleanup(func() { _ = pub.Close() })

	svc := webhooks.New(tc, whmocks.NewWebhookRepository(), whmocks.NewForwarder(), uuid.NewMock())

	return testEnv{
		svc:  whevents.NewEventStoreMiddleware(svc, tc, pub),
		auth: ac,
	}
}

// identityCtx returns a context carrying the token owner's identity, as the HTTP API sets it.
func (env testEnv) identityCtx(token string) context.Context {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx := authn.HTTPTokenToContext(context.Background(), req)

	var identityCtx context.Context
	capture := func(ctx context.Context, _ any) (any, error) {
		identityCtx = ctx
		return nil, nil
	}
	_, _ = authn.IdentityMiddleware(env.auth, logger.NewMock())(capture)(ctx, nil)

	return identityCtx
}

func (env testEnv) createWebhook(t *testing.T, token, thingID, name string) webhooks.Webhook {
	t.Helper()
	wh := webhooks.Webhook{Name: name, Url: webhookURL, Headers: headers, Metadata: metadata}
	whs, err := env.svc.CreateWebhooks(context.Background(), token, thingID, wh)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	return whs[0]
}

func nextEvent(t *testing.T, desc, lastID string) (map[string]any, string) {
	t.Helper()
	streams := redisClient.XRead(context.Background(), &r.XReadArgs{
		Streams: []string{events.WebhooksStream, lastID},
		Count:   1,
		Block:   time.Second,
	}).Val()

	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, lastID
	}

	msg := streams[0].Messages[0]
	assert.NotEmpty(t, msg.Values["occurred_at"], desc)
	delete(msg.Values, "occurred_at")

	return msg.Values, msg.ID
}

// skipEvents reads past the given number of events, such as those of a test's setup.
func skipEvents(t *testing.T, count int) string {
	t.Helper()
	lastID := "0"
	for i := 0; i < count; i++ {
		_, lastID = nextEvent(t, "setup", lastID)
	}

	return lastID
}

// webhookEvent returns the expected fields of an event the actor performed on a webhook in the given group and org.
func webhookEvent(operation string, actor users.User, groupID, orgID string, fields map[string]any) map[string]any {
	event := map[string]any{
		"operation":               operation,
		"jwt_identity_user_id":    actor.ID,
		"jwt_identity_user_email": actor.Email,
		"evt_org_id":              orgID,
		"evt_group_id":            groupID,
	}
	for k, v := range fields {
		event[k] = v
	}

	return event
}

func TestCreateWebhooks(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(t)

	cases := []struct {
		desc    string
		actor   users.User
		token   string
		thingID string
		groupID string
		orgID   string
		err     error
	}{
		{
			desc:    "create webhook with invalid token",
			token:   invalidToken,
			thingID: thingID,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "create webhook",
			actor:   user,
			token:   user.Email,
			thingID: thingID,
			groupID: groupID,
			orgID:   orgID,
			err:     nil,
		},
		{
			desc:    "create webhook in a group whose org can't be looked up",
			actor:   orphanUser,
			token:   orphanUserToken,
			thingID: orphanThingID,
			groupID: orphanGroupID,
			orgID:   "",
			err:     nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		wh := webhooks.Webhook{Name: webhookName, Url: webhookURL, Headers: headers, Metadata: metadata}
		created, err := env.svc.CreateWebhooks(env.identityCtx(tc.token), tc.token, tc.thingID, wh)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var expected map[string]any
		if tc.err == nil {
			expected = webhookEvent(events.WebhookCreate, tc.actor, tc.groupID, tc.orgID, map[string]any{
				"id":       created[0].ID,
				"thing_id": tc.thingID,
				"name":     webhookName,
				"metadata": `{"key":"value"}`,
			})
		}

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
		assert.Equal(t, expected, event, msg)
	}
}

func TestUpdateWebhook(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(t)
	wh := env.createWebhook(t, user.Email, thingID, webhookName)

	update := webhooks.Webhook{ID: wh.ID, Name: updatedName, Url: webhookURL, Headers: headers, Metadata: metadata}

	cases := []struct {
		desc      string
		token     string
		webhook   webhooks.Webhook
		err       error
		published bool
	}{
		{
			desc:      "update webhook with invalid token",
			token:     invalidToken,
			webhook:   update,
			err:       errors.ErrAuthentication,
			published: false,
		},
		{
			desc:      "update unknown webhook",
			token:     user.Email,
			webhook:   webhooks.Webhook{ID: unknownWebhook, Name: updatedName},
			err:       dbutil.ErrNotFound,
			published: false,
		},
		{
			desc:      "update webhook",
			token:     user.Email,
			webhook:   update,
			err:       nil,
			published: true,
		},
	}

	lastID := skipEvents(t, 1)
	for _, tc := range cases {
		err := env.svc.UpdateWebhook(env.identityCtx(tc.token), tc.token, tc.webhook)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var expected map[string]any
		if tc.published {
			expected = webhookEvent(events.WebhookUpdate, user, groupID, orgID, map[string]any{
				"id":       wh.ID,
				"name":     updatedName,
				"metadata": `{"key":"value"}`,
			})
		}

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
		assert.Equal(t, expected, event, msg)
	}
}

func TestRemoveWebhooks(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(t)
	first := env.createWebhook(t, user.Email, thingID, webhookName)
	second := env.createWebhook(t, user.Email, thingID, updatedName)

	cases := []struct {
		desc    string
		token   string
		ids     []string
		err     error
		removed []string
	}{
		{
			desc:    "remove webhooks with invalid token",
			token:   invalidToken,
			ids:     []string{first.ID},
			err:     errors.ErrAuthentication,
			removed: nil,
		},
		{
			desc:    "remove webhooks including an unknown one",
			token:   user.Email,
			ids:     []string{first.ID, unknownWebhook},
			err:     dbutil.ErrNotFound,
			removed: nil,
		},
		{
			desc:    "remove webhooks",
			token:   user.Email,
			ids:     []string{first.ID, second.ID},
			err:     nil,
			removed: []string{first.ID, second.ID},
		},
	}

	lastID := skipEvents(t, 2)
	for _, tc := range cases {
		_, err := env.svc.RemoveWebhooks(env.identityCtx(tc.token), tc.token, tc.ids...)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		for _, id := range tc.removed {
			expected := webhookEvent(events.WebhookRemove, user, groupID, orgID, map[string]any{"id": id})
			var event map[string]any
			event, lastID = nextEvent(t, tc.desc, lastID)
			msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
			assert.Equal(t, expected, event, msg)
		}

		event, _ := nextEvent(t, tc.desc, lastID)
		assert.Nil(t, event, fmt.Sprintf("%s: unexpected event %v\n", tc.desc, event))
	}
}
