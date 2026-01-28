// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package profiles_test

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
	email                  = "user@example.com"
	adminEmail             = "admin@example.com"
	otherUserEmail         = "other_user@example.com"
	token                  = email
	otherToken             = otherUserEmail
	adminToken             = adminEmail
	wrongValue             = "wrong_value"
	emptyValue             = ""
	emptyJson              = "{}"
	wrongID                = 0
	password               = "password"
	maxNameSize            = 1024
	nameKey                = "name"
	ascKey                 = "asc"
	descKey                = "desc"
	orgID                  = "374106f7-030e-4881-8ab0-151195c29f92"
	orgID2                 = "374106f7-030e-4881-8ab0-151195c29f93"
	prefix                 = "fe6b4e92-cc98-425e-b0aa-"
	n                      = 101
	noLimit                = -1
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
	thing = things.Thing{
		Name:     "test_app",
		Metadata: metadata,
	}
	profile = things.Profile{
		Name:     "test",
		Metadata: metadata,
	}
	profile1 = things.Profile{
		Name:     "test1",
		Metadata: metadata,
	}
	user            = users.User{ID: "574106f7-030e-4881-8ab0-151195c29f94", Email: email, Password: password, Role: auth.Owner}
	otherUser       = users.User{ID: "ecf9e48b-ba3b-41c4-82a9-72e063b17868", Email: otherUserEmail, Password: password, Role: auth.Editor}
	admin           = users.User{ID: "2e248e36-2d26-46ea-97b0-1e38d674cbe4", Email: adminEmail, Password: password, Role: auth.RootSub}
	usersList       = []users.User{admin, user, otherUser}
	group           = things.Group{Name: "test-group", Description: "test-group-desc", OrgID: orgID}
	orgsList        = []auth.Org{{ID: orgID, OwnerID: user.ID}, {ID: orgID2, OwnerID: user.ID}}
	metadata        = map[string]any{"test": "data"}
	invalidName     = strings.Repeat("m", maxNameSize+1)
	invalidNameData = fmt.Sprintf(`{"limit":5,"offset":0,"name":"%s"}`, invalidName)
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

