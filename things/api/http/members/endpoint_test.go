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
	emptyValue     = ""
	contentType    = "application/json"
	userEmail      = "user@example.com"
	adminEmail     = "admin@example.com"
	viewerEmail    = "viewer@gmail.com"
	editorEmail    = "editor@gmail.com"
	otherUserEmail = "other_user@example.com"
	token          = userEmail
	viewerToken    = viewerEmail
	editorToken    = editorEmail
	wrongValue     = "wrong_value"
	password       = "password"
	orgID          = "374106f7-030e-4881-8ab0-151195c29f92"
	n              = 5
)

var (
	user          = users.User{ID: "574106f7-030e-4881-8ab0-151195c29f94", Email: userEmail, Password: password, Role: auth.Owner}
	otherUser     = users.User{ID: "ecf9e48b-ba3b-41c4-82a9-72e063b17868", Email: otherUserEmail, Password: password, Role: auth.Editor}
	admin         = users.User{ID: "2e248e36-2d26-46ea-97b0-1e38d674cbe4", Email: adminEmail, Password: password, Role: auth.RootSub}
	viewer        = users.User{ID: "874106f7-030e-4881-8ab0-151195c29f99", Email: viewerToken, Password: password, Role: auth.Viewer}
	editor        = users.User{ID: "874106f7-030e-4881-8ab0-151195c29f91", Email: editorToken, Password: password, Role: auth.Editor}
	usersList     = []users.User{admin, user, otherUser}
	usersByEmails = map[string]users.User{userEmail: {ID: user.ID, Email: user.Email}, otherUserEmail: {ID: otherUser.ID, Email: otherUser.Email}, viewerEmail: {ID: viewer.ID, Email: viewer.Email},
		editorEmail: {ID: editor.ID, Email: editor.Email}}
	usersByIDs = map[string]users.User{user.ID: {ID: user.ID, Email: user.Email}, otherUser.ID: {ID: otherUser.ID, Email: otherUser.Email}, viewer.ID: {ID: viewer.ID, Email: viewer.Email},
		editor.ID: {ID: editor.ID, Email: editor.Email}}
	members = []things.GroupMember{
		{MemberID: otherUser.ID, Email: otherUser.Email, Role: things.Admin},
		{MemberID: viewer.ID, Email: viewer.Email, Role: things.Viewer},
		{MemberID: editor.ID, Email: editor.Email, Role: things.Editor},
	}
	group    = things.Group{Name: "test-group", Description: "test-group-desc", OrgID: orgID}
	orgsList = []auth.Org{{ID: orgID, OwnerID: user.ID}}
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
		req.Header.Set("Authorization", apiutil.ThingPrefix+tr.key)
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
	groupMembersRepo := thmocks.NewGroupMembersRepository()
	groupsRepo := thmocks.NewGroupRepository(groupMembersRepo)
	profileCache := thmocks.NewProfileCache()
	thingCache := thmocks.NewThingCache()
	groupCache := thmocks.NewGroupCache()
	idProvider := uuid.NewMock()

	return things.New(auth, uc, thingsRepo, profilesRepo, groupsRepo, groupMembersRepo, profileCache, thingCache, groupCache, idProvider)
}

