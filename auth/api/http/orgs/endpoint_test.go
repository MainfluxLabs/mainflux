package orgs_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
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
	secret        = "secret"
	contentType   = "application/json"
	id            = "123e4567-e89b-12d3-a456-000000000022"
	adminID       = "adminID"
	editorID      = "editorID"
	viewerID      = "viewerID"
	email         = "user@example.com"
	adminEmail    = "admin@example.com"
	editorEmail   = "editor@example.com"
	viewerEmail   = "viewer@example.com"
	wrongValue    = "wrong_value"
	name          = "testName"
	description   = "testDesc"
	n             = 10
	loginDuration = 30 * time.Minute
)

var (
	org = auth.Org{
		Name:        name,
		Description: description,
		Metadata:    map[string]interface{}{"key": "value"},
	}
	idProvider    = uuid.New()
	viewerMember  = auth.OrgMember{MemberID: viewerID, Email: viewerEmail, Role: auth.Viewer}
	editorMember  = auth.OrgMember{MemberID: editorID, Email: editorEmail, Role: auth.Editor}
	adminMember   = auth.OrgMember{MemberID: adminID, Email: adminEmail, Role: auth.Admin}
	usersByEmails = map[string]users.User{adminEmail: {ID: adminID, Email: adminEmail}, editorEmail: {ID: editorID, Email: editorEmail}, viewerEmail: {ID: viewerID, Email: viewerEmail}, email: {ID: id, Email: email}}
	usersByIDs    = map[string]users.User{adminID: {ID: adminID, Email: adminEmail}, editorID: {ID: editorID, Email: editorEmail}, viewerID: {ID: viewerID, Email: viewerEmail}, id: {ID: id, Email: email}}
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
	membsRepo := mocks.NewMembersRepository()
	orgsRepo := mocks.NewOrgRepository(membsRepo)
	rolesRepo := mocks.NewRolesRepository()

	idProvider := uuid.NewMock()
	t := jwt.New(secret)
	uc := mocks.NewUsersService(usersByIDs, usersByEmails)
	tc := thmocks.NewThingsServiceClient(nil, nil, nil)

	return auth.New(orgsRepo, tc, uc, nil, rolesRepo, membsRepo, idProvider, t, loginDuration)
}

func newServer(svc auth.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(svc, mocktracer.New(), logger)
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) string {
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
		Metadata:    map[string]interface{}{"newKey": "newValue"},
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

func TestBackup(t *testing.T) {
	svc := newService()
	_, adminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	o, err := svc.CreateOrg(context.Background(), adminToken, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	members := []auth.OrgMember{viewerMember, editorMember, adminMember}
	err = svc.AssignMembers(context.Background(), adminToken, o.ID, members...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.AssignRole(context.Background(), id, auth.RoleAdmin)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	or := []orgRes{
		{
			ID:          o.ID,
			OwnerID:     o.OwnerID,
			Name:        o.Name,
			Description: o.Description,
			Metadata:    o.Metadata,
		},
	}

	m := []viewOrgMembers{
		{
			MemberID: id,
			OrgID:    o.ID,
			Role:     auth.Owner,
		},
		{
			MemberID: adminID,
			OrgID:    o.ID,
			Role:     auth.Admin,
		},
		{
			MemberID: editorID,
			OrgID:    o.ID,
			Role:     auth.Editor,
		},
		{
			MemberID: viewerID,
			OrgID:    o.ID,
			Role:     auth.Viewer,
		},
	}

	data := backup{or, m}

	cases := []struct {
		desc   string
		token  string
		res    backup
		status int
	}{
		{
			desc:   "backup with invalid auth token",
			token:  wrongValue,
			res:    backup{},
			status: http.StatusUnauthorized,
		},
		{
			desc:   "backup without auth token",
			token:  "",
			res:    backup{},
			status: http.StatusUnauthorized,
		},
		{
			desc:   "backup with unauthorized credentials",
			token:  viewerToken,
			res:    backup{},
			status: http.StatusForbidden,
		},
		{
			desc:   "backup with admin credentials",
			token:  adminToken,
			res:    data,
			status: http.StatusOK,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/backup", ts.URL),
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var data backup
		err = json.NewDecoder(res.Body).Decode(&data)

		sort.Slice(data.OrgMembers, func(i, j int) bool {
			return data.OrgMembers[i].MemberID < data.OrgMembers[j].MemberID
		})

		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.res, data, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

	}
}

func TestRestore(t *testing.T) {
	svc := newService()
	_, adminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	err = svc.AssignRole(context.Background(), id, auth.RoleAdmin)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	orgID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	or := []orgRes{
		{
			ID:          orgID,
			OwnerID:     id,
			Name:        org.Name,
			Description: org.Description,
			Metadata:    org.Metadata,
		},
	}

	m := []viewOrgMembers{
		{
			MemberID: viewerID,
			OrgID:    orgID,
			Role:     auth.Viewer,
		},
		{
			MemberID: editorID,
			OrgID:    orgID,
			Role:     auth.Editor,
		},
		{
			MemberID: adminID,
			OrgID:    orgID,
			Role:     auth.Admin,
		},
	}

	data := toJSON(backup{
		Orgs:       or,
		OrgMembers: m,
	})

	cases := []struct {
		desc   string
		token  string
		req    string
		status int
	}{
		{
			desc:   "restore from backup",
			token:  adminToken,
			req:    data,
			status: http.StatusCreated,
		},
		{
			desc:   "restore from backup with invalid auth token",
			token:  wrongValue,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "restore from backup without auth token",
			token:  "",
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "restore from backup with invalid request body",
			token:  adminToken,
			req:    "}",
			status: http.StatusBadRequest,
		},
		{
			desc:   "restore from backup with unauthorized credentials",
			token:  viewerToken,
			req:    data,
			status: http.StatusForbidden,
		},
		{
			desc:   "restore from backup without request body",
			token:  adminToken,
			req:    "",
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodPost,
			url:    fmt.Sprintf("%s/restore", ts.URL),
			token:  tc.token,
			body:   strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

	}
}

type orgRes struct {
	ID          string                 `json:"id"`
	OwnerID     string                 `json:"owner_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
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

type viewOrgMembers struct {
	MemberID string `json:"member_id"`
	OrgID    string `json:"org_id"`
	Role     string `json:"role"`
}
type backup struct {
	Orgs       []orgRes         `json:"orgs"`
	OrgMembers []viewOrgMembers `json:"org_members"`
}
