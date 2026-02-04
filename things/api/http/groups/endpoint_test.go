package groups_test

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
	contentTypeJSON        = "application/json"
	contentTypeOctetStream = "application/octet-stream"
	userEmail              = "user@example.com"
	adminEmail             = "admin@example.com"
	otherUserEmail         = "other_user@example.com"
	token                  = userEmail
	otherToken             = otherUserEmail
	adminToken             = adminEmail
	wrongValue             = "wrong_value"
	emptyValue             = ""
	password               = "password"
	orgID                  = "374106f7-030e-4881-8ab0-151195c29f92"
	prefix                 = "fe6b4e92-cc98-425e-b0aa-"
	n                      = 101
	noLimit                = -1
	emptyJson              = "{}"
	maxNameSize            = 1024
	nameKey                = "name"
	ascKey                 = "asc"
	descKey                = "desc"
	validData              = `{"limit":5,"offset":0}`
	descData               = `{"limit":5,"offset":0,"dir":"desc","order":"name"}`
	ascData                = `{"limit":5,"offset":0,"dir":"asc","order":"name"}`
	invalidOrderData       = `{"limit":5,"offset":0,"dir":"asc","order":"wrong"}`
	zeroLimitData          = `{"limit":0,"offset":0}`
	invalidDirData         = `{"limit":5,"offset":0,"dir":"wrong"}`
	invalidLimitData       = `{"limit":210,"offset":0}`
	invalidData            = `{"limit": "invalid"}`
)

