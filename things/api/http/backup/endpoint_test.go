// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup_test

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
	contentTypeJSON = "application/json"
	email           = "user@example.com"
	adminEmail      = "admin@example.com"
	otherUserEmail  = "other_user@example.com"
	token           = email
	adminToken      = adminEmail
	wrongValue      = "wrong_value"
	emptyValue      = ""
	emptyJson       = "{}"
	password        = "password"
	nameKey         = "name"
	orgID           = "374106f7-030e-4881-8ab0-151195c29f92"
	orgID2          = "374106f7-030e-4881-8ab0-151195c29f93"
	prefix          = "fe6b4e92-cc98-425e-b0aa-"
	n               = 101
)

var (
	user      = users.User{ID: "574106f7-030e-4881-8ab0-151195c29f94", Email: email, Password: password, Role: auth.Owner}
	otherUser = users.User{ID: "ecf9e48b-ba3b-41c4-82a9-72e063b17868", Email: otherUserEmail, Password: password, Role: auth.Editor}
	admin     = users.User{ID: "2e248e36-2d26-46ea-97b0-1e38d674cbe4", Email: adminEmail, Password: password, Role: auth.RootSub}
	usersList = []users.User{admin, user, otherUser}
	orgsList  = []auth.Org{{ID: orgID, OwnerID: user.ID}, {ID: orgID2, OwnerID: user.ID}}
	metadata  = map[string]any{"test": "data"}
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

func TestBackup(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()

	var groups []things.Group
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

		groups = append(groups, gr)
	}
	gr := groups[0]

	profiles := []things.Profile{}
	for i := 0; i < 10; i++ {
		name := "name_" + fmt.Sprintf("%03d", i+1)
		prs, err := svc.CreateProfiles(context.Background(), token, gr.ID,
			things.Profile{
				Name:     name,
				Metadata: metadata,
			})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		pr := prs[0]

		profiles = append(profiles, pr)
	}
	pr := profiles[0]

	ths := []things.Thing{}

	for i := 0; i < 10; i++ {
		name := "name_" + fmt.Sprintf("%03d", i+1)
		things, err := svc.CreateThings(context.Background(), token, pr.ID,
			things.Thing{
				Name:        name,
				Metadata:    metadata,
				ExternalKey: fmt.Sprintf("external_key_%d", i),
			})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th := things[0]

		ths = append(ths, th)
	}

	var thingsRes []viewThingRes
	for _, th := range ths {
		thingsRes = append(thingsRes, viewThingRes{
			ID:          th.ID,
			GroupID:     th.GroupID,
			ProfileID:   th.ProfileID,
			Name:        th.Name,
			Key:         th.Key,
			ExternalKey: th.ExternalKey,
			Metadata:    th.Metadata,
		})
	}

	var profilesRes []backupProfileRes
	for _, pr := range profiles {
		profilesRes = append(profilesRes, backupProfileRes{
			ID:       pr.ID,
			GroupID:  pr.GroupID,
			Name:     pr.Name,
			Metadata: pr.Metadata,
		})
	}

	var groupsRes []viewGroupRes
	for _, gr := range groups {
		groupsRes = append(groupsRes, viewGroupRes{
			ID:          gr.ID,
			OrgID:       gr.OrgID,
			Name:        gr.Name,
			Description: gr.Description,
			Metadata:    gr.Metadata,
		})
	}

	backup := backupRes{
		Groups:   groupsRes,
		Things:   thingsRes,
		Profiles: profilesRes,
	}

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    backupRes
	}{
		{
			desc:   "backup all things, profiles and groups",
			auth:   adminToken,
			status: http.StatusOK,
			res:    backup,
		},
		{
			desc:   "backup with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			res:    backupRes{},
		},
		{
			desc:   "backup with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			res:    backupRes{},
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/backup", ts.URL),
			token:  tc.auth,
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		var body backupRes
		err = json.NewDecoder(res.Body).Decode(&body)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res.Profiles, body.Profiles, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res.Profiles, body.Profiles))
		assert.ElementsMatch(t, tc.res.Things, body.Things, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res.Things, body.Things))
		assert.ElementsMatch(t, tc.res.Groups, body.Groups, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res.Groups, body.Groups))
	}
}

