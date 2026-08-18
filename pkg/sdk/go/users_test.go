// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	sdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	httpapi "github.com/MainfluxLabs/mainflux/users/api/http"
	usmocks "github.com/MainfluxLabs/mainflux/users/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

const (
	invalidEmail = "userexample.com"
	userEmail    = "user@example.com"
	validPass    = "validPass"
	registerUser = "register@example.com"

	inviteDuration = 7 * 24 * time.Hour
)

var (
	passRegex = regexp.MustCompile(`^\S{8,}$`)
	user      = users.User{Email: userEmail, ID: "574106f7-030e-4881-8ab0-151195c29f94", Password: validPass, Role: auth.Owner}
	otherUser = users.User{Email: otherEmail, ID: "371106m2-131g-5286-2mc1-540295c29f96", Password: validPass, Role: auth.Editor}
	admin     = users.User{Email: adminEmail, ID: "371106m2-131g-5286-2mc1-540295c29f95", Password: validPass, Role: auth.RootSub}

	usersList = []users.User{admin, user, otherUser}
	orgsList  = []auth.Org{{ID: orgID, OwnerID: user.ID}}
)

func newUserService() users.Service {
	usersRepo := usmocks.NewUserRepository(usersList)
	verificationsRepo := usmocks.NewEmailVerificationRepository(nil)
	platformInvitesRepo := usmocks.NewPlatformInvitesRepository()
	identityRepo := usmocks.NewIdentityRepository()
	hasher := usmocks.NewHasher()
	idProvider := uuid.New()
	admin.ID, _ = idProvider.ID()
	auth := mocks.NewAuthService(admin.ID, usersList, orgsList)
	emailer := usmocks.NewEmailer()
	oauthGoogleCfg := oauth2.Config{}
	oauthGithubCfg := oauth2.Config{}
	cfgURLs := users.ConfigURLs{}
	c := users.Config{
		InviteDuration:      inviteDuration,
		EmailVerifyEnabled:  true,
		SelfRegisterEnabled: true,
		GoogleOAuth:         oauthGoogleCfg,
		GitHubOAuth:         oauthGithubCfg,
		URLs:                cfgURLs,
	}
	return users.New(usersRepo, verificationsRepo, platformInvitesRepo, identityRepo, hasher, auth, emailer, idProvider, c)
}

func newUserServer(svc users.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(svc, mocks.NewAuthService(admin.ID, usersList, orgsList), mocktracer.New(), logger, passRegex)
	return httptest.NewServer(mux)
}

/*func TestCreateUser(t *testing.T) {
	svc := newUserService()
	ts := newUserServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		UsersURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	sdkUser := sdk.User{Email: registerUser, Password: validPass}

	tokenPair, err := svc.Login(context.Background(), admin)
	token := tokenPair.AccessToken
	require.Nil(t, err, fmt.Sprintf("unexpected error login: %s", err))

	mainfluxSDK := sdk.NewSDK(sdkConf)
	cases := []struct {
		desc  string
		user  sdk.User
		token string
		err   error
	}{
		{
			desc:  "create new user",
			user:  sdkUser,
			token: token,
			err:   nil,
		},
		{
			desc:  "create existing user",
			user:  sdkUser,
			token: token,
			err:   createError(sdk.ErrFailedCreation, http.StatusConflict),
		},
		{
			desc:  "create user with invalid email address",
			user:  sdk.User{Email: invalidEmail, Password: "password"},
			token: token,
			err:   createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
		{
			desc:  "create user with empty password",
			user:  sdk.User{Email: registerUser, Password: ""},
			token: token,
			err:   createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
		{
			desc:  "create user without password",
			user:  sdk.User{Email: registerUser},
			token: token,
			err:   createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
		{
			desc:  "create user without email",
			user:  sdk.User{Password: "password"},
			token: token,
			err:   createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
		{
			desc:  "create empty user",
			user:  sdk.User{},
			token: token,
			err:   createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		_, err := mainfluxSDK.CreateUser(tc.token, tc.user)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
	}
}*/