func TestCreateProfiles(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	data := `[{"name": "1"}, {"name": "2"}]`
	invalidData := fmt.Sprintf(`[{"name": "%s"}]`, invalidName)

	cases := []struct {
		desc        string
		data        string
		contentType string
		auth        string
		status      int
		response    string
	}{
		{
			desc:        "create valid profiles",
			data:        data,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusCreated,
			response:    emptyValue,
		},
		{
			desc:        "create profile with empty request",
			data:        emptyValue,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create profiles with empty JSON",
			data:        "[]",
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create profile with invalid auth token",
			data:        data,
			contentType: contentTypeJSON,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
			response:    emptyValue,
		},
		{
			desc:        "create profile with empty auth token",
			data:        data,
			contentType: contentTypeJSON,
			auth:        emptyValue,
			status:      http.StatusUnauthorized,
			response:    emptyValue,
		},
		{
			desc:     "create profile with invalid request format",
			data:     "}",
			auth:     token,
			status:   http.StatusUnsupportedMediaType,
			response: emptyValue,
		},
		{
			desc:        "create profile without content type",
			data:        data,
			contentType: emptyValue,
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
			response:    emptyValue,
		},
		{
			desc:        "create profile with invalid name",
			data:        invalidData,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/groups/%s/profiles", ts.URL, gr.ID),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.data),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		location := res.Header.Get("Location")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.response, location, fmt.Sprintf("%s: expected response %s got %s", tc.desc, tc.response, location))
	}
}

func TestUpdateProfile(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	prs, err := svc.CreateProfiles(context.Background(), token, gr.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	pr := prs[0]

	c := profile
	c.Name = "updated_profile"
	updateData := toJSON(c)

	c.Name = invalidName
	invalidData := toJSON(c)

	cases := []struct {
		desc        string
		req         string
		id          string
		contentType string
		auth        string
		status      int
	}{
		{
			desc:        "update existing profile",
			req:         updateData,
			id:          pr.ID,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update non-existing profile",
			req:         updateData,
			id:          strconv.FormatUint(wrongID, 10),
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update profile with invalid id",
			req:         updateData,
			id:          wrongValue,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update profile with invalid token",
			req:         updateData,
			id:          pr.ID,
			contentType: contentTypeJSON,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update profile with empty token",
			req:         updateData,
			id:          pr.ID,
			contentType: contentTypeJSON,
			auth:        emptyValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update profile with invalid data format",
			req:         "}",
			id:          pr.ID,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update profile with empty JSON object",
			req:         "{}",
			id:          pr.ID,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update profile with empty request",
			req:         emptyValue,
			id:          pr.ID,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update profile with missing content type",
			req:         updateData,
			id:          pr.ID,
			contentType: emptyValue,
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "update profile with invalid name",
			req:         invalidData,
			contentType: contentTypeJSON,
			auth:        token,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/profiles/%s", ts.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestViewProfile(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	prs, err := svc.CreateProfiles(context.Background(), token, gr.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	pr := prs[0]

	data := profileRes{
		ID:       pr.ID,
		Name:     pr.Name,
		GroupID:  pr.GroupID,
		Metadata: pr.Metadata,
	}

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
		res    profileRes
	}{
		{
			desc:   "view existing profile",
			id:     pr.ID,
			auth:   token,
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "view non-existent profile",
			id:     strconv.FormatUint(wrongID, 10),
			auth:   token,
			status: http.StatusNotFound,
			res:    profileRes{},
		},
		{
			desc:   "view profile with invalid token",
			id:     pr.ID,
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			res:    profileRes{},
		},
		{
			desc:   "view profile with empty token",
			id:     pr.ID,
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			res:    profileRes{},
		},
		{
			desc:   "view profile with invalid id",
			id:     wrongValue,
			auth:   token,
			status: http.StatusNotFound,
			res:    profileRes{},
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/profiles/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var body profileRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, body, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body))
	}
}

func TestListProfiles(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profiles := make([]things.Profile, n)
	for i := 0; i < n; i++ {
		name := "name_" + fmt.Sprintf("%03d", i+1)
		id := fmt.Sprintf("%s%012d", prefix, i+1)
		profiles[i] = things.Profile{ID: id, Name: name, Metadata: metadata}
	}

	prs, err := svc.CreateProfiles(context.Background(), token, gr.ID, profiles...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	data := []profileRes{}
	for _, profile := range prs {
		data = append(data, profileRes{
			ID:       profile.ID,
			Name:     profile.Name,
			Metadata: profile.Metadata,
			GroupID:  profile.GroupID,
			Config:   profile.Config,
		})
	}
	profileURL := fmt.Sprintf("%s/profiles", ts.URL)

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []profileRes
	}{
		{
			desc:   "get a list of profiles",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", profileURL, 0, 6),
			res:    data[0:6],
		},
		{
			desc:   "get a list of profiles with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", profileURL, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of profiles with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", profileURL, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of profiles with negative offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", profileURL, -1, 5),
			res:    nil,
		},
		{
			desc:   "get a list of profiles with negative limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", profileURL, 5, -2),
			res:    nil,
		},
		{
			desc:   "get a list of profiles with zero limit and offset 1",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", profileURL, 1, 0),
			res:    nil,
		},
		{
			desc:   "get a list of profiles without offset",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?limit=%d", profileURL, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of profiles without limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d", profileURL, 1),
			res:    data[1:11],
		},
		{
			desc:   "get a list of profiles with redundant query params",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&value=something", profileURL, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of profiles with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", profileURL, 0, 210),
			res:    nil,
		},
		{
			desc:   "get a list of profiles with default URL",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s%s", profileURL, emptyValue),
			res:    data[0:10],
		},
		{
			desc:   "get a list of profiles with invalid number of params",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", profileURL, "?offset=4&limit=4&limit=5&offset=5"),
			res:    nil,
		},
		{
			desc:   "get a list of profiles with invalid offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", profileURL, "?offset=e&limit=5"),
			res:    nil,
		},
		{
			desc:   "get a list of profiles with invalid limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", profileURL, "?offset=5&limit=e"),
			res:    nil,
		},
		{
			desc:   "get a list of profiles with invalid name",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&name=%s", profileURL, 0, 10, invalidName),
			res:    nil,
		},
		{
			desc:   "get a list of profiles sorted by name ascendant",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", profileURL, 0, 6, nameKey, ascKey),
			res:    data[0:6],
		},
		{
			desc:   "get a list of profiles sorted by name descendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", profileURL, 0, 6, nameKey, descKey),
			res:    data[0:6],
		},
		{
			desc:   "get a list of profiles sorted with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", profileURL, 0, 6, "wrong", ascKey),
			res:    nil,
		},
		{
			desc:   "get a list of profiles sorted by name with invalid direction",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", profileURL, 0, 6, nameKey, "wrong"),
			res:    nil,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var body profilesPageRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, body.Profiles, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body.Profiles))
	}
}

