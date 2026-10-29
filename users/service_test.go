// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	usmocks "github.com/MainfluxLabs/mainflux/users/mocks"
	jose "github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

const (
	wrong   = "wrong-value"
	userNum = 101

	inviteDuration     = 7 * 24 * time.Hour
	inviteRedirectPath = "/register/invite"
)

var (
	admin           = users.User{Email: "admin@example.com", ID: "574106f7-030e-4881-8ab0-151195c29f94", Role: "root", Password: "password"}
	noPasswordAdmin = users.User{Email: "admin@example.com", ID: "574106f7-030e-4881-8ab0-151195c29f94", Role: "root"}
	unauthUser      = users.User{Email: "unauth-user@example.com", ID: "6a32810a-4451-4ae8-bf7f-4b1752856eef", Password: "password"}
	selfRegister    = users.User{Email: "self-register@example.com", Password: "password"}
	registerUser    = users.User{Email: "register-user@example.com", ID: "574106f7-030e-4881-8ab0-151195c29f95", Password: "password"}
	user            = users.User{Email: "user@example.com", ID: "574106f7-030e-4881-8ab0-151195c29f96", Password: "password"}
	noPasswordUser  = users.User{Email: "user@example.com", ID: "574106f7-030e-4881-8ab0-151195c29f96"}
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

func newService(srv ...*httptest.Server) users.Service {
	hasher := usmocks.NewHasher()
	allUsers := usersList
	var oauthGoogleCfg, oauthGithubCfg oauth2.Config
	var cfgURLs users.ConfigURLs

	if len(srv) > 0 && srv[0] != nil {
		oauthUsers := []users.User{
			{Email: "new-google-user@example.com", ID: "oauth-google-id-001", Status: users.EnabledStatusKey},
			{Email: "new-github-user@example.com", ID: "oauth-github-id-001", Status: users.EnabledStatusKey},
		}
		allUsers = append(usersList, oauthUsers...)
		oauthGoogleCfg = oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: srv[0].URL + "/token"}}
		oauthGithubCfg = oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: srv[0].URL + "/token"}}
		cfgURLs = users.ConfigURLs{
			GoogleUserInfoURL:   srv[0].URL + "/google/userinfo",
			GitHubUserInfoURL:   srv[0].URL + "/github/user",
			GitHubUserEmailsURL: srv[0].URL + "/github/emails",
			RedirectLoginURL:    "https://example.com/login",
		}
	}

	userRepo := usmocks.NewUserRepository(allUsers)
	verificationRepo := usmocks.NewEmailVerificationRepository(verificationsList)
	invitesRepo := usmocks.NewPlatformInvitesRepository()
	identityRepo := usmocks.NewIdentityRepository()
	authSvc := mocks.NewAuthService(admin.ID, allUsers, nil)
	e := usmocks.NewEmailer()
	c := users.Config{
		InviteDuration:      inviteDuration,
		EmailVerifyEnabled:  true,
		SelfRegisterEnabled: true,
		GoogleOAuth:         oauthGoogleCfg,
		GitHubOAuth:         oauthGithubCfg,
		URLs:                cfgURLs,
	}
	return users.New(userRepo, verificationRepo, invitesRepo, identityRepo, hasher, authSvc, e, idProvider, c)
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

