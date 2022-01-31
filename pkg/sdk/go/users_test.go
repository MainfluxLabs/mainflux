// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/mainflux/mainflux/logger"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/users"
	"github.com/mainflux/mainflux/users/api"
	"github.com/mainflux/mainflux/users/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	invalidEmail      = "userexample.com"
	userEmail         = "user@example.com"
	validPass         = "validPass"
	memberRelationKey = "member"
	authoritiesObjKey = "authorities"
)

var (
	passRegex = regexp.MustCompile("^.{8,}$")
	admin     = users.User{Email: adminEmail, Password: validPass}
)

func newUserService() users.Service {
	usersRepo := mocks.NewUserRepository()
	hasher := mocks.NewHasher()

	idProvider := uuid.New()
	id, _ := idProvider.ID()
	admin.ID = id
	mockAuthzDB := map[string][]mocks.SubjectSet{}
	mockAuthzDB[admin.ID] = []mocks.SubjectSet{{Object: authoritiesObjKey, Relation: memberRelationKey}}
	mockAuthzDB["*"] = []mocks.SubjectSet{{Object: "user", Relation: "create"}}

	auth := mocks.NewAuthService(map[string]users.User{adminEmail: admin}, mockAuthzDB)

	emailer := mocks.NewEmailer()

	return users.New(usersRepo, hasher, auth, emailer, idProvider, passRegex)
}

func newUserServer(svc users.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := api.MakeHandler(svc, mocktracer.New(), logger)
	return httptest.NewServer(mux)
}

func TestCreateUser(t *testing.T) {
	svc := newUserService()
	ts := newUserServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		UsersURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	sdkUser := sdk.User{Email: "new-user@example.com", Password: "password"}

	// mockAuthzDB := map[string][]mocks.SubjectSet{}
	// mockAuthzDB[user.Email] = append(mockAuthzDB[user.Email], mocks.SubjectSet{Object: "authorities", Relation: "member"})
	// auth := mocks.NewAuthService(map[string]users.User{userEmail: user}, mockAuthzDB)
	token, _ := svc.Login(context.Background(), admin)
	// tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Id: admin.ID, Email: admin.Email, Type: 0})
	// token := tkn.GetValue()

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
			user:  sdk.User{Email: "user2@example.com", Password: ""},
			token: token,
			err:   createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
		{
			desc:  "create user without password",
			user:  sdk.User{Email: "user2@example.com"},
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
}

func TestRegisterUser(t *testing.T) {
	svc := newUserService()
	ts := newUserServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		UsersURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	sdkUser := sdk.User{Email: "user@example.com", Password: "password"}

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
			desc: "register existing user",
			user: sdkUser,
			err:  createError(sdk.ErrFailedCreation, http.StatusConflict),
		},
		{
			desc: "register user with invalid email address",
			user: sdk.User{Email: invalidEmail, Password: "password"},
			err:  createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
		{
			desc: "register user with empty password",
			user: sdk.User{Email: "user2@example.com", Password: ""},
			err:  createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
		{
			desc: "register user without password",
			user: sdk.User{Email: "user2@example.com"},
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
	sdkUser := sdk.User{Email: "user@example.com", Password: "password"}

	token, _ := svc.Login(context.Background(), admin)
	_, err := mainfluxSDK.CreateUser(token, sdkUser)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	token, _ = svc.Login(context.Background(), users.User{Email: sdkUser.Email, Password: sdkUser.Password})

	cases := []struct {
		desc  string
		user  sdk.User
		token string
		err   error
	}{
		{
			desc:  "create token for user",
			user:  sdkUser,
			token: token,
			err:   nil,
		},
		{
			desc:  "create token for non existing user",
			user:  sdk.User{Email: "user2@example.com", Password: "password"},
			token: "",
			err:   createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
		},
		{
			desc:  "create user with empty email",
			user:  sdk.User{Email: "", Password: "password"},
			token: "",
			err:   createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		token, err := mainfluxSDK.CreateToken(tc.user)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.token, token, fmt.Sprintf("%s: expected response: %s, got:  %s", tc.desc, token, tc.token))
	}
}
