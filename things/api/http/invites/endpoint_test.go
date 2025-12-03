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
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	httpapi "github.com/MainfluxLabs/mainflux/things/api/http"
	thmocks "github.com/MainfluxLabs/mainflux/things/mocks"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	emptyValue             = ""
	contentTypeJSON        = "application/json"
	contentTypeOctetStream = "application/octet-stream"
	userEmail              = "user@example.com"
	adminEmail             = "admin@example.com"
	viewerEmail            = "viewer@gmail.com"
	editorEmail            = "editor@gmail.com"
	otherUserEmail         = "other_user@example.com"
	userToken              = userEmail
	otherToken             = otherUserEmail
	adminToken             = adminEmail
	viewerToken            = viewerEmail
	editorToken            = editorEmail
	wrongValue             = "wrong_value"
	password               = "password"
	orgID                  = "374106f7-030e-4881-8ab0-151195c29f92"

	responseAccept  = "accept"
	responseDecline = "decline"
	invalidResponse = "wrong"

	redirectPathViewInvite = "/view-group-invite"
)

var (
	user          = users.User{ID: "574106f7-030e-4881-8ab0-151195c29f94", Email: userEmail, Password: password, Role: auth.Owner}
	otherUser     = users.User{ID: "ecf9e48b-ba3b-41c4-82a9-72e063b17868", Email: otherUserEmail, Password: password, Role: auth.Editor}
	admin         = users.User{ID: "2e248e36-2d26-46ea-97b0-1e38d674cbe4", Email: adminEmail, Password: password, Role: auth.RootSub}
	viewer        = users.User{ID: "874106f7-030e-4881-8ab0-151195c29f99", Email: viewerToken, Password: password, Role: auth.Viewer}
	editor        = users.User{ID: "874106f7-030e-4881-8ab0-151195c29f91", Email: editorToken, Password: password, Role: auth.Editor}
	usersList     = []users.User{admin, user, otherUser, viewer, editor, admin}
	usersByEmails = map[string]users.User{userEmail: {ID: user.ID, Email: user.Email}, otherUserEmail: {ID: otherUser.ID, Email: otherUser.Email}, viewerEmail: {ID: viewer.ID, Email: viewer.Email},
		editorEmail: {ID: editor.ID, Email: editor.Email}, adminEmail: admin}
	usersByIDs = map[string]users.User{user.ID: {ID: user.ID, Email: user.Email}, otherUser.ID: {ID: otherUser.ID, Email: otherUser.Email}, viewer.ID: {ID: viewer.ID, Email: viewer.Email},
		editor.ID: {ID: editor.ID, Email: editor.Email}, admin.ID: admin}
	group    = things.Group{Name: "test-group", Description: "test-group-desc", OrgID: orgID}
	orgsList = []auth.Org{{ID: orgID, OwnerID: user.ID}}

	inviteDuration = 7 * 24 * time.Hour
)

type testRequest struct {
	client      *http.Client
	method      string
	url         string
	contentType string
	key         string
	token       string
	body        io.Reader
}

func (tr testRequest) make() (*http.Response, error) {
	req, err := http.NewRequest(tr.method, tr.url, tr.body)
	if err != nil {
		return nil, err
	}
	if tr.key != "" {
		req.Header.Set("Authorization", apiutil.ThingKeyPrefixInternal+tr.key)
	}
	if tr.token != "" {
		req.Header.Set("Authorization", apiutil.BearerPrefix+tr.token)
	}
	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}
	return tr.client.Do(req)
}

func newService() things.Service {
	auth := mocks.NewAuthService(admin.ID, usersList, orgsList)
	uc := thmocks.NewUsersService(usersByIDs, usersByEmails)
	thingsRepo := thmocks.NewThingRepository()
	profilesRepo := thmocks.NewProfileRepository(thingsRepo)
	groupMembershipsRepo := thmocks.NewGroupMembershipsRepository()
	groupsRepo := thmocks.NewGroupRepository(groupMembershipsRepo)
	invitesRepo := thmocks.NewInvitesRepository()
	profileCache := thmocks.NewProfileCache()
	thingCache := thmocks.NewThingCache()
	groupCache := thmocks.NewGroupCache()
	idProvider := uuid.NewMock()

	emailerMock := thmocks.NewEmailer()

	return things.New(auth, uc, thingsRepo, profilesRepo, groupsRepo, invitesRepo, groupMembershipsRepo, profileCache, thingCache, groupCache, idProvider, emailerMock, inviteDuration)
}

