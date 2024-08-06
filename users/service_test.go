// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	usmocks "github.com/MainfluxLabs/mainflux/users/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	wrong   = "wrong-value"
	userNum = 101
)

var (
	admin           = users.User{Email: "admin@example.com", ID: "574106f7-030e-4881-8ab0-151195c29f94", Role: "root"}
	unauthUser      = users.User{Email: "unauth-user@example.com", ID: "6a32810a-4451-4ae8-bf7f-4b1752856eef"}
	selfRegister    = users.User{Email: "self-register@example.com", Password: "password"}
	registerUser    = users.User{Email: "register-user@example.com", ID: "574106f7-030e-4881-8ab0-151195c29f95", Password: "password"}
	user            = users.User{Email: "user@example.com", ID: "574106f7-030e-4881-8ab0-151195c29f96"}
	nonExistingUser = users.User{Email: "non-ex-user@example.com", Password: "password"}
	usersList       = []users.User{admin, registerUser, user, unauthUser}
	host            = "example.com"

	idProvider = uuid.New()
	passRegex  = regexp.MustCompile("^.{8,}$")
)

func newService() users.Service {
	hasher := usmocks.NewHasher()
	userRepo := usmocks.NewUserRepository(usersList)
	authSvc := mocks.NewAuthService(admin.ID, usersList)
	e := usmocks.NewEmailer()

	return users.New(userRepo, hasher, authSvc, e, idProvider, passRegex)
}

