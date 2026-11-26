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

	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	ntevents "github.com/MainfluxLabs/mainflux/consumers/notifiers/events"
	ntmocks "github.com/MainfluxLabs/mainflux/consumers/notifiers/mocks"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/authn"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/events"
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	r "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	groupID         = "c5a1e8f2-6b3d-4a9e-8f7c-2d1b0e9a8c7f"
	orgID           = "f8d2c1b4-7a6e-4d3c-9b8a-5e4f3d2c1b0a"
	orphanGroupID   = "b7f2d3e5-4c6e-4a8f-8d9b-2e3f4a5b6c7d"
	notifierName    = "notifier"
	updatedName     = "updated-notifier"
	invalidToken    = "invalid"
	unknownNotifier = "unknown-notifier"
	orphanUserToken = "orphan@example.com"
)

var (
	user = users.User{
		ID:    "574106f7-030e-4881-8ab0-151195c29f94",
		Email: "user@example.com",
	}
	// orphanUser edits notifiers in a group whose org can't be looked up.
	orphanUser = users.User{
		ID:    "e0d1c2b3-a495-4867-8f9a-0b1c2d3e4f5a",
		Email: orphanUserToken,
	}
	metadata        = map[string]any{"key": "value"}
	contacts        = []string{"first@example.com", "second@example.com"}
	updatedContacts = []string{"third@example.com"}
)

type testEnv struct {
	svc  notifiers.Service
	auth domain.AuthClient
}

