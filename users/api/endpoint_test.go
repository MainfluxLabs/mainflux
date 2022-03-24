// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/users"
	"github.com/mainflux/mainflux/users/api"
	"github.com/mainflux/mainflux/users/bcrypt"
	"github.com/mainflux/mainflux/users/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	contentType       = "application/json"
	validEmail        = "user@example.com"
	adminEmail        = "admin@example.com"
	invalidEmail      = "userexample.com"
	validPass         = "password"
	invalidPass       = "wrong"
	memberRelationKey = "member"
	authoritiesObjKey = "authorities"
)

var (
	user               = users.User{Email: validEmail, Password: validPass}
	admin              = users.User{Email: adminEmail, Password: validPass}
	notFoundRes        = toJSON(errorRes{errors.ErrNotFound.Error()})
	unauthRes          = toJSON(errorRes{errors.ErrAuthentication.Error()})
	malformedRes       = toJSON(errorRes{errors.ErrMalformedEntity.Error()})
	weakPassword       = toJSON(errorRes{users.ErrPasswordFormat.Error()})
	unsupportedRes     = toJSON(errorRes{errors.ErrUnsupportedContentType.Error()})
	missingTokRes      = toJSON(apiutil.ErrorRes{Err: apiutil.ErrBearerToken.Error()})
	missingEmailRes    = toJSON(apiutil.ErrorRes{Err: apiutil.ErrMissingEmail.Error()})
	missingPassRes     = toJSON(apiutil.ErrorRes{Err: apiutil.ErrMissingPass.Error()})
	invalidRestPassRes = toJSON(apiutil.ErrorRes{Err: apiutil.ErrInvalidResetPass.Error()})
	failDecodeRes      = toJSON(apiutil.ErrorRes{Err: errors.ErrMalformedEntity.Error()})
	passRegex          = regexp.MustCompile("^.{8,}$")
)

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
	usersRepo := mocks.NewUserRepository()
	hasher := bcrypt.New()

	idProvider := uuid.New()
	id, _ := idProvider.ID()
	admin.ID = id
	mockAuthzDB := map[string][]mocks.SubjectSet{}
	mockAuthzDB[admin.ID] = []mocks.SubjectSet{{Object: authoritiesObjKey, Relation: memberRelationKey}}
	mockAuthzDB["*"] = []mocks.SubjectSet{{Object: "user", Relation: "create"}}

	auth := mocks.NewAuthService(map[string]users.User{admin.Email: admin}, mockAuthzDB)
	email := mocks.NewEmailer()
	return users.New(usersRepo, hasher, auth, email, idProvider, passRegex)
}

