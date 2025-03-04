package members_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
			method: http.MethodPatch,
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
	updtToEditor.Role = auth.Editor

	updtToOwner := viewerMember
	updtToOwner.Role = auth.Owner

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
		Role:  auth.Owner,
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
			status: http.StatusNotFound,
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
