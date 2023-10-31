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
	"github.com/MainfluxLabs/mainflux/internal/apiutil"
	"github.com/MainfluxLabs/mainflux/logger"
	thmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
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
	groupID       = "groupID"
	groupID2      = "groupID2"
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
	viewerMember  = auth.OrgMember{MemberID: viewerID, Email: viewerEmail, Role: auth.ViewerRole}
	editorMember  = auth.OrgMember{MemberID: editorID, Email: editorEmail, Role: auth.EditorRole}
	adminMember   = auth.OrgMember{MemberID: adminID, Email: adminEmail, Role: auth.AdminRole}
	usersByEmails = map[string]users.User{adminEmail: {ID: adminID, Email: adminEmail}, editorEmail: {ID: editorID, Email: editorEmail}, viewerEmail: {ID: viewerID, Email: viewerEmail}, email: {ID: id, Email: email}}
	usersByIDs    = map[string]users.User{adminID: {ID: adminID, Email: adminEmail}, editorID: {ID: editorID, Email: editorEmail}, viewerID: {ID: viewerID, Email: viewerEmail}, id: {ID: id, Email: email}}
	groups        = map[string]things.Group{groupID: {ID: groupID, OwnerID: id, Name: name, Description: description}, groupID2: {ID: groupID2, OwnerID: id, Name: name, Description: description}}
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
	orgsRepo := mocks.NewOrgRepository()
	rolesRepo := mocks.NewRolesRepository()
	policiesRepo := mocks.NewPoliciesRepository()
	idProvider := uuid.NewMock()
	t := jwt.New(secret)
	uc := mocks.NewUsersService(usersByIDs, usersByEmails)
	tc := thmocks.NewThingsServiceClient(nil, groups)

	return auth.New(orgsRepo, tc, uc, nil, rolesRepo, policiesRepo, idProvider, t, loginDuration)
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

	sort.Slice(orgs, func(i, j int) bool {
		return orgs[i].ID < orgs[j].ID
	})

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
			url:    fmt.Sprintf("%s/orgs?limit=%d&offset=%d", ts.URL, 5, 0),
			status: http.StatusOK,
			res:    orgs[:5],
		},
		{
			desc:   "list orgs filtering by name",
			token:  token,
			url:    fmt.Sprintf("%s/orgs?limit=%d&offset=%d&name=%s", ts.URL, n, 0, "1"),
			status: http.StatusOK,
			res:    orgs[1:2],
		},
		{
			desc:   "list orgs with invalid auth token",
			token:  wrongValue,
			url:    fmt.Sprintf("%s/orgs?limit=%d&offset=%d", ts.URL, 5, 0),
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list orgs with empty auth token",
			token:  "",
			url:    fmt.Sprintf("%s/orgs?limit=%d&offset=%d", ts.URL, 5, 0),
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list orgs with negative offset",
			token:  token,
			url:    fmt.Sprintf("%s/orgs?limit=%d&offset=%d", ts.URL, 0, -5),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list orgs with negative limit",
			token:  token,
			url:    fmt.Sprintf("%s/orgs?limit=%d&offset=%d", ts.URL, -5, 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list orgs without offset",
			token:  token,
			url:    fmt.Sprintf("%s/orgs?limit=%d", ts.URL, 5),
			status: http.StatusOK,
			res:    orgs[:5],
		},
		{
			desc:   "list orgs without limit",
			token:  token,
			url:    fmt.Sprintf("%s/orgs?offset=%d", ts.URL, 0),
			status: http.StatusOK,
			res:    orgs,
		},
		{
			desc:   "list orgs with redundant query params",
			token:  token,
			url:    fmt.Sprintf("%s/orgs?offset=%d&limit=%d&value=something", ts.URL, 0, 5),
			status: http.StatusOK,
			res:    orgs[:5],
		},
		{
			desc:   "list orgs with default URL",
			token:  token,
			url:    fmt.Sprintf("%s/orgs", ts.URL),
			status: http.StatusOK,
			res:    orgs,
		},
		{
			desc:   "list orgs with invalid limit",
			token:  token,
			url:    fmt.Sprintf("%s/orgs?limit=%s&offset=%d", ts.URL, "i", 5),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list orgs with invalid offset",
			token:  token,
			url:    fmt.Sprintf("%s/orgs?limit=%d&offset=%s", ts.URL, 5, "i"),
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
		assert.Equal(t, tc.res, data.Orgs, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data.Orgs))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestListMemberships(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	data := []orgRes{}
	for i := 0; i < n; i++ {
		org.Name = fmt.Sprintf("org-%d", i)
		org.Description = fmt.Sprintf("org-%d", i)

		or, err := svc.CreateOrg(context.Background(), token, org)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		data = append(data, orgRes{
			ID:          or.ID,
			OwnerID:     or.OwnerID,
			Name:        or.Name,
			Description: or.Description,
			Metadata:    or.Metadata,
		})

		err = svc.AssignMembers(context.Background(), token, or.ID, editorMember)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	cases := []struct {
		desc   string
		token  string
		status int
		url    string
		res    []orgRes
	}{
		{
			desc:   "list org memberships",
			token:  token,
			url:    fmt.Sprintf("%s/members/%s/orgs?limit=%d&offset=%d", ts.URL, id, n, 0),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list memberships filtering with name",
			token:  token,
			url:    fmt.Sprintf("%s/members/%s/orgs?limit=%d&offset=%d&name=%s", ts.URL, id, n, 0, "1"),
			status: http.StatusOK,
			res:    data[1:2],
		},
		{
			desc:   "list memberships with invalid auth token",
			token:  wrongValue,
			url:    fmt.Sprintf("%s/members/%s/orgs?limit=%d&offset=%d", ts.URL, id, n, 0),
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list memberships with empty auth token",
			token:  "",
			url:    fmt.Sprintf("%s/members/%s/orgs?limit=%d&offset=%d", ts.URL, id, 5, 0),
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list memberships with negative offset",
			token:  token,
			url:    fmt.Sprintf("%s/members/%s/orgs?limit=%d&offset=%d", ts.URL, id, 0, -5),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list memberships with negative limit",
			token:  token,
			url:    fmt.Sprintf("%s/members/%s/orgs?limit=%d&offset=%d", ts.URL, id, -5, 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list memberships without offset",
			token:  token,
			url:    fmt.Sprintf("%s/members/%s/orgs?limit=%d", ts.URL, id, n),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list memberships without limit",
			token:  token,
			url:    fmt.Sprintf("%s/members/%s/orgs?offset=%d", ts.URL, id, 0),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list memberships with redundant query params",
			token:  token,
			url:    fmt.Sprintf("%s/members/%s/orgs?limit=%d&offset=%d&value=something", ts.URL, id, n, 0),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list memberships with default URL",
			token:  token,
			url:    fmt.Sprintf("%s/members/%s/orgs", ts.URL, id),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list memberships with invalid limit",
			token:  token,
			url:    fmt.Sprintf("%s/members/%s/orgs?limit=%s&offset=%d", ts.URL, id, "i", 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list memberships with invalid offset",
			token:  token,
			url:    fmt.Sprintf("%s/members/%s/orgs?limit=%d&offset=%s", ts.URL, id, n, "i"),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list memberships with invalid member id",
			token:  token,
			url:    fmt.Sprintf("%s/members/%s/orgs?limit=%d&offset=%d", ts.URL, wrongValue, n, 0),
			status: http.StatusForbidden,
			res:    nil,
		},
		{
			desc:   "list memberships without member id",
			token:  token,
			url:    fmt.Sprintf("%s/members/%s/orgs?limit=%d&offset=%d", ts.URL, "", n, 0),
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
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, data.Orgs, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data.Orgs))

	}
}

func TestAssignMembers(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	invalidMember := viewerMember
	invalidMember.Role = wrongValue

	or, err := svc.CreateOrg(context.Background(), token, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	data := toJSON(membersReq{OrgMembers: []auth.OrgMember{editorMember}})
	invalidData := toJSON(membersReq{OrgMembers: []auth.OrgMember{invalidMember}})

	cases := []struct {
		desc   string
		token  string
		id     string
		req    string
		status int
	}{
		{
			desc:   "assign member to org",
			token:  token,
			id:     or.ID,
			req:    data,
			status: http.StatusOK,
		},
		{
			desc:   "assign member to org with invalid member role",
			token:  token,
			id:     or.ID,
			req:    invalidData,
			status: http.StatusBadRequest,
		},
		{
			desc:   "assign member to org with invalid auth token",
			token:  wrongValue,
			id:     or.ID,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "assign member to org with empty token",
			token:  "",
			id:     or.ID,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "assign member to non-existing org",
			token:  token,
			id:     wrongValue,
			req:    data,
			status: http.StatusNotFound,
		},
		{
			desc:   "assign member to org without org id",
			token:  token,
			id:     "",
			req:    data,
			status: http.StatusBadRequest,
		},
		{
			desc:   "assign member to org with invalid request body",
			token:  token,
			id:     or.ID,
			req:    "{",
			status: http.StatusBadRequest,
		},
		{
			desc:   "assign member to org without request body",
			token:  token,
			id:     or.ID,
			req:    "",
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodPost,
			url:    fmt.Sprintf("%s/orgs/%s/members", ts.URL, tc.id),
			token:  tc.token,
			body:   strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

	}
}

func TestUnassignMembers(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	or, err := svc.CreateOrg(context.Background(), token, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	members := []auth.OrgMember{editorMember, viewerMember}

	err = svc.AssignMembers(context.Background(), token, or.ID, members...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	data := toJSON(unassignMembersReq{MemberIDs: []string{viewerMember.MemberID, editorMember.MemberID}})

	cases := []struct {
		desc   string
		token  string
		id     string
		req    string
		status int
	}{
		{
			desc:   "unassign members from org",
			token:  token,
			id:     or.ID,
			req:    data,
			status: http.StatusNoContent,
		},
		{
			desc:   "unassign members from org with invalid auth token",
			token:  wrongValue,
			id:     or.ID,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "unassign members from org with empty token",
			token:  "",
			id:     or.ID,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "uassign members from non-existing org",
			token:  token,
			id:     wrongValue,
			req:    data,
			status: http.StatusNotFound,
		},
		{
			desc:   "unassign members from org without org id",
			token:  token,
			id:     "",
			req:    data,
			status: http.StatusBadRequest,
		},
		{
			desc:   "unassign members from org with invalid request body",
			token:  token,
			id:     or.ID,
			req:    "{",
			status: http.StatusBadRequest,
		},
		{
			desc:   "unassign members from org without request body",
			token:  token,
			id:     or.ID,
			req:    "",
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/orgs/%s/members", ts.URL, tc.id),
			token:  tc.token,
			body:   strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

	}
}

func TestUpdateMembers(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	updtToEditor := viewerMember
	updtToEditor.Role = auth.EditorRole

	updtToOwner := viewerMember
	updtToOwner.Role = auth.OwnerRole

	or, err := svc.CreateOrg(context.Background(), token, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.AssignMembers(context.Background(), token, or.ID, viewerMember)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	ViewerRoleData := toJSON(membersReq{OrgMembers: []auth.OrgMember{updtToEditor}})
	ownerRoleData := toJSON(membersReq{OrgMembers: []auth.OrgMember{updtToOwner}})

	cases := []struct {
		desc   string
		token  string
		id     string
		req    string
		status int
	}{
		{
			desc:   "update org member role",
			token:  token,
			id:     or.ID,
			req:    ViewerRoleData,
			status: http.StatusOK,
		},
		{
			desc:   "update org member role with invalid auth token",
			token:  wrongValue,
			id:     or.ID,
			req:    ViewerRoleData,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "update org member role with empty token",
			token:  "",
			id:     or.ID,
			req:    ViewerRoleData,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "update org member role with non-existing org",
			token:  token,
			id:     wrongValue,
			req:    ViewerRoleData,
			status: http.StatusNotFound,
		},
		{
			desc:   "update org member role without org id",
			token:  token,
			id:     "",
			req:    ViewerRoleData,
			status: http.StatusBadRequest,
		},
		{
			desc:   "update org member role with invalid request body",
			token:  token,
			id:     or.ID,
			req:    "{",
			status: http.StatusBadRequest,
		},
		{
			desc:   "update org member role without request body",
			token:  token,
			id:     or.ID,
			req:    "",
			status: http.StatusBadRequest,
		},
		{
			desc:   "update org member role to owner",
			token:  token,
			id:     or.ID,
			req:    ownerRoleData,
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodPut,
			url:    fmt.Sprintf("%s/orgs/%s/members", ts.URL, tc.id),
			token:  tc.token,
			body:   strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

	}
}

func TestListMembers(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	members := []auth.OrgMember{viewerMember, editorMember, adminMember}

	or, err := svc.CreateOrg(context.Background(), token, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.AssignMembers(context.Background(), token, or.ID, members...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	var data []viewMemberRes
	for _, m := range members {
		data = append(data, viewMemberRes{
			ID:    m.MemberID,
			Email: m.Email,
			Role:  m.Role,
		})
	}

	owner := viewMemberRes{
		ID:    id,
		Email: email,
		Role:  auth.OwnerRole,
	}

	data = append(data, owner)

	cases := []struct {
		desc   string
		token  string
		url    string
		status int
		res    []viewMemberRes
	}{
		{
			desc:   "list org members",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/members?limit=%d&offset=%d", ts.URL, or.ID, n, 0),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list org members with invalid auth token",
			token:  wrongValue,
			url:    fmt.Sprintf("%s/orgs/%s/members?limit=%d&offset=%d", ts.URL, or.ID, n, 0),
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list org members without auth token",
			token:  "",
			url:    fmt.Sprintf("%s/orgs/%s/members?limit=%d&offset=%d", ts.URL, or.ID, n, 0),
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list org members without org id",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/members?limit=%d&offset=%d", ts.URL, "", n, 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list org members with invalid org id",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/members?limit=%d&offset=%d", ts.URL, wrongValue, n, 0),
			status: http.StatusOK,
			res:    nil,
		},
		{
			desc:   "list org members with negative offset",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/members?limit=%d&offset=%d", ts.URL, or.ID, n, -5),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list org members with negative limit",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/members?limit=%d&offset=%d", ts.URL, or.ID, -5, 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list org members with invalid offset",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/members?limit=%d&offset=%s", ts.URL, or.ID, n, "i"),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list org members with invalid limit",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/members?limit=%s&offset=%d", ts.URL, or.ID, "i", 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list org members without limit",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/members?offset=%d", ts.URL, or.ID, 0),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list org members without offset",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/members?limit=%d", ts.URL, or.ID, n),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list org members with default URL",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/members", ts.URL, or.ID),
			status: http.StatusOK,
			res:    data,
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
		var data memberPageRes
		err = json.NewDecoder(res.Body).Decode(&data)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.ElementsMatch(t, tc.res, data.Members, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data.Members))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestAssignOrgGroups(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	or, err := svc.CreateOrg(context.Background(), token, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	var groupIDs []string
	for i := 0; i < n; i++ {
		groupID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		groupIDs = append(groupIDs, groupID)
	}

	data := toJSON(groupsReq{GroupIDs: groupIDs})

	cases := []struct {
		desc   string
		token  string
		orgID  string
		req    string
		status int
	}{
		{
			desc:   "assign groups to org",
			token:  token,
			orgID:  or.ID,
			req:    data,
			status: http.StatusOK,
		},
		{
			desc:   "assign groups to org with invalid auth token",
			token:  wrongValue,
			orgID:  or.ID,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "assign groups to org without auth token",
			token:  "",
			orgID:  or.ID,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "assign groups to org without org id",
			token:  token,
			orgID:  "",
			req:    data,
			status: http.StatusBadRequest,
		},
		{
			desc:   "assign groups to org with invalid org id",
			token:  token,
			orgID:  wrongValue,
			req:    data,
			status: http.StatusNotFound,
		},
		{
			desc:   "assign groups to org with invalid request body",
			token:  token,
			orgID:  or.ID,
			req:    "}",
			status: http.StatusBadRequest,
		},
		{
			desc:   "assign groups to org without request body",
			token:  token,
			orgID:  or.ID,
			req:    "",
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodPost,
			url:    fmt.Sprintf("%s/orgs/%s/groups", ts.URL, tc.orgID),
			token:  tc.token,
			body:   strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

	}
}

func TestUnassignOrgGroups(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	or, err := svc.CreateOrg(context.Background(), token, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	var groupIDs []string
	for i := 0; i < n; i++ {
		groupID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		groupIDs = append(groupIDs, groupID)
	}

	err = svc.AssignGroups(context.Background(), token, or.ID, groupIDs...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	data := toJSON(groupsReq{GroupIDs: groupIDs})

	cases := []struct {
		desc   string
		token  string
		orgID  string
		req    string
		status int
	}{
		{
			desc:   "unassign groups from org",
			token:  token,
			orgID:  or.ID,
			req:    data,
			status: http.StatusNoContent,
		},
		{
			desc:   "unassign groups from org with invalid auth token",
			token:  wrongValue,
			orgID:  or.ID,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "unassign groups from org without auth token",
			token:  "",
			orgID:  or.ID,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "unassign groups from org without org id",
			token:  token,
			orgID:  "",
			req:    data,
			status: http.StatusBadRequest,
		},
		{
			desc:   "unassign groups from org with invalid org id",
			token:  token,
			orgID:  wrongValue,
			req:    data,
			status: http.StatusNotFound,
		},
		{
			desc:   "unassign groups from org with invalid request body",
			token:  token,
			orgID:  or.ID,
			req:    "}",
			status: http.StatusBadRequest,
		},
		{
			desc:   "unassign groups from org without request body",
			token:  token,
			orgID:  or.ID,
			req:    "",
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/orgs/%s/groups", ts.URL, tc.orgID),
			token:  tc.token,
			body:   strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

	}
}

func TestListGroups(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	or, err := svc.CreateOrg(context.Background(), token, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	groupIDs := []string{groupID, groupID2}

	err = svc.AssignGroups(context.Background(), token, or.ID, groupIDs...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	data := []viewGroupRes{
		{
			ID:          groupID,
			OwnerID:     id,
			Name:        name,
			Description: description,
		},
		{
			ID:          groupID2,
			OwnerID:     id,
			Name:        name,
			Description: description,
		},
	}

	cases := []struct {
		desc   string
		token  string
		status int
		url    string
		res    []viewGroupRes
	}{
		{
			desc:   "list org groups",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/groups?limit=%d&offset=%d", ts.URL, or.ID, n, 0),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list org groups with invalid auth token",
			token:  wrongValue,
			url:    fmt.Sprintf("%s/orgs/%s/groups?limit=%d&offset=%d", ts.URL, or.ID, n, 0),
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list org groups with empty auth token",
			token:  "",
			url:    fmt.Sprintf("%s/orgs/%s/groups?limit=%d&offset=%d", ts.URL, or.ID, n, 0),
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list org groups with negative offset",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/groups?limit=%d&offset=%d", ts.URL, or.ID, n, -5),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list org groups with negative limit",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/groups?limit=%d&offset=%d", ts.URL, or.ID, -5, 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list org groups without offset",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/groups?limit=%d", ts.URL, or.ID, n),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list org groups without limit",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/groups?offset=%d", ts.URL, or.ID, 0),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list org groups with redundant query params",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/groups?limit=%d&offset=%d&value=value", ts.URL, or.ID, n, 0),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list org groups with default URL",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/groups", ts.URL, or.ID),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list org groups with invalid limit",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/groups?limit=%s&offset=%d", ts.URL, or.ID, "i", 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list org groups with invalid offset",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/groups?limit=%d&offset=%s", ts.URL, or.ID, n, "i"),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list org groups with invalid org id",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/groups?limit=%d&offset=%d", ts.URL, wrongValue, n, 0),
			status: http.StatusOK,
			res:    nil,
		},
		{
			desc:   "list org groups without org id",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/groups?limit=%d&offset=%d", ts.URL, "", n, 0),
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
		var data groupsPageRes
		err = json.NewDecoder(res.Body).Decode(&data)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, data.Groups, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data.Groups))

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

	var grIDs []string
	for i := 0; i < 2; i++ {
		groupID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		grIDs = append(grIDs, groupID)
	}

	err = svc.AssignGroups(context.Background(), adminToken, o.ID, grIDs...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	members := []auth.OrgMember{viewerMember, editorMember, adminMember}
	err = svc.AssignMembers(context.Background(), adminToken, o.ID, members...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.AssignRole(context.Background(), id, auth.RoleAdmin)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	groupPolicy := auth.GroupPolicyByID{
		MemberID: viewerID,
		Policy:   auth.RPolicy,
	}

	err = svc.CreateGroupPolicies(context.Background(), adminToken, grIDs[0], groupPolicy)
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
		},
		{
			MemberID: adminID,
			OrgID:    o.ID,
		},
		{
			MemberID: editorID,
			OrgID:    o.ID,
		},
		{
			MemberID: viewerID,
			OrgID:    o.ID,
		},
	}

	g := []viewOrgGroups{
		{
			GroupID: grIDs[0],
			OrgID:   o.ID,
		},
		{
			GroupID: grIDs[1],
			OrgID:   o.ID,
		},
	}

	gp := []viewGroupPolicies{
		{
			MemberID: o.OwnerID,
			Policy:   auth.RwPolicy,
		},
		{
			MemberID: viewerID,
			Policy:   auth.RPolicy,
		},
	}

	sort.Slice(g, func(i, j int) bool {
		return g[i].GroupID < g[j].GroupID
	})

	data := backup{or, m, g, gp}

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

		sort.Slice(data.OrgGroups, func(i, j int) bool {
			return data.OrgGroups[i].GroupID < data.OrgGroups[j].GroupID
		})

		sort.Slice(data.GroupPolicies, func(i, j int) bool {
			return data.GroupPolicies[i].MemberID < data.GroupPolicies[j].MemberID
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

	var grIDs []string
	for i := 0; i < 2; i++ {
		groupID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		grIDs = append(grIDs, groupID)
	}

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
			Role:     auth.ViewerRole,
		},
		{
			MemberID: editorID,
			OrgID:    orgID,
			Role:     auth.EditorRole,
		},
		{
			MemberID: adminID,
			OrgID:    orgID,
			Role:     auth.AdminRole,
		},
	}

	g := []viewOrgGroups{
		{
			GroupID: grIDs[0],
			OrgID:   orgID,
		},
		{
			GroupID: grIDs[1],
			OrgID:   orgID,
		},
	}

	gp := []viewGroupPolicies{
		{
			GroupID:  grIDs[0],
			MemberID: viewerID,
			Policy:   auth.ViewerRole,
		},
	}

	data := toJSON(backup{
		Orgs:          or,
		OrgMembers:    m,
		OrgGroups:     g,
		GroupPolicies: gp,
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

type membersReq struct {
	OrgMembers []auth.OrgMember `json:"org_members"`
}

type unassignMembersReq struct {
	MemberIDs []string `json:"member_ids"`
}

type viewMemberRes struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type memberPageRes struct {
	pageRes
	Members []viewMemberRes `json:"members"`
}

type groupsReq struct {
	GroupIDs []string `json:"group_ids"`
}

type viewGroupRes struct {
	ID          string `json:"id"`
	OwnerID     string `json:"owner_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type groupsPageRes struct {
	pageRes
	Groups []viewGroupRes `json:"groups"`
}

type viewOrgMembers struct {
	MemberID string `json:"member_id"`
	OrgID    string `json:"org_id"`
	Role     string `json:"role"`
}

type viewOrgGroups struct {
	GroupID string `json:"group_id"`
	OrgID   string `json:"org_id"`
}

type viewGroupPolicies struct {
	GroupID  string `json:"group_id"`
	MemberID string `json:"member_id"`
	Policy   string `json:"policy"`
}

type backup struct {
	Orgs          []orgRes            `json:"orgs"`
	OrgMembers    []viewOrgMembers    `json:"org_members"`
	OrgGroups     []viewOrgGroups     `json:"org_groups"`
	GroupPolicies []viewGroupPolicies `json:"group_policies"`
}