var (
	user            = users.User{ID: "574106f7-030e-4881-8ab0-151195c29f94", Email: userEmail, Password: password, Role: auth.Owner}
	otherUser       = users.User{ID: "ecf9e48b-ba3b-41c4-82a9-72e063b17868", Email: otherUserEmail, Password: password, Role: auth.Editor}
	admin           = users.User{ID: "2e248e36-2d26-46ea-97b0-1e38d674cbe4", Email: adminEmail, Password: password, Role: auth.RootSub}
	group           = things.Group{Name: "test-group", Description: "test-group-desc", OrgID: orgID}
	usersList       = []users.User{admin, user, otherUser}
	orgsList        = []auth.Org{{ID: orgID, OwnerID: user.ID}}
	metadata        = map[string]any{"test": "data"}
	invalidNameData = fmt.Sprintf(`{"limit":5,"offset":0,"name":"%s"}`, strings.Repeat("m", maxNameSize+1))
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
	thingsRepo := thmocks.NewThingRepository()
	profilesRepo := thmocks.NewProfileRepository(thingsRepo)
	groupMembershipsRepo := thmocks.NewGroupMembershipsRepository()
	groupsRepo := thmocks.NewGroupRepository(groupMembershipsRepo)
	profileCache := thmocks.NewProfileCache()
	thingCache := thmocks.NewThingCache()
	groupCache := thmocks.NewGroupCache()
	idProvider := uuid.NewMock()
	emailerMock := thmocks.NewEmailer()

	return things.New(auth, nil, thingsRepo, profilesRepo, groupsRepo, groupMembershipsRepo, profileCache, thingCache, groupCache, idProvider, emailerMock)
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

func TestCreateGroups(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	client := ts.Client()
	data := toJSON([]things.Group{group})

	cases := []struct {
		desc   string
		req    string
		ct     string
		orgID  string
		token  string
		status int
	}{
		{
			desc:   "create groups",
			req:    data,
			ct:     contentTypeJSON,
			orgID:  orgID,
			token:  token,
			status: http.StatusCreated,
		},
		{
			desc:   "create groups with invalid auth token",
			req:    data,
			ct:     contentTypeJSON,
			orgID:  orgID,
			token:  wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "create groups with empty auth token",
			req:    data,
			ct:     contentTypeJSON,
			orgID:  orgID,
			token:  emptyValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "create groups without org",
			req:    data,
			ct:     contentTypeJSON,
			orgID:  emptyValue,
			token:  token,
			status: http.StatusBadRequest,
		},
		{
			desc:   "create groups with empty request",
			req:    emptyValue,
			ct:     contentTypeJSON,
			orgID:  orgID,
			token:  token,
			status: http.StatusBadRequest,
		},
		{
			desc:   "create groups with empty JSON array",
			req:    "[]",
			ct:     contentTypeJSON,
			orgID:  orgID,
			token:  token,
			status: http.StatusBadRequest,
		},
		{
			desc:   "create groups with invalid request format",
			req:    "{",
			ct:     contentTypeJSON,
			orgID:  orgID,
			token:  token,
			status: http.StatusBadRequest,
		},
		{
			desc:   "create groups without content type",
			req:    data,
			orgID:  orgID,
			token:  token,
			status: http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/orgs/%s/groups", ts.URL, tc.orgID),
			contentType: tc.ct,
			token:       tc.token,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestViewGroup(t *testing.T) {
	svc := newService()

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	data := groupRes{
		ID:          gr.ID,
		OrgID:       gr.OrgID,
		Name:        gr.Name,
		Description: gr.Description,
		Metadata:    gr.Metadata,
	}

	cases := []struct {
		desc   string
		id     string
		token  string
		status int
		res    groupRes
	}{
		{
			desc:   "view group",
			id:     gr.ID,
			token:  token,
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "view non-existing group",
			id:     wrongValue,
			token:  token,
			status: http.StatusNotFound,
			res:    groupRes{},
		},
		{
			desc:   "view group without group id",
			id:     emptyValue,
			token:  token,
			status: http.StatusBadRequest,
			res:    groupRes{},
		},
		{
			desc:   "view group with invalid auth token",
			id:     gr.ID,
			token:  wrongValue,
			status: http.StatusUnauthorized,
			res:    groupRes{},
		},
		{
			desc:   "view group with empty auth token",
			id:     gr.ID,
			token:  emptyValue,
			status: http.StatusUnauthorized,
			res:    groupRes{},
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/groups/%s", ts.URL, tc.id),
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var data groupRes
		err = json.NewDecoder(res.Body).Decode(&data)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, data, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data))
	}
}

func TestViewGroupByThing(t *testing.T) {
	svc := newService()

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profile := things.Profile{Name: "test"}
	prs, err := svc.CreateProfiles(context.Background(), token, gr.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	thing := things.Thing{Name: "test"}
	ths, err := svc.CreateThings(context.Background(), token, prID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th := ths[0]

	data := groupRes{
		ID:          gr.ID,
		OrgID:       gr.OrgID,
		Name:        gr.Name,
		Description: gr.Description,
		Metadata:    gr.Metadata,
	}

	cases := []struct {
		desc   string
		id     string
		token  string
		status int
		res    groupRes
	}{
		{
			desc:   "view group by thing",
			id:     th.ID,
			token:  token,
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "view group by non-existing thing",
			id:     wrongValue,
			token:  token,
			status: http.StatusNotFound,
			res:    groupRes{},
		},
		{
			desc:   "view group by thing without thing id",
			id:     emptyValue,
			token:  token,
			status: http.StatusBadRequest,
			res:    groupRes{},
		},
		{
			desc:   "view group by thing with invalid auth token",
			id:     th.ID,
			token:  wrongValue,
			status: http.StatusUnauthorized,
			res:    groupRes{},
		},
		{
			desc:   "view group by thing with empty auth token",
			id:     th.ID,
			token:  emptyValue,
			status: http.StatusUnauthorized,
			res:    groupRes{},
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/things/%s/groups", ts.URL, tc.id),
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var data groupRes
		err = json.NewDecoder(res.Body).Decode(&data)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, data, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data))
	}
}

func TestViewGroupByProfile(t *testing.T) {
	svc := newService()

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profile := things.Profile{Name: "test"}
	prs, err := svc.CreateProfiles(context.Background(), token, gr.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	data := groupRes{
		ID:          gr.ID,
		OrgID:       gr.OrgID,
		Name:        gr.Name,
		Description: gr.Description,
		Metadata:    gr.Metadata,
	}

	cases := []struct {
		desc   string
		id     string
		token  string
		status int
		res    groupRes
	}{
		{
			desc:   "view group by profile",
			id:     prID,
			token:  token,
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "view group by non-existing profile",
			id:     wrongValue,
			token:  token,
			status: http.StatusNotFound,
			res:    groupRes{},
		},
		{
			desc:   "view group by profile without profile id",
			id:     emptyValue,
			token:  token,
			status: http.StatusBadRequest,
			res:    groupRes{},
		},
		{
			desc:   "view group by profile with invalid auth token",
			id:     prID,
			token:  wrongValue,
			status: http.StatusUnauthorized,
			res:    groupRes{},
		},
		{
			desc:   "view group by profile with empty auth token",
			id:     prID,
			token:  emptyValue,
			status: http.StatusUnauthorized,
			res:    groupRes{},
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/profiles/%s/groups", ts.URL, tc.id),
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var data groupRes
		err = json.NewDecoder(res.Body).Decode(&data)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, data, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data))
	}
}

func TestUpdateGroup(t *testing.T) {
	svc := newService()

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	grID := grs[0].ID

	ug := things.Group{
		ID:          grID,
		Name:        "updated_name",
		Description: "updated_description",
	}
	data := toJSON(ug)

	cases := []struct {
		desc   string
		req    string
		id     string
		ct     string
		token  string
		status int
	}{
		{
			desc:   "update group",
			req:    data,
			id:     ug.ID,
			ct:     contentTypeJSON,
			token:  token,
			status: http.StatusOK,
		},
		{
			desc:   "update group with invalid auth token",
			req:    data,
			id:     ug.ID,
			ct:     contentTypeJSON,
			token:  wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "update group with empty auth token",
			req:    data,
			id:     ug.ID,
			ct:     contentTypeJSON,
			token:  emptyValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "update group with invalid group id",
			req:    data,
			id:     wrongValue,
			ct:     contentTypeJSON,
			token:  token,
			status: http.StatusNotFound,
		},
		{
			desc:   "update group without group id",
			req:    data,
			id:     emptyValue,
			ct:     contentTypeJSON,
			token:  token,
			status: http.StatusBadRequest,
		},
		{
			desc:   "update group with invalid request format",
			req:    "{",
			id:     ug.ID,
			ct:     contentTypeJSON,
			token:  token,
			status: http.StatusBadRequest,
		},
		{
			desc:   "update group with empty request",
			req:    emptyValue,
			id:     ug.ID,
			ct:     contentTypeJSON,
			token:  token,
			status: http.StatusBadRequest,
		},
		{
			desc:   "update group without content type",
			req:    data,
			id:     ug.ID,
			ct:     emptyValue,
			token:  token,
			status: http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/groups/%s", ts.URL, tc.id),
			token:       tc.token,
			contentType: tc.ct,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestListGroups(t *testing.T) {
	svc := newService()

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	var groups []groupRes
	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		group.Name = fmt.Sprintf("group-%d", i)
		group.Description = fmt.Sprintf("desc-%d", i)
		grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		gr := grs[0]

		groups = append(groups, groupRes{
			ID:          gr.ID,
			OrgID:       gr.OrgID,
			Name:        gr.Name,
			Description: gr.Description,
			Metadata:    gr.Metadata,
		})
	}

	groupsURL := fmt.Sprintf("%s/groups", ts.URL)

	cases := []struct {
		desc   string
		token  string
		status int
		url    string
		res    []groupRes
	}{
		{
			desc:   "list groups",
			token:  token,
			url:    fmt.Sprintf("%s?limit=%d&offset=%d", groupsURL, 5, 0),
			status: http.StatusOK,
			res:    groups[:5],
		},
		{
			desc:   "list groups filtering by name",
			token:  token,
			url:    fmt.Sprintf("%s?limit=%d&offset=%d&name=%s", groupsURL, n, 0, "1"),
			status: http.StatusOK,
			res:    groups[1:2],
		},
		{
			desc:   "list groups with invalid auth token",
			token:  wrongValue,
			url:    fmt.Sprintf("%s?limit=%d&offset=%d", groupsURL, 5, 0),
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list groups with empty auth token",
			token:  "",
			url:    fmt.Sprintf("%s?limit=%d&offset=%d", groupsURL, 5, 0),
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list groups with negative offset",
			token:  token,
			url:    fmt.Sprintf("%s?limit=%d&offset=%d", groupsURL, 0, -5),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list groups with negative limit",
			token:  token,
			url:    fmt.Sprintf("%s?limit=%d&offset=%d", groupsURL, -5, 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list groups without offset",
			token:  token,
			url:    fmt.Sprintf("%s?limit=%d", groupsURL, 5),
			status: http.StatusOK,
			res:    groups[:5],
		},
		{
			desc:   "list groups without limit",
			token:  token,
			url:    fmt.Sprintf("%s?offset=%d", groupsURL, 0),
			status: http.StatusOK,
			res:    groups,
		},
		{
			desc:   "list groups with redundant query params",
			token:  token,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&value=something", groupsURL, 0, 5),
			status: http.StatusOK,
			res:    groups[:5],
		},
		{
			desc:   "list groups with default URL",
			token:  token,
			url:    groupsURL,
			status: http.StatusOK,
			res:    groups,
		},
		{
			desc:   "list groups with invalid limit",
			token:  token,
			url:    fmt.Sprintf("%s?limit=%s&offset=%d", groupsURL, "i", 5),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list groups with invalid offset",
			token:  token,
			url:    fmt.Sprintf("%s?limit=%d&offset=%s", groupsURL, 5, "i"),
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
		assert.ElementsMatch(t, tc.res, data.Groups, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data.Groups))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestListGroupsByOrg(t *testing.T) {
	svc := newService()

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	var groups []groupRes
	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		group.Name = fmt.Sprintf("group-%d", i)
		group.Description = fmt.Sprintf("desc-%d", i)
		grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		gr := grs[0]

		groups = append(groups, groupRes{
			ID:          gr.ID,
			OrgID:       gr.OrgID,
			Name:        gr.Name,
			Description: gr.Description,
			Metadata:    gr.Metadata,
		})
	}

	groupsURL := fmt.Sprintf("%s/orgs", ts.URL)

	cases := []struct {
		desc   string
		token  string
		orgID  string
		status int
		url    string
		res    []groupRes
	}{
		{
			desc:   "list groups by org",
			token:  token,
			url:    fmt.Sprintf("%s/%s/groups?limit=%d&offset=%d", groupsURL, orgID, 5, 0),
			status: http.StatusOK,
			res:    groups[:5],
		},
		{
			desc:   "list groups by org without org",
			token:  token,
			url:    fmt.Sprintf("%s/%s/groups?limit=%d&offset=%d", groupsURL, emptyValue, 5, 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list groups by org filtering by name",
			token:  token,
			url:    fmt.Sprintf("%s/%s/groups?limit=%d&offset=%d&name=%s", groupsURL, orgID, n, 0, "1"),
			status: http.StatusOK,
			res:    groups[1:2],
		},
		{
			desc:   "list groups by org with invalid auth token",
			token:  wrongValue,
			url:    fmt.Sprintf("%s/%s/groups?limit=%d&offset=%d", groupsURL, orgID, 5, 0),
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list groups by org with empty auth token",
			token:  "",
			url:    fmt.Sprintf("%s/%s/groups?limit=%d&offset=%d", groupsURL, orgID, 5, 0),
			status: http.StatusUnauthorized,
			res:    nil,
		},
		{
			desc:   "list groups by org with negative offset",
			token:  token,
			url:    fmt.Sprintf("%s/%s/groups?limit=%d&offset=%d", groupsURL, orgID, 0, -5),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list groups by org with negative limit",
			token:  token,
			url:    fmt.Sprintf("%s/%s/groups?limit=%d&offset=%d", groupsURL, orgID, -5, 0),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list groups by org without offset",
			token:  token,
			url:    fmt.Sprintf("%s/%s/groups?limit=%d", groupsURL, orgID, 5),
			status: http.StatusOK,
			res:    groups[:5],
		},
		{
			desc:   "list groups by org without limit",
			token:  token,
			url:    fmt.Sprintf("%s/%s/groups?offset=%d", groupsURL, orgID, 0),
			status: http.StatusOK,
			res:    groups,
		},
		{
			desc:   "list groups by org with redundant query params",
			token:  token,
			url:    fmt.Sprintf("%s/%s/groups?offset=%d&limit=%d&value=something", groupsURL, orgID, 0, 5),
			status: http.StatusOK,
			res:    groups[:5],
		},
		{
			desc:   "list groups by org with invalid limit",
			token:  token,
			url:    fmt.Sprintf("%s/%s/groups?limit=%s&offset=%d", groupsURL, orgID, "i", 5),
			status: http.StatusBadRequest,
			res:    nil,
		},
		{
			desc:   "list groups by org with invalid offset",
			token:  token,
			url:    fmt.Sprintf("%s/%s/groups?limit=%d&offset=%s", groupsURL, orgID, 5, "i"),
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
		assert.ElementsMatch(t, tc.res, data.Groups, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data.Groups))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestSearchGroups(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	groups := []groupRes{}
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("group_%03d", i+1)
		id := fmt.Sprintf("%s%012d", prefix, i+1)
		gr := things.Group{ID: id, OrgID: orgID, Name: name, Description: "desc", Metadata: metadata}

		grs, err := svc.CreateGroups(context.Background(), token, orgID, gr)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		group := grs[0]

		groups = append(groups, groupRes{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Metadata:    group.Metadata,
			OrgID:       group.OrgID,
		})
	}

	cases := []struct {
		desc   string
		auth   string
		status int
		req    string
		res    []groupRes
	}{
		{
			desc:   "search groups",
			auth:   token,
			status: http.StatusOK,
			req:    validData,
			res:    groups[0:5],
		},
		{
			desc:   "search groups ordered by name asc",
			auth:   token,
			status: http.StatusOK,
			req:    ascData,
			res:    groups[0:5],
		},
		{
			desc:   "search groups ordered by name desc",
			auth:   token,
			status: http.StatusOK,
			req:    descData,
			res:    groups[0:5],
		},
		{
			desc:   "search groups with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidOrderData,
			res:    nil,
		},
		{
			desc:   "search groups with invalid dir",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidDirData,
			res:    nil,
		},
		{
			desc:   "search groups with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search groups with invalid data",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidData,
			res:    nil,
		},
		{
			desc:   "search groups with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search groups with zero limit",
			auth:   token,
			status: http.StatusOK,
			req:    zeroLimitData,
			res:    groups[0:10],
		},
		{
			desc:   "search groups with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidLimitData,
			res:    nil,
		},
		{
			desc:   "search groups filtering with invalid name",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidNameData,
			res:    nil,
		},
		{
			desc:   "search groups with empty JSON body",
			auth:   token,
			status: http.StatusOK,
			req:    emptyJson,
			res:    groups[0:10],
		},
		{
			desc:   "search groups with no body",
			auth:   token,
			status: http.StatusOK,
			req:    emptyValue,
			res:    groups[0:10],
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodPost,
			url:    fmt.Sprintf("%s/groups/search", ts.URL),
			token:  tc.auth,
			body:   strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		var body groupsPageRes
		_ = json.NewDecoder(res.Body).Decode(&body)

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, body.Groups, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body.Groups))
	}
}

func TestSearchGroupsByOrg(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	groups := []groupRes{}
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("group_%03d", i+1)
		id := fmt.Sprintf("%s%012d", prefix, i+1)
		gr := things.Group{ID: id, OrgID: orgID, Name: name, Description: "desc", Metadata: metadata}

		grs, err := svc.CreateGroups(context.Background(), token, orgID, gr)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		group := grs[0]

		groups = append(groups, groupRes{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Metadata:    group.Metadata,
			OrgID:       group.OrgID,
		})
	}

	cases := []struct {
		desc   string
		auth   string
		status int
		req    string
		res    []groupRes
	}{
		{
			desc:   "search groups by org",
			auth:   token,
			status: http.StatusOK,
			req:    validData,
			res:    groups[0:5],
		},
		{
			desc:   "search groups by org ordered by name asc",
			auth:   token,
			status: http.StatusOK,
			req:    ascData,
			res:    groups[0:5],
		},
		{
			desc:   "search groups by org ordered by name desc",
			auth:   token,
			status: http.StatusOK,
			req:    descData,
			res:    groups[0:5],
		},
		{
			desc:   "search groups by org with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidOrderData,
			res:    nil,
		},
		{
			desc:   "search groups by org with invalid dir",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidDirData,
			res:    nil,
		},
		{
			desc:   "search groups by org with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search groups by org with invalid data",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidData,
			res:    nil,
		},
		{
			desc:   "search groups by org with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search groups by org with zero limit",
			auth:   token,
			status: http.StatusOK,
			req:    zeroLimitData,
			res:    groups[0:10],
		},
		{
			desc:   "search groups by org with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidLimitData,
			res:    nil,
		},
		{
			desc:   "search groups by org filtering with invalid name",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidNameData,
			res:    nil,
		},
		{
			desc:   "search groups by org with empty JSON body",
			auth:   token,
			status: http.StatusOK,
			req:    emptyJson,
			res:    groups[0:10],
		},
		{
			desc:   "search groups by org with no body",
			auth:   token,
			status: http.StatusOK,
			req:    emptyValue,
			res:    groups[0:10],
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodPost,
			url:    fmt.Sprintf("%s/orgs/%s/groups/search", ts.URL, orgID),
			token:  tc.auth,
			body:   strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		var body groupsPageRes
		_ = json.NewDecoder(res.Body).Decode(&body)

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, body.Groups, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body.Groups))
	}
}

func TestRemoveGroups(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	var groupIDs []string
	for i := uint64(0); i < 10; i++ {
		num := strconv.FormatUint(i, 10)
		group := things.Group{
			OrgID:       orgID,
			Name:        "test-group-" + num,
			Description: "test group desc",
		}
		grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		gr := grs[0]

		groupIDs = append(groupIDs, gr.ID)
	}
	cases := []struct {
		desc        string
		data        []string
		auth        string
		contentType string
		status      int
	}{
		{
			desc:        "remove existing groups",
			data:        groupIDs[:5],
			auth:        token,
			contentType: contentTypeJSON,
			status:      http.StatusNoContent,
		},
		{
			desc:        "remove non-existent groups",
			data:        []string{wrongValue},
			auth:        token,
			contentType: contentTypeJSON,
			status:      http.StatusNotFound,
		},
		{
			desc:        "remove groups with invalid token",
			data:        groupIDs[len(groupIDs)-5:],
			auth:        wrongValue,
			contentType: contentTypeJSON,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove groups without group ids",
			data:        []string{},
			auth:        token,
			contentType: contentTypeJSON,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove groups with empty group ids",
			data:        []string{emptyValue},
			auth:        token,
			contentType: contentTypeJSON,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove groups with empty token",
			data:        groupIDs,
			auth:        emptyValue,
			contentType: contentTypeJSON,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove groups with invalid content type",
			data:        groupIDs,
			auth:        token,
			contentType: wrongValue,
			status:      http.StatusUnsupportedMediaType,
		},
	}
	for _, tc := range cases {
		data := struct {
			GroupIDs []string `json:"group_ids"`
		}{
			tc.data,
		}
		body := toJSON(data)
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/groups", ts.URL),
			token:       tc.auth,
			contentType: tc.contentType,
			body:        strings.NewReader(body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

type groupRes struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	OrgID       string         `json:"org_id"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type groupsPageRes struct {
	pageRes
	Groups []groupRes `json:"groups"`
}

type pageRes struct {
	Limit  uint64 `json:"limit"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
}