func newServer(svc things.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(svc, mocktracer.New(), logger)
	return httptest.NewServer(mux)
}

func toJSON(data any) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func TestCreateGroupInvite(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	client := ts.Client()

	groups, err := svc.CreateGroups(context.Background(), userToken, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := groups[0]

	cases := []struct {
		desc   string
		body   string
		ct     string
		token  string
		status int
	}{
		{
			desc: "create group invite",
			body: toJSON(createInvitesReq{
				Email:        viewerEmail,
				Role:         viewer.Role,
				RedirectPath: redirectPathViewInvite,
			}),
			ct:     contentTypeJSON,
			token:  userToken,
			status: http.StatusCreated,
		},
		{
			desc: "create group invite with invalid auth token",
			body: toJSON(createInvitesReq{
				Email:        viewerEmail,
				Role:         viewer.Role,
				RedirectPath: redirectPathViewInvite,
			}),
			ct:     contentTypeJSON,
			token:  "invalid-token",
			status: http.StatusUnauthorized,
		},
		{
			desc: "create group invite with empty auth token",
			body: toJSON(createInvitesReq{
				Email:        viewerEmail,
				Role:         viewer.Role,
				RedirectPath: redirectPathViewInvite,
			}),
			ct:     contentTypeJSON,
			token:  emptyValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "create group invite with empty request",
			body:   "",
			ct:     contentTypeJSON,
			token:  userToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "create group invite with invalid request format",
			body:   "{",
			ct:     contentTypeJSON,
			token:  userToken,
			status: http.StatusBadRequest,
		},
		{
			desc: "create group invite without content type",
			body: toJSON(createInvitesReq{
				Email:        viewerEmail,
				Role:         viewer.Role,
				RedirectPath: redirectPathViewInvite,
			}),
			ct:     emptyValue,
			token:  userToken,
			status: http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/groups/%s/invites", ts.URL, gr.ID),
			contentType: tc.ct,
			token:       tc.token,
			body:        strings.NewReader(tc.body),
		}

		res, err := req.make()

		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}

}

func TestViewInvite(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	client := ts.Client()

	groups, err := svc.CreateGroups(context.Background(), userToken, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := groups[0]

	invite, err := svc.CreateGroupInvite(context.Background(), userToken, viewerEmail, viewer.Role, gr.ID, redirectPathViewInvite)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc   string
		token  string
		status int
	}{
		{
			desc:   "view group invite as invitee",
			token:  viewerToken,
			status: http.StatusOK,
		},
		{
			desc:   "view group invite as inviter",
			token:  userToken,
			status: http.StatusOK,
		},
		{
			desc:   "view group invite as unauthorized user",
			token:  editorToken,
			status: http.StatusForbidden,
		},
		{
			desc:   "view group invite with invalid auth token",
			token:  "invalid",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "view group invite with empty auth token",
			token:  emptyValue,
			status: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/invites/%s", ts.URL, invite.ID),
			token:  tc.token,
		}

		res, err := req.make()

		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}

}
func TestRevokeInvite(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	client := ts.Client()

	groups, err := svc.CreateGroups(context.Background(), userToken, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := groups[0]

	invite, err := svc.CreateGroupInvite(context.Background(), userToken, viewerEmail, viewer.Role, gr.ID, redirectPathViewInvite)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc   string
		token  string
		status int
	}{
		{
			desc:   "revoke group invite as inviter",
			token:  userToken,
			status: http.StatusNoContent,
		},
		{
			desc:   "revoke group invite as unauthorized user",
			token:  editorToken,
			status: http.StatusForbidden,
		},
		{
			desc:   "revoke group invite with invalid auth token",
			token:  "invalid",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "revoke group invite with empty auth token",
			token:  emptyValue,
			status: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/invites/%s", ts.URL, invite.ID),
			token:  tc.token,
		}

		res, err := req.make()

		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestRespondInvite(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	client := ts.Client()

	groups, err := svc.CreateGroups(context.Background(), userToken, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := groups[0]

	invite, err := svc.CreateGroupInvite(context.Background(), userToken, viewerEmail, viewer.Role, gr.ID, redirectPathViewInvite)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	invite2, err := svc.CreateGroupInvite(context.Background(), userToken, editorEmail, viewer.Role, gr.ID, redirectPathViewInvite)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	invite3, err := svc.CreateGroupInvite(context.Background(), userToken, adminEmail, viewer.Role, gr.ID, redirectPathViewInvite)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc     string
		inviteID string
		response string
		token    string
		status   int
	}{
		{
			desc:     "accept invite",
			inviteID: invite.ID,
			token:    viewerToken,
			response: responseAccept,
			status:   http.StatusCreated,
		},
		{
			desc:     "decline invite",
			inviteID: invite2.ID,
			token:    editorToken,
			response: responseDecline,
			status:   http.StatusNoContent,
		},
		{
			desc:     "respond to invite with invalid response action",
			inviteID: invite3.ID,
			response: invalidResponse,
			token:    adminToken,
			status:   http.StatusBadRequest,
		},
		{
			desc:     "respond to invite with invalid auth token",
			inviteID: invite3.ID,
			response: responseAccept,
			token:    "invalid",
			status:   http.StatusUnauthorized,
		},
		{
			desc:     "respond to invite with empty auth token",
			inviteID: invite3.ID,
			response: responseAccept,
			token:    "",
			status:   http.StatusUnauthorized,
		},
		{
			desc:     "respond to invite with non-existent id",
			inviteID: "invalid",
			response: responseAccept,
			token:    adminToken,
			status:   http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodPost,
			url:    fmt.Sprintf("%s/invites/%s/%s", ts.URL, tc.inviteID, tc.response),
			token:  tc.token,
		}

		res, err := req.make()

		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestListInvitesByUser(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	client := ts.Client()

	var invites []inviteRes

	n := 5
	for i := 1; i <= n; i++ {
		groups, err := svc.CreateGroups(context.Background(), userToken, orgID, things.Group{
			Name:        fmt.Sprintf("gr%d", i),
			Description: "test group",
			OrgID:       orgID,
		})

		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		gr := groups[0]

		inv, err := svc.CreateGroupInvite(context.Background(), userToken, viewerEmail, viewer.Role, gr.ID, redirectPathViewInvite)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		invites = append(invites, inviteRes{
			ID:          inv.ID,
			InviteeID:   inv.InviteeID.String,
			InviterID:   inv.InviterID,
			GroupID:     inv.GroupID,
			InviteeRole: inv.InviteeRole,
			CreatedAt:   inv.CreatedAt,
			ExpiresAt:   inv.ExpiresAt,
			State:       inv.State,
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
			desc:   "list received invites",
			url:    fmt.Sprintf("%s/users/%s/invites/received", ts.URL, viewer.ID),
			token:  viewerToken,
			status: http.StatusOK,
			res:    invites,
		},
		{
			desc:   "list sent invites",
			url:    fmt.Sprintf("%s/users/%s/invites/sent", ts.URL, user.ID),
			token:  userToken,
			status: http.StatusOK,
			res:    invites,
		},
		{
			desc:   "list received invites invalid auth token",
			url:    fmt.Sprintf("%s/users/%s/invites/received", ts.URL, viewer.ID),
			token:  "invalid",
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list invites with empty auth token",
			url:    fmt.Sprintf("%s/users/%s/invites/received", ts.URL, viewer.ID),
			token:  "",
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list received invites with negative offset",
			url:    fmt.Sprintf("%s/users/%s/invites/received?offset=%d", ts.URL, viewer.ID, -1),
			token:  "",
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list received invites with negative limit",
			url:    fmt.Sprintf("%s/users/%s/invites/received?offset=%d&limit=%d", ts.URL, viewer.ID, 0, -1),
			token:  "",
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list invites without offset",
			url:    fmt.Sprintf("%s/users/%s/invites/received?limit=%d", ts.URL, viewer.ID, 2),
			token:  viewerToken,
			status: http.StatusOK,
			res:    invites[:2],
		},
		{
			desc:   "list invites without limit",
			url:    fmt.Sprintf("%s/users/%s/invites/received?offset=%d", ts.URL, viewer.ID, 0),
			token:  viewerToken,
			status: http.StatusOK,
			res:    invites,
		},
		{
			desc:   "list invites with invalid limit",
			url:    fmt.Sprintf("%s/users/%s/invites/received?offset=%d&limit=%s", ts.URL, viewer.ID, 0, "l"),
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

func TestListInvitesByGroup(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	client := ts.Client()

	groups, err := svc.CreateGroups(context.Background(), userToken, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	gr := groups[0]

	var invites []inviteRes

	users := []users.User{viewer, editor}

	for _, user := range users {
		inv, err := svc.CreateGroupInvite(context.Background(), userToken, user.Email, user.Role, gr.ID, redirectPathViewInvite)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		invites = append(invites, inviteRes{
			ID:          inv.ID,
			InviteeID:   inv.InviteeID.String,
			InviterID:   inv.InviterID,
			GroupID:     inv.GroupID,
			InviteeRole: inv.InviteeRole,
			CreatedAt:   inv.CreatedAt,
			ExpiresAt:   inv.ExpiresAt,
			State:       inv.State,
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
			desc:   "list group invites",
			url:    fmt.Sprintf("%s/groups/%s/invites", ts.URL, gr.ID),
			token:  userToken,
			status: http.StatusOK,
			res:    invites,
		},
		{
			desc:   "list group invites as unauthorized user",
			url:    fmt.Sprintf("%s/groups/%s/invites", ts.URL, gr.ID),
			token:  viewerToken,
			status: http.StatusForbidden,
			res:    nil,
		},
		{
			desc:   "list group invites invalid auth token",
			url:    fmt.Sprintf("%s/groups/%s/invites", ts.URL, gr.ID),
			token:  "invalid",
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list group invites with empty auth token",
			url:    fmt.Sprintf("%s/groups/%s/invites", ts.URL, gr.ID),
			token:  "",
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list group invites with negative offset",
			url:    fmt.Sprintf("%s/groups/%s/invites?offset=%d", ts.URL, gr.ID, -1),
			token:  userToken,
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list group invites with negative limit",
			url:    fmt.Sprintf("%s/groups/%s/invites?offset=%d&limit=%d", ts.URL, gr.ID, 0, -1),
			token:  userToken,
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list group invites without offset",
			url:    fmt.Sprintf("%s/groups/%s/invites?limit=%d", ts.URL, gr.ID, 2),
			token:  userToken,
			status: http.StatusOK,
			res:    invites[:2],
		},
		{
			desc:   "list group invites without limit",
			url:    fmt.Sprintf("%s/groups/%s/invites?offset=%d", ts.URL, gr.ID, 0),
			token:  userToken,
			status: http.StatusOK,
			res:    invites,
		},
		{
			desc:   "list group invites with invalid limit",
			url:    fmt.Sprintf("%s/groups/%s/invites?offset=%d&limit=%s", ts.URL, gr.ID, 0, "l"),
			token:  userToken,
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

type createInvitesReq struct {
	Email        string `json:"email,omitempty"`
	Role         string `json:"role,omitempty"`
	RedirectPath string `json:"redirect_path,omitempty"`
}

type inviteRes struct {
	ID          string    `json:"id"`
	InviteeID   string    `json:"invitee_id"`
	InviterID   string    `json:"inviter_id"`
	GroupID     string    `json:"group_id"`
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