func newEnv(t *testing.T) testEnv {
	t.Helper()
	ac := pkgmocks.NewAuthService("", []users.User{user, orphanUser}, nil)

	// The things mock authorizes by token and looks groups up by ID.
	groups := map[string]domain.Group{
		user.Email:      {ID: groupID},
		groupID:         {ID: groupID, OrgID: orgID},
		orphanUserToken: {ID: orphanGroupID},
	}
	tc := pkgmocks.NewThingsServiceClient(nil, nil, groups)

	pub, err := events.NewPublisher(events.PublisherConfig{
		URL:                  redisURL,
		Stream:               events.SMTPNotifierStream,
		BufferSize:           events.DefBufferSize,
		DrainInitialInterval: 10 * time.Millisecond,
		DrainMaxBackoff:      100 * time.Millisecond,
		ShutdownDrainTimeout: 2 * time.Second,
	}, logger.NewMock())
	require.NoError(t, err)
	t.Cleanup(func() { _ = pub.Close() })

	svc := notifiers.New(uuid.NewMock(), ntmocks.NewNotifier(), ntmocks.NewNotifierRepository(), tc)

	return testEnv{
		svc:  ntevents.NewEventStoreMiddleware(svc, tc, pub),
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

func (env testEnv) createNotifier(t *testing.T, token, groupID, name string) notifiers.Notifier {
	t.Helper()
	nf := notifiers.Notifier{Name: name, Contacts: contacts, Metadata: metadata}
	nfs, err := env.svc.CreateNotifiers(context.Background(), token, groupID, nf)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	return nfs[0]
}

func nextEvent(t *testing.T, desc, lastID string) (map[string]any, string) {
	t.Helper()
	streams := redisClient.XRead(context.Background(), &r.XReadArgs{
		Streams: []string{events.SMTPNotifierStream, lastID},
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

// notifierEvent returns the expected fields of an event the actor performed on a notifier in the given group and org.
func notifierEvent(operation string, actor users.User, groupID, orgID string, fields map[string]any) map[string]any {
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

func TestCreateNotifiers(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(t)

	cases := []struct {
		desc    string
		actor   users.User
		token   string
		groupID string
		orgID   string
		err     error
	}{
		{
			desc:    "create notifier with invalid token",
			token:   invalidToken,
			groupID: groupID,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "create notifier",
			actor:   user,
			token:   user.Email,
			groupID: groupID,
			orgID:   orgID,
			err:     nil,
		},
		{
			desc:    "create notifier in a group whose org can't be looked up",
			actor:   orphanUser,
			token:   orphanUserToken,
			groupID: orphanGroupID,
			orgID:   "",
			err:     nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		nf := notifiers.Notifier{Name: notifierName, Contacts: contacts, Metadata: metadata}
		created, err := env.svc.CreateNotifiers(env.identityCtx(tc.token), tc.token, tc.groupID, nf)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var expected map[string]any
		if tc.err == nil {
			expected = notifierEvent(events.NotifierCreate, tc.actor, tc.groupID, tc.orgID, map[string]any{
				"id":       created[0].ID,
				"name":     notifierName,
				"contacts": "first@example.com,second@example.com",
				"metadata": `{"key":"value"}`,
			})
		}

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
		assert.Equal(t, expected, event, msg)
	}
}

func TestUpdateNotifier(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(t)
	nf := env.createNotifier(t, user.Email, groupID, notifierName)

	update := notifiers.Notifier{ID: nf.ID, Name: updatedName, Contacts: updatedContacts, Metadata: metadata}

	cases := []struct {
		desc      string
		token     string
		notifier  notifiers.Notifier
		err       error
		published bool
	}{
		{
			desc:      "update notifier with invalid token",
			token:     invalidToken,
			notifier:  update,
			err:       errors.ErrAuthentication,
			published: false,
		},
		{
			desc:      "update unknown notifier",
			token:     user.Email,
			notifier:  notifiers.Notifier{ID: unknownNotifier, Name: updatedName, Contacts: updatedContacts},
			err:       dbutil.ErrNotFound,
			published: false,
		},
		{
			desc:      "update notifier",
			token:     user.Email,
			notifier:  update,
			err:       nil,
			published: true,
		},
	}

	lastID := skipEvents(t, 1)
	for _, tc := range cases {
		err := env.svc.UpdateNotifier(env.identityCtx(tc.token), tc.token, tc.notifier)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var expected map[string]any
		if tc.published {
			expected = notifierEvent(events.NotifierUpdate, user, groupID, orgID, map[string]any{
				"id":       nf.ID,
				"name":     updatedName,
				"contacts": "third@example.com",
				"metadata": `{"key":"value"}`,
			})
		}

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
		assert.Equal(t, expected, event, msg)
	}
}

func TestRemoveNotifiers(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(t)
	first := env.createNotifier(t, user.Email, groupID, notifierName)
	second := env.createNotifier(t, user.Email, groupID, updatedName)

	cases := []struct {
		desc    string
		token   string
		ids     []string
		err     error
		removed []string
	}{
		{
			desc:    "remove notifiers with invalid token",
			token:   invalidToken,
			ids:     []string{first.ID},
			err:     errors.ErrAuthentication,
			removed: nil,
		},
		{
			desc:    "remove notifiers including an unknown one",
			token:   user.Email,
			ids:     []string{first.ID, unknownNotifier},
			err:     dbutil.ErrNotFound,
			removed: nil,
		},
		{
			desc:    "remove notifiers",
			token:   user.Email,
			ids:     []string{first.ID, second.ID},
			err:     nil,
			removed: []string{first.ID, second.ID},
		},
	}

	lastID := skipEvents(t, 2)
	for _, tc := range cases {
		_, err := env.svc.RemoveNotifiers(env.identityCtx(tc.token), tc.token, tc.ids...)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		for _, id := range tc.removed {
			expected := notifierEvent(events.NotifierRemove, user, groupID, orgID, map[string]any{"id": id})
			var event map[string]any
			event, lastID = nextEvent(t, tc.desc, lastID)
			msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
			assert.Equal(t, expected, event, msg)
		}

		event, _ := nextEvent(t, tc.desc, lastID)
		assert.Nil(t, event, fmt.Sprintf("%s: unexpected event %v\n", tc.desc, event))
	}
}
