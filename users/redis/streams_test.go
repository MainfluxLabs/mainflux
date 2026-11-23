// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/authn"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	usmocks "github.com/MainfluxLabs/mainflux/users/mocks"
	"github.com/MainfluxLabs/mainflux/users/redis"
	r "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

const (
	password       = "brave-ocean-lantern"
	newPassword    = "fresh-river-stone"
	inviteDuration = 7 * 24 * time.Hour
	redirectPath   = "/register/invite"
	verifyToken    = "verification-token"
	orgID          = "org-id"
)

var (
	admin = users.User{
		ID:       "574106f7-030e-4881-8ab0-151195c29f94",
		Email:    "admin@example.com",
		Password: password,
		Role:     domain.RoleRootAdmin,
		Status:   users.EnabledStatusKey,
	}
	user = users.User{
		ID:       "574106f7-030e-4881-8ab0-151195c29f96",
		Email:    "user@example.com",
		Password: password,
		Status:   users.EnabledStatusKey,
	}
	newUser = users.User{
		Email:    "new-user@example.com",
		Password: password,
	}
	usersList = []users.User{admin, user}
)

type testEnv struct {
	svc  users.Service
	auth domain.AuthClient
}

// newEnv returns a users service and the auth client it uses, without an event store.
func newEnv(emailVerifyEnabled bool) testEnv {
	c := users.Config{
		InviteDuration:      inviteDuration,
		EmailVerifyEnabled:  emailVerifyEnabled,
		SelfRegisterEnabled: true,
		PassMinLength:       users.MinPasswordLength,
	}

	return newEnvWithConfig(c, usersList)
}

// newEnvWithConfig is newEnv with a custom config and the users known to the auth client.
func newEnvWithConfig(c users.Config, authUsers []users.User) testEnv {
	verifications := []users.EmailVerification{
		{
			User:      newUser,
			Token:     verifyToken,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		},
	}
	ac := mocks.NewAuthService(admin.ID, authUsers, nil)
	svc := users.New(
		usmocks.NewUserRepository(usersList),
		usmocks.NewEmailVerificationRepository(verifications),
		usmocks.NewPlatformInvitesRepository(),
		usmocks.NewIdentityRepository(),
		usmocks.NewHasher(),
		usmocks.NewLegacyHasher(),
		ac,
		usmocks.NewEmailer(),
		uuid.New(),
		logger.NewMock(),
		c,
	)

	return testEnv{svc: svc, auth: ac}
}