func newServer(svc things.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(svc, mocktracer.New(), logger)
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func TestCreateGroupMembers(t *testing.T) {
	svc := newService()

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	gm := groupMember{ID: editor.ID, Email: editor.Email, Role: things.Editor}
	invalidMember := gm
	invalidMember.Role = wrongValue

	data := toJSON(membersReq{GroupMembers: []groupMember{gm}})
	invalidData := toJSON(membersReq{GroupMembers: []groupMember{invalidMember}})

	cases := []struct {
		desc   string
		token  string
		id     string
		req    string
		status int
	}{
		{
			desc:   "create group member",
			token:  token,
			id:     gr.ID,
			req:    data,
			status: http.StatusCreated,
		},
		{
			desc:   "create group member with invalid member role",
			token:  token,
			id:     gr.ID,
			req:    invalidData,
			status: http.StatusBadRequest,
		},
		{
			desc:   "create group member with invalid auth token",
			token:  wrongValue,
			id:     gr.ID,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "create group member with empty token",
			token:  emptyValue,
			id:     gr.ID,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "create group member without group id",
			token:  token,
			id:     emptyValue,
			req:    data,
			status: http.StatusBadRequest,
		},
		{
			desc:   "create group member with invalid request body",
			token:  token,
			id:     gr.ID,
			req:    "{",
			status: http.StatusBadRequest,
		},
		{
			desc:   "create group member without request body",
			token:  token,
			id:     gr.ID,
			req:    emptyValue,
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/groups/%s/members", ts.URL, tc.id),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

	}
}

func TestRemoveGroupMembers(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	for i := range members {
		members[i].GroupID = gr.ID
	}

	err = svc.CreateGroupMembers(context.Background(), token, members...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	data := toJSON(removeMembersReq{MemberIDs: []string{members[1].MemberID, members[2].MemberID}})

	cases := []struct {
		desc   string
		token  string
		id     string
		req    string
		status int
	}{
		{
			desc:   "remove members from group",
			token:  token,
			id:     gr.ID,
			req:    data,
			status: http.StatusNoContent,
		},
		{
			desc:   "remove members from group with invalid auth token",
			token:  wrongValue,
			id:     gr.ID,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove members from group with empty token",
			token:  emptyValue,
			id:     gr.ID,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove members from non-existing group",
			token:  token,
			id:     wrongValue,
			req:    data,
			status: http.StatusNotFound,
		},
		{
			desc:   "remove members from group without group id",
			token:  token,
			id:     emptyValue,
			req:    data,
			status: http.StatusBadRequest,
		},
		{
			desc:   "remove members from group with invalid request body",
			token:  token,
			id:     gr.ID,
			req:    "{",
			status: http.StatusBadRequest,
		},
		{
			desc:   "remove members from group without request body",
			token:  token,
			id:     gr.ID,
			req:    emptyValue,
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/groups/%s/members", ts.URL, tc.id),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

	}
}

func TestUpdateMembers(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	for i := range members {
		members[i].GroupID = gr.ID
	}

	err = svc.CreateGroupMembers(context.Background(), token, members...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	viewerMember := groupMember{ID: viewer.ID, Email: viewer.Email, Role: things.Viewer}
	updtToEditor := viewerMember
	updtToEditor.Role = auth.Editor

	updtToOwner := viewerMember
	updtToOwner.Role = auth.Owner

	viewerRoleData := toJSON(membersReq{GroupMembers: []groupMember{updtToEditor}})
	ownerRoleData := toJSON(membersReq{GroupMembers: []groupMember{updtToOwner}})

	cases := []struct {
		desc   string
		token  string
		id     string
		req    string
		status int
	}{
		{
			desc:   "update group member role",
			token:  token,
			id:     gr.ID,
			req:    viewerRoleData,
			status: http.StatusOK,
		},
		{
			desc:   "update group member role with invalid auth token",
			token:  wrongValue,
			id:     gr.ID,
			req:    viewerRoleData,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "update group member role with empty token",
			token:  emptyValue,
			id:     gr.ID,
			req:    viewerRoleData,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "update group member role with non-existing group",
			token:  token,
			id:     wrongValue,
			req:    viewerRoleData,
			status: http.StatusNotFound,
		},
		{
			desc:   "update group member role without group id",
			token:  token,
			id:     emptyValue,
			req:    viewerRoleData,
			status: http.StatusBadRequest,
		},
		{
			desc:   "update group member role with invalid request body",
			token:  token,
			id:     gr.ID,
			req:    "{",
			status: http.StatusBadRequest,
		},
		{
			desc:   "update group member role without request body",
			token:  token,
			id:     gr.ID,
			req:    emptyValue,
			status: http.StatusBadRequest,
		},
		{
			desc:   "update group member role to owner",
			token:  token,
			id:     gr.ID,
			req:    ownerRoleData,
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/groups/%s/members", ts.URL, tc.id),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

	}
}

func TestListGroupMembers(t *testing.T) {
	svc := newService()

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	for i := range members {
		members[i].GroupID = gr.ID
	}

	err = svc.CreateGroupMembers(context.Background(), token, members...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	var data []groupMember
	for _, m := range members {
		data = append(data, groupMember{
			ID:    m.MemberID,
			Email: m.Email,
			Role:  m.Role,
		})
	}

	owner := groupMember{
		ID:    user.ID,
		Email: userEmail,
		Role:  auth.Owner,
	}
	data = append(data, owner)

	cases := []struct {
		desc   string
		token  string
		url    string
		status int
		res    []groupMember
	}{
		{
			desc:   "list group members",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/members?limit=%d&offset=%d", ts.URL, gr.ID, n, 0),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list group members with invalid auth token",
			token:  wrongValue,
			url:    fmt.Sprintf("%s/groups/%s/members?limit=%d&offset=%d", ts.URL, gr.ID, n, 0),
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list group members without auth token",
			token:  emptyValue,
			url:    fmt.Sprintf("%s/groups/%s/members?limit=%d&offset=%d", ts.URL, gr.ID, n, 0),
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list group members without group id",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/members?limit=%d&offset=%d", ts.URL, emptyValue, n, 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list group members with invalid group id",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/members?limit=%d&offset=%d", ts.URL, wrongValue, n, 0),
			status: http.StatusNotFound,
			res:    nil,
		},
		{
			desc:   "list group members with negative offset",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/members?limit=%d&offset=%d", ts.URL, gr.ID, n, -5),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list group members with negative limit",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/members?limit=%d&offset=%d", ts.URL, gr.ID, -5, 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list group members with invalid offset",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/members?limit=%d&offset=%s", ts.URL, gr.ID, n, "i"),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list group members with invalid limit",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/members?limit=%s&offset=%d", ts.URL, gr.ID, "i", 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list group members without limit",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/members?offset=%d", ts.URL, gr.ID, 0),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list group members without offset",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/members?limit=%d", ts.URL, gr.ID, n),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list group members with default URL",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/members", ts.URL, gr.ID),
			status: http.StatusOK,
			res:    data,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodGet,
			url:         tc.url,
			contentType: contentType,
			token:       tc.token,
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
}

type groupMember struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type membersReq struct {
	GroupMembers []groupMember `json:"group_members"`
}

type removeMembersReq struct {
	MemberIDs []string `json:"member_ids"`
}

type memberPageRes struct {
	pageRes
	Members []groupMember `json:"group_members"`
}
