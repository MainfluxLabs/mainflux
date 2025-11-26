// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
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

	inviteDuration     = 7 * 24 * time.Hour
	inviteRedirectPath = "/register/invite"
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

	verification = users.EmailVerification{
		User:      users.User{Email: "example@verify.com", Password: "12345678"},
		Token:     "8a813b28-6f91-4fa5-8a18-783ffd2d27fb",
		CreatedAt: time.Now().Add(-7 * 24 * time.Hour),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	duplicateVerification = users.EmailVerification{
		User:      user,
		Token:     "8a813b28-6f91-4fa5-8a18-783ffd2d27fc",
		CreatedAt: time.Now().Add(-7 * 24 * time.Hour),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	expiredVerification = users.EmailVerification{
		User:      user,
		Token:     "8a813b28-6f91-4fa5-8a18-783ffd2d27fd",
		CreatedAt: time.Now().Add(-7 * 24 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	verificationsList = []users.EmailVerification{verification, duplicateVerification, expiredVerification}

	idProvider = uuid.New()
	passRegex  = regexp.MustCompile(`^\S{8,}$`)
)

func newService() users.Service {
	hasher := usmocks.NewHasher()
	userRepo := usmocks.NewUserRepository(usersList)
	verificationRepo := usmocks.NewEmailVerificationRepository(verificationsList)
	invitesRepo := usmocks.NewPlatformInvitesRepository()
	authSvc := mocks.NewAuthService(admin.ID, usersList, nil)
	e := usmocks.NewEmailer()

	return users.New(userRepo, verificationRepo, invitesRepo, inviteDuration, true, true, hasher, authSvc, e, idProvider)
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
			desc: "self register user with pending e-mail confirmation",
			user: selfRegister,
			err:  nil,
		},

		{
			desc:  "self register existing user",
			user:  registerUser,
			token: admin.Email,
			err:   dbutil.ErrConflict,
		},
	}

	for _, tc := range cases {
		_, err := svc.SelfRegister(context.Background(), tc.user, "")
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestVerifyEmail(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc         string
		verification users.EmailVerification
		err          error
	}{
		{
			desc:         "confirm valid verification",
			verification: verification,
			err:          nil,
		},
		{
			desc:         "confirm verification with already registered e-mail",
			verification: duplicateVerification,
			err:          dbutil.ErrConflict,
		},
		{
			desc:         "confirm expired verification",
			verification: expiredVerification,
			err:          users.ErrEmailVerificationExpired,
		},
		{
			desc:         "confirm verification with invalid token",
			verification: users.EmailVerification{Token: "7571b76e-9ce4-4128-a0d7-568f438b0e39", User: user},
			err:          errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		_, err := svc.VerifyEmail(context.Background(), tc.verification.Token)
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
		"view user as unauthorized user": {
			user:   users.User{},
			token:  token,
			userID: registerUser.ID,
			err:    errors.ErrAuthorization,
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

	registerUser.Metadata = map[string]any{"meta": "test"}

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
		"invalid user rest token": {nonExistingUser.Email, dbutil.ErrNotFound},
	}

	for desc, tc := range cases {
		err := svc.GenerateResetToken(context.Background(), tc.email, host)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestChangePassword(t *testing.T) {
	svc := newService()
	userToken, _ := svc.Login(context.Background(), registerUser)
	adminToken, _ := svc.Login(context.Background(), admin)

	cases := map[string]struct {
		token       string
		email       string
		password    string
		oldPassword string
		err         error
	}{
		"valid user change password ":                    {userToken, "", "newpassword", registerUser.Password, nil},
		"valid user change password with wrong password": {userToken, "", "newpassword", "wrongpassword", errors.ErrInvalidPassword},
		"valid user change password invalid token":       {"", "", "newpassword", registerUser.Password, errors.ErrAuthentication},

		"valid admin change user password ":            {adminToken, registerUser.Email, "newpassword", "", nil},
		"valid admin change password with wrong email": {adminToken, "wrongemail@example.com", "newpassword", "", dbutil.ErrNotFound},
		"valid admin change password invalid token":    {"", registerUser.Email, "newpassword", "", errors.ErrAuthentication},
	}

	for desc, tc := range cases {
		err := svc.ChangePassword(context.Background(), tc.token, tc.email, tc.password, tc.oldPassword)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

	}
}

func TestResetPassword(t *testing.T) {
	svc := newService()
	authSvc := mocks.NewAuthService("", []users.User{registerUser}, nil)

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

func TestCreatePlatformInvite(t *testing.T) {
	svc := newService()
	tokenAdmin, err := svc.Login(context.Background(), admin)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	tokenUser, err := svc.Login(context.Background(), user)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	existingInvite, err := svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, "existingUser@example.com", "", "")
	assert.Nil(t, err, fmt.Sprintf("Creating platform invite expected to succeed: %s", err))

	cases := map[string]struct {
		token string
		email string
		err   error
	}{
		"create valid platform invite":                               {tokenAdmin, "newUser@example.com", nil},
		"create platform invite towards reigstered user to platform": {tokenAdmin, existingInvite.InviteeEmail, dbutil.ErrConflict},
		"create platform invite as non-root-admin user":              {tokenUser, "brandNewUser@example.com", errors.ErrAuthorization},
	}

	for desc, tc := range cases {
		_, err := svc.CreatePlatformInvite(context.Background(), tc.token, inviteRedirectPath, tc.email, "", "")
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRevokePlatformInvite(t *testing.T) {
	svc := newService()
	tokenAdmin, err := svc.Login(context.Background(), admin)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	tokenUser, err := svc.Login(context.Background(), user)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	pendingInvite, err := svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, "test1@example.com", "", "")
	assert.Nil(t, err, fmt.Sprintf("Creating platform invite expected to succeed: %s", err))

	acceptedInvite, err := svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, "test2@example.com", "", "")
	assert.Nil(t, err, fmt.Sprintf("Creating platform invite expected to succeed: %s", err))
	err = svc.ValidatePlatformInvite(context.Background(), acceptedInvite.ID, acceptedInvite.InviteeEmail)
	assert.Nil(t, err, fmt.Sprintf("Validating platform invite expected to succeed: %s", err))

	cases := map[string]struct {
		token    string
		inviteID string
		err      error
	}{
		"revoke pending platform invite":                {tokenAdmin, pendingInvite.ID, nil},
		"revoke already accepted platform invite":       {tokenAdmin, acceptedInvite.ID, apiutil.ErrInvalidInviteState},
		"revoke platform invite as non-root-admin user": {tokenUser, pendingInvite.ID, errors.ErrAuthorization},
	}

	for desc, tc := range cases {
		err := svc.RevokePlatformInvite(context.Background(), tc.token, tc.inviteID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestViewPlatformInvite(t *testing.T) {
	svc := newService()
	tokenAdmin, err := svc.Login(context.Background(), admin)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	tokenUser, err := svc.Login(context.Background(), user)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	pendingInvite, err := svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, "test1@example.com", "", "")
	assert.Nil(t, err, fmt.Sprintf("Creating platform invite expected to succeed: %s", err))

	cases := map[string]struct {
		token    string
		inviteID string
		err      error
	}{
		"view platform invite":                        {tokenAdmin, pendingInvite.ID, nil},
		"view platform invite as non-root-admin user": {tokenUser, pendingInvite.ID, errors.ErrAuthorization},
	}

	for desc, tc := range cases {
		_, err := svc.ViewPlatformInvite(context.Background(), tc.token, tc.inviteID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListPlatformInvites(t *testing.T) {
	svc := newService()
	tokenAdmin, err := svc.Login(context.Background(), admin)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	tokenUser, err := svc.Login(context.Background(), user)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	n := uint64(10)

	for i := uint64(0); i < n; i++ {
		_, err := svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, fmt.Sprintf("test%d@example.com", i), "", "")
		assert.Nil(t, err, fmt.Sprintf("Creating platform invite expected to succeed: %s", err))
	}

	cases := map[string]struct {
		token string
		pm    users.PageMetadataInvites
		size  uint64
		err   error
	}{
		"list platform invites":                        {tokenAdmin, users.PageMetadataInvites{PageMetadata: apiutil.PageMetadata{Limit: n}}, n, nil},
		"list half platform invites":                   {tokenAdmin, users.PageMetadataInvites{PageMetadata: apiutil.PageMetadata{Limit: n / 2}}, n / 2, nil},
		"list last platform invite":                    {tokenAdmin, users.PageMetadataInvites{PageMetadata: apiutil.PageMetadata{Limit: 1, Offset: n - 1}}, 1, nil},
		"list platform invites as non-root-admin user": {tokenUser, users.PageMetadataInvites{}, 0, errors.ErrAuthorization},
	}

	for desc, tc := range cases {
		invitesPage, err := svc.ListPlatformInvites(context.Background(), tc.token, tc.pm)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
		assert.Equal(t, tc.size, uint64(len(invitesPage.Invites)), fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, invitesPage.Total))
	}
}

func TestValidatePlatformInvite(t *testing.T) {
	svc := newService()
	tokenAdmin, err := svc.Login(context.Background(), admin)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	pendingInvite, err := svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, "test1@example.com", "", "")
	assert.Nil(t, err, fmt.Sprintf("Creating platform invite expected to succeed: %s", err))

	pendingInvite2, err := svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, "test11@example.com", "", "")
	assert.Nil(t, err, fmt.Sprintf("Creating platform invite expected to succeed: %s", err))

	revokedInvite, err := svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, "test2@example.com", "", "")
	assert.Nil(t, err, fmt.Sprintf("Creating platform invite expected to succeed: %s", err))
	err = svc.RevokePlatformInvite(context.Background(), tokenAdmin, revokedInvite.ID)
	assert.Nil(t, err, fmt.Sprintf("Revoking platform invite expected to succeed: %s", err))

	acceptedInvite, err := svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, "test3@example.com", "", "")
	assert.Nil(t, err, fmt.Sprintf("Creating platform invite expected to succeed: %s", err))
	err = svc.ValidatePlatformInvite(context.Background(), acceptedInvite.ID, acceptedInvite.InviteeEmail)
	assert.Nil(t, err, fmt.Sprintf("Validating platform invite expected to succeed: %s", err))

	cases := map[string]struct {
		inviteID string
		email    string
		err      error
	}{
		"validate pending platform invite with matching email":           {pendingInvite.ID, pendingInvite.InviteeEmail, nil},
		"validate pending platform invite with non-matching email":       {pendingInvite2.ID, "random@email.com", errors.ErrAuthorization},
		"validate revoked platform invite with matching email":           {revokedInvite.ID, revokedInvite.InviteeEmail, errors.ErrAuthorization},
		"validate already accepted platform invite with arbitrary email": {acceptedInvite.ID, "random@email.com", errors.ErrAuthorization},
	}

	for desc, tc := range cases {
		err := svc.ValidatePlatformInvite(context.Background(), tc.inviteID, tc.email)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}
