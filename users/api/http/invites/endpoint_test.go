// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package invites_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	httpapi "github.com/MainfluxLabs/mainflux/users/api/http"
	usmocks "github.com/MainfluxLabs/mainflux/users/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

const (
	contentType = "application/json"
	validEmail  = "user@example.com"
	adminEmail  = "admin@example.com"
	validPass   = "password"

	inviteDuration     = 7 * 24 * time.Hour
	inviteRedirectPath = "/register/invite"
)

var (
	user       = users.User{Email: validEmail, ID: "574106f7-030e-4881-8ab0-151195c29f94", Password: validPass, Status: "enabled"}
	admin      = users.User{Email: adminEmail, ID: "371106m2-131g-5286-2mc1-540295c29f95", Password: validPass, Status: "enabled"}
	usersList  = []users.User{admin, user}
	idProvider = uuid.New()
	passRegex  = regexp.MustCompile(`^\S{8,}$`)

	verification = users.EmailVerification{
		User:      users.User{Email: "example@verify.com", Password: "12345678"},
		Token:     "697463fd-2708-4ca9-bf3f-c4d5d8da18f5",
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
)

type platformInviteReq struct {
	Email        string `json:"email,omitempty"`
	RedirectPath string `json:"redirect_path,omitempty"`
}

type testRequest struct {
	client      *http.Client
	method      string
	url         string
	contentType string
	token       string
	body        io.Reader
}

func (tr testRequest) make() (*http.Response, error) {
	req, err := http.NewRequest(tr.method, tr.url, tr.body)
	if err != nil {
		return nil, err
	}
	if tr.token != "" {
		req.Header.Set("Authorization", apiutil.BearerPrefix+tr.token)
	}
	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}

	req.Header.Set("Referer", "http://localhost")
	return tr.client.Do(req)
}

func newService() users.Service {
	usersRepo := usmocks.NewUserRepository(usersList)
	verificationsRepo := usmocks.NewEmailVerificationRepository(verificationsList)
	invitesRepo := usmocks.NewPlatformInvitesRepository()
	identityRepo := usmocks.NewIdentityRepository()
	hasher := usmocks.NewHasher()
	auth := mocks.NewAuthService(admin.ID, usersList, nil)
	email := usmocks.NewEmailer()
	oauthGoogleCfg := oauth2.Config{}
	oauthGithubCfg := oauth2.Config{}
	cfgURLs := users.ConfigURLs{}
	return users.New(usersRepo, verificationsRepo, invitesRepo, identityRepo, inviteDuration, true, true, hasher, auth, email, idProvider, oauthGoogleCfg, oauthGithubCfg, cfgURLs)
}

func newServer(svc users.Service) *httptest.Server {
	mux := httpapi.MakeHandler(svc, mocktracer.New(), logger.NewMock(), passRegex)
	return httptest.NewServer(mux)
}