// withEventStore wraps the service in the event store middleware backed by the test Redis.
func (env testEnv) withEventStore(t *testing.T) users.Service {
	t.Helper()
	pub, err := events.NewPublisher(events.PublisherConfig{
		URL:                  redisURL,
		Stream:               events.UsersStream,
		BufferSize:           events.DefBufferSize,
		DrainInitialInterval: 10 * time.Millisecond,
		DrainMaxBackoff:      100 * time.Millisecond,
		ShutdownDrainTimeout: 2 * time.Second,
	}, logger.NewMock())
	require.NoError(t, err)
	t.Cleanup(func() { _ = pub.Close() })

	return redis.NewEventStoreMiddleware(env.svc, env.auth, pub)
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

func nextEvent(t *testing.T, desc, lastID string) (map[string]any, string) {
	t.Helper()
	streams := redisClient.XRead(context.Background(), &r.XReadArgs{
		Streams: []string{events.UsersStream, lastID},
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

// actorEvent returns the expected event fields for an operation performed by the given user.
func actorEvent(operation string, actor users.User, fields map[string]any) map[string]any {
	event := map[string]any{
		"operation":               operation,
		"jwt_identity_user_id":    actor.ID,
		"jwt_identity_user_email": actor.Email,
		"evt_org_id":              "",
		"evt_group_id":            "",
	}
	for k, v := range fields {
		event[k] = v
	}

	return event
}

func TestSelfRegister(t *testing.T) {
	cases := []struct {
		desc        string
		emailVerify bool
		user        users.User
		err         error
		published   bool
	}{
		{
			desc:        "self register without e-mail verification",
			emailVerify: false,
			user:        newUser,
			err:         nil,
			published:   true,
		},
		{
			desc:        "self register with pending e-mail verification",
			emailVerify: true,
			user:        newUser,
			err:         nil,
			published:   false,
		},
		{
			desc:        "self register existing user",
			emailVerify: false,
			user:        user,
			err:         dbutil.ErrConflict,
			published:   false,
		},
	}

	for _, tc := range cases {
		_ = redisClient.FlushAll(context.Background()).Err()
		svc := newEnv(tc.emailVerify).withEventStore(t)

		id, err := svc.SelfRegister(context.Background(), tc.user, redirectPath)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var expected map[string]any
		if tc.published {
			registered := users.User{ID: id, Email: tc.user.Email}
			expected = actorEvent(events.UserRegister, registered, map[string]any{
				"id":    id,
				"email": tc.user.Email,
			})
		}

		event, _ := nextEvent(t, tc.desc, "0")
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
		assert.Equal(t, expected, event, msg)
	}
}

func TestVerifyEmail(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	svc := newEnv(true).withEventStore(t)

	cases := []struct {
		desc      string
		token     string
		err       error
		published bool
	}{
		{
			desc:      "verify e-mail",
			token:     verifyToken,
			err:       nil,
			published: true,
		},
		{
			desc:      "verify e-mail with invalid token",
			token:     "invalid",
			err:       errors.ErrAuthentication,
			published: false,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		id, err := svc.VerifyEmail(context.Background(), tc.token)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var expected map[string]any
		if tc.published {
			registered := users.User{ID: id, Email: newUser.Email}
			expected = actorEvent(events.UserRegister, registered, map[string]any{
				"id":    id,
				"email": newUser.Email,
			})
		}

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
		assert.Equal(t, expected, event, msg)
	}
}

func TestRegisterByInvite(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(false)

	invite, err := env.svc.CreatePlatformInvite(
		context.Background(),
		admin.Email,
		redirectPath,
		newUser.Email,
		domain.OrgInvite{},
	)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	svc := env.withEventStore(t)

	cases := []struct {
		desc      string
		inviteID  string
		err       error
		published bool
	}{
		{
			desc:      "register by invalid invite",
			inviteID:  "invalid",
			err:       dbutil.ErrNotFound,
			published: false,
		},
		{
			desc:      "register by invite",
			inviteID:  invite.ID,
			err:       nil,
			published: true,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		id, err := svc.RegisterByInvite(context.Background(), newUser, tc.inviteID, redirectPath)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var expected map[string]any
		if tc.published {
			registered := users.User{ID: id, Email: newUser.Email}
			expected = actorEvent(events.UserRegister, registered, map[string]any{
				"id":        id,
				"email":     newUser.Email,
				"invite_id": invite.ID,
			})
		}

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
		assert.Equal(t, expected, event, msg)
	}
}

func TestRegister(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(false)
	svc := env.withEventStore(t)

	cases := []struct {
		desc      string
		token     string
		err       error
		published bool
	}{
		{
			desc:      "register user as admin",
			token:     admin.Email,
			err:       nil,
			published: true,
		},
		{
			desc:      "register user as regular user",
			token:     user.Email,
			err:       errors.ErrAuthorization,
			published: false,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		id, err := svc.Register(env.identityCtx(tc.token), tc.token, newUser)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var expected map[string]any
		if tc.published {
			expected = actorEvent(events.UserRegister, admin, map[string]any{
				"id":    id,
				"email": newUser.Email,
			})
		}

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
		assert.Equal(t, expected, event, msg)
	}
}

func TestOAuthCallback(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	googleUser := users.User{Email: "google-user@example.com"}
	githubUser := users.User{Email: "github-user@example.com"}

	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "access-token", "token_type": "Bearer"})
	})
	mux.HandleFunc("/google/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "google-id", "email": googleUser.Email})
	})
	mux.HandleFunc("/github/user", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})
	})
	mux.HandleFunc("/github/emails", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{"email": githubUser.Email, "primary": true, "verified": true}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := users.Config{
		InviteDuration:      inviteDuration,
		SelfRegisterEnabled: true,
		PassMinLength:       users.MinPasswordLength,
		GoogleOAuth:         oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: srv.URL + "/token"}},
		GitHubOAuth:         oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: srv.URL + "/token"}},
		URLs: users.ConfigURLs{
			GoogleUserInfoURL:   srv.URL + "/google/userinfo",
			GitHubUserInfoURL:   srv.URL + "/github/user",
			GitHubUserEmailsURL: srv.URL + "/github/emails",
			RedirectLoginURL:    "https://example.com/login",
		},
	}
	// The auth client must know the new users to issue their login keys.
	env := newEnvWithConfig(c, append([]users.User{googleUser, githubUser}, usersList...))

	invite, err := env.svc.CreatePlatformInvite(
		context.Background(),
		admin.Email,
		redirectPath,
		githubUser.Email,
		domain.OrgInvite{},
	)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	svc := env.withEventStore(t)

	cases := []struct {
		desc      string
		provider  string
		inviteID  string
		err       error
		published bool
	}{
		{
			desc:      "oauth callback registers a new user",
			provider:  users.GoogleProvider,
			err:       nil,
			published: true,
		},
		{
			desc:      "oauth callback logs in a linked user",
			provider:  users.GoogleProvider,
			err:       nil,
			published: false,
		},
		{
			desc:      "oauth callback registers a new user by invite",
			provider:  users.GitHubProvider,
			inviteID:  invite.ID,
			err:       nil,
			published: true,
		},
		{
			desc:      "oauth callback with unknown provider",
			provider:  "unknown",
			err:       apiutil.ErrInvalidProvider,
			published: false,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		res, err := svc.OAuthCallback(context.Background(), users.OAuthCallbackData{
			Provider: tc.provider,
			Code:     "code",
			Verifier: oauth2.GenerateVerifier(),
			InviteID: tc.inviteID,
		})
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var expected map[string]any
		if tc.published {
			fields := map[string]any{
				"id":    res.User.ID,
				"email": res.User.Email,
			}
			if tc.inviteID != "" {
				fields["invite_id"] = tc.inviteID
			}
			expected = actorEvent(events.UserRegister, res.User, fields)
		}

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
		assert.Equal(t, expected, event, msg)
	}
}

