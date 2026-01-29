package backup_test

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
	secret         = "secret"
	contentType    = "application/json"
	id             = "123e4567-e89b-12d3-a456-000000000022"
	adminID        = "adminID"
	editorID       = "editorID"
	viewerID       = "viewerID"
	email          = "user@example.com"
	adminEmail     = "admin@example.com"
	editorEmail    = "editor@example.com"
	viewerEmail    = "viewer@example.com"
	wrongValue     = "wrong_value"
	name           = "testName"
	description    = "testDesc"
	n              = 10
	loginDuration  = 30 * time.Minute
	inviteDuration = 7 * 24 * time.Hour
)

var (
	org = auth.Org{
		Name:        name,
		Description: description,
		Metadata:    map[string]any{"key": "value"},
	}
	idProvider    = uuid.New()
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

	memberships := []auth.OrgMembership{viewer, editor, admin}
	err = svc.CreateOrgMemberships(context.Background(), adminToken, o.ID, memberships...)
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

	m := []viewOrgMembership{
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

		sort.Slice(data.OrgMemberships, func(i, j int) bool {
			return data.OrgMemberships[i].MemberID < data.OrgMemberships[j].MemberID
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

	m := []viewOrgMembership{
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
		Orgs:           or,
		OrgMemberships: m,
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
	ID          string         `json:"id"`
	OwnerID     string         `json:"owner_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Metadata    map[string]any `json:"metadata"`
}

type viewOrgMembership struct {
	MemberID string `json:"member_id"`
	OrgID    string `json:"org_id"`
	Role     string `json:"role"`
}
type backup struct {
	Orgs           []orgRes            `json:"orgs"`
	OrgMemberships []viewOrgMembership `json:"org_memberships"`
}
