package orgs_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	httpapi "github.com/MainfluxLabs/mainflux/auth/api/http"
	"github.com/MainfluxLabs/mainflux/auth/jwt"
	"github.com/MainfluxLabs/mainflux/auth/mocks"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	thmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	secret           = "secret"
	contentType      = "application/json"
	id               = "123e4567-e89b-12d3-a456-000000000022"
	adminID          = "adminID"
	editorID         = "editorID"
	viewerID         = "viewerID"
	email            = "user@example.com"
	adminEmail       = "admin@example.com"
	editorEmail      = "editor@example.com"
	viewerEmail      = "viewer@example.com"
	wrongValue       = "wrong_value"
	name             = "testName"
	description      = "testDesc"
	n                = 10
	loginDuration    = 30 * time.Minute
	inviteDuration   = 7 * 24 * time.Hour
	maxNameSize      = 1024
	emptyValue       = ""
	emptyJson        = "{}"
	validData        = `{"limit":5,"offset":0}`
	descData         = `{"limit":5,"offset":0,"dir":"desc","order":"name"}`
	ascData          = `{"limit":5,"offset":0,"dir":"asc","order":"name"}`
	invalidOrderData = `{"limit":5,"offset":0,"dir":"asc","order":"wrong"}`
	zeroLimitData    = `{"limit":0,"offset":0}`
	invalidDirData   = `{"limit":5,"offset":0,"dir":"wrong"}`
	invalidLimitData = `{"limit":210,"offset":0}`
	invalidData      = `{"limit": "invalid"}`
)