func TestListProfilesByOrg(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), adminToken, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grs2, err := svc.CreateGroups(context.Background(), otherToken, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr, gr2 := grs[0], grs2[0]

	profiles := make([]things.Profile, n)
	for i := 0; i < n; i++ {
		suffix := i + 1
		name := "name_" + fmt.Sprintf("%03d", suffix)
		id := fmt.Sprintf("%s%012d", prefix, suffix)
		profiles[i] = things.Profile{ID: id, Name: name, Metadata: metadata}
	}

	prs, err := svc.CreateProfiles(context.Background(), adminToken, gr.ID, profiles...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	data := []profileRes{}
	for _, profile := range prs {
		data = append(data, profileRes{
			ID:       profile.ID,
			Name:     profile.Name,
			Metadata: profile.Metadata,
			GroupID:  profile.GroupID,
			Config:   profile.Config,
		})
	}

	profiles2 := make([]things.Profile, n)
	for i := 0; i < n; i++ {
		suffix := n + i + 1
		name := "name_" + fmt.Sprintf("%03d", suffix)
		id := fmt.Sprintf("%s%012d", prefix, suffix)
		profiles2[i] = things.Profile{ID: id, Name: name, Metadata: metadata}
	}

	prs2, err := svc.CreateProfiles(context.Background(), otherToken, gr2.ID, profiles2...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	data2 := []profileRes{}
	for _, profile := range prs2 {
		data2 = append(data2, profileRes{
			ID:       profile.ID,
			Name:     profile.Name,
			Metadata: profile.Metadata,
			GroupID:  profile.GroupID,
			Config:   profile.Config,
		})
	}

	profileURL := fmt.Sprintf("%s/orgs", ts.URL)

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []profileRes
	}{
		{
			desc:   "get a list of profiles by org the user belongs to",
			auth:   otherToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d", profileURL, orgID, n, 5),
			res:    data2[0:5],
		},
		{
			desc:   "get a list of profiles by org as org owner",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d", profileURL, orgID, 0, 100),
			res:    data[0:100],
		},
		{
			desc:   "get a list of profiles by org with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d", profileURL, orgID, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of profiles by org with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d", profileURL, orgID, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of profiles by org with negative offset",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d", profileURL, orgID, -2, 5),
			res:    nil,
		},
		{
			desc:   "get a list of profiles by org with negative limit",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d", profileURL, orgID, 1, -5),
			res:    nil,
		},
		{
			desc:   "get a list of profiles by org with no limit",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d", profileURL, orgID, 1, 0),
			res:    nil,
		},
		{
			desc:   "get a list of profiles by org without offset",
			auth:   adminToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/profiles?limit=%d", profileURL, orgID, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of profiles by org without limit",
			auth:   adminToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d", profileURL, orgID, 1),
			res:    data[1:11],
		},
		{
			desc:   "get a list of profiles by org with redundant query params",
			auth:   adminToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d&value=something", profileURL, orgID, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of profiles by org with limit greater than max",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d", profileURL, orgID, 0, 210),
			res:    nil,
		},
		{
			desc:   "get a list of things by org with default URL",
			auth:   adminToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/profiles", profileURL, orgID),
			res:    data[0:10],
		},
		{
			desc:   "get a list of profiles by org with invalid number of params",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/profiles%s", profileURL, orgID, "?offset=4&limit=4&limit=5&offset=5"),
			res:    nil,
		},
		{
			desc:   "get a list of profiles by org with invalid offset",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/profiles%s", profileURL, orgID, "?offset=e&limit=5"),
			res:    nil,
		},
		{
			desc:   "get a list of profiles by org with invalid limit",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/profiles%s", profileURL, orgID, "?offset=5&limit=e"),
			res:    nil,
		},
		{
			desc:   "get a list of profiles by org sorted by name ascendant",
			auth:   adminToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d&order=%s&dir=%s", profileURL, orgID, 0, 5, nameKey, ascKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of profiles by org sorted by name descendent",
			auth:   adminToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d&order=%s&dir=%s", profileURL, orgID, 0, 5, nameKey, descKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of profiles by org sorted with invalid order",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d&order=%s&dir=%s", profileURL, orgID, 0, 5, "wrong", ascKey),
			res:    nil,
		},
		{
			desc:   "get a list of profiles by org sorted by name with invalid direction",
			auth:   adminToken,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/profiles?offset=%d&limit=%d&order=%s&dir=%s", profileURL, orgID, 0, 5, nameKey, "wrong"),
			res:    nil,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var body profilesPageRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, body.Profiles, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body.Profiles))
	}
}