func TestRestore(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	idProvider := uuid.New()

	thId, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thKey, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	testThing := things.Thing{
		ID:          thId,
		Name:        nameKey,
		Key:         thKey,
		ExternalKey: "abc123",
		Metadata:    metadata,
	}

	var groups []things.Group
	for i := uint64(0); i < 10; i++ {
		num := strconv.FormatUint(i, 10)
		gr := things.Group{
			ID:          fmt.Sprintf("%s%012d", prefix, i+1),
			Name:        "test-group-" + num,
			Description: "test group desc",
		}

		groups = append(groups, gr)
	}

	profiles := []things.Profile{}
	for i := 0; i < n; i++ {
		prID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		name := "name_" + fmt.Sprintf("%03d", i+1)
		profiles = append(profiles, things.Profile{
			ID:       prID,
			GroupID:  emptyValue,
			Name:     name,
			Metadata: metadata,
		})
	}

	thr := []restoreThingReq{
		{
			ID:          testThing.ID,
			Name:        testThing.Name,
			Key:         testThing.Key,
			ExternalKey: testThing.ExternalKey,
			Metadata:    testThing.Metadata,
		},
	}

	var prr []restoreProfileReq
	for _, pr := range profiles {
		prr = append(prr, restoreProfileReq{
			ID:       pr.ID,
			Name:     pr.Name,
			Metadata: pr.Metadata,
		})
	}

	var gr []restoreGroupReq
	for _, group := range groups {
		gr = append(gr, restoreGroupReq{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Metadata:    group.Metadata,
		})
	}

	resReq := restoreReq{
		Things:   thr,
		Profiles: prr,
		Groups:   gr,
	}

	data := toJSON(resReq)
	invalidData := toJSON(restoreReq{})

	cases := []struct {
		desc        string
		auth        string
		status      int
		req         string
		contentType string
	}{
		{
			desc:        "restore all things, profiles and groups",
			auth:        adminToken,
			status:      http.StatusCreated,
			req:         data,
			contentType: contentTypeJSON,
		},
		{
			desc:        "restore with invalid token",
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
			req:         data,
			contentType: contentTypeJSON,
		},
		{
			desc:        "restore with empty token",
			auth:        emptyValue,
			status:      http.StatusUnauthorized,
			req:         data,
			contentType: contentTypeJSON,
		},
		{
			desc:        "restore with invalid request",
			auth:        token,
			status:      http.StatusBadRequest,
			req:         invalidData,
			contentType: contentTypeJSON,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/restore", ts.URL),
			token:       tc.auth,
			contentType: tc.contentType,
			body:        strings.NewReader(tc.req),
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

	}
}

type viewThingRes struct {
	ID          string         `json:"id"`
	GroupID     string         `json:"group_id,omitempty"`
	ProfileID   string         `json:"profile_id"`
	Name        string         `json:"name,omitempty"`
	Key         string         `json:"key"`
	ExternalKey string         `json:"external_key,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type backupProfileRes struct {
	ID       string         `json:"id"`
	GroupID  string         `json:"group_id"`
	Name     string         `json:"name,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type viewGroupRes struct {
	ID          string         `json:"id"`
	OrgID       string         `json:"org_id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type backupRes struct {
	Things   []viewThingRes     `json:"things"`
	Profiles []backupProfileRes `json:"profiles"`
	Groups   []viewGroupRes     `json:"groups"`
}

type restoreThingReq struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Key         string         `json:"key"`
	ExternalKey string         `json:"external_key"`
	Metadata    map[string]any `json:"metadata"`
}

type restoreProfileReq struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Metadata map[string]any `json:"metadata"`
}

type restoreGroupReq struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type restoreReq struct {
	Things   []restoreThingReq   `json:"things"`
	Profiles []restoreProfileReq `json:"profiles"`
	Groups   []restoreGroupReq   `json:"groups"`
}
