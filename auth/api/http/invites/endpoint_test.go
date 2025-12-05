package invites_test

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
	invitesCommon "github.com/MainfluxLabs/mainflux/pkg/invites"
	thmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

const (
	redirectPathInvite      = "/view-invite"
	redirectPathGroupInvite = "/view-group-invite"
	secret                  = "secret"
	contentType             = "application/json"
	id                      = "123e4567-e89b-12d3-a456-000000000022"
	adminID                 = "adminID"
	editorID                = "editorID"
	viewerID                = "viewerID"
	email                   = "user@example.com"
	adminEmail              = "admin@example.com"
	editorEmail             = "editor@example.com"
	viewerEmail             = "viewer@example.com"
	name                    = "testName"
	description             = "testDesc"
	n                       = 10

	responseAccept  = "accept"
	responseDecline = "decline"
	invalidResponse = "wrong"

	loginDuration  = 30 * time.Minute
	inviteDuration = 7 * 24 * time.Hour
)

var (
	org = auth.Org{
		Name:        name,
		Description: description,
		Metadata:    map[string]any{"key": "value"},
	}

	viewer = auth.OrgMembership{MemberID: viewerID, Email: viewerEmail, Role: auth.Viewer}
	editor = auth.OrgMembership{MemberID: editorID, Email: editorEmail, Role: auth.Editor}
	admin  = auth.OrgMembership{MemberID: adminID, Email: adminEmail, Role: auth.Admin}

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

type invitesReq struct {
	Email        string `json:"email,omitempty"`
	Role         string `json:"role,omitempty"`
	RedirectPath string `json:"redirect_path,omitempty"`
}

type respondInviteReq struct {
	RedirectPath string `json:"redirect_path,omitempty"`
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
	emailerMock := mocks.NewEmailer()

	return auth.New(orgsRepo, tc, uc, nil, rolesRepo, membsRepo, invitesRepo, emailerMock, idProvider, t, loginDuration, inviteDuration)
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

func TestCreateOrgInvite(t *testing.T) {
	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	org, err := svc.CreateOrg(context.Background(), ownerToken, org)
	assert.Nil(t, err, fmt.Sprintf("Creating Org expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()

	client := ts.Client()

	cases := []struct {
		desc   string
		req    string
		ct     string
		token  string
		status int
	}{
		{
			desc:   "create org invite",
			req:    toJSON(invitesReq{Email: viewer.Email, Role: viewer.Role, RedirectPath: redirectPathInvite}),
			ct:     contentType,
			token:  ownerToken,
			status: http.StatusCreated,
		},
		{
			desc:   "create org invite with invalid auth token",
			req:    toJSON(invitesReq{Email: viewer.Email, Role: viewer.Role, RedirectPath: redirectPathInvite}),
			ct:     contentType,
			token:  "invalid-token",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "create org invite with empty auth token",
			req:    toJSON(invitesReq{Email: viewer.Email, Role: viewer.Role, RedirectPath: redirectPathInvite}),
			ct:     contentType,
			token:  "",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "create org invite with empty request",
			req:    "",
			ct:     contentType,
			token:  "",
			status: http.StatusBadRequest,
		},
		{
			desc:   "create org invite with empty JSON array",
			req:    "[]",
			ct:     contentType,
			token:  "",
			status: http.StatusBadRequest,
		},
		{
			desc:   "create org invite with invalid request format",
			req:    "{",
			ct:     contentType,
			token:  ownerToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "create org invite without content type",
			req:    toJSON(invitesReq{Email: viewer.Email, Role: viewer.Role, RedirectPath: redirectPathInvite}),
			ct:     "",
			token:  ownerToken,
			status: http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/orgs/%s/invites", ts.URL, org.ID),
			contentType: tc.ct,
			token:       tc.token,
			body:        strings.NewReader(tc.req),
		}

		res, err := req.make()

		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestViewInvite(t *testing.T) {
	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	_, inviteeToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	org, err := svc.CreateOrg(context.Background(), ownerToken, org)
	assert.Nil(t, err, fmt.Sprintf("Creating Org expected to succeed: %s", err))

	invite, err := svc.CreateOrgInvite(context.Background(), ownerToken, viewer.Email, viewer.Role, org.ID, redirectPathInvite)
	assert.Nil(t, err, fmt.Sprintf("Inviting member expected to succeed: %s", err))

	inviteID := invite.ID

	ts := newServer(svc)
	defer ts.Close()

	client := ts.Client()

	cases := []struct {
		desc     string
		inviteID string
		token    string
		status   int
	}{
		{
			desc:     "view invite",
			token:    inviteeToken,
			inviteID: inviteID,
			status:   http.StatusOK,
		},
		{
			desc:     "view invite with non-existent invite id",
			token:    inviteeToken,
			inviteID: "invalid",
			status:   http.StatusNotFound,
		},
		{
			desc:     "view invite with invalid auth token",
			token:    "invalid",
			inviteID: inviteID,
			status:   http.StatusUnauthorized,
		},
		{
			desc:     "view invite with empty auth token",
			token:    "",
			inviteID: inviteID,
			status:   http.StatusUnauthorized,
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

func TestRevokeInvite(t *testing.T) {
	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	org, err := svc.CreateOrg(context.Background(), ownerToken, org)
	assert.Nil(t, err, fmt.Sprintf("Creating Org expected to succeed: %s", err))

	invite, err := svc.CreateOrgInvite(context.Background(), ownerToken, viewer.Email, viewer.Role, org.ID, redirectPathInvite)
	assert.Nil(t, err, fmt.Sprintf("Inviting member expected to succeed: %s", err))

	inviteID := invite.ID

	ts := newServer(svc)
	defer ts.Close()

	client := ts.Client()

	cases := []struct {
		desc     string
		inviteID string
		token    string
		status   int
	}{
		{
			desc:     "revoke invite",
			token:    ownerToken,
			inviteID: inviteID,
			status:   http.StatusNoContent,
		},
		{
			desc:     "revoke invite non-existent invite id",
			token:    ownerToken,
			inviteID: "invalid",
			status:   http.StatusNotFound,
		},
		{
			desc:     "revoke invite with invalid auth token",
			token:    "invalid",
			inviteID: inviteID,
			status:   http.StatusUnauthorized,
		},
		{
			desc:     "revoke invite with empty auth token",
			token:    "",
			inviteID: inviteID,
			status:   http.StatusUnauthorized,
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

func TestRespondInvite(t *testing.T) {
	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	org, err := svc.CreateOrg(context.Background(), ownerToken, org)
	assert.Nil(t, err, fmt.Sprintf("Creating Org expected to succeed: %s", err))

	_, viewerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	_, editorToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: editorID, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	_, adminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	memberships := []auth.OrgMembership{viewer, editor, admin}
	invites := []auth.OrgInvite{}
	for _, membership := range memberships {
		inv, err := svc.CreateOrgInvite(context.Background(), ownerToken, membership.Email, membership.Role, org.ID, redirectPathInvite)
		assert.Nil(t, err, fmt.Sprintf("Inviting members expected to succeed: %s", err))

		invites = append(invites, inv)
	}

	ts := newServer(svc)
	defer ts.Close()

	client := ts.Client()

	cases := []struct {
		desc     string
		body     string
		inviteID string
		response string
		token    string
		status   int
	}{
		{
			desc:     "accept invite",
			inviteID: invites[0].ID,
			body:     toJSON(respondInviteReq{RedirectPath: redirectPathGroupInvite}),
			response: responseAccept,
			token:    viewerToken,
			status:   http.StatusCreated,
		},
		{
			desc:     "decline invite",
			body:     toJSON(respondInviteReq{RedirectPath: redirectPathGroupInvite}),
			inviteID: invites[1].ID,
			response: responseDecline,
			token:    editorToken,
			status:   http.StatusNoContent,
		},
		{
			desc:     "respond to invite with invalid response action",
			body:     toJSON(respondInviteReq{RedirectPath: redirectPathGroupInvite}),
			inviteID: invites[2].ID,
			response: invalidResponse,
			token:    adminToken,
			status:   http.StatusBadRequest,
		},
		{
			desc:     "respond to invite with invalid auth token",
			body:     toJSON(respondInviteReq{RedirectPath: redirectPathGroupInvite}),
			inviteID: invites[2].ID,
			response: responseAccept,
			token:    "invalid",
			status:   http.StatusUnauthorized,
		},
		{
			desc:     "respond to invite with empty auth token",
			body:     toJSON(respondInviteReq{RedirectPath: redirectPathGroupInvite}),
			inviteID: invites[2].ID,
			response: responseAccept,
			token:    "",
			status:   http.StatusUnauthorized,
		},
		{
			desc:     "respond to invite with non-existent id",
			body:     toJSON(respondInviteReq{RedirectPath: redirectPathGroupInvite}),
			inviteID: "invalid",
			response: responseAccept,
			token:    adminToken,
			status:   http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			contentType: contentType,
			url:         fmt.Sprintf("%s/invites/%s/%s", ts.URL, tc.inviteID, tc.response),
			body:        strings.NewReader(tc.body),
			token:       tc.token,
		}

		res, err := req.make()

		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestListInvitesByInvitee(t *testing.T) {
	svc := newService()

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	_, viewerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	orgIDs := []string{}
	invites := []inviteRes{}

	n := 5
	for i := 1; i <= n; i++ {
		org, err := svc.CreateOrg(context.Background(), ownerToken, auth.Org{
			Name: fmt.Sprintf("org%d", i),
		})

		assert.Nil(t, err, fmt.Sprintf("Creating Org expected to succeed: %s", err))
		orgIDs = append(orgIDs, org.ID)

		inv, err := svc.CreateOrgInvite(context.Background(), ownerToken, viewerEmail, auth.Viewer, org.ID, redirectPathInvite)

		assert.Nil(t, err, fmt.Sprintf("Inviting member expected to succeed: %s", err))
		invites = append(invites, inviteRes{
			ID:          inv.ID,
			InviteeID:   inv.InviteeID.String,
			InviterID:   inv.InviterID,
			OrgID:       inv.OrgID,
			InviteeRole: inv.InviteeRole,
			CreatedAt:   inv.CreatedAt,
			ExpiresAt:   inv.ExpiresAt,
			State:       invitesCommon.InviteStatePending,
		})
	}

	cases := []struct {
		desc   string
		url    string
		token  string
		status int
		res    []inviteRes
	}{
		{
			desc:   "list invites",
			url:    fmt.Sprintf("%s/users/%s/invites/received", ts.URL, viewerID),
			token:  viewerToken,
			status: http.StatusOK,
			res:    invites,
		},
		{
			desc:   "list invites with invalid auth token",
			url:    fmt.Sprintf("%s/users/%s/invites/received", ts.URL, viewerID),
			token:  "invalid",
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list invites with empty auth token",
			url:    fmt.Sprintf("%s/users/%s/invites/received", ts.URL, viewerID),
			token:  "",
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list invites with negative offset",
			url:    fmt.Sprintf("%s/users/%s/invites/received?offset=%d", ts.URL, viewerID, -1),
			token:  "",
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list invites with negative limit",
			url:    fmt.Sprintf("%s/users/%s/invites/received?offset=%d&limit=%d", ts.URL, viewerID, 0, -1),
			token:  "",
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list invites without offset",
			url:    fmt.Sprintf("%s/users/%s/invites/received?limit=%d", ts.URL, viewerID, 2),
			token:  viewerToken,
			status: http.StatusOK,
			res:    invites[:2],
		},
		{
			desc:   "list invites without limit",
			url:    fmt.Sprintf("%s/users/%s/invites/received?offset=%d", ts.URL, viewerID, 0),
			token:  viewerToken,
			status: http.StatusOK,
			res:    invites,
		},
		{
			desc:   "list invites with invalid limit",
			url:    fmt.Sprintf("%s/users/%s/invites/received?offset=%d&limit=%s", ts.URL, viewerID, 0, "l"),
			token:  viewerToken,
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
		var data invitesRes
		err = json.NewDecoder(res.Body).Decode(&data)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.ElementsMatch(t, tc.res, data.Invites, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data.Invites))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

type inviteRes struct {
	ID          string    `json:"id"`
	InviteeID   string    `json:"invitee_id"`
	InviterID   string    `json:"inviter_id"`
	OrgID       string    `json:"org_id"`
	InviteeRole string    `json:"invitee_role"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	State       string    `json:"state"`
}

type pageRes struct {
	Limit  uint64 `json:"limit"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
}

type invitesRes struct {
	pageRes
	Invites []inviteRes `json:"invites"`
}