func TestSelfRegister(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc  string
		user  users.User
		token string
		err   error
	}{
		{
			desc: "self register new user",
			user: selfRegister,
			err:  nil,
		},
		{
			desc: "self register existing user",
			user: selfRegister,
			err:  errors.ErrConflict,
		},
		{
			desc:  "register new user",
			user:  nonExistingUser,
			token: admin.Email,
			err:   nil,
		},
		{
			desc:  "register existing user",
			user:  registerUser,
			token: admin.Email,
			err:   errors.ErrConflict,
		},
		{
			desc: "register new user with weak password",
			user: users.User{
				Email:    registerUser.Email,
				Password: "weak",
			},
			token: admin.Email,
			err:   users.ErrPasswordFormat,
		},
	}

	for _, tc := range cases {
		_, err := svc.SelfRegister(context.Background(), tc.user)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestLogin(t *testing.T) {
	svc := newService()

	cases := map[string]struct {
		user users.User
		err  error
	}{
		"login with good credentials": {
			user: registerUser,
			err:  nil,
		},
		"login with wrong e-mail": {
			user: users.User{
				Email:    wrong,
				Password: registerUser.Password,
			},
			err: errors.ErrAuthentication,
		},
		"login with wrong password": {
			user: users.User{
				Email:    registerUser.Email,
				Password: wrong,
			},
			err: errors.ErrAuthentication,
		},
		"login failed auth": {
			user: nonExistingUser,
			err:  errors.ErrAuthentication,
		},
	}

	for desc, tc := range cases {
		_, err := svc.Login(context.Background(), tc.user)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestViewUser(t *testing.T) {
	svc := newService()

	token, err := svc.Login(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		user   users.User
		token  string
		userID string
		err    error
	}{
		"view user with authorized token": {
			user:   user,
			token:  token,
			userID: user.ID,
			err:    nil,
		},
		"view user with empty token": {
			user:   users.User{},
			token:  "",
			userID: registerUser.ID,
			err:    errors.ErrAuthentication,
		},
		"view user with valid token and invalid user id": {
			user:   users.User{},
			token:  token,
			userID: "",
			err:    errors.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		user, err := svc.ViewUser(context.Background(), tc.token, tc.userID)
		assert.Equal(t, tc.user, user, fmt.Sprintf("%s: expected %v got %v\n", desc, tc.user, user))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestViewProfile(t *testing.T) {
	svc := newService()

	token, err := svc.Login(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	adminToken, err := svc.Login(context.Background(), admin)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		user  users.User
		token string
		err   error
	}{
		"valid token's user info": {
			user:  user,
			token: token,
			err:   nil,
		},
		"valid token's admin info": {
			user:  admin,
			token: adminToken,
			err:   nil,
		},
		"invalid token's user info": {
			user:  users.User{},
			token: "",
			err:   errors.ErrAuthentication,
		},
	}

	for desc, tc := range cases {
		u, err := svc.ViewProfile(context.Background(), tc.token)
		assert.Equal(t, tc.user, u, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.user, u))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListUsers(t *testing.T) {
	svc := newService()

	token, err := svc.Login(context.Background(), admin)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	unauthUserToken, err := svc.Login(context.Background(), unauthUser)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	page, err := svc.ListUsers(context.Background(), token, users.PageMetadata{})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	totUser := page.Total

	var nUsers = uint64(userNum)

	for i := uint64(1); i <= nUsers; i++ {
		email := fmt.Sprintf("TestListUsers%d@example.com", i)
		user := users.User{
			Email:    email,
			Password: "passpass",
		}
		_, err := svc.SelfRegister(context.Background(), user)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}
	totUser = totUser + nUsers

	cases := map[string]struct {
		token  string
		offset uint64
		limit  uint64
		email  string
		size   uint64
		err    error
	}{
		"list users with authorized token": {
			token: token,
			size:  totUser,
			err:   nil,
		},
		"list users with invalid token": {
			token: wrong,
			size:  0,
			err:   errors.ErrAuthentication,
		},
		"list users with empty token": {
			token: "",
			size:  0,
			err:   errors.ErrAuthentication,
		},
		"list users without permission": {
			token: unauthUserToken,
			size:  0,
			err:   errors.ErrAuthorization,
		},
		"list users with offset and limit": {
			token:  token,
			offset: 1,
			limit:  totUser,
			size:   totUser - 1,
			err:    nil,
		},
		"list last user": {
			token:  token,
			offset: totUser - 1,
			limit:  totUser,
			size:   1,
			err:    nil,
		},
		"list empty set": {
			token:  token,
			offset: totUser + 1,
			limit:  totUser,
			size:   0,
			err:    nil,
		},

		"list users with no limit": {
			token: token,
			limit: 0,
			size:  totUser,
			err:   nil,
		},
	}

	for desc, tc := range cases {
		pm := users.PageMetadata{
			Offset: tc.offset,
			Limit:  tc.limit,
			Email:  tc.email,
			Status: "all",
		}
		page, err := svc.ListUsers(context.Background(), tc.token, pm)
		size := uint64(len(page.Users))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateUser(t *testing.T) {
	svc := newService()

	token, err := svc.Login(context.Background(), registerUser)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	registerUser.Metadata = map[string]interface{}{"meta": "test"}

	cases := map[string]struct {
		user  users.User
		token string
		err   error
	}{
		"update user with valid token": {
			user:  registerUser,
			token: token,
			err:   nil,
		},
		"update user with invalid token": {
			user:  registerUser,
			token: "non-existent",
			err:   errors.ErrAuthentication,
		},
	}

	for desc, tc := range cases {
		err := svc.UpdateUser(context.Background(), tc.token, tc.user)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestGenerateResetToken(t *testing.T) {
	svc := newService()

	cases := map[string]struct {
		email string
		err   error
	}{
		"valid user reset token":  {registerUser.Email, nil},
		"invalid user rest token": {nonExistingUser.Email, errors.ErrNotFound},
	}

	for desc, tc := range cases {
		err := svc.GenerateResetToken(context.Background(), tc.email, host)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestChangePassword(t *testing.T) {
	svc := newService()
	token, _ := svc.Login(context.Background(), registerUser)

	cases := map[string]struct {
		token       string
		password    string
		oldPassword string
		err         error
	}{
		"valid user change password ":                    {token, "newpassword", registerUser.Password, nil},
		"valid user change password with wrong password": {token, "newpassword", "wrongpassword", errors.ErrAuthentication},
		"valid user change password invalid token":       {"", "newpassword", registerUser.Password, errors.ErrAuthentication},
	}

	for desc, tc := range cases {
		err := svc.ChangePassword(context.Background(), tc.token, tc.password, tc.oldPassword)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

	}
}

func TestResetPassword(t *testing.T) {
	svc := newService()
	authSvc := mocks.NewAuthService("", []users.User{registerUser})

	resetToken, err := authSvc.Issue(context.Background(), &protomfx.IssueReq{Id: registerUser.ID, Email: registerUser.Email, Type: 2})
	assert.Nil(t, err, fmt.Sprintf("Generating reset token expected to succeed: %s", err))
	cases := map[string]struct {
		token    string
		password string
		err      error
	}{
		"valid user reset password ":   {resetToken.GetValue(), registerUser.Email, nil},
		"invalid user reset password ": {"", "newpassword", errors.ErrAuthentication},
	}

	for desc, tc := range cases {
		err := svc.ResetPassword(context.Background(), tc.token, tc.password)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestSendPasswordReset(t *testing.T) {
	svc := newService()
	token, _ := svc.Login(context.Background(), registerUser)

	cases := map[string]struct {
		token string
		email string
		err   error
	}{
		"valid user reset password ": {token, registerUser.Email, nil},
	}

	for desc, tc := range cases {
		err := svc.SendPasswordReset(context.Background(), host, tc.email, tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

	}
}