func newServer(svc users.Service) *httptest.Server {
	mux := api.MakeHandler(svc, mocktracer.New(), logger.NewMock())
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func TestRegister(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	data := toJSON(user)
	invalidData := toJSON(users.User{Email: invalidEmail, Password: validPass})
	invalidPasswordData := toJSON(users.User{Email: validEmail, Password: invalidPass})
	invalidFieldData := fmt.Sprintf(`{"email": "%s", "pass": "%s"}`, user.Email, user.Password)

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		token       string
	}{
		{"register new user", data, contentType, http.StatusCreated, ""},
		{"register existing user", data, contentType, http.StatusConflict, ""},
		{"register user with invalid email address", invalidData, contentType, http.StatusBadRequest, ""},
		{"register user with weak password", invalidPasswordData, contentType, http.StatusBadRequest, ""},
		{"register user with invalid request format", "{", contentType, http.StatusBadRequest, ""},
		{"register user with empty JSON request", "{}", contentType, http.StatusBadRequest, ""},
		{"register user with empty request", "", contentType, http.StatusBadRequest, ""},
		{"register user with invalid field name", invalidFieldData, contentType, http.StatusBadRequest, ""},
		{"register user with missing content type", data, "", http.StatusUnsupportedMediaType, ""},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/register", ts.URL),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestUserCreate(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	data := toJSON(user)
	invalidData := toJSON(users.User{Email: invalidEmail, Password: validPass})
	invalidPasswordData := toJSON(users.User{Email: validEmail, Password: invalidPass})
	invalidFieldData := fmt.Sprintf(`{"email": "%s", "pass": "%s"}`, user.Email, user.Password)
	unauthzEmail := "unauthz@example.com"

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		token       string
	}{
		{"create new user", data, contentType, http.StatusCreated, admin.Email},
		{"create user with empty token", data, contentType, http.StatusForbidden, ""},
		{"create existing user", data, contentType, http.StatusConflict, admin.Email},
		{"create user with invalid email address", invalidData, contentType, http.StatusBadRequest, admin.Email},
		{"create user with weak password", invalidPasswordData, contentType, http.StatusBadRequest, admin.Email},
		{"create new user with unauthorized access", data, contentType, http.StatusUnauthorized, unauthzEmail},
		{"create existing user with unauthorized access", data, contentType, http.StatusUnauthorized, unauthzEmail},
		{"create user with invalid request format", "{", contentType, http.StatusBadRequest, admin.Email},
		{"create user with empty JSON request", "{}", contentType, http.StatusBadRequest, admin.Email},
		{"create user with empty request", "", contentType, http.StatusBadRequest, admin.Email},
		{"create user with invalid field name", invalidFieldData, contentType, http.StatusBadRequest, admin.Email},
		{"create user with missing content type", data, "", http.StatusUnsupportedMediaType, admin.Email},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/users", ts.URL),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestLogin(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	mockAuthzDB := map[string][]mocks.SubjectSet{}

	auth := mocks.NewAuthService(map[string]users.User{}, mockAuthzDB)

	data := toJSON(user)
	invalidEmailData := toJSON(users.User{
		Email:    invalidEmail,
		Password: validPass,
	})
	invalidData := toJSON(users.User{
		Email:    validEmail,
		Password: "invalid_password",
	})
	nonexistentData := toJSON(users.User{
		Email:    "non-existentuser@example.com",
		Password: validPass,
	})

	id, err := svc.Register(context.Background(), "", user)
	require.Nil(t, err, fmt.Sprintf("register user got unexpected error: %s", err))
	_, err = auth.AddPolicy(context.Background(), &mainflux.AddPolicyReq{
		Sub: id,
		Act: memberRelationKey,
		Obj: authoritiesObjKey,
	})
	require.Nil(t, err, fmt.Sprintf("adding policy for user got unexpected error: %s", err))

	mfxTok, err := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: 0})
	require.Nil(t, err, fmt.Sprintf("issue token for user got unexpected error: %s", err))
	token := mfxTok.GetValue()
	tokenData := toJSON(map[string]string{"token": token})

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		res         string
	}{
		{"login with valid credentials", data, contentType, http.StatusCreated, tokenData},
		{"login with invalid credentials", invalidData, contentType, http.StatusUnauthorized, unauthRes},
		{"login with invalid email address", invalidEmailData, contentType, http.StatusBadRequest, malformedRes},
		{"login non-existent user", nonexistentData, contentType, http.StatusUnauthorized, unauthRes},
		{"login with invalid request format", "{", contentType, http.StatusBadRequest, malformedRes},
		{"login with empty JSON request", "{}", contentType, http.StatusBadRequest, malformedRes},
		{"login with empty request", "", contentType, http.StatusBadRequest, malformedRes},
		{"login with missing content type", data, "", http.StatusUnsupportedMediaType, unsupportedRes},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/tokens", ts.URL),
			contentType: tc.contentType,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		body, err := ioutil.ReadAll(res.Body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		token := strings.Trim(string(body), "\n")

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, token, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, token))
	}
}

func TestUser(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	mockAuthzDB := map[string][]mocks.SubjectSet{}

	auth := mocks.NewAuthService(map[string]users.User{}, mockAuthzDB)

	userID, err := svc.Register(context.Background(), "", user)
	require.Nil(t, err, fmt.Sprintf("register user got unexpected error: %s", err))

	tkn, err := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: 0})
	require.Nil(t, err, fmt.Sprintf("issue token got unexpected error: %s", err))
	token := tkn.GetValue()

	cases := []struct {
		desc   string
		token  string
		status int
		res    string
	}{
		{"user info with valid token", token, http.StatusOK, ""},
		{"user info with invalid token", "", http.StatusUnauthorized, ""},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/users/%s", ts.URL, userID),
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		body, err := ioutil.ReadAll(res.Body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		token := strings.Trim(string(body), "\n")

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, "", fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, token))
	}
}

func TestPasswordResetRequest(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()
	data := toJSON(user)

	nonexistentData := toJSON(users.User{
		Email:    "non-existentuser@example.com",
		Password: validPass,
	})

	expectedExisting := toJSON(struct {
		Msg string `json:"msg"`
	}{
		api.MailSent,
	})

	_, err := svc.Register(context.Background(), "", user)
	require.Nil(t, err, fmt.Sprintf("register user got unexpected error: %s", err))

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		res         string
	}{
		{"password reset request with valid email", data, contentType, http.StatusCreated, expectedExisting},
		{"password reset request with invalid email", nonexistentData, contentType, http.StatusNotFound, notFoundRes},
		{"password reset request with invalid request format", "{", contentType, http.StatusBadRequest, malformedRes},
		{"password reset request with empty JSON request", "{}", contentType, http.StatusBadRequest, missingEmailRes},
		{"password reset request with empty request", "", contentType, http.StatusBadRequest, malformedRes},
		{"password reset request with missing content type", data, "", http.StatusUnsupportedMediaType, unsupportedRes},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/password/reset-request", ts.URL),
			contentType: tc.contentType,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		body, err := ioutil.ReadAll(res.Body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		token := strings.Trim(string(body), "\n")

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, token, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, token))
	}
}

