// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	httpapi "github.com/MainfluxLabs/mainflux/users/api/http"
	usmocks "github.com/MainfluxLabs/mainflux/users/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	contentType  = "application/json"
	validEmail   = "user@example.com"
	adminEmail   = "admin@example.com"
	invalidEmail = "userexample.com"
	validPass    = "password"
	invalidToken = "invalid"
	invalidPass  = "wrong"
	prefix       = "fe6b4e92-cc98-425e-b0aa-"
	userNum      = 101
)

var (
	user               = users.User{Email: validEmail, ID: "574106f7-030e-4881-8ab0-151195c29f94", Password: validPass, Status: "enabled"}
	admin              = users.User{Email: adminEmail, ID: "371106m2-131g-5286-2mc1-540295c29f95", Password: validPass, Status: "enabled"}
	newUser            = users.User{Email: "newuser@example.com", Password: validPass, Status: "enabled"}
	usersList          = []users.User{admin, user}
	metadata           = map[string]interface{}{"key": "value"}
	notFoundRes        = toJSON(apiutil.ErrorRes{Err: errors.ErrNotFound.Error()})
	unauthRes          = toJSON(apiutil.ErrorRes{Err: errors.ErrAuthentication.Error()})
	weakPassword       = toJSON(apiutil.ErrorRes{Err: users.ErrPasswordFormat.Error()})
	malformedRes       = toJSON(apiutil.ErrorRes{Err: apiutil.ErrMalformedEntity.Error()})
	unsupportedRes     = toJSON(apiutil.ErrorRes{Err: apiutil.ErrUnsupportedContentType.Error()})
	missingTokRes      = toJSON(apiutil.ErrorRes{Err: apiutil.ErrBearerToken.Error()})
	missingEmailRes    = toJSON(apiutil.ErrorRes{Err: apiutil.ErrMissingEmail.Error()})
	missingPassRes     = toJSON(apiutil.ErrorRes{Err: apiutil.ErrMissingPass.Error()})
	invalidRestPassRes = toJSON(apiutil.ErrorRes{Err: apiutil.ErrInvalidResetPass.Error()})
	idProvider         = uuid.New()
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
	usersRepo := usmocks.NewUserRepository(usersList)
	hasher := usmocks.NewHasher()
	auth := mocks.NewAuthService(admin.ID, usersList)
	email := usmocks.NewEmailer()
	return users.New(usersRepo, hasher, auth, email, idProvider, passRegex)
}