var (
	org = auth.Org{
		Name:        name,
		Description: description,
		Metadata:    map[string]any{"key": "value"},
	}
	idProvider      = uuid.New()
	usersByEmails   = map[string]users.User{adminEmail: {ID: adminID, Email: adminEmail}, editorEmail: {ID: editorID, Email: editorEmail}, viewerEmail: {ID: viewerID, Email: viewerEmail}, email: {ID: id, Email: email}}
	usersByIDs      = map[string]users.User{adminID: {ID: adminID, Email: adminEmail}, editorID: {ID: editorID, Email: editorEmail}, viewerID: {ID: viewerID, Email: viewerEmail}, id: {ID: id, Email: email}}
	metadata        = map[string]any{"test": "data"}
	invalidNameData = fmt.Sprintf(`{"limit":5,"offset":0,"name":"%s"}`, strings.Repeat("m", maxNameSize+1))
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

func newService() auth.Service {
	membsRepo := mocks.NewOrgMembershipsRepository()
	orgsRepo := mocks.NewOrgRepository(membsRepo)
	rolesRepo := mocks.NewRolesRepository()
	invitesRepo := mocks.NewInvitesRepository()

	idProvider := uuid.NewMock()
	t := jwt.New(secret)
	uc := mocks.NewUsersService(usersByIDs, usersByEmails)
	tc := thmocks.NewThingsServiceClient(nil, nil, nil)

	return auth.New(orgsRepo, tc, uc, nil, rolesRepo, membsRepo, invitesRepo, nil, idProvider, t, loginDuration, inviteDuration)
}

func newServer(svc auth.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(svc, mocktracer.New(), logger)
	return httptest.NewServer(mux)
}

func toJSON(data any) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func TestCreateOrg(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()

	client := ts.Client()
	data := toJSON(org)

	cases := []struct {
		desc   string
		req    string
		ct     string
		token  string
		status int
	}{
		{
			desc:   "create org",
			req:    data,
			ct:     contentType,
			token:  token,
			status: http.StatusCreated,
		},
		{
			desc:   "create org with invalid auth token",
			req:    data,
			ct:     contentType,
			token:  wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "create org with empty auth token",
			req:    data,
			ct:     contentType,
			token:  "",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "create org with empty request",
			req:    "",
			ct:     contentType,
			token:  token,
			status: http.StatusBadRequest,
		},
		{
			desc:   "create orgs with empty JSON array",
			req:    "[]",
			ct:     contentType,
			token:  token,
			status: http.StatusBadRequest,
		},
		{
			desc:   "create org with invalid request format",
			req:    "{",
			ct:     contentType,
			token:  token,
			status: http.StatusBadRequest,
		},
		{
			desc:   "create org without content type",
			req:    data,
			ct:     "",
			token:  token,
			status: http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/orgs", ts.URL),
			contentType: tc.ct,
			token:       tc.token,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestViewOrg(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	or, err := svc.CreateOrg(context.Background(), token, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	data := orgRes{
		ID:          or.ID,
		OwnerID:     or.OwnerID,
		Name:        or.Name,
		Description: or.Description,
		Metadata:    or.Metadata,
	}

	cases := []struct {
		desc   string
		id     string
		token  string
		status int
		res    orgRes
	}{
		{
			desc:   "view org",
			id:     or.ID,
			token:  token,
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "view non-existing org",
			id:     wrongValue,
			token:  token,
			status: http.StatusNotFound,
			res:    orgRes{},
		},
		{
			desc:   "view org with without org id",
			id:     "",
			token:  token,
			status: http.StatusBadRequest,
			res:    orgRes{},
		},
		{
			desc:   "view org with invalid auth token",
			id:     or.ID,
			token:  wrongValue,
			status: http.StatusUnauthorized,
			res:    orgRes{},
		},
		{
			desc:   "view org with empty auth token",
			id:     or.ID,
			token:  "",
			status: http.StatusUnauthorized,
			res:    orgRes{},
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/orgs/%s", ts.URL, tc.id),
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var data orgRes
		err = json.NewDecoder(res.Body).Decode(&data)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, data, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data))
	}
}

func TestUpdateOrg(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	or, err := svc.CreateOrg(context.Background(), token, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	updtOrg := auth.Org{
		Name:        "updatedName",
		Description: "updatedDesc",
		Metadata:    map[string]any{"newKey": "newValue"},
	}

	data := toJSON(updtOrg)

	cases := []struct {
		desc   string
		req    string
		id     string
		ct     string
		token  string
		status int
	}{
		{
			desc:   "update org",
			req:    data,
			id:     or.ID,
			ct:     contentType,
			token:  token,
			status: http.StatusOK,
		},
		{
			desc:   "update org with invalid auth token",
			req:    data,
			id:     or.ID,
			ct:     contentType,
			token:  wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "update org with empty auth token",
			req:    data,
			id:     or.ID,
			ct:     contentType,
			token:  "",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "update org with invalid org id",
			req:    data,
			id:     wrongValue,
			ct:     contentType,
			token:  token,
			status: http.StatusNotFound,
		},
		{
			desc:   "update org with without org id",
			req:    data,
			id:     "",
			ct:     contentType,
			token:  token,
			status: http.StatusBadRequest,
		},
		{
			desc:   "update org with invalid request format",
			req:    "{",
			id:     or.ID,
			ct:     contentType,
			token:  token,
			status: http.StatusBadRequest,
		},
		{
			desc:   "update org with empty request",
			req:    "",
			id:     or.ID,
			ct:     contentType,
			token:  token,
			status: http.StatusBadRequest,
		},
		{
			desc:   "update org without content type",
			req:    data,
			id:     or.ID,
			ct:     "",
			token:  token,
			status: http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/orgs/%s", ts.URL, tc.id),
			token:       tc.token,
			contentType: tc.ct,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestDeleteOrg(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	or, err := svc.CreateOrg(context.Background(), token, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	unknownID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc   string
		id     string
		token  string
		status int
	}{
		{
			desc:   "delete org",
			id:     or.ID,
			token:  token,
			status: http.StatusNoContent,
		},
		{
			desc:   "delete deleted org",
			id:     or.ID,
			token:  token,
			status: http.StatusNotFound,
		},
		{
			desc:   "delete non-existing org",
			id:     unknownID,
			token:  token,
			status: http.StatusNotFound,
		},
		{
			desc:   "delete org with invalid org id",
			id:     wrongValue,
			token:  token,
			status: http.StatusNotFound,
		},
		{
			desc:   "delete org without org id",
			id:     "",
			token:  token,
			status: http.StatusBadRequest,
		},
		{
			desc:   "delete org with invalid auth token",
			id:     or.ID,
			token:  wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "delete org with empty auth token",
			id:     or.ID,
			token:  "",
			status: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/orgs/%s", ts.URL, tc.id),
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestDeleteOrgs(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()

	var orgIDs []string
	for i := uint64(0); i < 10; i++ {
		num := strconv.FormatUint(i, 10)
		org := auth.Org{Name: "org-" + num}
		o, err := svc.CreateOrg(context.Background(), token, org)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		orgIDs = append(orgIDs, o.ID)
	}

	cases := []struct {
		desc        string
		data        []string
		auth        string
		contentType string
		status      int
	}{
		{
			desc:        "remove orgs with invalid token",
			data:        orgIDs,
			auth:        wrongValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove orgs with empty token",
			data:        orgIDs,
			auth:        emptyValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove orgs with invalid content type",
			data:        orgIDs,
			auth:        token,
			contentType: wrongValue,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "remove existing orgs",
			data:        orgIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNoContent,
		},
		{
			desc:        "remove non-existent orgs",
			data:        []string{wrongValue},
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "remove orgs with empty org ids",
			data:        []string{emptyValue},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		data := struct {
			OrgIDs []string `json:"org_ids"`
		}{
			tc.data,
		}

		body := toJSON(data)

		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/orgs", ts.URL),
			token:       tc.auth,
			contentType: tc.contentType,
			body:        strings.NewReader(body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestListOrgs(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	var orgs []orgRes
	for i := 0; i < n; i++ {
		org.Name = fmt.Sprintf("org-%d", i)
		org.Description = fmt.Sprintf("org-%d description", i)

		or, err := svc.CreateOrg(context.Background(), token, org)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		orgs = append(orgs, orgRes{
			ID:          or.ID,
			OwnerID:     or.OwnerID,
			Name:        or.Name,
			Description: or.Description,
			Metadata:    or.Metadata,
		})
	}

	orgsURL := fmt.Sprintf("%s/orgs", ts.URL)

	cases := []struct {
		desc   string
		token  string
		status int
		url    string
		res    []orgRes
	}{
		{
			desc:   "list orgs",
			token:  token,
			url:    fmt.Sprintf("%s?limit=%d&offset=%d", orgsURL, 5, 0),
			status: http.StatusOK,
			res:    orgs[:5],
		},
		{
			desc:   "list orgs filtering by name",
			token:  token,
			url:    fmt.Sprintf("%s?limit=%d&offset=%d&name=%s", orgsURL, n, 0, "1"),
			status: http.StatusOK,
			res:    orgs[1:2],
		},
		{
			desc:   "list orgs with invalid auth token",
			token:  wrongValue,
			url:    fmt.Sprintf("%s?limit=%d&offset=%d", orgsURL, 5, 0),
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list orgs with empty auth token",
			token:  "",
			url:    fmt.Sprintf("%s?limit=%d&offset=%d", orgsURL, 5, 0),
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list orgs with negative offset",
			token:  token,
			url:    fmt.Sprintf("%s?limit=%d&offset=%d", orgsURL, 0, -5),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list orgs with negative limit",
			token:  token,
			url:    fmt.Sprintf("%s?limit=%d&offset=%d", orgsURL, -5, 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list orgs without offset",
			token:  token,
			url:    fmt.Sprintf("%s?limit=%d", orgsURL, 5),
			status: http.StatusOK,
			res:    orgs[:5],
		},
		{
			desc:   "list orgs without limit",
			token:  token,
			url:    fmt.Sprintf("%s?offset=%d", orgsURL, 0),
			status: http.StatusOK,
			res:    orgs,
		},
		{
			desc:   "list orgs with redundant query params",
			token:  token,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&value=something", orgsURL, 0, 5),
			status: http.StatusOK,
			res:    orgs[:5],
		},
		{
			desc:   "list orgs with default URL",
			token:  token,
			url:    orgsURL,
			status: http.StatusOK,
			res:    orgs,
		},
		{
			desc:   "list orgs with invalid limit",
			token:  token,
			url:    fmt.Sprintf("%s?limit=%s&offset=%d", orgsURL, "i", 5),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list orgs with invalid offset",
			token:  token,
			url:    fmt.Sprintf("%s?limit=%d&offset=%s", orgsURL, 5, "i"),
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
		var data orgsPageRes
		err = json.NewDecoder(res.Body).Decode(&data)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.ElementsMatch(t, tc.res, data.Orgs, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data.Orgs))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestSearchOrgs(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()

	orgs := []orgRes{}
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("org_%03d", i+1)
		description := fmt.Sprintf("description for %s", name)
		org := auth.Org{Name: name, Description: description, Metadata: metadata}

		or, err := svc.CreateOrg(context.Background(), token, org)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		orgs = append(orgs, orgRes{
			ID:          or.ID,
			OwnerID:     or.OwnerID,
			Name:        or.Name,
			Description: or.Description,
			Metadata:    or.Metadata,
		})
	}

	cases := []struct {
		desc   string
		auth   string
		status int
		req    string
		res    []orgRes
	}{
		{
			desc:   "search orgs",
			auth:   token,
			status: http.StatusOK,
			req:    validData,
			res:    orgs[0:5],
		},
		{
			desc:   "search orgs ordered by name asc",
			auth:   token,
			status: http.StatusOK,
			req:    ascData,
			res:    orgs[0:5],
		},
		{
			desc:   "search orgs ordered by name desc",
			auth:   token,
			status: http.StatusOK,
			req:    descData,
			res:    orgs[0:5],
		},
		{
			desc:   "search orgs with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidOrderData,
			res:    nil,
		},
		{
			desc:   "search orgs with invalid dir",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidDirData,
			res:    nil,
		},
		{
			desc:   "search orgs with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search orgs with invalid data",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidData,
			res:    nil,
		},
		{
			desc:   "search orgs with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search orgs with zero limit",
			auth:   token,
			status: http.StatusOK,
			req:    zeroLimitData,
			res:    orgs[0:10],
		},
		{
			desc:   "search orgs with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidLimitData,
			res:    nil,
		},
		{
			desc:   "search orgs filtering with invalid name",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidNameData,
			res:    nil,
		},
		{
			desc:   "search orgs with empty JSON body",
			auth:   token,
			status: http.StatusOK,
			req:    emptyJson,
			res:    orgs[0:10],
		},
		{
			desc:   "search orgs with no body",
			auth:   token,
			status: http.StatusOK,
			req:    emptyValue,
			res:    orgs[0:10],
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodPost,
			url:    fmt.Sprintf("%s/orgs/search", ts.URL),
			token:  tc.auth,
			body:   strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		var body orgsPageRes
		_ = json.NewDecoder(res.Body).Decode(&body)

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, body.Orgs, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body.Orgs))
	}
}

type orgRes struct {
	ID          string         `json:"id"`
	OwnerID     string         `json:"owner_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Metadata    map[string]any `json:"metadata"`
}

type orgsPageRes struct {
	pageRes
	Orgs []orgRes `json:"orgs"`
}

type pageRes struct {
	Limit  uint64 `json:"limit"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
	Name   string `json:"name"`
}