func TestPasswordReset(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()
	reqData := struct {
		Token    string `json:"token,omitempty"`
		Password string `json:"password,omitempty"`
		ConfPass string `json:"confirm_password,omitempty"`
	}{}

	mockAuthzDB := map[string][]mocks.SubjectSet{}

	auth := mocks.NewAuthService(map[string]users.User{}, mockAuthzDB)

	id, err := svc.Register(context.Background(), "", user)
	require.Nil(t, err, fmt.Sprintf("register user got unexpected error: %s", err))

	tkn, err := auth.Issue(context.Background(), &mainflux.IssueReq{Id: id, Email: user.Email, Type: 0})
	require.Nil(t, err, fmt.Sprintf("issue user token error: %s", err))

	token := tkn.GetValue()

	reqData.Password = user.Password
	reqData.ConfPass = user.Password
	reqData.Token = token
	reqExisting := toJSON(reqData)

	reqData.Token = "wrong"

	reqNoExist := toJSON(reqData)

	reqData.Token = token

	reqData.ConfPass = invalidPass
	reqPassNoMatch := toJSON(reqData)

	reqData.Password = invalidPass
	reqPassWeak := toJSON(reqData)

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		res         string
		tok         string
	}{
		{"password reset with valid token", reqExisting, contentType, http.StatusCreated, "{}", token},
		{"password reset with invalid token", reqNoExist, contentType, http.StatusUnauthorized, unauthRes, token},
		{"password reset with confirm password not matching", reqPassNoMatch, contentType, http.StatusBadRequest, invalidRestPassRes, token},
		{"password reset request with invalid request format", "{", contentType, http.StatusBadRequest, malformedRes, token},
		{"password reset request with empty JSON request", "{}", contentType, http.StatusBadRequest, missingPassRes, token},
		{"password reset request with empty request", "", contentType, http.StatusBadRequest, malformedRes, token},
		{"password reset request with missing content type", reqExisting, "", http.StatusUnsupportedMediaType, unsupportedRes, token},
		{"password reset with weak password", reqPassWeak, contentType, http.StatusBadRequest, weakPassword, token},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/password/reset", ts.URL),
			contentType: tc.contentType,
			body:        strings.NewReader(tc.req),
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		body, err := ioutil.ReadAll(res.Body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		token := strings.Trim(string(body), "\n")

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, token, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, token))
	}
}

func TestPasswordChange(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	mockAuthzDB := map[string][]mocks.SubjectSet{}
	auth := mocks.NewAuthService(map[string]users.User{}, mockAuthzDB)

	reqData := struct {
		Token    string `json:"token,omitempty"`
		Password string `json:"password,omitempty"`
		OldPassw string `json:"old_password,omitempty"`
	}{}

	id, err := svc.Register(context.Background(), "", user)
	require.Nil(t, err, fmt.Sprintf("register user got unexpected error: %s", err))

	tkn, err := auth.Issue(context.Background(), &mainflux.IssueReq{Id: id, Email: user.Email, Type: 0})
	require.Nil(t, err, fmt.Sprintf("issue token got unexpected error: %s", err))
	token := tkn.GetValue()

	reqData.Password = user.Password
	reqData.OldPassw = user.Password
	reqData.Token = token
	dataResExisting := toJSON(reqData)

	reqNoExist := toJSON(reqData)

	reqData.OldPassw = invalidPass
	reqWrongPass := toJSON(reqData)

	reqData.OldPassw = user.Password
	reqData.Password = invalidPass
	reqWeakPass := toJSON(reqData)

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		res         string
		tok         string
	}{
		{"password change with valid token", dataResExisting, contentType, http.StatusCreated, "{}", token},
		{"password change with empty token", reqNoExist, contentType, http.StatusUnauthorized, missingTokRes, ""},
		{"password change with invalid old password", reqWrongPass, contentType, http.StatusUnauthorized, unauthRes, token},
		{"password change with invalid new password", reqWeakPass, contentType, http.StatusBadRequest, weakPassword, token},
		{"password change with empty JSON request", "{}", contentType, http.StatusBadRequest, missingPassRes, token},
		{"password change empty request", "", contentType, http.StatusBadRequest, malformedRes, token},
		{"password change missing content type", dataResExisting, "", http.StatusUnsupportedMediaType, unsupportedRes, token},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/password", ts.URL),
			contentType: tc.contentType,
			body:        strings.NewReader(tc.req),
			token:       tc.tok,
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		body, err := ioutil.ReadAll(res.Body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		token := strings.Trim(string(body), "\n")

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, token, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, token))
	}
}

type errorRes struct {
	Err string `json:"error"`
}