func TestRegisterUser(t *testing.T) {
	svc := newUserService()
	ts := newUserServer(svc)
	defer ts.Close()

	sdkConf := sdk.Config{
		UsersURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}
	sdkUser := sdk.User{Email: registerUser, Password: validPass}
	mainfluxSDK := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc string
		user sdk.User
		err  error
	}{
		{
			desc: "register new user",
			user: sdkUser,
			err:  nil,
		},
		{
			desc: "register user with pending e-mail confirmation",
			user: sdkUser,
			err:  nil,
		},
		{
			desc: "register existing user",
			user: sdk.User{Email: user.Email, Password: user.Password},
			err:  createError(sdk.ErrFailedCreation, http.StatusConflict),
		},
		{
			desc: "register user with invalid email address",
			user: sdk.User{Email: invalidEmail, Password: "password"},
			err:  createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
		{
			desc: "register user with empty password",
			user: sdk.User{Email: registerUser, Password: ""},
			err:  createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
		{
			desc: "register user without password",
			user: sdk.User{Email: registerUser},
			err:  createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
		{
			desc: "register user without email",
			user: sdk.User{Password: "password"},
			err:  createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
		{
			desc: "register empty user",
			user: sdk.User{},
			err:  createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		_, err := mainfluxSDK.RegisterUser(tc.user)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
	}
}

func TestCreateToken(t *testing.T) {
	svc := newUserService()
	ts := newUserServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		UsersURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	sdkUser := sdk.User{Email: userEmail, Password: validPass}

	tokenPair, err := svc.Login(context.Background(), users.User{Email: sdkUser.Email, Password: sdkUser.Password})
	token := tokenPair.AccessToken
	require.Nil(t, err, fmt.Sprintf("unexpected error login: %s", err))

	cases := []struct {
		desc    string
		user    sdk.User
		access  string
		refresh string
		err     error
	}{
		{
			desc:    "create token for user",
			user:    sdkUser,
			access:  token,
			refresh: tokenPair.RefreshToken,
			err:     nil,
		},
		{
			desc:   "create token for non existing user",
			user:   sdk.User{Email: registerUser, Password: "password"},
			access: "",
			err:    createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
		},
		{
			desc:   "create user with empty email",
			user:   sdk.User{Email: "", Password: "password"},
			access: "",
			err:    createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		tokens, err := mainfluxSDK.CreateToken(tc.user)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.access, tokens.AccessToken, fmt.Sprintf("%s: expected access token %s, got %s", tc.desc, tc.access, tokens.AccessToken))
		assert.Equal(t, tc.refresh, tokens.RefreshToken, fmt.Sprintf("%s: expected refresh token %s, got %s", tc.desc, tc.refresh, tokens.RefreshToken))
	}
}

func TestRefreshToken(t *testing.T) {
	svc := newUserService()
	ts := newUserServer(svc)
	defer ts.Close()

	mainfluxSDK := sdk.NewSDK(sdk.Config{
		UsersURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	})

	tokenPair, err := mainfluxSDK.CreateToken(sdk.User{Email: userEmail, Password: validPass})
	require.Nil(t, err, fmt.Sprintf("unexpected error creating token: %s", err))

	cases := []struct {
		desc    string
		refresh string
		access  string
		err     error
	}{
		{
			desc:    "refresh with valid refresh token",
			refresh: tokenPair.RefreshToken,
			access:  tokenPair.AccessToken,
			err:     nil,
		},
		{
			desc:    "refresh with an access token",
			refresh: tokenPair.AccessToken,
			access:  "",
			err:     createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
		},
		{
			desc:    "refresh with empty token",
			refresh: "",
			access:  "",
			err:     createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
		},
	}

	for _, tc := range cases {
		tokens, err := mainfluxSDK.RefreshToken(tc.refresh)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.access, tokens.AccessToken, fmt.Sprintf("%s: expected access token %s, got %s", tc.desc, tc.access, tokens.AccessToken))
	}
}

// TestAutoRefresh drives the retry path in sendRequest with a stub that rejects
// the first authenticated call, standing in for an access token that expired
// mid-session.
func TestAutoRefresh(t *testing.T) {
	const (
		staleAccess = "stale-access"
		freshAccess = "fresh-access"
		refresh     = "refresh-token"
	)

	var (
		mu           sync.Mutex
		refreshCalls int
		authHeaders  []string
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tokens" {
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"token": staleAccess, "refresh_token": refresh})
			return
		}

		if r.URL.Path == "/tokens/refresh" {
			mu.Lock()
			refreshCalls++
			mu.Unlock()
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"token": freshAccess, "refresh_token": refresh})
			return
		}

		auth := r.Header.Get("Authorization")
		mu.Lock()
		authHeaders = append(authHeaders, auth)
		mu.Unlock()

		// Only the renewed token is accepted, so the first attempt must 401.
		if auth != "Bearer "+freshAccess {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"total": 0, "users": []any{}})
	}))
	defer ts.Close()

	mainfluxSDK := sdk.NewSDK(sdk.Config{
		UsersURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
		AutoRefresh:     true,
	})

	tokens, err := mainfluxSDK.CreateToken(sdk.User{Email: userEmail, Password: validPass})
	require.Nil(t, err, fmt.Sprintf("unexpected error creating token: %s", err))
	require.Equal(t, staleAccess, tokens.AccessToken, "expected the stub's access token")

	_, err = mainfluxSDK.ListUsers(sdk.PageMetadata{}, tokens.AccessToken)
	assert.Nil(t, err, fmt.Sprintf("expected the retried request to succeed: %s", err))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, refreshCalls, "expected exactly one refresh")
	assert.Equal(t, []string{"Bearer " + staleAccess, "Bearer " + freshAccess}, authHeaders,
		"expected the stale token first, then the refreshed one")
}

// TestAutoRefreshDisabled confirms the retry stays opt-in: without AutoRefresh a
// 401 surfaces to the caller untouched.
func TestAutoRefreshDisabled(t *testing.T) {
	var refreshCalls int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tokens/refresh" {
			refreshCalls++
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	mainfluxSDK := sdk.NewSDK(sdk.Config{
		UsersURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	})

	_, err := mainfluxSDK.ListUsers(sdk.PageMetadata{}, "some-token")
	assert.NotNil(t, err, "expected the 401 to reach the caller")
	assert.Equal(t, 0, refreshCalls, "expected no refresh attempt when AutoRefresh is off")
}
