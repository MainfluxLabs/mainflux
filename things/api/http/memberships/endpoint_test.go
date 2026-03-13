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
	token                  = userEmail
	otherToken             = otherUserEmail
	adminToken             = adminEmail
	viewerToken            = viewerEmail
	editorToken            = editorEmail
	wrongValue             = "wrong_value"
	password               = "password"
	orgID                  = "374106f7-030e-4881-8ab0-151195c29f92"
	n                      = 5
	emailKey               = "email"
	idKey                  = "id"
	ascKey                 = "asc"
	descKey                = "desc"
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
	memberships = []things.GroupMembership{
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
	profileCache := thmocks.NewProfileCache()
	thingCache := thmocks.NewThingCache()
	groupCache := thmocks.NewGroupCache()
	idProvider := uuid.NewMock()
	emailerMock := thmocks.NewEmailer()

	return things.New(auth, uc, thingsRepo, profilesRepo, groupsRepo, groupMembershipsRepo, profileCache, thingCache, groupCache, idProvider, emailerMock)
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

func TestCreateGroupMemberships(t *testing.T) {
	svc := newService()

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	gm := groupMembership{MemberID: editor.ID, Email: editor.Email, Role: things.Editor}
	invalidMembership := gm
	invalidMembership.Role = wrongValue

	data := toJSON(membershipsReq{GroupMemberships: []groupMembership{gm}})
	invalidData := toJSON(membershipsReq{GroupMemberships: []groupMembership{invalidMembership}})

	cases := []struct {
		desc   string
		token  string
		id     string
		req    string
		status int
	}{
		{
			desc:   "create group memberships",
			token:  token,
			id:     gr.ID,
			req:    data,
			status: http.StatusCreated,
		},
		{
			desc:   "create group memberships with invalid member role",
			token:  token,
			id:     gr.ID,
			req:    invalidData,
			status: http.StatusBadRequest,
		},
		{
			desc:   "create group memberships with invalid auth token",
			token:  wrongValue,
			id:     gr.ID,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "create group memberships with empty token",
			token:  emptyValue,
			id:     gr.ID,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "create group memberships without group id",
			token:  token,
			id:     emptyValue,
			req:    data,
			status: http.StatusBadRequest,
		},
		{
			desc:   "create group memberships with invalid request body",
			token:  token,
			id:     gr.ID,
			req:    "{",
			status: http.StatusBadRequest,
		},
		{
			desc:   "create group memberships without request body",
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
			url:         fmt.Sprintf("%s/groups/%s/memberships", ts.URL, tc.id),
			contentType: contentTypeJSON,
			token:       tc.token,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

	}
}

func TestRemoveGroupMemberships(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	for i := range memberships {
		memberships[i].GroupID = gr.ID
	}

	err = svc.CreateGroupMemberships(context.Background(), token, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	data := toJSON(removeMembershipsReq{MemberIDs: []string{memberships[1].MemberID, memberships[2].MemberID}})

	cases := []struct {
		desc   string
		token  string
		id     string
		req    string
		status int
	}{
		{
			desc:   "remove memberships from group",
			token:  token,
			id:     gr.ID,
			req:    data,
			status: http.StatusNoContent,
		},
		{
			desc:   "remove memberships from group with invalid auth token",
			token:  wrongValue,
			id:     gr.ID,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove memberships from group with empty token",
			token:  emptyValue,
			id:     gr.ID,
			req:    data,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove memberships from non-existing group",
			token:  token,
			id:     wrongValue,
			req:    data,
			status: http.StatusNotFound,
		},
		{
			desc:   "remove memberships from group without group id",
			token:  token,
			id:     emptyValue,
			req:    data,
			status: http.StatusBadRequest,
		},
		{
			desc:   "remove memberships from group with invalid request body",
			token:  token,
			id:     gr.ID,
			req:    "{",
			status: http.StatusBadRequest,
		},
		{
			desc:   "remove memberships from group without request body",
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
			url:         fmt.Sprintf("%s/groups/%s/memberships", ts.URL, tc.id),
			contentType: contentTypeJSON,
			token:       tc.token,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

	}
}

func TestUpdateMemberships(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	for i := range memberships {
		memberships[i].GroupID = gr.ID
	}

	err = svc.CreateGroupMemberships(context.Background(), token, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	gm := groupMembership{MemberID: viewer.ID, Email: viewer.Email, Role: things.Viewer}
	editor := gm
	editor.Role = auth.Editor

	owner := gm
	owner.Role = auth.Owner

	viewerData := toJSON(membershipsReq{GroupMemberships: []groupMembership{editor}})
	ownerData := toJSON(membershipsReq{GroupMemberships: []groupMembership{owner}})

	cases := []struct {
		desc   string
		token  string
		id     string
		req    string
		status int
	}{
		{
			desc:   "update group membership",
			token:  token,
			id:     gr.ID,
			req:    viewerData,
			status: http.StatusOK,
		},
		{
			desc:   "update group membership with invalid auth token",
			token:  wrongValue,
			id:     gr.ID,
			req:    viewerData,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "update group membership with empty token",
			token:  emptyValue,
			id:     gr.ID,
			req:    viewerData,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "update group membership with non-existing group",
			token:  token,
			id:     wrongValue,
			req:    viewerData,
			status: http.StatusNotFound,
		},
		{
			desc:   "update group membership without group id",
			token:  token,
			id:     emptyValue,
			req:    viewerData,
			status: http.StatusBadRequest,
		},
		{
			desc:   "update group membership with invalid request body",
			token:  token,
			id:     gr.ID,
			req:    "{",
			status: http.StatusBadRequest,
		},
		{
			desc:   "update group membership without request body",
			token:  token,
			id:     gr.ID,
			req:    emptyValue,
			status: http.StatusBadRequest,
		},
		{
			desc:   "update group membership role to owner",
			token:  token,
			id:     gr.ID,
			req:    ownerData,
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/groups/%s/memberships", ts.URL, tc.id),
			contentType: contentTypeJSON,
			token:       tc.token,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

	}
}

func TestListGroupMemberships(t *testing.T) {
	svc := newService()

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	for i := range memberships {
		memberships[i].GroupID = gr.ID
	}

	err = svc.CreateGroupMemberships(context.Background(), token, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	var data []groupMembership
	for _, m := range memberships {
		data = append(data, groupMembership{
			MemberID: m.MemberID,
			Email:    m.Email,
			Role:     m.Role,
		})
	}

	owner := groupMembership{
		MemberID: user.ID,
		Email:    userEmail,
		Role:     auth.Owner,
	}
	data = append(data, owner)

	dataByEmailAsc := make([]groupMembership, len(data))
	copy(dataByEmailAsc, data)
	sort.Slice(dataByEmailAsc, func(i, j int) bool {
		return dataByEmailAsc[i].Email < dataByEmailAsc[j].Email
	})

	dataByEmailDesc := make([]groupMembership, len(data))
	copy(dataByEmailDesc, data)
	sort.Slice(dataByEmailDesc, func(i, j int) bool {
		return dataByEmailDesc[i].Email > dataByEmailDesc[j].Email
	})

	dataByIDAsc := make([]groupMembership, len(data))
	copy(dataByIDAsc, data)
	sort.Slice(dataByIDAsc, func(i, j int) bool {
		return dataByIDAsc[i].MemberID < dataByIDAsc[j].MemberID
	})

	dataByIDDesc := make([]groupMembership, len(data))
	copy(dataByIDDesc, data)
	sort.Slice(dataByIDDesc, func(i, j int) bool {
		return dataByIDDesc[i].MemberID > dataByIDDesc[j].MemberID
	})

	cases := []struct {
		desc   string
		token  string
		url    string
		status int
		res    []groupMembership
	}{
		{
			desc:   "list group memberships",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/memberships?limit=%d&offset=%d", ts.URL, gr.ID, n, 0),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list group memberships with invalid auth token",
			token:  wrongValue,
			url:    fmt.Sprintf("%s/groups/%s/memberships?limit=%d&offset=%d", ts.URL, gr.ID, n, 0),
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list group memberships without auth token",
			token:  emptyValue,
			url:    fmt.Sprintf("%s/groups/%s/memberships?limit=%d&offset=%d", ts.URL, gr.ID, n, 0),
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list group memberships without group id",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/memberships?limit=%d&offset=%d", ts.URL, emptyValue, n, 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list group memberships with invalid group id",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/memberships?limit=%d&offset=%d", ts.URL, wrongValue, n, 0),
			status: http.StatusNotFound,
			res:    nil,
		},
		{
			desc:   "list group memberships with negative offset",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/memberships?limit=%d&offset=%d", ts.URL, gr.ID, n, -5),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list group memberships with negative limit",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/memberships?limit=%d&offset=%d", ts.URL, gr.ID, -5, 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list group memberships with invalid offset",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/memberships?limit=%d&offset=%s", ts.URL, gr.ID, n, "i"),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list group memberships with invalid limit",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/memberships?limit=%s&offset=%d", ts.URL, gr.ID, "i", 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list group memberships without limit",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/memberships?offset=%d", ts.URL, gr.ID, 0),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list group memberships without offset",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/memberships?limit=%d", ts.URL, gr.ID, n),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list group memberships with default URL",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/memberships", ts.URL, gr.ID),
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "list group memberships filtered by email",
			token:  token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/groups/%s/memberships?email=%s", ts.URL, gr.ID, viewerEmail),
			res: []groupMembership{
				{
					MemberID: viewer.ID,
					Email:    viewer.Email,
					Role:     things.Viewer,
				},
			},
		},
		{
			desc:   "list group memberships filtered by email that doesn't match",
			token:  token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/groups/%s/memberships?email=%s", ts.URL, gr.ID, wrongValue),
			res:    []groupMembership{},
		},
		{
			desc:   "list group memberships sorted by email ascendant",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/memberships?order=%s&dir=%s", ts.URL, gr.ID, emailKey, ascKey),
			status: http.StatusOK,
			res:    dataByEmailAsc,
		},
		{
			desc:   "list group memberships sorted by email descendent",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/memberships?order=%s&dir=%s", ts.URL, gr.ID, emailKey, descKey),
			status: http.StatusOK,
			res:    dataByEmailDesc,
		},
		{
			desc:   "list group memberships sorted by id ascendant",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/memberships?order=%s&dir=%s", ts.URL, gr.ID, idKey, ascKey),
			status: http.StatusOK,
			res:    dataByIDAsc,
		},
		{
			desc:   "list group memberships sorted by id descendent",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/memberships?order=%s&dir=%s", ts.URL, gr.ID, idKey, descKey),
			status: http.StatusOK,
			res:    dataByIDDesc,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodGet,
			url:         tc.url,
			contentType: contentTypeJSON,
			token:       tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var data membershipPageRes
		err = json.NewDecoder(res.Body).Decode(&data)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.ElementsMatch(t, tc.res, data.GroupMemberships, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data.GroupMemberships))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

type pageRes struct {
	Limit  uint64 `json:"limit"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
}

type groupMembership struct {
	MemberID string `json:"member_id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type membershipsReq struct {
	GroupMemberships []groupMembership `json:"group_memberships"`
}

type removeMembershipsReq struct {
	MemberIDs []string `json:"member_ids"`
}

type membershipPageRes struct {
	pageRes
	GroupMemberships []groupMembership `json:"group_memberships"`
}