func toJSON(data any) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func TestCreatePlatformInvite(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	client := ts.Client()

	tokenAdmin, err := svc.Login(context.Background(), admin)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s\n", err))

	tokenRegular, err := svc.Login(context.Background(), user)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s\n", err))

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		token       string
	}{
		{
			"create platform invite",
			toJSON(platformInviteReq{Email: "new@user.com", RedirectPath: inviteRedirectPath}),
			contentType,
			http.StatusCreated,
			tokenAdmin,
		},
		{
			"create platform invite as non-root-admin user",
			toJSON(platformInviteReq{Email: "new@user.com", RedirectPath: inviteRedirectPath}),
			contentType,
			http.StatusForbidden,
			tokenRegular,
		},
		{
			"create platform invite with invalid auth token",
			toJSON(platformInviteReq{Email: "new@user.com", RedirectPath: inviteRedirectPath}),
			contentType,
			http.StatusUnauthorized,
			"invalid",
		},
		{
			"create platform invite with empty redirect path",
			toJSON(platformInviteReq{Email: "new@user.com", RedirectPath: ""}),
			contentType,
			http.StatusBadRequest,
			tokenAdmin,
		},
		{
			"create platform invite with empty email",
			toJSON(platformInviteReq{Email: "", RedirectPath: inviteRedirectPath}),
			contentType,
			http.StatusBadRequest,
			tokenAdmin,
		},
		{
			"create platform invite with empty auth token",
			toJSON(platformInviteReq{Email: "new@user.com", RedirectPath: inviteRedirectPath}),
			contentType,
			http.StatusUnauthorized,
			"",
		},
		{
			"create platform invite with empty request",
			"",
			contentType,
			http.StatusBadRequest,
			tokenAdmin,
		},
		{
			"create platform invite with invalid request fromat",
			"{,",
			contentType,
			http.StatusBadRequest,
			tokenAdmin,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/invites", ts.URL),
			contentType: tc.contentType,
			body:        strings.NewReader(tc.req),
			token:       tc.token,
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestViewPlatformInvite(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	client := ts.Client()

	tokenAdmin, err := svc.Login(context.Background(), admin)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s\n", err))

	tokenRegular, err := svc.Login(context.Background(), user)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s\n", err))

	invite, err := svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, "new@user.com", auth.OrgInvite{})
	assert.Nil(t, err, fmt.Sprintf("Inviting platform member expected to succeed: %s\n", err))

	cases := []struct {
		desc     string
		inviteID string
		status   int
		token    string
	}{
		{
			"view platform invite",
			invite.ID,
			http.StatusOK,
			tokenAdmin,
		},
		{
			"view platform invite with invalid id",
			"invalid-123",
			http.StatusNotFound,
			tokenAdmin,
		},
		{
			"view platform invite as non-root-admin user",
			invite.ID,
			http.StatusForbidden,
			tokenRegular,
		},
		{
			"view platform invite with empty auth token",
			invite.ID,
			http.StatusUnauthorized,
			"",
		},
		{
			"view platform invite with invalid auth token",
			invite.ID,
			http.StatusUnauthorized,
			"invalid123",
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/invites/%s", ts.URL, tc.inviteID),
			token:  tc.token,
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestListPlatformInvites(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	client := ts.Client()

	tokenAdmin, err := svc.Login(context.Background(), admin)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s\n", err))

	tokenRegular, err := svc.Login(context.Background(), user)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s\n", err))

	_, err = svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, "new@user.com", auth.OrgInvite{})
	assert.Nil(t, err, fmt.Sprintf("Inviting platform member expected to succeed: %s\n", err))

	_, err = svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, "new1@user.com", auth.OrgInvite{})
	assert.Nil(t, err, fmt.Sprintf("Inviting platform member expected to succeed: %s\n", err))

	cases := []struct {
		desc   string
		status int
		token  string
	}{
		{
			"view platform invites",
			http.StatusOK,
			tokenAdmin,
		},
		{
			"view platform invites as non-root-admin user",
			http.StatusForbidden,
			tokenRegular,
		},
		{
			"view platform invites with empty auth token",
			http.StatusUnauthorized,
			"",
		},
		{
			"view platform invites with invalid auth token",
			http.StatusUnauthorized,
			"invalid123",
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/invites", ts.URL),
			token:  tc.token,
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestRevokePlatformInvite(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	client := ts.Client()

	tokenAdmin, err := svc.Login(context.Background(), admin)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s\n", err))

	tokenRegular, err := svc.Login(context.Background(), user)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s\n", err))

	invite, err := svc.CreatePlatformInvite(context.Background(), tokenAdmin, inviteRedirectPath, "new@user.com", auth.OrgInvite{})
	assert.Nil(t, err, fmt.Sprintf("Inviting platform member expected to succeed: %s\n", err))

	cases := []struct {
		desc     string
		inviteID string
		status   int
		token    string
	}{
		{
			"revoke platform invite",
			invite.ID,
			http.StatusNoContent,
			tokenAdmin,
		},
		{
			"revoke platform invite as non-root-admin user",
			invite.ID,
			http.StatusForbidden,
			tokenRegular,
		},
		{
			"revoke platform invite with invalid invite id",
			"invalid123",
			http.StatusNotFound,
			tokenAdmin,
		},
		{
			"revoke platform invite with empty invalid auth token",
			invite.ID,
			http.StatusUnauthorized,
			"invalid123",
		},
		{
			"revoke platform invite with empty auth token",
			invite.ID,
			http.StatusUnauthorized,
			"",
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/invites/%s", ts.URL, tc.inviteID),
			token:  tc.token,
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}
