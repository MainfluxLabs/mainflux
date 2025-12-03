package memberships_test

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
	contentTypeOctetStream = "application/octet-stream"
	secret                 = "secret"
	id                     = "123e4567-e89b-12d3-a456-000000000022"
	adminID                = "adminID"
	editorID               = "editorID"
	viewerID               = "viewerID"
	email                  = "user@example.com"
	adminEmail             = "admin@example.com"
	editorEmail            = "editor@example.com"
	viewerEmail            = "viewer@example.com"
	wrongValue             = "wrong_value"
	name                   = "testName"
	description            = "testDesc"
	n                      = 10
	loginDuration          = 30 * time.Minute
	inviteDuration         = 7 * 24 * time.Hour
	emailKey               = "email"
	idKey                  = "id"
	ascKey                 = "asc"
	descKey                = "desc"
	emptyValue             = ""
)

var (
	org = auth.Org{
		Name:        name,
		Description: description,
		Metadata:    map[string]any{"key": "value"},
	}
	viewer        = auth.OrgMembership{MemberID: viewerID, Email: viewerEmail, Role: auth.Viewer}
	editor        = auth.OrgMembership{MemberID: editorID, Email: editorEmail, Role: auth.Editor}
	admin         = auth.OrgMembership{MemberID: adminID, Email: adminEmail, Role: auth.Admin}
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
	membershipsRepo := mocks.NewOrgMembershipsRepository()
	orgsRepo := mocks.NewOrgRepository(membershipsRepo)
	rolesRepo := mocks.NewRolesRepository()
	invitesRepo := mocks.NewInvitesRepository()

	idProvider := uuid.NewMock()
	t := jwt.New(secret)
	uc := mocks.NewUsersService(usersByIDs, usersByEmails)
	tc := thmocks.NewThingsServiceClient(nil, nil, nil)

	return auth.New(orgsRepo, tc, uc, nil, rolesRepo, membershipsRepo, invitesRepo, nil, idProvider, t, loginDuration, inviteDuration)
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

func TestCreateOrgMemberships(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	invalidMembership := viewer
	invalidMembership.Role = wrongValue

	or, err := svc.CreateOrg(context.Background(), token, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	data := toJSON(orgMembershipsReq{OrgMemberships: []auth.OrgMembership{editor}})
	invalidData := toJSON(orgMembershipsReq{OrgMemberships: []auth.OrgMembership{invalidMembership}})

	cases := []struct {
		desc   string
		token  string
		id     string
		req    string
		status int
	}{
		{
			desc:   "create org membership",
			token:  token,
			id:     or.ID,
			req:    data,
			status: http.StatusOK,
		},
		{
			desc:   "create org membership with invalid member role",
			token:  token,
			id:     or.ID,
			req:    invalidData,
			status: http.StatusBadRequest,
		},
		{
			desc:   "create org membership with invalid auth token",
			token:  wrongValue,
			id:     or.ID,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "create org membership with empty token",
			token:  "",
			id:     or.ID,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "create org membership without org id",
			token:  token,
			id:     "",
			req:    data,
			status: http.StatusBadRequest,
		},
		{
			desc:   "create org membership with invalid request body",
			token:  token,
			id:     or.ID,
			req:    "{",
			status: http.StatusBadRequest,
		},
		{
			desc:   "create org membership without request body",
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
			url:    fmt.Sprintf("%s/orgs/%s/memberships", ts.URL, tc.id),
			token:  tc.token,
			body:   strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

	}
}

func TestRemoveOrgMemberships(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	or, err := svc.CreateOrg(context.Background(), token, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	memberships := []auth.OrgMembership{editor, viewer}

	err = svc.CreateOrgMemberships(context.Background(), token, or.ID, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	data := toJSON(removeOrgMembershipsReq{MemberIDs: []string{viewer.MemberID, editor.MemberID}})

	cases := []struct {
		desc   string
		token  string
		id     string
		req    string
		status int
	}{
		{
			desc:   "remove memberships from org",
			token:  token,
			id:     or.ID,
			req:    data,
			status: http.StatusNoContent,
		},
		{
			desc:   "remove memberships from org with invalid auth token",
			token:  wrongValue,
			id:     or.ID,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove memberships from org with empty token",
			token:  "",
			id:     or.ID,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove memberships from non-existing org",
			token:  token,
			id:     wrongValue,
			req:    data,
			status: http.StatusNotFound,
		},
		{
			desc:   "remove memberships from org without org id",
			token:  token,
			id:     "",
			req:    data,
			status: http.StatusBadRequest,
		},
		{
			desc:   "remove memberships from org with invalid request body",
			token:  token,
			id:     or.ID,
			req:    "{",
			status: http.StatusBadRequest,
		},
		{
			desc:   "remove memberships from org without request body",
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
			url:    fmt.Sprintf("%s/orgs/%s/memberships", ts.URL, tc.id),
			token:  tc.token,
			body:   strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

	}
}

func TestUpdateOrgMemberships(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	editor := viewer
	editor.Role = auth.Editor

	owner := viewer
	owner.Role = auth.Owner

	or, err := svc.CreateOrg(context.Background(), token, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.CreateOrgMemberships(context.Background(), token, or.ID, viewer)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	viewerData := toJSON(orgMembershipsReq{OrgMemberships: []auth.OrgMembership{editor}})
	ownerData := toJSON(orgMembershipsReq{OrgMemberships: []auth.OrgMembership{owner}})

	cases := []struct {
		desc   string
		token  string
		id     string
		req    string
		status int
	}{
		{
			desc:   "update org membership",
			token:  token,
			id:     or.ID,
			req:    viewerData,
			status: http.StatusOK,
		},
		{
			desc:   "update org membership with invalid auth token",
			token:  wrongValue,
			id:     or.ID,
			req:    viewerData,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "update org membership with empty token",
			token:  "",
			id:     or.ID,
			req:    viewerData,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "update org membership with non-existing org",
			token:  token,
			id:     wrongValue,
			req:    viewerData,
			status: http.StatusNotFound,
		},
		{
			desc:   "update org membership without org id",
			token:  token,
			id:     "",
			req:    viewerData,
			status: http.StatusBadRequest,
		},
		{
			desc:   "update org membership with invalid request body",
			token:  token,
			id:     or.ID,
			req:    "{",
			status: http.StatusBadRequest,
		},
		{
			desc:   "update org membership without request body",
			token:  token,
			id:     or.ID,
			req:    "",
			status: http.StatusBadRequest,
		},
		{
			desc:   "update org member role to owner",
			token:  token,
			id:     or.ID,
			req:    ownerData,
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodPut,
			url:    fmt.Sprintf("%s/orgs/%s/memberships", ts.URL, tc.id),
			token:  tc.token,
			body:   strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

	}
}

func TestListOrgMemberships(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	memberships := []auth.OrgMembership{viewer, editor, admin}

	or, err := svc.CreateOrg(context.Background(), token, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.CreateOrgMemberships(context.Background(), token, or.ID, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	var data []viewOrgMembershipRes
	for _, m := range memberships {
		data = append(data, viewOrgMembershipRes{
			OrgID:    or.ID,
			MemberID: m.MemberID,
			Email:    m.Email,
			Role:     m.Role,
		})
	}

	owner := viewOrgMembershipRes{
		OrgID:    or.ID,
		MemberID: id,
		Email:    email,
		Role:     auth.Owner,
	}

	data = append(data, owner)

	dataByEmailAsc := make([]viewOrgMembershipRes, len(data))
	copy(dataByEmailAsc, data)
	sort.Slice(dataByEmailAsc, func(i, j int) bool {
		return dataByEmailAsc[i].Email < dataByEmailAsc[j].Email
	})

	dataByEmailDesc := make([]viewOrgMembershipRes, len(data))
	copy(dataByEmailDesc, data)
	sort.Slice(dataByEmailDesc, func(i, j int) bool {
		return dataByEmailDesc[i].Email > dataByEmailDesc[j].Email
	})

	dataByIDAsc := make([]viewOrgMembershipRes, len(data))
	copy(dataByIDAsc, data)
	sort.Slice(dataByIDAsc, func(i, j int) bool {
		return dataByIDAsc[i].MemberID < dataByIDAsc[j].MemberID
	})

	dataByIDDesc := make([]viewOrgMembershipRes, len(data))
	copy(dataByIDDesc, data)
	sort.Slice(dataByIDDesc, func(i, j int) bool {
		return dataByIDDesc[i].MemberID > dataByIDDesc[j].MemberID
	})

	cases := []struct {
		desc   string
		token  string
		url    string
		status int
		res    []viewOrgMembershipRes
	}{
		{
			desc:   "list org memberships",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/memberships?limit=%d&offset=%d", ts.URL, or.ID, n, 0),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list org memberships with invalid auth token",
			token:  wrongValue,
			url:    fmt.Sprintf("%s/orgs/%s/memberships?limit=%d&offset=%d", ts.URL, or.ID, n, 0),
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list org memberships without auth token",
			token:  "",
			url:    fmt.Sprintf("%s/orgs/%s/memberships?limit=%d&offset=%d", ts.URL, or.ID, n, 0),
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list org memberships without org id",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/memberships?limit=%d&offset=%d", ts.URL, "", n, 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list org memberships with invalid org id",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/memberships?limit=%d&offset=%d", ts.URL, wrongValue, n, 0),
			status: http.StatusNotFound,
			res:    nil,
		},
		{
			desc:   "list org memberships with negative offset",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/memberships?limit=%d&offset=%d", ts.URL, or.ID, n, -5),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list org memberships with negative limit",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/memberships?limit=%d&offset=%d", ts.URL, or.ID, -5, 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list org memberships with invalid offset",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/memberships?limit=%d&offset=%s", ts.URL, or.ID, n, "i"),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list org memberships with invalid limit",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/memberships?limit=%s&offset=%d", ts.URL, or.ID, "i", 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list org memberships without limit",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/memberships?offset=%d", ts.URL, or.ID, 0),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list org memberships without offset",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/memberships?limit=%d", ts.URL, or.ID, n),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list org memberships with default URL",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/memberships", ts.URL, or.ID),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list org memberships filtered by email",
			token:  token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/orgs/%s/memberships?email=%s", ts.URL, or.ID, viewerEmail),
			res: []viewOrgMembershipRes{
				{
					OrgID:    or.ID,
					MemberID: viewerID,
					Email:    viewerEmail,
					Role:     auth.Viewer,
				},
			},
		},
		{
			desc:   "list org memberships filtered by email that doesn't match",
			token:  token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/orgs/%s/memberships?email=%s", ts.URL, or.ID, wrongValue),
			res:    []viewOrgMembershipRes{},
		},
		{
			desc:   "list group memberships sorted by email ascendant",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/memberships?order=%s&dir=%s", ts.URL, or.ID, emailKey, ascKey),
			status: http.StatusOK,
			res:    dataByEmailAsc,
		},
		{
			desc:   "list group memberships sorted by email descendent",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/memberships?order=%s&dir=%s", ts.URL, or.ID, emailKey, descKey),
			status: http.StatusOK,
			res:    dataByEmailDesc,
		},
		{
			desc:   "list group memberships sorted by id ascendant",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/memberships?order=%s&dir=%s", ts.URL, or.ID, idKey, ascKey),
			status: http.StatusOK,
			res:    dataByIDAsc,
		},
		{
			desc:   "list group memberships sorted by id descendent",
			token:  token,
			url:    fmt.Sprintf("%s/orgs/%s/memberships?order=%s&dir=%s", ts.URL, or.ID, idKey, descKey),
			status: http.StatusOK,
			res:    dataByIDDesc,
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
		var data orgMembershipPageRes
		err = json.NewDecoder(res.Body).Decode(&data)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.ElementsMatch(t, tc.res, data.OrgMemberships, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data.OrgMemberships))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestBackupOrgMemberships(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	memberships := []auth.OrgMembership{viewer, editor, admin}

	or, err := svc.CreateOrg(context.Background(), token, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.CreateOrgMemberships(context.Background(), token, or.ID, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	var data []viewOrgMembershipRes
	for _, m := range memberships {
		data = append(data, viewOrgMembershipRes{
			OrgID:    or.ID,
			MemberID: m.MemberID,
			Email:    m.Email,
			Role:     m.Role,
		})
	}

	owner := viewOrgMembershipRes{
		OrgID:    or.ID,
		MemberID: id,
		Email:    email,
		Role:     auth.Owner,
	}

	data = append(data, owner)

	orgMembershipURL := fmt.Sprintf("%s/orgs", ts.URL)

	cases := []struct {
		desc   string
		token  string
		url    string
		status int
		res    []viewOrgMembershipRes
	}{
		{
			desc:   "backup org memberships as org owner",
			token:  token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/memberships/backup", orgMembershipURL, or.ID),
			res:    data,
		},
		{
			desc:   "backup org memberships as admin",
			token:  adminToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/memberships/backup", orgMembershipURL, or.ID),
			res:    data,
		},
		{
			desc:   "backup org memberships without org id",
			token:  token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/memberships/backup", orgMembershipURL, emptyValue),
			res:    nil,
		},
		{
			desc:   "backup org memberships with invalid token",
			token:  wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/memberships/backup", orgMembershipURL, or.ID),
			res:    nil,
		},
		{
			desc:   "backup org memberships with empty token",
			token:  emptyValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/memberships/backup", orgMembershipURL, or.ID),
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
		var body []viewOrgMembershipRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, body, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, body))
	}
}

func TestRestoreOrgMemberships(t *testing.T) {
	svc := newService()
	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	memberships := []auth.OrgMembership{viewer, editor, admin}

	or, err := svc.CreateOrg(context.Background(), token, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	var data []viewOrgMembershipRes
	for _, m := range memberships {
		data = append(data, viewOrgMembershipRes{
			OrgID:    or.ID,
			MemberID: m.MemberID,
			Email:    m.Email,
			Role:     m.Role,
		})
	}

	dataBytes, err := json.Marshal(data)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	dataString := string(dataBytes)

	orgMembershipURL := fmt.Sprintf("%s/orgs", ts.URL)

	cases := []struct {
		desc        string
		auth        string
		contentType string
		data        string
		status      int
		url         string
		res         string
	}{
		{
			desc:        "restore org memberships as org owner",
			auth:        token,
			data:        dataString,
			contentType: contentTypeOctetStream,
			status:      http.StatusCreated,
			url:         fmt.Sprintf("%s/%s/memberships/restore", orgMembershipURL, or.ID),
			res:         emptyValue,
		},
		{
			desc:        "restore org memberships as admin",
			auth:        adminToken,
			data:        dataString,
			contentType: contentTypeOctetStream,
			status:      http.StatusCreated,
			url:         fmt.Sprintf("%s/%s/memberships/restore", orgMembershipURL, or.ID),
			res:         emptyValue,
		},
		{
			desc:        "restore org memberships without org id",
			auth:        token,
			data:        dataString,
			contentType: contentTypeOctetStream,
			status:      http.StatusBadRequest,
			url:         fmt.Sprintf("%s/%s/memberships/restore", orgMembershipURL, emptyValue),
			res:         emptyValue,
		},
		{
			desc:        "restore org memberships with invalid token",
			auth:        wrongValue,
			data:        dataString,
			contentType: contentTypeOctetStream,
			status:      http.StatusUnauthorized,
			url:         fmt.Sprintf("%s/%s/memberships/restore", orgMembershipURL, or.ID),
			res:         emptyValue,
		},
		{
			desc:        "restore org memberships with empty token",
			auth:        emptyValue,
			data:        dataString,
			contentType: contentTypeOctetStream,
			status:      http.StatusUnauthorized,
			url:         fmt.Sprintf("%s/%s/memberships/restore", orgMembershipURL, or.ID),
			res:         emptyValue,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         tc.url,
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.data),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

type pageRes struct {
	Limit  uint64 `json:"limit"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
	Name   string `json:"name"`
}

type orgMembershipsReq struct {
	OrgMemberships []auth.OrgMembership `json:"org_memberships"`
}

type removeOrgMembershipsReq struct {
	MemberIDs []string `json:"member_ids"`
}

type viewOrgMembershipRes struct {
	OrgID    string `json:"org_id,omitempty"`
	MemberID string `json:"member_id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type orgMembershipPageRes struct {
	pageRes
	OrgMemberships []viewOrgMembershipRes `json:"org_memberships"`
}
