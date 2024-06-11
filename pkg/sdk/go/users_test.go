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
)

const (
	invalidEmail = "userexample.com"
	userEmail    = "user@example.com"
	validPass    = "validPass"
	registerUser = "register@example.com"
)

var (
	passRegex = regexp.MustCompile("^.{8,}$")
	user      = users.User{Email: userEmail, ID: "574106f7-030e-4881-8ab0-151195c29f94", Password: validPass}
	otherUser = users.User{Email: otherEmail, ID: "371106m2-131g-5286-2mc1-540295c29f96", Password: validPass}
	admin     = users.User{Email: adminEmail, ID: "371106m2-131g-5286-2mc1-540295c29f95", Password: validPass}
	usersList = []users.User{admin, user, otherUser}
)

func newUserService() users.Service {
	usersRepo := usmocks.NewUserRepository(usersList)
	hasher := usmocks.NewHasher()
	idProvider := uuid.New()
	admin.ID, _ = idProvider.ID()
	auth := mocks.NewAuthService(admin.ID, usersList)
	emailer := usmocks.NewEmailer()

	return users.New(usersRepo, hasher, auth, emailer, idProvider, passRegex)
}

func newUserServer(svc users.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(svc, mocktracer.New(), logger)
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

	token, err := svc.Login(context.Background(), admin)
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

	token, err := svc.Login(context.Background(), users.User{Email: sdkUser.Email, Password: sdkUser.Password})
	require.Nil(t, err, fmt.Sprintf("unexpected error login: %s", err))

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
			user:  sdk.User{Email: registerUser, Password: "password"},
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