func TestRefresh(t *testing.T) {
	svc := newService()

	token, err := svc.Login(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		token string
		err   error
	}{
		"refresh with a valid token": {
			token: token,
			err:   nil,
		},
		"refresh with an invalid token": {
			token: wrong,
			err:   errors.ErrAuthentication,
		},
		"refresh with an empty token": {
			token: "",
			err:   errors.ErrAuthentication,
		},
	}

	for desc, tc := range cases {
		_, err := svc.RefreshToken(context.Background(), tc.token)
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
			user:   noPasswordUser,
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
			user:  noPasswordUser,
			token: token,
			err:   nil,
		},
		"valid token's admin info": {
			user:  noPasswordAdmin,
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

	resetToken, err := authSvc.Issue(context.Background(), registerUser.ID, registerUser.Email, domain.RecoveryKey)
	assert.Nil(t, err, fmt.Sprintf("Generating reset token expected to succeed: %s", err))
	cases := map[string]struct {
		token    string
		password string
		err      error
	}{
		"valid user reset password ":   {resetToken, registerUser.Email, nil},
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

	existingInvite, err := svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, "existingUser@example.com", auth.OrgInvite{})
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
		_, err := svc.CreatePlatformInvite(context.Background(), tc.token, inviteRedirectPath, tc.email, auth.OrgInvite{})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRevokePlatformInvite(t *testing.T) {
	svc := newService()
	tokenAdmin, err := svc.Login(context.Background(), admin)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	tokenUser, err := svc.Login(context.Background(), user)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	pendingInvite, err := svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, "test1@example.com", auth.OrgInvite{})
	assert.Nil(t, err, fmt.Sprintf("Creating platform invite expected to succeed: %s", err))

	acceptedInvite, err := svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, "test2@example.com", auth.OrgInvite{})
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

	pendingInvite, err := svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, "test1@example.com", auth.OrgInvite{})
	assert.Nil(t, err, fmt.Sprintf("Creating platform invite expected to succeed: %s", err))

	cases := map[string]struct {
		inviteID string
		err      error
	}{
		"view platform invite":                 {pendingInvite.ID, nil},
		"view platform invite with invalid id": {"invalid-id", dbutil.ErrNotFound},
	}

	for desc, tc := range cases {
		_, err := svc.ViewPlatformInvite(context.Background(), tc.inviteID)
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
		_, err := svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, fmt.Sprintf("test%d@example.com", i), auth.OrgInvite{})
		assert.Nil(t, err, fmt.Sprintf("Creating platform invite expected to succeed: %s", err))
	}

	cases := map[string]struct {
		token string
		pm    users.PageMetadata
		size  uint64
		err   error
	}{
		"list platform invites":                        {tokenAdmin, users.PageMetadata{Limit: n}, n, nil},
		"list half platform invites":                   {tokenAdmin, users.PageMetadata{Limit: n / 2}, n / 2, nil},
		"list last platform invite":                    {tokenAdmin, users.PageMetadata{Limit: 1, Offset: n - 1}, 1, nil},
		"list platform invites as non-root-admin user": {tokenUser, users.PageMetadata{}, 0, errors.ErrAuthorization},
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

	pendingInvite, err := svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, "test1@example.com", auth.OrgInvite{})
	assert.Nil(t, err, fmt.Sprintf("Creating platform invite expected to succeed: %s", err))

	pendingInvite2, err := svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, "test11@example.com", auth.OrgInvite{})
	assert.Nil(t, err, fmt.Sprintf("Creating platform invite expected to succeed: %s", err))

	revokedInvite, err := svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, "test2@example.com", auth.OrgInvite{})
	assert.Nil(t, err, fmt.Sprintf("Creating platform invite expected to succeed: %s", err))
	err = svc.RevokePlatformInvite(context.Background(), tokenAdmin, revokedInvite.ID)
	assert.Nil(t, err, fmt.Sprintf("Revoking platform invite expected to succeed: %s", err))

	acceptedInvite, err := svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, "test3@example.com", auth.OrgInvite{})
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

func TestOAuthLogin(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc          string
		provider      string
		stateNotEmpty bool
		err           error
	}{
		{
			desc:          "oauth login with google provider",
			provider:      users.GoogleProvider,
			stateNotEmpty: true,
			err:           nil,
		},
		{
			desc:          "oauth login with github provider",
			provider:      users.GitHubProvider,
			stateNotEmpty: true,
			err:           nil,
		},
		{
			desc:          "oauth login with unconfigured oidc provider",
			provider:      users.OIDCProvider,
			stateNotEmpty: false,
			err:           apiutil.ErrInvalidProvider,
		},
		{
			desc:          "oauth login with unknown provider",
			provider:      "unknown",
			stateNotEmpty: false,
			err:           apiutil.ErrInvalidProvider,
		},
	}

	for _, tc := range cases {
		data, err := svc.OAuthLogin(tc.provider)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected err %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.stateNotEmpty, data.State != "", fmt.Sprintf("%s: expected state non-empty=%v\n", tc.desc, tc.stateNotEmpty))
	}

	data1, err := svc.OAuthLogin(users.GoogleProvider)
	require.Nil(t, err)
	data2, err := svc.OAuthLogin(users.GoogleProvider)
	require.Nil(t, err)
	assert.NotEqual(t, data1.State, data2.State, "each OAuthLogin call must generate a unique state")
}

