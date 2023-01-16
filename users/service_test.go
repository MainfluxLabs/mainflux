// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"

	"github.com/MainfluxLabs/mainflux/users/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	wrong   = "wrong-value"
	userNum = 101
)

var (
	userAdmin       = users.User{Email: "admin@example.com", ID: "574106f7-030e-4881-8ab0-151195c29f94", Password: "password", Status: "enabled"}
	unauthUser      = users.User{Email: "unauthUser@example.com", ID: "6a32810a-4451-4ae8-bf7f-4b1752856eef", Password: "password", Metadata: map[string]interface{}{"role": "user"}}
	selfRegister    = users.User{Email: "selfRegister@example.com", Password: "password", Metadata: map[string]interface{}{"role": "user"}}
	user            = users.User{Email: "user@example.com", Password: "password", Metadata: map[string]interface{}{"role": "user"}}
	nonExistingUser = users.User{Email: "non-ex-user@example.com", Password: "password", Metadata: map[string]interface{}{"role": "user"}}
	host            = "example.com"

	idProvider = uuid.New()
	passRegex  = regexp.MustCompile("^.{8,}$")
)

func newService() users.Service {
	userRepo := mocks.NewUserRepository()
	hasher := mocks.NewHasher()

	mockAuthzDB := map[string][]mocks.SubjectSet{}

	mockAuthzDB[userAdmin.ID] = []mocks.SubjectSet{{Object: "authorities", Relation: "member"}}
	mockAuthzDB["*"] = []mocks.SubjectSet{{Object: "user", Relation: "create"}}

	mockUsers := map[string]users.User{userAdmin.Email: userAdmin, unauthUser.Email: unauthUser}

	authSvc := mocks.NewAuthService(mockUsers, mockAuthzDB)
	e := mocks.NewEmailer()

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
			user:  user,
			token: userAdmin.Email,
			err:   nil,
		},
		{
			desc:  "register existing user",
			user:  user,
			token: userAdmin.Email,
			err:   errors.ErrConflict,
		},
		{
			desc: "register new user with weak password",
			user: users.User{
				Email:    user.Email,
				Password: "weak",
			},
			token: userAdmin.Email,
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
	_, err := svc.SelfRegister(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	noAuthUser := users.User{
		Email:    "email@test.com",
		Password: "12345678",
	}

	cases := map[string]struct {
		user users.User
		err  error
	}{
		"login with good credentials": {
			user: user,
			err:  nil,
		},
		"login with wrong e-mail": {
			user: users.User{
				Email:    wrong,
				Password: user.Password,
			},
			err: errors.ErrAuthentication,
		},
		"login with wrong password": {
			user: users.User{
				Email:    user.Email,
				Password: wrong,
			},
			err: errors.ErrAuthentication,
		},
		"login failed auth": {
			user: noAuthUser,
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
	id, err := svc.SelfRegister(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	token, err := svc.Login(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	u := user
	u.Password = ""

	cases := map[string]struct {
		user   users.User
		token  string
		userID string
		err    error
	}{
		"view user with authorized token": {
			user:   u,
			token:  token,
			userID: id,
			err:    nil,
		},
		"view user with empty token": {
			user:   users.User{},
			token:  "",
			userID: id,
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
		_, err := svc.ViewUser(context.Background(), tc.token, tc.userID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestViewProfile(t *testing.T) {
	svc := newService()
	_, err := svc.SelfRegister(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	token, err := svc.Login(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	u := user
	u.Password = ""

	cases := map[string]struct {
		user  users.User
		token string
		err   error
	}{
		"valid token's user info": {
			user:  u,
			token: token,
			err:   nil,
		},
		"invalid token's user info": {
			user:  users.User{},
			token: "",
			err:   errors.ErrAuthentication,
		},
	}

	for desc, tc := range cases {
		_, err := svc.ViewProfile(context.Background(), tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListUsers(t *testing.T) {
	svc := newService()

	token, err := svc.Login(context.Background(), userAdmin)
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

	_, err := svc.SelfRegister(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	token, err := svc.Login(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	user.Metadata = map[string]interface{}{"role": "test"}

	cases := map[string]struct {
		user  users.User
		token string
		err   error
	}{
		"update user with valid token": {
			user:  user,
			token: token,
			err:   nil,
		},
		"update user with invalid token": {
			user:  user,
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
	_, err := svc.SelfRegister(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		email string
		err   error
	}{
		"valid user reset token":  {user.Email, nil},
		"invalid user rest token": {nonExistingUser.Email, errors.ErrNotFound},
	}

	for desc, tc := range cases {
		err := svc.GenerateResetToken(context.Background(), tc.email, host)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestChangePassword(t *testing.T) {
	svc := newService()
	_, err := svc.SelfRegister(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("register user error: %s", err))
	token, _ := svc.Login(context.Background(), user)

	cases := map[string]struct {
		token       string
		password    string
		oldPassword string
		err         error
	}{
		"valid user change password ":                    {token, "newpassword", user.Password, nil},
		"valid user change password with wrong password": {token, "newpassword", "wrongpassword", errors.ErrAuthentication},
		"valid user change password invalid token":       {"", "newpassword", user.Password, errors.ErrAuthentication},
	}

	for desc, tc := range cases {
		err := svc.ChangePassword(context.Background(), tc.token, tc.password, tc.oldPassword)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

	}
}

func TestResetPassword(t *testing.T) {
	svc := newService()
	_, err := svc.SelfRegister(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	mockAuthzDB := map[string][]mocks.SubjectSet{}
	mockAuthzDB[user.Email] = append(mockAuthzDB[user.Email], mocks.SubjectSet{Object: "authorities", Relation: "member"})
	authSvc := mocks.NewAuthService(map[string]users.User{user.Email: user}, mockAuthzDB)

	resetToken, err := authSvc.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: 2})
	assert.Nil(t, err, fmt.Sprintf("Generating reset token expected to succeed: %s", err))
	cases := map[string]struct {
		token    string
		password string
		err      error
	}{
		"valid user reset password ":   {resetToken.GetValue(), user.Email, nil},
		"invalid user reset password ": {"", "newpassword", errors.ErrAuthentication},
	}

	for desc, tc := range cases {
		err := svc.ResetPassword(context.Background(), tc.token, tc.password)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestSendPasswordReset(t *testing.T) {
	svc := newService()
	_, err := svc.SelfRegister(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("register user error: %s", err))
	token, _ := svc.Login(context.Background(), user)

	cases := map[string]struct {
		token string
		email string
		err   error
	}{
		"valid user reset password ": {token, user.Email, nil},
	}

	for desc, tc := range cases {
		err := svc.SendPasswordReset(context.Background(), host, tc.email, tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

	}
}