func TestLogin(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	svc := newEnv(false).withEventStore(t)

	cases := []struct {
		desc  string
		user  users.User
		err   error
		event map[string]any
	}{
		{
			desc: "login with valid credentials",
			user: user,
			err:  nil,
		},
		{
			desc: "login with wrong password",
			user: users.User{Email: user.Email, Password: "wrong-password"},
			err:  errors.ErrAuthentication,
			event: map[string]any{
				"operation":    events.UserLoginFail,
				"email":        user.Email,
				"evt_org_id":   "",
				"evt_group_id": "",
			},
		},
		{
			desc: "login with unknown e-mail",
			user: users.User{Email: "unknown@example.com", Password: password},
			err:  errors.ErrAuthentication,
			event: map[string]any{
				"operation":    events.UserLoginFail,
				"email":        "unknown@example.com",
				"evt_org_id":   "",
				"evt_group_id": "",
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, err := svc.Login(context.Background(), tc.user)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestUpdateUser(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(false)
	svc := env.withEventStore(t)

	metadata := map[string]any{"key": "value"}

	cases := []struct {
		desc  string
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "update user",
			token: user.Email,
			err:   nil,
			event: actorEvent(events.UserUpdate, user, map[string]any{
				"id":       user.ID,
				"metadata": `{"key":"value"}`,
			}),
		},
		{
			desc:  "update user with invalid token",
			token: "invalid",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.UpdateUser(env.identityCtx(tc.token), tc.token, users.User{Metadata: metadata})
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestChangePassword(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(false)
	svc := env.withEventStore(t)

	cases := []struct {
		desc        string
		token       string
		email       string
		oldPassword string
		err         error
		event       map[string]any
	}{
		{
			desc:        "change own password with wrong old password",
			token:       user.Email,
			oldPassword: "wrong-password",
			err:         errors.ErrInvalidPassword,
			event:       nil,
		},
		{
			desc:        "change own password",
			token:       user.Email,
			oldPassword: password,
			err:         nil,
			event: actorEvent(events.UserChangePassword, user, map[string]any{
				"id": user.ID,
			}),
		},
		{
			desc:  "change another user's password as admin",
			token: admin.Email,
			email: user.Email,
			err:   nil,
			event: actorEvent(events.UserChangePassword, admin, map[string]any{
				"id": user.ID,
			}),
		},
	}

	lastID := "0"
	for _, tc := range cases {
		ctx := env.identityCtx(tc.token)
		err := svc.ChangePassword(ctx, tc.token, tc.email, newPassword, tc.oldPassword)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestResetPassword(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(false)

	resetToken, err := env.auth.Issue(context.Background(), user.ID, user.Email, domain.RecoveryKey)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	svc := env.withEventStore(t)

	cases := []struct {
		desc  string
		token string
		err   error
		event map[string]any
	}{
		{
			desc:  "reset password with invalid token",
			token: "invalid",
			err:   errors.ErrAuthentication,
			event: nil,
		},
		{
			desc:  "reset password",
			token: resetToken,
			err:   nil,
			event: actorEvent(events.UserResetPassword, user, map[string]any{
				"id": user.ID,
			}),
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.ResetPassword(context.Background(), tc.token, newPassword)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestChangeUserStatus(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(false)
	svc := env.withEventStore(t)

	cases := []struct {
		desc   string
		token  string
		change func(ctx context.Context, token, id string) error
		err    error
		event  map[string]any
	}{
		{
			desc:   "disable user as regular user",
			token:  user.Email,
			change: svc.DisableUser,
			err:    errors.ErrAuthorization,
			event:  nil,
		},
		{
			desc:   "disable user",
			token:  admin.Email,
			change: svc.DisableUser,
			err:    nil,
			event:  actorEvent(events.UserDisable, admin, map[string]any{"id": user.ID}),
		},
		{
			desc:   "enable user",
			token:  admin.Email,
			change: svc.EnableUser,
			err:    nil,
			event:  actorEvent(events.UserEnable, admin, map[string]any{"id": user.ID}),
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := tc.change(env.identityCtx(tc.token), tc.token, user.ID)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestChangeAdminRole(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(false)
	svc := env.withEventStore(t)

	cases := []struct {
		desc   string
		token  string
		change func(ctx context.Context, token, id string) error
		err    error
		event  map[string]any
	}{
		{
			desc:   "assign admin role as regular user",
			token:  user.Email,
			change: svc.AssignAdminRole,
			err:    errors.ErrAuthorization,
			event:  nil,
		},
		{
			desc:   "assign admin role",
			token:  admin.Email,
			change: svc.AssignAdminRole,
			err:    nil,
			event: actorEvent(events.UserAssignAdminRole, admin, map[string]any{
				"id": user.ID,
			}),
		},
		{
			desc:   "unassign admin role",
			token:  admin.Email,
			change: svc.UnassignAdminRole,
			err:    nil,
			event: actorEvent(events.UserUnassignAdminRole, admin, map[string]any{
				"id": user.ID,
			}),
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := tc.change(env.identityCtx(tc.token), tc.token, user.ID)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}

func TestCreatePlatformInvite(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(false)
	svc := env.withEventStore(t)

	orgInvite := domain.OrgInvite{
		OrgID:        orgID,
		InviteeRole:  "editor",
		GroupInvites: []domain.GroupInvite{{GroupID: "group-id", MemberRole: "viewer"}},
	}

	cases := []struct {
		desc      string
		token     string
		email     string
		orgInvite domain.OrgInvite
		err       error
		published bool
	}{
		{
			desc:      "create platform invite as regular user",
			token:     user.Email,
			email:     newUser.Email,
			err:       errors.ErrAuthorization,
			published: false,
		},
		{
			desc:      "create platform invite for registered user",
			token:     admin.Email,
			email:     user.Email,
			err:       dbutil.ErrConflict,
			published: false,
		},
		{
			desc:      "create platform invite with org invite",
			token:     admin.Email,
			email:     newUser.Email,
			orgInvite: orgInvite,
			err:       nil,
			published: true,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		ctx := env.identityCtx(tc.token)
		invite, err := svc.CreatePlatformInvite(ctx, tc.token, redirectPath, tc.email, tc.orgInvite)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var expected map[string]any
		if tc.published {
			expected = actorEvent(events.PlatformInviteCreate, admin, map[string]any{
				"id":            invite.ID,
				"invitee_email": newUser.Email,
				"invitee_role":  "editor",
				"group_invites": `[{"group_id":"group-id","member_role":"viewer"}]`,
				"evt_org_id":    orgID,
			})
		}

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
		assert.Equal(t, expected, event, msg)
	}
}

func TestRevokePlatformInvite(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(false)

	invite, err := env.svc.CreatePlatformInvite(
		context.Background(),
		admin.Email,
		redirectPath,
		newUser.Email,
		domain.OrgInvite{},
	)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	svc := env.withEventStore(t)

	cases := []struct {
		desc     string
		token    string
		inviteID string
		err      error
		event    map[string]any
	}{
		{
			desc:     "revoke platform invite as regular user",
			token:    user.Email,
			inviteID: invite.ID,
			err:      errors.ErrAuthorization,
			event:    nil,
		},
		{
			desc:     "revoke platform invite",
			token:    admin.Email,
			inviteID: invite.ID,
			err:      nil,
			event: actorEvent(events.PlatformInviteRevoke, admin, map[string]any{
				"id": invite.ID,
			}),
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.RevokePlatformInvite(env.identityCtx(tc.token), tc.token, tc.inviteID)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event)
		assert.Equal(t, tc.event, event, msg)
	}
}