func TestSearchProfiles(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profiles := make([]things.Profile, n)
	for i := 0; i < n; i++ {
		suffix := i + 1
		name := "name_" + fmt.Sprintf("%03d", suffix)
		id := fmt.Sprintf("%s%012d", prefix, suffix)
		profiles[i] = things.Profile{ID: id, Name: name, Metadata: metadata}
	}

	prs, err := svc.CreateProfiles(context.Background(), adminToken, gr.ID, profiles...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	data := []profileRes{}
	for _, profile := range prs {
		data = append(data, profileRes{
			ID:       profile.ID,
			Name:     profile.Name,
			Metadata: profile.Metadata,
			GroupID:  profile.GroupID,
			Config:   profile.Config,
		})
	}

	cases := []struct {
		desc   string
		auth   string
		status int
		req    string
		res    []profileRes
	}{
		{
			desc:   "search profiles",
			auth:   token,
			status: http.StatusOK,
			req:    validData,
			res:    data[0:5],
		},
		{
			desc:   "search profiles ordered by name asc",
			auth:   token,
			status: http.StatusOK,
			req:    ascData,
			res:    data[0:5],
		},
		{
			desc:   "search profiles ordered by name desc",
			auth:   token,
			status: http.StatusOK,
			req:    descData,
			res:    data[0:5],
		},
		{
			desc:   "search profiles with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidOrderData,
			res:    nil,
		},
		{
			desc:   "search profiles with invalid dir",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidDirData,
			res:    nil,
		},
		{
			desc:   "search profiles with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search profiles with invalid data",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidData,
			res:    nil,
		},
		{
			desc:   "search profiles with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search profiles with zero limit",
			auth:   token,
			status: http.StatusOK,
			req:    zeroLimitData,
			res:    data[0:10],
		},
		{
			desc:   "search profiles with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidLimitData,
			res:    nil,
		},
		{
			desc:   "search profiles filtering with invalid name",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidNameData,
			res:    nil,
		},
		{
			desc:   "search profiles with empty JSON body",
			auth:   token,
			status: http.StatusOK,
			req:    emptyJson,
			res:    data[0:10],
		},
		{
			desc:   "search profiles with no body",
			auth:   token,
			status: http.StatusOK,
			req:    emptyValue,
			res:    data[0:10],
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodPost,
			url:    fmt.Sprintf("%s/profiles/search", ts.URL),
			token:  tc.auth,
			body:   strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		var body profilesPageRes
		_ = json.NewDecoder(res.Body).Decode(&body)

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, body.Profiles, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body.Profiles))
	}
}