func newServer(svc users.Service) *httptest.Server {
	mux := httpapi.MakeHandler(svc, mocktracer.New(), logger.NewMock())
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func TestSelfRegister(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	data := toJSON(newUser)
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

func TestRegister(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()
	data := toJSON(newUser)
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
		{"create new user", data, contentType, http.StatusCreated, admin.Email},
		{"create user with empty token", data, contentType, http.StatusForbidden, ""},
		{"create existing user", data, contentType, http.StatusConflict, admin.Email},
		{"create user with invalid email address", invalidData, contentType, http.StatusBadRequest, admin.Email},
		{"create user with weak password", invalidPasswordData, contentType, http.StatusBadRequest, admin.Email},
		{"create new user with unauthorized access", data, contentType, http.StatusUnauthorized, "wrong"},
		{"create existing user with unauthorized access", data, contentType, http.StatusUnauthorized, "wrong"},
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

	auth := mocks.NewAuthService("", usersList)

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

	mfxTok, err := auth.Issue(context.Background(), &protomfx.IssueReq{Id: user.ID, Email: user.Email, Type: 0})
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

	auth := mocks.NewAuthService("", usersList)

	tkn, err := auth.Issue(context.Background(), &protomfx.IssueReq{Id: user.ID, Email: user.Email, Type: 0})
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
			url:    fmt.Sprintf("%s/users/%s", ts.URL, user.ID),
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

func TestListUsers(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	token, err := svc.Login(context.Background(), admin)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var data []viewUserRes
	data = append(data, viewUserRes{admin.ID, admin.Email}, viewUserRes{user.ID, user.Email})
	for i := 1; i < userNum; i++ {
		id := fmt.Sprintf("%s%012d", prefix, i)
		email := fmt.Sprintf("users%d@example.com", i)
		user := users.User{
			ID:       id,
			Email:    email,
			Password: "password",
			Status:   "enabled",
		}
		usersList, err := svc.Register(context.Background(), token, user)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		data = append(data, viewUserRes{usersList, email})
	}

	sort.Slice(data, func(i, j int) bool {
		return data[i].Email < data[j].Email
	})

	cases := []struct {
		desc   string
		url    string
		token  string
		status int
		res    []viewUserRes
	}{
		{
			desc:   "get list of users",
			url:    fmt.Sprintf("%s/users%s", ts.URL, ""),
			token:  token,
			status: http.StatusOK,
			res:    data[0:10],
		},
		{
			desc:   "get list of users with limit",
			url:    fmt.Sprintf("%s/users?offset=%d&limit=%d", ts.URL, 0, 10),
			token:  token,
			status: http.StatusOK,
			res:    data[0:10],
		},
		{
			desc:   "get list of all users with no limit",
			url:    fmt.Sprintf("%s/users?limit=%d", ts.URL, -1),
			token:  token,
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "get list of users with limit ordered by name descendent",
			url:    fmt.Sprintf("%s/users?offset=%d&limit=%d&order=name&dir=desc", ts.URL, 0, 5),
			token:  token,
			status: http.StatusOK,
			res:    data[0:5],
		},
		{
			desc:   "get list of users with invalid token",
			url:    fmt.Sprintf("%s/users?offset=%d&limit=%d&order=wrong", ts.URL, 0, 5),
			token:  invalidToken,
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "get list of users with empty token",
			url:    fmt.Sprintf("%s/users?offset=%d&limit=%d", ts.URL, 0, 1),
			token:  "",
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "get list of users with invalid offset",
			url:    fmt.Sprintf("%s/users?offset=%d&limit=%d", ts.URL, -1, 5),
			token:  token,
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "get list of users with invalid limit",
			url:    fmt.Sprintf("%s/users?offset=%d&limit=%d", ts.URL, 1, -5),
			token:  token,
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "get list of users with zero limit",
			url:    fmt.Sprintf("%s/users?offset=%d&limit=%d", ts.URL, 1, 0),
			token:  token,
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "get list of users without offset",
			url:    fmt.Sprintf("%s/users?limit=%d", ts.URL, 5),
			token:  token,
			status: http.StatusOK,
			res:    data[0:5],
		},
		{
			desc:   "get list of users with redundant query params",
			url:    fmt.Sprintf("%s/users?offset=%d&limit=%d&value=something", ts.URL, 0, 5),
			token:  token,
			status: http.StatusOK,
			res:    data[0:5],
		},
		{
			desc:   "get list of users with limit greater than max",
			url:    fmt.Sprintf("%s/users?offset=%d&limit=%d", ts.URL, 0, 110),
			token:  token,
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "get list of users with invalid number of params",
			url:    fmt.Sprintf("%s/users?offset=%d&limit=%d&limit=%d&offset=%d", ts.URL, 4, 4, 5, 5),
			token:  token,
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "get list of users with invalid offset",
			url:    fmt.Sprintf("%s/users?offset=%s&limit=%d", ts.URL, "s", 5),
			token:  token,
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "get list of users with invalid limit",
			url:    fmt.Sprintf("%s/users?offset=%d&limit=%s", ts.URL, 0, "s"),
			token:  token,
			status: http.StatusBadRequest,
			res:    nil,
		},
	}
	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var data userRes
		err = json.NewDecoder(res.Body).Decode(&data)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, data.Users, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, data.Users))
	}

}

func TestUpdateUser(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	token, err := svc.Login(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	data := toJSON(metadata)
	emptyData := toJSON(map[string]interface{}{})

	cases := []struct {
		desc     string
		token    string
		metadata string
		status   int
	}{
		{
			desc:     "update existing users metadata",
			token:    token,
			metadata: data,
			status:   http.StatusOK,
		},
		{
			desc:     "update existing users metadata with empty metadata",
			token:    token,
			metadata: emptyData,
			status:   http.StatusOK,
		},
		{
			desc:     "update existing users metadata with empty token",
			token:    "",
			metadata: data,
			status:   http.StatusUnauthorized,
		},
		{
			desc:     "update existing users metadata with invalid token",
			token:    invalidToken,
			metadata: data,
			status:   http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodPut,
			url:    fmt.Sprintf("%s/users", ts.URL),
			token:  tc.token,
			body:   strings.NewReader(tc.metadata),
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
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
		httpapi.MailSent,
	})

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

	auth := mocks.NewAuthService("", usersList)

	tkn, err := auth.Issue(context.Background(), &protomfx.IssueReq{Id: user.ID, Email: user.Email, Type: 0})
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

	auth := mocks.NewAuthService("", usersList)

	reqData := struct {
		Token    string `json:"token,omitempty"`
		Password string `json:"password,omitempty"`
		OldPassw string `json:"old_password,omitempty"`
	}{}

	tkn, err := auth.Issue(context.Background(), &protomfx.IssueReq{Id: user.ID, Email: user.Email, Type: 0})
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

type viewUserRes struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
}

type userRes struct {
	pageRes
	Users []viewUserRes `json:"users"`
}
