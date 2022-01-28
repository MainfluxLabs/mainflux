// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/users"

	"github.com/mainflux/mainflux/users/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const wrong string = "wrong-value"

var (
	userAdmin       = users.User{Email: "admin@example.com", ID: "574106f7-030e-4881-8ab0-151195c29f94", Password: "password", Metadata: map[string]interface{}{"role": "user"}}
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

func TestRegister(t *testing.T) {
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
		{
			desc:  "register a new user with unauthorized access",
			user:  users.User{Email: "newuser@example.com", Password: "12345678"},
			err:   errors.ErrAuthorization,
			token: unauthUser.Email,
		},
	}

	for _, tc := range cases {
		_, err := svc.Register(context.Background(), tc.token, tc.user)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestLogin(t *testing.T) {
	svc := newService()
	_, err := svc.Register(context.Background(), "", user)
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
	id, err := svc.Register(context.Background(), "", user)
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
			err:    errors.ErrAuthentication,
		},
	}

	for desc, tc := range cases {
		_, err := svc.ViewUser(context.Background(), tc.token, tc.userID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestViewProfile(t *testing.T) {
	svc := newService()
	_, err := svc.Register(context.Background(), "", user)
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

	page, err := svc.ListUsers(context.Background(), token, 0, 0, "", nil)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	totUser := page.Total

	var nUsers = uint64(10)

	for i := uint64(1); i <= nUsers; i++ {
		email := fmt.Sprintf("TestListUsers%d@example.com", i)
		user := users.User{
			Email:    email,
			Password: "passpass",
		}
		_, err := svc.Register(context.Background(), token, user)
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
			size:  0,
			err:   nil,
		},
		"list user with emtpy token": {
			token: "",
			size:  0,
			err:   errors.ErrAuthentication,
		},
		"list users with offset and limit": {
			token:  token,
			offset: 6,
			limit:  totUser,
			size:   totUser - 6,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListUsers(context.Background(), tc.token, tc.offset, tc.limit, tc.email, nil)
		size := uint64(len(page.Users))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateUser(t *testing.T) {
	svc := newService()

	_, err := svc.Register(context.Background(), "", user)
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
	_, err := svc.Register(context.Background(), "", user)
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
	_, err := svc.Register(context.Background(), "", user)
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
	_, err := svc.Register(context.Background(), "", user)
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
	_, err := svc.Register(context.Background(), "", user)
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