func TestSearchProfilesByGroup(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profiles := make([]things.Profile, n)
	for i := 0; i < n; i++ {
		suffix := i + 1
		name := "name_" + fmt.Sprintf("%03d", suffix)
		id := fmt.Sprintf("%s%012d", prefix, suffix)
		profiles[i] = things.Profile{ID: id, Name: name, Metadata: metadata}
	}

	prs, err := svc.CreateProfiles(context.Background(), adminToken, gr.ID, profiles...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	data := []profileRes{}
	for _, profile := range prs {
		data = append(data, profileRes{
			ID:       profile.ID,
			Name:     profile.Name,
			Metadata: profile.Metadata,
			GroupID:  profile.GroupID,
			Config:   profile.Config,
		})
	}

	cases := []struct {
		desc   string
		auth   string
		status int
		req    string
		res    []profileRes
	}{
		{
			desc:   "search profiles by group",
			auth:   token,
			status: http.StatusOK,
			req:    validData,
			res:    data[0:5],
		},
		{
			desc:   "search profiles by group ordered by name asc",
			auth:   token,
			status: http.StatusOK,
			req:    ascData,
			res:    data[0:5],
		},
		{
			desc:   "search profiles by group ordered by name desc",
			auth:   token,
			status: http.StatusOK,
			req:    descData,
			res:    data[0:5],
		},
		{
			desc:   "search profiles by group with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidOrderData,
			res:    nil,
		},
		{
			desc:   "search profiles by group with invalid dir",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidDirData,
			res:    nil,
		},
		{
			desc:   "search profiles by group with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search profiles by group with invalid data",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidData,
			res:    nil,
		},
		{
			desc:   "search profiles by group with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search profiles by group with zero limit",
			auth:   token,
			status: http.StatusOK,
			req:    zeroLimitData,
			res:    data[0:10],
		},
		{
			desc:   "search profiles by group with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidLimitData,
			res:    nil,
		},
		{
			desc:   "search profiles by group with invalid name",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidNameData,
			res:    nil,
		},
		{
			desc:   "search profiles by group with empty JSON body",
			auth:   token,
			status: http.StatusOK,
			req:    emptyJson,
			res:    data[0:10],
		},
		{
			desc:   "search profiles by group with no body",
			auth:   token,
			status: http.StatusOK,
			req:    emptyValue,
			res:    data[0:10],
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodPost,
			url:    fmt.Sprintf("%s/groups/%s/profiles/search", ts.URL, gr.ID),
			token:  tc.auth,
			body:   strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		var body profilesPageRes
		_ = json.NewDecoder(res.Body).Decode(&body)

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, body.Profiles, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body.Profiles))
	}
}

func TestSearchProfilesByOrg(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profiles := make([]things.Profile, n)
	for i := 0; i < n; i++ {
		suffix := i + 1
		name := "name_" + fmt.Sprintf("%03d", suffix)
		id := fmt.Sprintf("%s%012d", prefix, suffix)
		profiles[i] = things.Profile{ID: id, Name: name, Metadata: metadata}
	}

	prs, err := svc.CreateProfiles(context.Background(), adminToken, gr.ID, profiles...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	data := []profileRes{}
	for _, profile := range prs {
		data = append(data, profileRes{
			ID:       profile.ID,
			Name:     profile.Name,
			Metadata: profile.Metadata,
			GroupID:  profile.GroupID,
			Config:   profile.Config,
		})
	}

	cases := []struct {
		desc   string
		auth   string
		status int
		req    string
		res    []profileRes
	}{
		{
			desc:   "search profiles by org",
			auth:   token,
			status: http.StatusOK,
			req:    validData,
			res:    data[0:5],
		},
		{
			desc:   "search profiles by org ordered by name asc",
			auth:   token,
			status: http.StatusOK,
			req:    ascData,
			res:    data[0:5],
		},
		{
			desc:   "search profiles by org ordered by name desc",
			auth:   token,
			status: http.StatusOK,
			req:    descData,
			res:    data[0:5],
		},
		{
			desc:   "search profiles by org with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidOrderData,
			res:    nil,
		},
		{
			desc:   "search profiles by org with invalid dir",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidDirData,
			res:    nil,
		},
		{
			desc:   "search profiles by org with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search profiles by org with invalid data",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidData,
			res:    nil,
		},
		{
			desc:   "search profiles by org with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search profiles by org with zero limit",
			auth:   token,
			status: http.StatusOK,
			req:    zeroLimitData,
			res:    data[0:10],
		},
		{
			desc:   "search profiles by org with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidLimitData,
			res:    nil,
		},
		{
			desc:   "search profiles by org with invalid name",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidNameData,
			res:    nil,
		},
		{
			desc:   "search profiles by org with empty JSON body",
			auth:   token,
			status: http.StatusOK,
			req:    emptyJson,
			res:    data[0:10],
		},
		{
			desc:   "search profiles by org with no body",
			auth:   token,
			status: http.StatusOK,
			req:    emptyValue,
			res:    data[0:10],
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodPost,
			url:    fmt.Sprintf("%s/orgs/%s/profiles/search", ts.URL, orgID),
			token:  tc.auth,
			body:   strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		var body profilesPageRes
		_ = json.NewDecoder(res.Body).Decode(&body)

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, body.Profiles, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body.Profiles))
	}
}

