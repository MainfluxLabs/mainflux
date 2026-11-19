// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis_test

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/auth/jwt"
	"github.com/MainfluxLabs/mainflux/auth/mocks"
	"github.com/MainfluxLabs/mainflux/auth/redis"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/events"
	thmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	r "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	secret      = "secret"
	ownerEmail  = "owner@test.com"
	ownerID     = "ownerID"
	memberEmail = "member@test.com"
	memberID    = "memberID"
	otherEmail  = "other@test.com"
	otherID     = "otherID"
	description = "description"
	name        = "name"

	loginDuration = 30 * time.Minute

	maxSessionDuration = 168 * time.Hour
	inviteDuration     = 7 * 24 * time.Hour
)

var (
	org           = auth.Org{Name: name, Description: description, Metadata: map[string]any{"test": "test"}}
	owner         = users.User{ID: ownerID, Email: ownerEmail}
	member        = users.User{ID: memberID, Email: memberEmail}
	other         = users.User{ID: otherID, Email: otherEmail}
	usersByEmails = map[string]users.User{ownerEmail: owner, memberEmail: member, otherEmail: other}
	usersByIDs    = map[string]users.User{ownerID: owner, memberID: member, otherID: other}
)

// withEventStore wraps svc in the event-store middleware backed by a fresh
// Publisher pointed at the test Redis. The publisher is closed via
// t.Cleanup, which drains any in-flight events before the test ends.
func withEventStore(t *testing.T, svc auth.Service) auth.Service {
	t.Helper()
	pub, err := events.NewPublisher(events.PublisherConfig{
		URL:                  redisURL,
		Stream:               events.AuthStream,
		BufferSize:           events.DefBufferSize,
		DrainInitialInterval: 10 * time.Millisecond,
		DrainMaxBackoff:      100 * time.Millisecond,
		ShutdownDrainTimeout: 2 * time.Second,
	}, logger.NewMock())
	require.NoError(t, err)
	t.Cleanup(func() { _ = pub.Close() })
	return redis.NewEventStoreMiddleware(svc, pub)
}

func newService() auth.Service {
	keyRepo := mocks.NewKeyRepository()
	idMockProvider := uuid.NewMock()
	membsRepo := mocks.NewOrgMembershipsRepository()
	orgRepo := mocks.NewOrgRepository(membsRepo)
	roleRepo := mocks.NewRolesRepository()
	invitesRepo := mocks.NewInvitesRepository()
	emailerMock := mocks.NewEmailer()

	uc := mocks.NewUsersService(usersByIDs, usersByEmails)
	tc := thmocks.NewThingsServiceClient(nil, nil, nil)
	t := jwt.New(secret)

	return auth.New(orgRepo, tc, uc, keyRepo, mocks.NewSessionRepository(), roleRepo, membsRepo, invitesRepo, emailerMock, idMockProvider, t, loginDuration, maxSessionDuration, inviteDuration)
}