func TestOAuthCallback(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"access_token": "valid-access-token", "token_type": "Bearer", "expires_in": 3600})
	})
	mux.HandleFunc("/google/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "new-google-provider-id", "email": "new-google-user@example.com"})
	})
	mux.HandleFunc("/github/user", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": 99999})
	})
	mux.HandleFunc("/github/emails", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{{"email": "new-github-user@example.com", "primary": true, "verified": true}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	svc := newService(srv)

	verifier := oauth2.GenerateVerifier()

	cases := []struct {
		desc                string
		provider            string
		code                string
		verifier            string
		err                 error
		redirectURLContains string
	}{
		{
			desc:                "callback creates new google user and returns login redirect",
			provider:            users.GoogleProvider,
			code:                "valid-code",
			verifier:            verifier,
			redirectURLContains: "token=",
		},
		{
			desc:                "callback creates new github user and returns login redirect",
			provider:            users.GitHubProvider,
			code:                "valid-code",
			verifier:            verifier,
			redirectURLContains: "token=",
		},
		{
			desc:                "callback authenticates already-linked google user",
			provider:            users.GoogleProvider,
			code:                "valid-code",
			verifier:            verifier,
			redirectURLContains: "token=",
		},
		{
			desc:     "callback with unknown provider returns error",
			provider: "unknown",
			code:     "valid-code",
			verifier: verifier,
			err:      apiutil.ErrInvalidProvider,
		},
	}

	for _, tc := range cases {
		redirectURL, err := svc.OAuthCallback(context.Background(), users.OAuthCallbackData{
			Provider: tc.provider,
			Code:     tc.code,
			Verifier: tc.verifier,
		})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected err %s got %s\n", tc.desc, tc.err, err))
		if tc.redirectURLContains != "" {
			assert.Contains(t, redirectURL, tc.redirectURLContains, fmt.Sprintf("%s: redirect URL should contain %q\n", tc.desc, tc.redirectURLContains))
		}
	}
}

func TestOIDCCallback(t *testing.T) {
	const (
		oidcClientID = "mfx-client"
		oidcKeyID    = "test-key"
	)

	signingKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.Nil(t, err)
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.Nil(t, err)

	var srv *httptest.Server
	var idToken string
	var userInfoClaims map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                srv.URL,
			"authorization_endpoint":                srv.URL + "/oidc/auth",
			"token_endpoint":                        srv.URL + "/oidc/token",
			"jwks_uri":                              srv.URL + "/oidc/jwks",
			"userinfo_endpoint":                     srv.URL + "/oidc/userinfo",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/oidc/jwks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jose.JSONWebKeySet{
			Keys: []jose.JSONWebKey{
				{
					Key:       &signingKey.PublicKey,
					KeyID:     oidcKeyID,
					Algorithm: string(jose.RS256),
					Use:       "sig",
				},
			},
		})
	})
	mux.HandleFunc("/oidc/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "valid-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
			"id_token":     idToken,
		})
	})
	mux.HandleFunc("/oidc/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(userInfoClaims)
	})
	srv = httptest.NewServer(mux)
	defer srv.Close()

	signIDToken := func(key *rsa.PrivateKey, claims map[string]any) string {
		signerOpts := (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", oidcKeyID)
		signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, signerOpts)
		require.Nil(t, err)

		payload, err := json.Marshal(claims)
		require.Nil(t, err)

		signed, err := signer.Sign(payload)
		require.Nil(t, err)

		serialized, err := signed.CompactSerialize()
		require.Nil(t, err)
		return serialized
	}

	tokenClaims := func(sub, audience string, extra map[string]any) map[string]any {
		claims := map[string]any{
			"iss": srv.URL,
			"aud": audience,
			"sub": sub,
			"iat": time.Now().Unix(),
			"exp": time.Now().Add(time.Hour).Unix(),
		}
		for key, value := range extra {
			claims[key] = value
		}
		return claims
	}

	newOIDCUser := users.User{Email: "new-oidc-user@example.com", ID: "oidc-user-id-001", Status: users.EnabledStatusKey}
	userRepo := usmocks.NewUserRepository(usersList)
	authSvc := mocks.NewAuthService(admin.ID, append(usersList, newOIDCUser), nil)
	c := users.Config{
		InviteDuration:      inviteDuration,
		SelfRegisterEnabled: true,
		OIDC: users.OIDCConfig{
			IssuerURL: srv.URL,
			OAuth: oauth2.Config{
				ClientID: oidcClientID,
			},
		},
		URLs: users.ConfigURLs{
			RedirectLoginURL: "https://example.com/login",
		},
	}
	svc := users.New(userRepo, usmocks.NewEmailVerificationRepository(nil), usmocks.NewPlatformInvitesRepository(), usmocks.NewIdentityRepository(), usmocks.NewHasher(), authSvc, usmocks.NewEmailer(), idProvider, c)

	loginData, err := svc.OAuthLogin(users.OIDCProvider)
	require.Nil(t, err)
	assert.Contains(t, loginData.RedirectURL, srv.URL+"/oidc/auth", "oidc login should redirect to the discovered authorization endpoint")

	verifier := oauth2.GenerateVerifier()

	cases := []struct {
		desc           string
		idToken        string
		userInfoClaims map[string]any
		err            error
	}{
		{
			desc:    "callback creates new oidc user",
			idToken: signIDToken(signingKey, tokenClaims("oidc-sub-001", oidcClientID, map[string]any{"email": newOIDCUser.Email, "email_verified": true})),
			err:     nil,
		},
		{
			desc:    "callback authenticates already-linked oidc user",
			idToken: signIDToken(signingKey, tokenClaims("oidc-sub-001", oidcClientID, map[string]any{"email": newOIDCUser.Email})),
			err:     nil,
		},
		{
			desc:    "callback links existing user using e-mail from userinfo endpoint",
			idToken: signIDToken(signingKey, tokenClaims("oidc-sub-002", oidcClientID, nil)),
			userInfoClaims: map[string]any{
				"sub":   "oidc-sub-002",
				"email": user.Email,
			},
			err: nil,
		},
		{
			desc:    "callback with userinfo for another subject returns error",
			idToken: signIDToken(signingKey, tokenClaims("oidc-sub-007", oidcClientID, nil)),
			userInfoClaims: map[string]any{
				"sub":   "another-sub",
				"email": user.Email,
			},
			err: errors.ErrAuthentication,
		},
		{
			desc:    "callback accepts email_verified sent as a string",
			idToken: signIDToken(signingKey, tokenClaims("oidc-sub-008", oidcClientID, map[string]any{"email": user.Email, "email_verified": "true"})),
			err:     nil,
		},
		{
			desc:    "callback with unverified e-mail sent as a string returns error",
			idToken: signIDToken(signingKey, tokenClaims("oidc-sub-009", oidcClientID, map[string]any{"email": user.Email, "email_verified": "false"})),
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "callback with unverified e-mail returns error",
			idToken: signIDToken(signingKey, tokenClaims("oidc-sub-003", oidcClientID, map[string]any{"email": user.Email, "email_verified": false})),
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "callback with id token for another audience returns error",
			idToken: signIDToken(signingKey, tokenClaims("oidc-sub-004", wrong, map[string]any{"email": user.Email})),
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "callback with id token signed by unknown key returns error",
			idToken: signIDToken(otherKey, tokenClaims("oidc-sub-005", oidcClientID, map[string]any{"email": user.Email})),
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "callback without e-mail claim returns error",
			idToken: signIDToken(signingKey, tokenClaims("oidc-sub-006", oidcClientID, nil)),
			userInfoClaims: map[string]any{
				"sub": "oidc-sub-006",
			},
			err: errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		idToken = tc.idToken
		userInfoClaims = tc.userInfoClaims

		redirectURL, err := svc.OAuthCallback(context.Background(), users.OAuthCallbackData{
			Provider: users.OIDCProvider,
			Code:     "valid-code",
			Verifier: verifier,
		})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected err %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Contains(t, redirectURL, "token=", fmt.Sprintf("%s: redirect URL should contain token\n", tc.desc))
		}
	}
}