func TestViewProfileByThing(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	prs, err := svc.CreateProfiles(context.Background(), token, gr.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	ths, err := svc.CreateThings(context.Background(), token, pr.ID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th := ths[0]

	prRes := profileRes{
		ID:       pr.ID,
		Name:     pr.Name,
		GroupID:  pr.GroupID,
		Metadata: pr.Metadata,
	}

	profileURL := fmt.Sprintf("%s/things", ts.URL)

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    profileRes
	}{
		{
			desc:   "view profile by thing",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/profiles", profileURL, th.ID),
			res:    prRes,
		},
		{
			desc:   "view profile by thing with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/profiles", profileURL, th.ID),
			res:    profileRes{},
		},
		{
			desc:   "view profile by thing with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/profiles", profileURL, th.ID),
			res:    profileRes{},
		},
		{
			desc:   "view profile by thing without thing id",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/profiles", profileURL, emptyValue),
			res:    profileRes{},
		},
		{
			desc:   "view profile by thing with wrong thing id",
			auth:   token,
			status: http.StatusNotFound,
			url:    fmt.Sprintf("%s/%s/profiles", profileURL, wrongValue),
			res:    profileRes{},
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var body profileRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, body, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body))
	}
}

func TestRemoveProfile(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID, prID1 := prs[0].ID, prs[1].ID

	_, err = svc.CreateThings(context.Background(), token, prID1, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
	}{
		{
			desc:   "remove profile with invalid token",
			id:     prID,
			auth:   wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove profile with empty token",
			id:     prID,
			auth:   emptyValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove profile with invalid token",
			id:     prID,
			auth:   wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove existing profile",
			id:     prID,
			auth:   token,
			status: http.StatusNoContent,
		},
		{
			desc:   "remove removed profile",
			id:     prID,
			auth:   token,
			status: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/profiles/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestRemoveProfiles(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	cList := []things.Profile{profile, profile, profile1}
	prs, err := svc.CreateProfiles(context.Background(), token, grID, cList...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	_, err = svc.CreateThings(context.Background(), token, prs[2].ID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	profileIDs := []string{prs[0].ID, prs[1].ID}

	cases := []struct {
		desc        string
		data        []string
		auth        string
		contentType string
		status      int
	}{
		{
			desc:        "remove profiles with invalid token",
			data:        profileIDs,
			auth:        wrongValue,
			contentType: contentTypeJSON,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove profiles with empty token",
			data:        profileIDs,
			auth:        emptyValue,
			contentType: contentTypeJSON,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove profiles with invalid content type",
			data:        profileIDs,
			auth:        token,
			contentType: wrongValue,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "remove existing profiles",
			data:        profileIDs,
			auth:        token,
			contentType: contentTypeJSON,
			status:      http.StatusNoContent,
		},
		{
			desc:        "remove non-existent profiles",
			data:        profileIDs,
			auth:        token,
			contentType: contentTypeJSON,
			status:      http.StatusNotFound,
		},
		{
			desc:        "remove profiles with empty profile ids",
			data:        []string{emptyValue},
			auth:        token,
			contentType: contentTypeJSON,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove profiles without profile ids",
			data:        []string{},
			auth:        token,
			contentType: contentTypeJSON,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		data := struct {
			ProfileIDs []string `json:"profile_ids"`
		}{
			tc.data,
		}

		body := toJSON(data)

		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/profiles", ts.URL),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

type profileRes struct {
	ID       string         `json:"id"`
	Name     string         `json:"name,omitempty"`
	GroupID  string         `json:"group_id,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
}

type profilesPageRes struct {
	Profiles []profileRes `json:"profiles"`
	Total    uint64       `json:"total"`
	Offset   uint64       `json:"offset"`
	Limit    uint64       `json:"limit"`
}