func TestCreateOrg(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: ownerID, Subject: ownerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	svc = withEventStore(t, svc)

	cases := []struct {
		desc  string
		token string
		org   auth.Org
		err   error
		event map[string]any
	}{
		{
			desc:  "create org successfully",
			org:   org,
			token: ownerToken,
			err:   nil,
			event: map[string]any{
				"operation":    events.OrgCreate,
				"evt_group_id": "",
			},
		},
		{
			desc:  "create org with invalid credentials",
			org:   org,
			token: "invalid",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, oc := range cases {
		created, err := svc.CreateOrg(context.Background(), oc.token, oc.org)
		assert.True(t, errors.Contains(err, oc.err), fmt.Sprintf("%s: expected %s got %s\n", oc.desc, oc.err, err))

		// The id is taken from the created org rather than hardcoded: it comes
		// from a counter shared with everything else that draws an id first.
		if oc.event != nil {
			oc.event["id"] = created.ID
			oc.event["evt_org_id"] = created.ID
		}

		streams := redisClient.XRead(context.Background(), &r.XReadArgs{
			Streams: []string{events.AuthStream, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]any
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			lastID = msg.ID

			assert.NotEmpty(t, event["occurred_at"], oc.desc)
			delete(event, "occurred_at")
		}

		assert.Equal(t, oc.event, event, fmt.Sprintf("%s: expected %v got %v\n", oc.desc, oc.event, event))
	}
}

func TestIssue(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := withEventStore(t, newService())

	impersonator := domain.Identity{
		ID:    "root-id",
		Email: "root@test.com",
	}

	cases := []struct {
		desc  string
		key   auth.Key
		event map[string]any
	}{
		{
			desc: "issue login key",
			key: auth.Key{
				Type:     auth.LoginKey,
				IssuedAt: time.Now(),
				IssuerID: ownerID,
				Subject:  ownerEmail,
			},
			event: map[string]any{
				"operation":               events.UserLogin,
				"jwt_identity_user_id":    ownerID,
				"jwt_identity_user_email": ownerEmail,
				"evt_org_id":              "",
				"evt_group_id":            "",
			},
		},
		{
			desc: "issue impersonated login key",
			key: auth.Key{
				Type:         auth.LoginKey,
				IssuedAt:     time.Now(),
				IssuerID:     ownerID,
				Subject:      ownerEmail,
				Impersonator: &impersonator,
			},
			event: map[string]any{
				"operation":               events.UserImpersonate,
				"id":                      ownerID,
				"email":                   ownerEmail,
				"jwt_identity_user_id":    impersonator.ID,
				"jwt_identity_user_email": impersonator.Email,
				"evt_org_id":              "",
				"evt_group_id":            "",
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, _, err := svc.Issue(context.Background(), "", tc.key)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))

		streams := redisClient.XRead(context.Background(), &r.XReadArgs{
			Streams: []string{events.AuthStream, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]any
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			lastID = msg.ID

			assert.NotEmpty(t, event["occurred_at"], tc.desc)
			delete(event, "occurred_at")
		}

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}

func TestRemoveOrg(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: ownerID, Subject: ownerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	org, err := svc.CreateOrg(context.Background(), ownerToken, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	svc = withEventStore(t, svc)

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "remove existing org successfully",
			id:    org.ID,
			token: ownerToken,
			err:   nil,
			event: map[string]any{
				"id":           org.ID,
				"operation":    events.OrgRemove,
				"evt_org_id":   org.ID,
				"evt_group_id": "",
			},
		},
		{
			desc:  "remove non-existent org",
			id:    strconv.FormatUint(math.MaxUint64, 10),
			token: ownerToken,
			err:   dbutil.ErrNotFound,
			event: nil,
		},
	}

	lastID := "0"
	for _, oc := range cases {
		err := svc.RemoveOrgs(context.Background(), oc.token, oc.id)
		assert.True(t, errors.Contains(err, oc.err), fmt.Sprintf("%s: expected %s got %s\n", oc.desc, oc.err, err))

		streams := redisClient.XRead(context.Background(), &r.XReadArgs{
			Streams: []string{events.AuthStream, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]any
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			lastID = msg.ID

			assert.NotEmpty(t, event["occurred_at"], oc.desc)
			delete(event, "occurred_at")
		}

		assert.Equal(t, oc.event, event, fmt.Sprintf("%s: expected %v got %v\n", oc.desc, oc.event, event))
	}
}

// nextEvent reads the event after lastID from the auth stream, without its occurred_at field.
func nextEvent(t *testing.T, desc, lastID string) (map[string]any, string) {
	t.Helper()
	streams := redisClient.XRead(context.Background(), &r.XReadArgs{
		Streams: []string{events.AuthStream, lastID},
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

// loginToken issues a login key for the user without sending events.
func loginToken(t *testing.T, svc auth.Service, u users.User) string {
	t.Helper()
	key := auth.Key{
		Type:     auth.LoginKey,
		IssuedAt: time.Now(),
		IssuerID: u.ID,
		Subject:  u.Email,
	}
	_, token, err := svc.Issue(context.Background(), "", key)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	return token
}

// setupOrg creates an org owned by the owner without sending events.
func setupOrg(t *testing.T, svc auth.Service) (string, auth.Org) {
	t.Helper()
	ownerToken := loginToken(t, svc, owner)
	o, err := svc.CreateOrg(context.Background(), ownerToken, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	return ownerToken, o
}

func TestIssueAPIKey(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	ownerToken := loginToken(t, svc, owner)
	svc = withEventStore(t, svc)

	expiresAt := time.Now().Add(time.Hour).UTC()

	cases := []struct {
		desc  string
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "issue API key successfully",
			token: ownerToken,
			err:   nil,
			event: map[string]any{
				"operation":    events.APIKeyIssue,
				"expires_at":   expiresAt.Format(time.RFC3339Nano),
				"evt_org_id":   "",
				"evt_group_id": "",
			},
		},
		{
			desc:  "issue API key with invalid credentials",
			token: "invalid",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		key := auth.Key{
			Type:      auth.APIKey,
			IssuedAt:  time.Now(),
			ExpiresAt: expiresAt,
		}
		issued, _, err := svc.Issue(context.Background(), tc.token, key)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)
		if tc.event != nil {
			tc.event["id"] = issued.ID
		}

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestRevokeAPIKey(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	ownerToken := loginToken(t, svc, owner)
	key := auth.Key{Type: auth.APIKey, IssuedAt: time.Now()}
	issued, _, err := svc.Issue(context.Background(), ownerToken, key)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	svc = withEventStore(t, svc)

	cases := []struct {
		desc  string
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "revoke API key with invalid credentials",
			token: "invalid",
			err:   errors.ErrAuthentication,
			event: nil,
		},
		{
			desc:  "revoke API key successfully",
			token: ownerToken,
			err:   nil,
			event: map[string]any{
				"id":           issued.ID,
				"operation":    events.APIKeyRevoke,
				"evt_org_id":   "",
				"evt_group_id": "",
			},
		},
		{
			desc:  "revoke already revoked API key",
			token: ownerToken,
			err:   dbutil.ErrNotFound,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.Revoke(context.Background(), tc.token, issued.ID)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestUpdateOrg(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	ownerToken, o := setupOrg(t, svc)
	svc = withEventStore(t, svc)

	update := auth.Org{
		ID:          o.ID,
		Name:        "updated",
		Description: "updated-desc",
		Metadata:    map[string]any{"test": "updated"},
	}

	cases := []struct {
		desc  string
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "update org successfully",
			token: ownerToken,
			err:   nil,
			event: map[string]any{
				"id":           o.ID,
				"name":         update.Name,
				"description":  update.Description,
				"metadata":     "{\"test\":\"updated\"}",
				"operation":    events.OrgUpdate,
				"evt_org_id":   o.ID,
				"evt_group_id": "",
			},
		},
		{
			desc:  "update org with invalid credentials",
			token: "invalid",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, err := svc.UpdateOrg(context.Background(), tc.token, update)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestCreateOrgMemberships(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	ownerToken, o := setupOrg(t, svc)
	svc = withEventStore(t, svc)

	om := auth.OrgMembership{Email: memberEmail, Role: auth.Viewer}

	cases := []struct {
		desc  string
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "create org memberships with invalid credentials",
			token: "invalid",
			err:   errors.ErrAuthentication,
			event: nil,
		},
		{
			desc:  "create org memberships successfully",
			token: ownerToken,
			err:   nil,
			event: map[string]any{
				"org_id":       o.ID,
				"member_id":    memberID,
				"role":         auth.Viewer,
				"operation":    events.OrgMembershipCreate,
				"evt_org_id":   o.ID,
				"evt_group_id": "",
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, err := svc.CreateOrgMemberships(context.Background(), tc.token, o.ID, om)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestUpdateOrgMemberships(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	ownerToken, o := setupOrg(t, svc)
	om := auth.OrgMembership{Email: memberEmail, Role: auth.Viewer}
	_, err := svc.CreateOrgMemberships(context.Background(), ownerToken, o.ID, om)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	svc = withEventStore(t, svc)

	om.Role = auth.Editor

	cases := []struct {
		desc  string
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "update org memberships successfully",
			token: ownerToken,
			err:   nil,
			event: map[string]any{
				"org_id":       o.ID,
				"member_id":    memberID,
				"role":         auth.Editor,
				"operation":    events.OrgMembershipUpdate,
				"evt_org_id":   o.ID,
				"evt_group_id": "",
			},
		},
		{
			desc:  "update org memberships with invalid credentials",
			token: "invalid",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, err := svc.UpdateOrgMemberships(context.Background(), tc.token, o.ID, om)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestRemoveOrgMemberships(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	ownerToken, o := setupOrg(t, svc)
	om := auth.OrgMembership{Email: memberEmail, Role: auth.Viewer}
	_, err := svc.CreateOrgMemberships(context.Background(), ownerToken, o.ID, om)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	svc = withEventStore(t, svc)

	cases := []struct {
		desc  string
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "remove org memberships with invalid credentials",
			token: "invalid",
			err:   errors.ErrAuthentication,
			event: nil,
		},
		{
			desc:  "remove org memberships successfully",
			token: ownerToken,
			err:   nil,
			event: map[string]any{
				"org_id":       o.ID,
				"member_id":    memberID,
				"operation":    events.OrgMembershipRemove,
				"evt_org_id":   o.ID,
				"evt_group_id": "",
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.RemoveOrgMemberships(context.Background(), tc.token, o.ID, memberID)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestCreateOrgInvite(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	ownerToken, o := setupOrg(t, svc)
	svc = withEventStore(t, svc)

	oi := auth.OrgInvite{OrgID: o.ID, InviteeEmail: memberEmail, InviteeRole: auth.Viewer}

	cases := []struct {
		desc  string
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "create org invite successfully",
			token: ownerToken,
			err:   nil,
			event: map[string]any{
				"invitee_id":    memberID,
				"invitee_email": memberEmail,
				"invitee_role":  auth.Viewer,
				"operation":     events.OrgInviteCreate,
				"evt_org_id":    o.ID,
				"evt_group_id":  "",
			},
		},
		{
			desc:  "create org invite with invalid credentials",
			token: "invalid",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		invite, err := svc.CreateOrgInvite(context.Background(), tc.token, oi, "")
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)
		if tc.event != nil {
			tc.event["id"] = invite.ID
		}

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestCreateDormantOrgInvite(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	ownerToken, o := setupOrg(t, svc)
	svc = withEventStore(t, svc)

	oi := auth.OrgInvite{OrgID: o.ID, InviteeRole: auth.Viewer}

	cases := []struct {
		desc  string
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "create dormant org invite successfully",
			token: ownerToken,
			err:   nil,
			event: map[string]any{
				"invitee_id":              "",
				"invitee_email":           "",
				"invitee_role":            auth.Viewer,
				"operation":               events.OrgInviteCreate,
				"jwt_identity_user_id":    ownerID,
				"jwt_identity_user_email": ownerEmail,
				"evt_org_id":              o.ID,
				"evt_group_id":            "",
			},
		},
		{
			desc:  "create dormant org invite with invalid credentials",
			token: "invalid",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		invite, err := svc.CreateDormantOrgInvite(context.Background(), tc.token, oi, "platform-invite-id")
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)
		if tc.event != nil {
			tc.event["id"] = invite.ID
		}

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestRevokeOrgInvite(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	ownerToken, o := setupOrg(t, svc)
	oi := auth.OrgInvite{OrgID: o.ID, InviteeEmail: memberEmail, InviteeRole: auth.Viewer}
	invite, err := svc.CreateOrgInvite(context.Background(), ownerToken, oi, "")
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	svc = withEventStore(t, svc)

	cases := []struct {
		desc  string
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "revoke org invite with invalid credentials",
			token: "invalid",
			err:   errors.ErrAuthentication,
			event: nil,
		},
		{
			desc:  "revoke org invite successfully",
			token: ownerToken,
			err:   nil,
			event: map[string]any{
				"id":           invite.ID,
				"operation":    events.OrgInviteRevoke,
				"evt_org_id":   o.ID,
				"evt_group_id": "",
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, err := svc.RevokeOrgInvite(context.Background(), tc.token, invite.ID)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestRespondOrgInvite(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	ownerToken, o := setupOrg(t, svc)
	memberToken := loginToken(t, svc, member)
	otherToken := loginToken(t, svc, other)

	oi := auth.OrgInvite{OrgID: o.ID, InviteeEmail: memberEmail, InviteeRole: auth.Viewer}
	declined, err := svc.CreateOrgInvite(context.Background(), ownerToken, oi, "")
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	oi.InviteeEmail = otherEmail
	accepted, err := svc.CreateOrgInvite(context.Background(), ownerToken, oi, "")
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	svc = withEventStore(t, svc)

	cases := []struct {
		desc     string
		token    string
		inviteID string
		accept   bool
		err      error
		event    map[string]any
	}{
		{
			desc:     "respond to org invite with invalid credentials",
			token:    "invalid",
			inviteID: declined.ID,
			accept:   false,
			err:      errors.ErrAuthentication,
			event:    nil,
		},
		{
			desc:     "decline org invite successfully",
			token:    memberToken,
			inviteID: declined.ID,
			accept:   false,
			err:      nil,
			event: map[string]any{
				"id":           declined.ID,
				"operation":    events.OrgInviteDecline,
				"evt_org_id":   o.ID,
				"evt_group_id": "",
			},
		},
		{
			desc:     "accept org invite successfully",
			token:    otherToken,
			inviteID: accepted.ID,
			accept:   true,
			err:      nil,
			event: map[string]any{
				"id":           accepted.ID,
				"operation":    events.OrgInviteAccept,
				"evt_org_id":   o.ID,
				"evt_group_id": "",
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, err := svc.RespondOrgInvite(context.Background(), tc.token, tc.inviteID, tc.accept)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestLogoutAll(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()
	ownerToken := loginToken(t, svc, owner)
	svc = withEventStore(t, svc)

	cases := []struct {
		desc  string
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "logout all sessions successfully",
			token: ownerToken,
			err:   nil,
			event: map[string]any{
				"operation":    events.SessionLogoutAll,
				"evt_org_id":   "",
				"evt_group_id": "",
			},
		},
		{
			desc:  "logout all sessions with invalid credentials",
			token: "invalid",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.LogoutAll(context.Background(), tc.token)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}
