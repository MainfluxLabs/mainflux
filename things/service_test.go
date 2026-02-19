// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	authmock "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/things/mocks"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	emptyValue      = ""
	wrongValue      = "wrong-value"
	adminEmail      = "admin@example.com"
	userEmail       = "user@example.com"
	otherUserEmail  = "other.user@example.com"
	unauthUserEmail = "unauth@example.com"
	viewerEmail     = "viewer@gmail.com"
	editorEmail     = "editor@gmail.com"
	adminToken      = adminEmail
	viewerToken     = viewerEmail
	editorToken     = editorEmail
	token           = userEmail
	otherToken      = otherUserEmail
	unauthToken     = unauthUserEmail
	password        = "password"
	n               = uint64(102)
	n2              = uint64(204)
	orgID           = "374106f7-030e-4881-8ab0-151195c29f92"
	prefixID        = "fe6b4e92-cc98-425e-b0aa-"
	prefixName      = "test-"
)

var (
	thing         = things.Thing{Name: "test"}
	thingList     = [n]things.Thing{}
	profileList   = [n]things.Profile{}
	profile       = things.Profile{Name: "test"}
	thsExtID      = []things.Thing{{ID: prefixID + "000000000001", Name: "a"}, {ID: prefixID + "000000000002", Name: "b"}}
	prsExtID      = []things.Profile{{ID: prefixID + "000000000001", Name: "a"}, {ID: prefixID + "000000000002", Name: "b"}}
	user          = users.User{ID: "574106f7-030e-4881-8ab0-151195c29f94", Email: userEmail, Password: password, Role: auth.Owner}
	otherUser     = users.User{ID: "674106f7-030e-4881-8ab0-151195c29f95", Email: otherUserEmail, Password: password, Role: auth.Editor}
	unauthUser    = users.User{ID: "674106f7-030e-4881-8ab0-151195c29f93", Email: unauthUserEmail, Password: password, Role: auth.Viewer}
	admin         = users.User{ID: "874106f7-030e-4881-8ab0-151195c29f97", Email: adminEmail, Password: password, Role: auth.RootSub}
	viewer        = users.User{ID: "874106f7-030e-4881-8ab0-151195c29f99", Email: viewerEmail, Password: password, Role: auth.Viewer}
	editor        = users.User{ID: "874106f7-030e-4881-8ab0-151195c29f91", Email: editorEmail, Password: password, Role: auth.Editor}
	usersList     = []users.User{admin, user, otherUser, viewer, editor, unauthUser}
	usersByEmails = map[string]users.User{userEmail: {ID: user.ID, Email: userEmail}, otherUserEmail: {ID: otherUser.ID, Email: otherToken}, viewerEmail: {ID: viewer.ID, Email: viewer.Email},
		editorEmail: {ID: editor.ID, Email: editor.Email}, unauthUserEmail: unauthUser}
	usersByIDs = map[string]users.User{user.ID: {ID: user.ID, Email: userEmail}, otherUser.ID: {ID: otherUser.ID, Email: otherUserEmail}, viewer.ID: {ID: viewer.ID, Email: viewerEmail},
		editor.ID: {ID: editor.ID, Email: editorEmail}, unauthUser.ID: unauthUser}
	memberships = []things.GroupMembership{
		{MemberID: otherUser.ID, Email: otherUser.Email, Role: things.Admin},
		{MemberID: viewer.ID, Email: viewer.Email, Role: things.Viewer},
		{MemberID: editor.ID, Email: editor.Email, Role: things.Editor},
	}
	createdGroup = things.Group{OrgID: orgID, Name: "test-group", Description: "test-group-desc"}
	orgsList     = []auth.Org{{ID: orgID, OwnerID: user.ID}}
	metadata     = map[string]any{"test": "data"}
)

func newService() things.Service {
	auth := authmock.NewAuthService(admin.ID, usersList, orgsList)
	uc := mocks.NewUsersService(usersByIDs, usersByEmails)
	thingsRepo := mocks.NewThingRepository()
	profilesRepo := mocks.NewProfileRepository(thingsRepo)
	groupMembershipsRepo := mocks.NewGroupMembershipsRepository()
	groupsRepo := mocks.NewGroupRepository(groupMembershipsRepo)
	profileCache := mocks.NewProfileCache()
	thingCache := mocks.NewThingCache()
	groupCache := mocks.NewGroupCache()
	idProvider := uuid.NewMock()
	emailerMock := mocks.NewEmailer()

	return things.New(auth, uc, thingsRepo, profilesRepo, groupsRepo, groupMembershipsRepo, profileCache, thingCache, groupCache, idProvider, emailerMock)
}

func TestInit(t *testing.T) {
	for i := uint64(0); i < n; i++ {
		thingList[i].Name = fmt.Sprintf("name-%d", i+1)
		thingList[i].ID = fmt.Sprintf("%s%012d", prefixID, i+1)
		thingList[i].Key = fmt.Sprintf("%s1%011d", prefixID, i+1)
	}
}

func TestCreateThings(t *testing.T) {
	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	thsExtID[0].GroupID = grID
	thsExtID[0].ProfileID = prID
	thsExtID[1].GroupID = grID
	thsExtID[1].ProfileID = prID

	cases := []struct {
		desc      string
		things    []things.Thing
		token     string
		profileID string
		err       error
	}{
		{
			desc:      "create new things",
			things:    []things.Thing{{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}},
			profileID: prID,
			token:     token,
			err:       nil,
		},
		{
			desc:      "create new thing with wrong profile id",
			things:    []things.Thing{{Name: "e"}},
			profileID: wrongValue,
			token:     token,
			err:       dbutil.ErrNotFound,
		},
		{
			desc:      "create thing with wrong credentials",
			things:    []things.Thing{{Name: "f"}},
			profileID: prID,
			token:     wrongValue,
			err:       errors.ErrAuthentication,
		},
		{
			desc:      "create new things with external UUID",
			things:    thsExtID,
			profileID: prID,
			token:     token,
			err:       nil,
		},
		{
			desc:      "create new things with external wrong UUID",
			things:    []things.Thing{{ID: "b0aa-000000000001", Name: "a"}, {ID: "b0aa-000000000002", Name: "b"}},
			profileID: prID,
			token:     token,
			err:       nil,
		},
	}

	for _, tc := range cases {
		_, err := svc.CreateThings(context.Background(), tc.token, tc.profileID, tc.things...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateThing(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	ths, err := svc.CreateThings(context.Background(), token, prID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	th.Name = "newName"
	th.Key = "newKey"

	other := things.Thing{ID: emptyValue, Key: "x", Name: "y"}

	cases := []struct {
		desc  string
		thing things.Thing
		token string
		err   error
	}{
		{
			desc:  "update name and key of existing thing",
			thing: th,
			token: token,
			err:   nil,
		},
		{
			desc:  "update name and key of thing with wrong credentials",
			thing: th,
			token: wrongValue,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "update name and key of non-existing thing",
			thing: other,
			token: token,
			err:   dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateThing(context.Background(), tc.token, tc.thing)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateThingGroupAndProfile(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup, createdGroup, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID, grID1, grID2 := grs[0].ID, grs[1].ID, grs[2].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	profile1 := profile
	prs1, err := svc.CreateProfiles(context.Background(), token, grID1, profile1)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID1 := prs1[0].ID

	ths, err := svc.CreateThings(context.Background(), token, prID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	invalidPrGr := things.Thing{
		ID:        th.ID,
		ProfileID: prID1,
		GroupID:   grID,
	}

	other := things.Thing{ID: emptyValue, Key: "x", ProfileID: prID}

	cases := []struct {
		desc  string
		thing things.Thing
		token string
		err   error
	}{
		{
			desc:  "update existing thing",
			thing: th,
			token: token,
			err:   nil,
		},
		{
			desc:  "update thing with wrong credentials",
			thing: th,
			token: wrongValue,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "update non-existing thing",
			thing: other,
			token: token,
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "update thing with profile from non-belonging group",
			thing: invalidPrGr,
			token: token,
			err:   errors.ErrAuthorization,
		},
		{
			desc: "update thing with group change and profile from non-belonging group",
			thing: things.Thing{
				ID:        th.ID,
				GroupID:   grID2,
				ProfileID: prID1,
			},
			token: token,
			err:   errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateThingGroupAndProfile(context.Background(), tc.token, tc.thing)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewThing(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	ths, err := svc.CreateThings(context.Background(), token, prID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	cases := map[string]struct {
		id    string
		token string
		err   error
	}{
		"view existing thing": {
			id:    th.ID,
			token: token,
			err:   nil,
		},
		"view existing thing as admin": {
			id:    th.ID,
			token: adminToken,
			err:   nil,
		},
		"view thing with wrong credentials": {
			id:    th.ID,
			token: wrongValue,
			err:   errors.ErrAuthentication,
		},
		"view non-existing thing": {
			id:    emptyValue,
			token: token,
			err:   dbutil.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := svc.ViewThing(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestViewMetadataByKey(t *testing.T) {
	svc := newService()
	idProvider := uuid.New()

	key, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	thing := things.Thing{
		Name:     "test-meta",
		Key:      key,
		Metadata: metadata,
	}
	_, err = svc.CreateThings(context.Background(), token, prID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	otherKey, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		key string
		err error
	}{
		"view thing metadata": {
			key: key,
			err: nil,
		},
		"view metadata from a non-existing thing": {
			key: otherKey,
			err: dbutil.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := svc.ViewMetadataByKey(context.Background(), things.ThingKey{Type: things.KeyTypeInternal, Value: tc.key})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListThings(t *testing.T) {
	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	grs2, err := svc.CreateGroups(context.Background(), otherToken, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID2 := grs2[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prs2, err := svc.CreateProfiles(context.Background(), otherToken, grID2, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID
	prID2 := prs2[0].ID
	thingList[0].Metadata = metadata

	var ths1 []things.Thing
	var suffix uint64
	for i := uint64(0); i < n; i++ {
		suffix = i + 1
		th := thingList[i]
		th.Name = fmt.Sprintf("%s%d", prefixName, suffix)
		th.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		th.Key = fmt.Sprintf("%s%d", prefixID, suffix)

		ths1 = append(ths1, th)
	}

	var ths2 []things.Thing
	for i := uint64(0); i < n; i++ {
		suffix = n + i + 1
		th := thingList[i]
		th.Name = fmt.Sprintf("%s%d", prefixName, suffix)
		th.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		th.Key = fmt.Sprintf("%s%d", prefixID, suffix)

		ths2 = append(ths2, th)
	}

	_, err = svc.CreateThings(context.Background(), token, prID, ths1...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	_, err = svc.CreateThings(context.Background(), otherToken, prID2, ths2...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		token        string
		pageMetadata apiutil.PageMetadata
		size         uint64
		err          error
	}{
		"list things as user from another group": {
			token: otherToken,
			pageMetadata: apiutil.PageMetadata{
				Offset: n,
				Limit:  5,
			},
			size: 5,
			err:  nil,
		},
		"list all things as admin": {
			token: adminToken,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n2,
			},
			size: n2,
			err:  nil,
		},
		"list all things with no limit": {
			token: token,
			pageMetadata: apiutil.PageMetadata{
				Limit: 0,
			},
			size: 0,
			err:  nil,
		},
		"list half": {
			token: token,
			pageMetadata: apiutil.PageMetadata{
				Offset: n,
				Limit:  n2,
			},
			size: n,
			err:  nil,
		},
		"list last thing": {
			token: token,
			pageMetadata: apiutil.PageMetadata{
				Offset: n2 - 1,
				Limit:  n2,
			},
			size: 1,
			err:  nil,
		},
		"list empty set": {
			token: token,
			pageMetadata: apiutil.PageMetadata{
				Offset: uint64(n2) + 1,
				Limit:  n2,
			},
			size: 0,
			err:  nil,
		},
		"list with wrong credentials": {
			token: wrongValue,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  0,
			},
			size: 0,
			err:  errors.ErrAuthentication,
		},
		"list with existing name": {
			token: token,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n2,
				Name:   "test-1",
			},
			size: 1,
			err:  nil,
		},
		"list with non-existent name": {
			token: token,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n2,
				Name:   "wrong",
			},
			size: 0,
			err:  nil,
		},
		"list with metadata": {
			token: token,
			pageMetadata: apiutil.PageMetadata{
				Offset:   0,
				Limit:    n2,
				Metadata: metadata,
			},
			size: n2,
			err:  nil,
		},
		"list all things sorted by name ascendant": {
			token: token,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n2,
				Order:  "name",
				Dir:    "asc",
			},
			size: n2,
			err:  nil,
		},
		"list all things sorted by name descendent": {
			token: token,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n2,
				Order:  "name",
				Dir:    "desc",
			},
			size: n2,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListThings(context.Background(), tc.token, tc.pageMetadata)
		size := uint64(len(page.Things))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

		testSortEntities(t, tc.pageMetadata, page.Things, func(t things.Thing) string { return t.Name })
	}
}

func TestListThingsByProfile(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	prs, err := svc.CreateProfiles(context.Background(), token, gr.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	var ths []things.Thing
	for i := uint64(0); i < n; i++ {
		suffix := i + 1
		th := thingList[i]
		th.Name = fmt.Sprintf("%s%012d", prefixName, suffix)
		th.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		ths = append(ths, th)
	}

	_, err = svc.CreateThings(context.Background(), token, pr.ID, ths...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		token        string
		prID         string
		pageMetadata apiutil.PageMetadata
		size         uint64
		err          error
	}{
		"list all things by existing profile": {
			token: token,
			prID:  pr.ID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		"list all things by existing profile with no limit": {
			token: token,
			prID:  pr.ID,
			pageMetadata: apiutil.PageMetadata{
				Limit: 0,
			},
			size: n,
			err:  nil,
		},
		"list half of things by existing profile": {
			token: token,
			prID:  pr.ID,
			pageMetadata: apiutil.PageMetadata{
				Offset: n / 2,
				Limit:  n,
			},
			size: n / 2,
			err:  nil,
		},
		"list last thing by existing profile": {
			token: token,
			prID:  pr.ID,
			pageMetadata: apiutil.PageMetadata{
				Offset: n - 1,
				Limit:  n,
			},
			size: 1,
			err:  nil,
		},
		"list empty set of things by existing profile": {
			token: token,
			prID:  pr.ID,
			pageMetadata: apiutil.PageMetadata{
				Offset: n + 1,
				Limit:  n,
			},
			size: 0,
			err:  nil,
		},
		"list things by existing profile with wrong credentials": {
			token: wrongValue,
			prID:  pr.ID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  0,
			},
			size: 0,
			err:  errors.ErrAuthentication,
		},
		"list things by non-existent profile with wrong credentials": {
			token: token,
			prID:  "non-existent",
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: 0,
			err:  dbutil.ErrNotFound,
		},
		"list all things by profile sorted by name ascendant": {
			token: token,
			prID:  pr.ID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "asc",
			},
			size: n,
			err:  nil,
		},
		"list all things by profile sorted by name descendent": {
			token: token,
			prID:  pr.ID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "desc",
			},
			size: n,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListThingsByProfile(context.Background(), tc.token, tc.prID, tc.pageMetadata)
		size := uint64(len(page.Things))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

		testSortEntities(t, tc.pageMetadata, page.Things, func(t things.Thing) string { return t.Name })
	}
}

func TestListThingsByOrg(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), adminToken, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grs2, err := svc.CreateGroups(context.Background(), otherToken, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr, gr2 := grs[0], grs2[0]

	prs, err := svc.CreateProfiles(context.Background(), adminToken, gr.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	prs2, err := svc.CreateProfiles(context.Background(), otherToken, gr2.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr, pr2 := prs[0], prs2[0]

	var ths []things.Thing
	for i := uint64(0); i < n; i++ {
		suffix := i + 1
		th := thing
		th.Name = fmt.Sprintf("%s%012d", prefixName, suffix)
		th.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		ths = append(ths, th)
	}

	var ths2 []things.Thing
	for i := uint64(0); i < n; i++ {
		suffix := n + i + 1
		th2 := thing
		th2.Name = fmt.Sprintf("%s%012d", prefixName, suffix)
		th2.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		ths2 = append(ths2, th2)
	}

	_, err = svc.CreateThings(context.Background(), adminToken, pr.ID, ths...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	_, err = svc.CreateThings(context.Background(), otherToken, pr2.ID, ths2...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		token        string
		orgID        string
		pageMetadata apiutil.PageMetadata
		size         uint64
		err          error
	}{
		"list things by org as admin": {
			token: adminToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n2,
			},
			size: n2,
			err:  nil,
		},
		"list things by org as org owner": {
			token: token,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n2,
			},
			size: n2,
			err:  nil,
		},
		"list things by org from groups the user belongs to": {
			token: otherToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: n,
				Limit:  5,
			},
			size: 5,
			err:  nil,
		},
		"list all things by org with no limit": {
			token: adminToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Limit: 0,
			},
			size: 0,
			err:  nil,
		},
		"list half of things by org": {
			token: adminToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: n,
				Limit:  n2,
			},
			size: n,
			err:  nil,
		},
		"list last thing by org": {
			token: adminToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: n2 - 1,
				Limit:  n2,
			},
			size: 1,
			err:  nil,
		},
		"list empty set of things by org": {
			token: adminToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: n2 + 1,
				Limit:  n2,
			},
			size: 0,
			err:  nil,
		},
		"list things by org with wrong credentials": {
			token: wrongValue,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  0,
			},
			size: 0,
			err:  errors.ErrAuthentication,
		},
		"list things by non-existent org": {
			token: adminToken,
			orgID: "non-existent",
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: 0,
			err:  nil,
		},
		"list all things by org sorted by name ascendant": {
			token: adminToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "asc",
			},
			size: n,
			err:  nil,
		},
		"list all things by org sorted by name descendent": {
			token: adminToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "desc",
			},
			size: n,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListThingsByOrg(context.Background(), tc.token, tc.orgID, tc.pageMetadata)
		size := uint64(len(page.Things))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

		testSortEntities(t, tc.pageMetadata, page.Things, func(t things.Thing) string { return t.Name })
	}
}

func TestRemoveThings(t *testing.T) {
	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	ths, err := svc.CreateThings(context.Background(), token, prID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "remove thing with wrong credentials",
			id:    th.ID,
			token: wrongValue,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "remove existing thing",
			id:    th.ID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove removed thing",
			id:    th.ID,
			token: token,
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "remove non-existing thing",
			id:    emptyValue,
			token: token,
			err:   dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveThings(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestCreateProfiles(t *testing.T) {
	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID
	prsExtID[0].GroupID = grID
	prsExtID[1].GroupID = grID
	cases := []struct {
		desc     string
		profiles []things.Profile
		token    string
		groupID  string
		err      error
	}{
		{
			desc:     "create new profiles",
			profiles: []things.Profile{{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}},
			token:    token,
			groupID:  grID,
			err:      nil,
		},
		{
			desc:     "create new profile with wrong group id",
			profiles: []things.Profile{{Name: "e"}},
			token:    token,
			groupID:  wrongValue,
			err:      dbutil.ErrNotFound,
		},
		{
			desc:     "create profile with wrong credentials",
			profiles: []things.Profile{{Name: "f"}},
			token:    wrongValue,
			groupID:  grID,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "create new profiles with external UUID",
			profiles: prsExtID,
			token:    token,
			groupID:  grID,
			err:      nil,
		},
		{
			desc:     "create new profiles with invalid external UUID",
			profiles: []things.Profile{{ID: "b0aa-000000000001", Name: "a"}, {ID: "b0aa-000000000002", Name: "b"}},
			token:    token,
			groupID:  grID,
			err:      nil,
		},
	}

	for _, cc := range cases {
		_, err := svc.CreateProfiles(context.Background(), cc.token, cc.groupID, cc.profiles...)
		assert.True(t, errors.Contains(err, cc.err), fmt.Sprintf("%s: expected %s got %s\n", cc.desc, cc.err, err))
	}
}

func TestUpdateProfile(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	prs, err := svc.CreateProfiles(context.Background(), token, gr.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	pr := prs[0]
	other := things.Profile{ID: emptyValue}

	cases := []struct {
		desc    string
		profile things.Profile
		token   string
		err     error
	}{
		{
			desc:    "update existing profile",
			profile: pr,
			token:   token,
			err:     nil,
		},
		{
			desc:    "update profile with wrong credentials",
			profile: pr,
			token:   wrongValue,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "update non-existing profile",
			profile: other,
			token:   token,
			err:     dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateProfile(context.Background(), tc.token, tc.profile)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewProfile(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	prs, err := svc.CreateProfiles(context.Background(), token, gr.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	pr := prs[0]

	cases := map[string]struct {
		id       string
		token    string
		err      error
		metadata map[string]any
	}{
		"view existing profile": {
			id:    pr.ID,
			token: token,
			err:   nil,
		},
		"view existing profile as admin": {
			id:    pr.ID,
			token: adminToken,
			err:   nil,
		},
		"view profile with wrong credentials": {
			id:    pr.ID,
			token: wrongValue,
			err:   errors.ErrAuthentication,
		},
		"view non-existing profile": {
			id:    emptyValue,
			token: token,
			err:   dbutil.ErrNotFound,
		},
		"view profile with metadata": {
			id:    emptyValue,
			token: token,
			err:   dbutil.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := svc.ViewProfile(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListProfiles(t *testing.T) {
	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	grs2, err := svc.CreateGroups(context.Background(), otherToken, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr2 := grs2[0]
	profileList[0].Metadata = metadata

	var prs1 []things.Profile
	var suffix uint64
	for i := uint64(0); i < n; i++ {
		suffix = i + 1
		pr := profileList[i]
		pr.Name = fmt.Sprintf("%s%d", prefixName, suffix)
		pr.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		prs1 = append(prs1, pr)
	}

	var prs2 []things.Profile
	for i := uint64(0); i < n; i++ {
		suffix = n + i + 1
		pr := profileList[i]
		pr.Name = fmt.Sprintf("%s%d", prefixName, suffix)
		pr.ID = fmt.Sprintf("%s%012d", prefixID, suffix)

		prs2 = append(prs2, pr)
	}

	_, err = svc.CreateProfiles(context.Background(), token, gr.ID, prs1...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	_, err = svc.CreateProfiles(context.Background(), otherToken, gr2.ID, prs2...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		token        string
		pageMetadata apiutil.PageMetadata
		size         uint64
		err          error
	}{
		"list profiles as user from another group": {
			token: otherToken,
			pageMetadata: apiutil.PageMetadata{
				Offset: n,
				Limit:  5,
			},
			size: 5,
			err:  nil,
		},
		"list all profiles as admin": {
			token: adminToken,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n2,
			},
			size: n2,
			err:  nil,
		},
		"list all profiles with no limit": {
			token: token,
			pageMetadata: apiutil.PageMetadata{
				Limit: 0,
			},
			size: 0,
			err:  nil,
		},
		"list half": {
			token: token,
			pageMetadata: apiutil.PageMetadata{
				Offset: n,
				Limit:  n2,
			},
			size: n,
			err:  nil,
		},
		"list last profile": {
			token: token,
			pageMetadata: apiutil.PageMetadata{
				Offset: n2 - 1,
				Limit:  n2,
			},
			size: 1,
			err:  nil,
		},
		"list empty set": {
			token: token,
			pageMetadata: apiutil.PageMetadata{
				Offset: uint64(n2) + 1,
				Limit:  n2,
			},
			size: 0,
			err:  nil,
		},
		"list with wrong credentials": {
			token: wrongValue,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  0,
			},
			size: 0,
			err:  errors.ErrAuthentication,
		},
		"list with existing name": {
			token: token,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n2,
				Name:   "test-1",
			},
			size: 1,
			err:  nil,
		},
		"list with non-existent name": {
			token: token,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n2,
				Name:   "wrong",
			},
			size: 0,
			err:  nil,
		},
		"list all profiles with metadata": {
			token: token,
			pageMetadata: apiutil.PageMetadata{
				Offset:   0,
				Limit:    n2,
				Metadata: metadata,
			},
			size: n2,
			err:  nil,
		},
		"list all profiles sorted by name ascendant": {
			token: token,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n2,
				Order:  "name",
				Dir:    "asc",
			},
			size: n2,
			err:  nil,
		},
		"list all profiles sorted by name descendent": {
			token: token,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n2,
				Order:  "name",
				Dir:    "desc",
			},
			size: n2,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListProfiles(context.Background(), tc.token, tc.pageMetadata)
		size := uint64(len(page.Profiles))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

		testSortEntities(t, tc.pageMetadata, page.Profiles, func(p things.Profile) string { return p.Name })
	}
}

func TestListProfilesByOrg(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), adminToken, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grs2, err := svc.CreateGroups(context.Background(), otherToken, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	gr, gr2 := grs[0], grs2[0]
	var prs1 []things.Profile
	for i := uint64(0); i < n; i++ {
		suffix := i + 1
		pr := profile
		pr.Name = fmt.Sprintf("%s%d", prefixName, suffix)
		pr.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		prs1 = append(prs1, pr)
	}

	var prs2 []things.Profile
	for i := uint64(0); i < n; i++ {
		suffix := n + i + 1
		pr2 := profile
		pr2.Name = fmt.Sprintf("%s%d", prefixName, suffix)
		pr2.ID = fmt.Sprintf("%s%012d", prefixID, suffix)

		prs2 = append(prs2, pr2)
	}

	_, err = svc.CreateProfiles(context.Background(), adminToken, gr.ID, prs1...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	_, err = svc.CreateProfiles(context.Background(), otherToken, gr2.ID, prs2...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		token        string
		orgID        string
		pageMetadata apiutil.PageMetadata
		size         uint64
		err          error
	}{
		"list profiles by org as admin": {
			token: adminToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n2,
			},
			size: n2,
			err:  nil,
		},
		"list profiles by org as org owner": {
			token: token,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n2,
			},
			size: n2,
			err:  nil,
		},
		"list profiles by org from groups the user belongs to": {
			token: otherToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: n,
				Limit:  5,
			},
			size: 5,
			err:  nil,
		},
		"list profiles by org with no limit": {
			token: adminToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Limit: 0,
			},
			size: 0,
			err:  nil,
		},
		"list half of profiles by org": {
			token: adminToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: n,
				Limit:  n2,
			},
			size: n,
			err:  nil,
		},
		"list last profile by org": {
			token: adminToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: n2 - 1,
				Limit:  n2,
			},
			size: 1,
			err:  nil,
		},
		"list empty set of profiles by org": {
			token: adminToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: n2 + 1,
				Limit:  n2,
			},
			size: 0,
			err:  nil,
		},
		"list profiles by org with wrong credentials": {
			token: wrongValue,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  0,
			},
			size: 0,
			err:  errors.ErrAuthentication,
		},
		"list profiles by non-existent org": {
			token: adminToken,
			orgID: "non-existent",
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: 0,
			err:  nil,
		},
		"list all profiles by org sorted by name ascendant": {
			token: adminToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "asc",
			},
			size: n,
			err:  nil,
		},
		"list all profiles by org sorted by name descendent": {
			token: adminToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "desc",
			},
			size: n,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListProfilesByOrg(context.Background(), tc.token, tc.orgID, tc.pageMetadata)
		size := uint64(len(page.Profiles))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

		testSortEntities(t, tc.pageMetadata, page.Profiles, func(p things.Profile) string { return p.Name })
	}
}

func TestViewProfileByThing(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	p := profile
	p.Name = "test-profile"

	prs, err := svc.CreateProfiles(context.Background(), token, gr.ID, p)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	ths, err := svc.CreateThings(context.Background(), token, pr.ID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th := ths[0]

	cases := map[string]struct {
		token   string
		thID    string
		profile things.Profile
		err     error
	}{
		"view profile by existing thing": {
			token:   token,
			thID:    th.ID,
			profile: pr,
			err:     nil,
		},
		"view profile by existing thing as admin": {
			token:   adminToken,
			thID:    th.ID,
			profile: pr,
			err:     nil,
		},
		"view profile by existing thing with wrong credentials": {
			token:   wrongValue,
			thID:    th.ID,
			profile: things.Profile{},
			err:     errors.ErrAuthentication,
		},
		"view profile by non-existent thing": {
			token:   token,
			thID:    "non-existent",
			profile: things.Profile{},
			err:     dbutil.ErrNotFound,
		},
		"view profile by existent thing with invalid token": {
			token:   wrongValue,
			thID:    th.ID,
			profile: things.Profile{},
			err:     errors.ErrAuthentication,
		},
	}

	for desc, tc := range cases {
		pr, err := svc.ViewProfileByThing(context.Background(), tc.token, tc.thID)
		assert.Equal(t, tc.profile, pr, fmt.Sprintf("%s: expected %v got %v\n", desc, tc.profile, pr))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRemoveProfile(t *testing.T) {
	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	prID, prID1 := prs[0].ID, prs[1].ID

	_, err = svc.CreateThings(context.Background(), token, prID1, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "remove profile with wrong credentials",
			id:    prID,
			token: wrongValue,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "remove existing profile",
			id:    prID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove removed profile",
			id:    prID,
			token: token,
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "remove non-existing profile",
			id:    emptyValue,
			token: token,
			err:   dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveProfiles(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestGetPubConfigByKey(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	prs, err := svc.CreateProfiles(context.Background(), token, gr.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	ths, err := svc.CreateThings(context.Background(), token, pr.ID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	cases := map[string]struct {
		key string
		err error
	}{
		"allowed access": {
			key: th.Key,
			err: nil,
		},
		"non-existing thing": {
			key: wrongValue,
			err: dbutil.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := svc.GetPubConfigByKey(context.Background(), things.ThingKey{Type: things.KeyTypeInternal, Value: tc.key})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected '%s' got '%s'\n", desc, tc.err, err))
	}
}

func TestIdentify(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	prs, err := svc.CreateProfiles(context.Background(), token, gr.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	ths, err := svc.CreateThings(context.Background(), token, pr.ID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	externalKey := "abc123"
	err = svc.UpdateExternalKey(context.Background(), token, externalKey, th.ID)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := map[string]struct {
		key     string
		keyType string
		id      string
		err     error
	}{
		"identify thing with internal key": {
			key:     th.Key,
			keyType: things.KeyTypeInternal,
			id:      th.ID,
			err:     nil,
		},
		"identify thing with invalid internal key": {
			key:     wrongValue,
			keyType: things.KeyTypeInternal,
			id:      emptyValue,
			err:     dbutil.ErrNotFound,
		},
		"identify thing with external key": {
			key:     externalKey,
			keyType: things.KeyTypeExternal,
			id:      th.ID,
			err:     nil,
		},
		"identify thing with invalid external key": {
			key:     wrongValue,
			keyType: things.KeyTypeExternal,
			id:      emptyValue,
			err:     dbutil.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		id, err := svc.Identify(context.Background(), things.ThingKey{Value: tc.key, Type: tc.keyType})
		assert.Equal(t, tc.id, id, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.id, id))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestCreateGroups(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc  string
		token string
		orgID string
		group things.Group
		err   error
	}{
		{
			desc:  "create group",
			token: token,
			orgID: orgID,
			group: createdGroup,
			err:   nil,
		},
		{
			desc:  "create group with wrong credentials",
			token: wrongValue,
			orgID: orgID,
			group: createdGroup,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "create group without credentials",
			token: emptyValue,
			orgID: orgID,
			group: createdGroup,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "create group without org",
			token: token,
			orgID: "",
			group: things.Group{Name: "test-group", Description: "test"},
			err:   dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := svc.CreateGroups(context.Background(), tc.token, tc.orgID, tc.group)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}

}

func TestListGroups(t *testing.T) {
	svc := newService()
	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		createdGroup.Name = fmt.Sprintf("group-%d", i)
		createdGroup.Description = fmt.Sprintf("desc-%d", i)
		_, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	cases := []struct {
		desc  string
		token string
		meta  apiutil.PageMetadata
		size  uint64
		err   error
	}{
		{
			desc:  "list groups",
			token: token,
			meta: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:  "list groups as system admin",
			token: adminToken,
			meta: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:  "list groups with wrong credentials",
			token: wrongValue,
			meta:  apiutil.PageMetadata{},
			size:  0,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "list groups without credentials",
			token: emptyValue,
			meta:  apiutil.PageMetadata{},
			size:  0,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "list half of total groups",
			token: token,
			meta: apiutil.PageMetadata{
				Offset: n / 2,
				Limit:  n,
			},
			size: n / 2,
			err:  nil,
		},
		{
			desc:  "list last group",
			token: token,
			meta: apiutil.PageMetadata{
				Offset: n - 1,
				Limit:  n,
			},
			size: 1,
			err:  nil,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListGroups(context.Background(), tc.token, tc.meta)
		size := uint64(len(page.Groups))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.size, size))
	}
}

func TestListGroupsByOrg(t *testing.T) {
	svc := newService()

	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		createdGroup.Name = fmt.Sprintf("group-%d", i)
		createdGroup.Description = fmt.Sprintf("desc-%d", i)
		_, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	cases := map[string]struct {
		token        string
		orgID        string
		pageMetadata apiutil.PageMetadata
		size         uint64
		err          error
	}{
		"list groups by org as root admin": {
			token: adminToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		"list groups by org as org owner": {
			token: token,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		"list groups by org with no limit": {
			token: adminToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Limit: 0,
			},
			size: 0,
			err:  nil,
		},
		"list half of groups by org": {
			token: adminToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		"list last group by org": {
			token: adminToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: n - 1,
				Limit:  n,
			},
			size: 1,
			err:  nil,
		},
		"list groups by org with wrong credentials": {
			token: wrongValue,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  0,
			},
			size: 0,
			err:  errors.ErrAuthentication,
		},
		"list groups by non-existent org": {
			token: adminToken,
			orgID: "non-existent",
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: 0,
			err:  nil,
		},
		"list all groups by org sorted by name ascendant": {
			token: adminToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "asc",
			},
			size: n,
			err:  nil,
		},
		"list all groups by org sorted by name descendent": {
			token: adminToken,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "desc",
			},
			size: n,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListGroupsByOrg(context.Background(), tc.token, tc.orgID, tc.pageMetadata)
		size := uint64(len(page.Groups))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

		testSortEntities(t, tc.pageMetadata, page.Groups, func(g things.Group) string { return g.Name })
	}
}

func TestRemoveGroup(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)

	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	grID := grs[0].ID

	for i := range memberships {
		memberships[i].GroupID = grID
	}
	err = svc.CreateGroupMemberships(context.Background(), token, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc  string
		token string
		id    string
		err   error
	}{
		{
			desc:  "remove group with wrong credentials",
			token: wrongValue,
			id:    grID,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "remove group without credentials",
			token: emptyValue,
			id:    grID,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "remove non-existing group",
			token: token,
			id:    wrongValue,
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "remove group as viewer",
			token: viewerToken,
			id:    grID,
			err:   errors.ErrAuthorization,
		},
		{
			desc:  "remove group as editor",
			token: editorToken,
			id:    grID,
			err:   errors.ErrAuthorization,
		},
		{
			desc:  "remove group as admin",
			token: otherToken,
			id:    grID,
			err:   errors.ErrAuthorization,
		},
		{
			desc:  "remove group as owner",
			token: token,
			id:    grID,
			err:   nil,
		},
		{
			desc:  "remove removed group",
			token: token,
			id:    grID,
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "remove non-existing group",
			token: token,
			id:    wrongValue,
			err:   dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveGroups(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateGroup(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	grID := grs[0].ID

	for i := range memberships {
		memberships[i].GroupID = grID
	}
	err = svc.CreateGroupMemberships(context.Background(), token, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	ug := things.Group{
		ID:          grID,
		Name:        "updated_name",
		Description: "updated_description",
	}

	cases := []struct {
		desc  string
		token string
		group things.Group
		err   error
	}{
		{
			desc:  "update group as viewer",
			token: viewerToken,
			group: ug,
			err:   errors.ErrAuthorization,
		},
		{
			desc:  "update group as editor",
			token: editorToken,
			group: ug,
			err:   errors.ErrAuthorization,
		},
		{
			desc:  "update group as admin",
			token: otherToken,
			group: ug,
			err:   nil,
		},
		{
			desc:  "update group as owner",
			token: token,
			group: ug,
			err:   nil,
		},
		{
			desc:  "update group with wrong credentials",
			token: wrongValue,
			group: ug,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "update group without credentials",
			token: emptyValue,
			group: ug,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "update non-existing group",
			token: token,
			group: things.Group{},
			err:   dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := svc.UpdateGroup(context.Background(), tc.token, tc.group)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewGroup(t *testing.T) {
	svc := newService()

	createdGroup.Metadata = map[string]any{"test": "meta"}
	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	gr := grs[0]

	for i := range memberships {
		memberships[i].GroupID = gr.ID
	}
	err = svc.CreateGroupMemberships(context.Background(), token, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	grRes := things.Group{
		ID:          gr.ID,
		OrgID:       gr.OrgID,
		Name:        gr.Name,
		Description: gr.Description,
		Metadata:    gr.Metadata,
	}

	cases := []struct {
		desc  string
		token string
		grID  string
		res   things.Group
		err   error
	}{
		{
			desc:  "view group as owner",
			token: token,
			grID:  gr.ID,
			res:   grRes,
			err:   nil,
		},
		{
			desc:  "view group as viewer",
			token: viewerToken,
			grID:  gr.ID,
			res:   grRes,
			err:   nil,
		},
		{
			desc:  "view group as editor",
			token: editorToken,
			grID:  gr.ID,
			res:   grRes,
			err:   nil,
		},
		{
			desc:  "view group as admin",
			token: adminToken,
			grID:  gr.ID,
			res:   grRes,
			err:   nil,
		},
		{
			desc:  "view group as system admin",
			token: adminToken,
			grID:  gr.ID,
			res:   grRes,
			err:   nil,
		},
		{
			desc:  "view group with wrong credentials",
			token: wrongValue,
			grID:  gr.ID,
			res:   things.Group{},
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "view group without credentials",
			token: emptyValue,
			grID:  gr.ID,
			res:   things.Group{},
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "view group without ID",
			token: token,
			grID:  emptyValue,
			res:   things.Group{},
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "view non-existing group",
			token: token,
			grID:  wrongValue,
			res:   things.Group{},
			err:   dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		res, err := svc.ViewGroup(context.Background(), tc.token, tc.grID)
		gr := things.Group{
			ID:          res.ID,
			OrgID:       res.OrgID,
			Name:        res.Name,
			Description: res.Description,
			Metadata:    res.Metadata,
		}
		assert.Equal(t, tc.res, gr, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.res, gr))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewGroupByThing(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	gr := grs[0]

	prs, err := svc.CreateProfiles(context.Background(), adminToken, gr.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	ths, err := svc.CreateThings(context.Background(), token, prID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th := ths[0]

	grRes := things.Group{
		ID:          gr.ID,
		OrgID:       gr.OrgID,
		Name:        gr.Name,
		Description: gr.Description,
		Metadata:    gr.Metadata,
	}

	cases := []struct {
		desc  string
		token string
		thID  string
		res   things.Group
		err   error
	}{
		{
			desc:  "view group by thing",
			token: token,
			thID:  th.ID,
			res:   grRes,
			err:   nil,
		},
		{
			desc:  "view group by thing with wrong credentials",
			token: wrongValue,
			thID:  th.ID,
			res:   things.Group{},
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "view group by thing without credentials",
			token: emptyValue,
			thID:  th.ID,
			res:   things.Group{},
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "view group by thing without ID",
			token: token,
			thID:  emptyValue,
			res:   things.Group{},
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "view group by non-existing thing",
			token: token,
			thID:  wrongValue,
			res:   things.Group{},
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "view group by thing without user rights",
			token: otherToken,
			thID:  th.ID,
			res:   things.Group{},
			err:   dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		res, err := svc.ViewGroupByThing(context.Background(), tc.token, tc.thID)
		gr := things.Group{
			ID:          res.ID,
			OrgID:       res.OrgID,
			Name:        res.Name,
			Description: res.Description,
			Metadata:    res.Metadata,
		}
		assert.Equal(t, tc.res, gr, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.res, gr))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewGroupByProfile(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	gr := grs[0]

	prs, err := svc.CreateProfiles(context.Background(), adminToken, gr.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	grRes := things.Group{
		ID:          gr.ID,
		OrgID:       gr.OrgID,
		Name:        gr.Name,
		Description: gr.Description,
		Metadata:    gr.Metadata,
	}

	cases := []struct {
		desc  string
		token string
		prID  string
		res   things.Group
		err   error
	}{
		{
			desc:  "view group by profile",
			token: token,
			prID:  prID,
			res:   grRes,
			err:   nil,
		},
		{
			desc:  "view group by profile with wrong credentials",
			token: wrongValue,
			prID:  prID,
			res:   things.Group{},
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "view group by profile without credentials",
			token: emptyValue,
			prID:  prID,
			res:   things.Group{},
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "view group by profile without ID",
			token: token,
			prID:  emptyValue,
			res:   things.Group{},
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "view group by profile with non-existing profile",
			token: token,
			prID:  wrongValue,
			res:   things.Group{},
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "view group by profile without user rights",
			token: otherToken,
			prID:  prID,
			res:   things.Group{},
			err:   dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		res, err := svc.ViewGroupByProfile(context.Background(), tc.token, tc.prID)
		gr := things.Group{
			ID:          res.ID,
			OrgID:       res.OrgID,
			Name:        res.Name,
			Description: res.Description,
			Metadata:    res.Metadata,
		}
		assert.Equal(t, tc.res, gr, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.res, gr))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestCreateGroupMemberships(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	for i := range memberships {
		memberships[i].GroupID = gr.ID
	}
	gms := []things.GroupMembership{memberships[1], memberships[2]}
	gm := things.GroupMembership{MemberID: "1", GroupID: gr.ID, Email: "member@gmail.com", Role: things.Viewer}

	cases := []struct {
		desc        string
		token       string
		memberships []things.GroupMembership
		err         error
	}{
		{
			desc:        "create group memberships as owner",
			token:       token,
			memberships: []things.GroupMembership{{MemberID: otherUser.ID, GroupID: gr.ID, Email: otherUserEmail, Role: things.Admin}},
			err:         nil,
		},
		{
			desc:        "create group memberships as admin",
			token:       otherToken,
			memberships: gms,
			err:         nil,
		},
		{
			desc:        "create group memberships as editor",
			token:       editorToken,
			memberships: []things.GroupMembership{gm},
			err:         errors.ErrAuthorization,
		},
		{
			desc:        "create group memberships as viewer",
			token:       viewerToken,
			memberships: []things.GroupMembership{gm},
			err:         errors.ErrAuthorization,
		},
		{
			desc:        "create group memberships with wrong credentials",
			token:       wrongValue,
			memberships: []things.GroupMembership{gm},
			err:         errors.ErrAuthentication,
		},
		{
			desc:        "create group memberships without credentials",
			token:       emptyValue,
			memberships: []things.GroupMembership{gm},
			err:         errors.ErrAuthentication,
		},
		{
			desc:        "create group memberships without group id",
			token:       token,
			memberships: []things.GroupMembership{{MemberID: "2", Email: "member2@gmail.com", Role: things.Viewer}},
			err:         dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.CreateGroupMemberships(context.Background(), tc.token, tc.memberships...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestListGroupMemberships(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	for i := range memberships {
		memberships[i].GroupID = gr.ID
	}
	err = svc.CreateGroupMemberships(context.Background(), token, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	var n uint64 = 4

	cases := []struct {
		desc    string
		token   string
		groupID string
		meta    apiutil.PageMetadata
		size    uint64
		err     error
	}{
		{
			desc:    "list group memberships as owner",
			token:   token,
			groupID: gr.ID,
			meta: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:    "list group memberships as admin",
			token:   adminToken,
			groupID: gr.ID,
			meta: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:    "list group memberships as editor",
			token:   editorToken,
			groupID: gr.ID,
			meta: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:    "list group memberships as viewer",
			token:   viewerToken,
			groupID: gr.ID,
			meta: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:    "list group memberships as system admin",
			token:   adminToken,
			groupID: gr.ID,
			meta: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:    "list half group memberships",
			token:   token,
			groupID: gr.ID,
			meta: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n / 2,
			},
			size: n / 2,
			err:  nil,
		},
		{
			desc:    "list last group membership",
			token:   token,
			groupID: gr.ID,
			meta: apiutil.PageMetadata{
				Offset: n - 1,
				Limit:  1,
			},
			size: 1,
			err:  nil,
		},
		{
			desc:    "list group memberships with wrong credentials",
			token:   wrongValue,
			groupID: gr.ID,
			meta:    apiutil.PageMetadata{},
			size:    0,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "list group memberships without credentials",
			token:   emptyValue,
			groupID: gr.ID,
			meta:    apiutil.PageMetadata{},
			size:    0,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "list memberships from non-existing group",
			token:   token,
			groupID: wrongValue,
			meta:    apiutil.PageMetadata{},
			size:    0,
			err:     dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListGroupMemberships(context.Background(), tc.token, tc.groupID, tc.meta)
		size := uint64(len(page.GroupMemberships))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s expected %d got %d\n", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateMemberships(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	for i := range memberships {
		memberships[i].GroupID = gr.ID
	}
	err = svc.CreateGroupMemberships(context.Background(), token, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	gm := things.GroupMembership{GroupID: gr.ID, MemberID: user.ID, Email: user.Email, Role: things.Owner}

	cases := []struct {
		desc       string
		token      string
		membership things.GroupMembership
		err        error
	}{
		{
			desc:       "update group membership as viewer",
			token:      viewerToken,
			membership: memberships[1],
			err:        errors.ErrAuthorization,
		},
		{
			desc:       "update group membership as editor",
			token:      editorToken,
			membership: memberships[2],
			err:        errors.ErrAuthorization,
		},
		{
			desc:       "update group membership as admin",
			token:      otherToken,
			membership: memberships[2],
			err:        nil,
		},
		{
			desc:       "update group membership as owner",
			token:      token,
			membership: memberships[1],
			err:        nil,
		},
		{
			desc:       "update group owner role as owner",
			token:      token,
			membership: gm,
			err:        errors.ErrAuthorization,
		},
		{
			desc:       "update group owner role as admin",
			token:      otherToken,
			membership: gm,
			err:        errors.ErrAuthorization,
		},
		{
			desc:       "update group membership with wrong credentials",
			token:      wrongValue,
			membership: memberships[1],
			err:        errors.ErrAuthentication,
		},
		{
			desc:       "update group membership without credentials",
			token:      emptyValue,
			membership: memberships[1],
			err:        errors.ErrAuthentication,
		},
		{
			desc:       "update group membership with non-existing group",
			token:      token,
			membership: things.GroupMembership{MemberID: editor.ID, GroupID: wrongValue, Email: editor.Email, Role: things.Editor},
			err:        dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateGroupMemberships(context.Background(), tc.token, tc.membership)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveGroupMemberships(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	for i := range memberships {
		memberships[i].GroupID = gr.ID
	}
	err = svc.CreateGroupMemberships(context.Background(), token, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc     string
		token    string
		groupID  string
		memberID string
		err      error
	}{
		{
			desc:     "remove membership from group as viewer",
			token:    viewerToken,
			groupID:  gr.ID,
			memberID: editor.ID,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "remove membership from group as editor",
			token:    editorToken,
			groupID:  gr.ID,
			memberID: viewer.ID,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "remove membership from group as admin",
			token:    otherToken,
			groupID:  gr.ID,
			memberID: viewer.ID,
			err:      nil,
		},
		{
			desc:     "remove owner from group as admin",
			token:    otherToken,
			groupID:  gr.ID,
			memberID: user.ID,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "remove membership from group as owner",
			token:    token,
			groupID:  gr.ID,
			memberID: editor.ID,
			err:      nil,
		},
		{
			desc:     "remove membership with wrong credentials",
			token:    wrongValue,
			groupID:  gr.ID,
			memberID: editor.ID,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "remove membership without credentials",
			token:    emptyValue,
			groupID:  gr.ID,
			memberID: editor.ID,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "remove membership from non-existing group",
			token:    token,
			groupID:  wrongValue,
			memberID: editor.ID,
			err:      dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveGroupMemberships(context.Background(), tc.token, tc.groupID, tc.memberID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateThingKey(t *testing.T) {
	svc := newService()

	createdGroups, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	createdGroup := createdGroups[0]

	for i := range memberships {
		memberships[i].GroupID = createdGroup.ID
	}
	err = svc.CreateGroupMemberships(context.Background(), token, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	createdProfiles, err := svc.CreateProfiles(context.Background(), token, createdGroup.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	createdProfileID := createdProfiles[0].ID

	createdThings, err := svc.CreateThings(context.Background(), token, createdProfileID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	createdThing := createdThings[0]

	cases := []struct {
		desc        string
		token       string
		externalKey string
		err         error
	}{
		{
			desc:        "update external thing key as admin",
			token:       otherToken,
			externalKey: "abc123",
			err:         nil,
		},
		{
			desc:        "update existing external thing key as admin",
			token:       otherToken,
			externalKey: "abc123",
			err:         dbutil.ErrConflict,
		},
		{
			desc:        "update external thing key as editor",
			token:       editorToken,
			externalKey: "def123",
			err:         nil,
		},
		{
			desc:        "update external thing key as viewer",
			token:       viewerToken,
			externalKey: "ghi123",
			err:         errors.ErrAuthorization,
		},
		{
			desc:        "update external thing key as unauthorized user",
			token:       unauthToken,
			externalKey: "ghi123",
			err:         errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateExternalKey(context.Background(), tc.token, tc.externalKey, createdThing.ID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveExternalKey(t *testing.T) {
	svc := newService()

	createdGroups, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	createdGroup := createdGroups[0]

	for i := range memberships {
		memberships[i].GroupID = createdGroup.ID
	}
	err = svc.CreateGroupMemberships(context.Background(), token, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	createdProfiles, err := svc.CreateProfiles(context.Background(), token, createdGroup.ID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	createdProfileID := createdProfiles[0].ID

	createdThings, err := svc.CreateThings(context.Background(), token, createdProfileID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	createdThing := createdThings[0]

	externalKey := "abc123"
	err = svc.UpdateExternalKey(context.Background(), adminToken, externalKey, createdThing.ID)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc    string
		token   string
		thingID string
		err     error
	}{
		{
			desc:    "remove external key as admin",
			token:   otherToken,
			thingID: createdThing.ID,
			err:     nil,
		},
		{
			desc:    "remove external key as editor",
			token:   editorToken,
			thingID: createdThing.ID,
			err:     nil,
		},
		{
			desc:    "remove external key as viewer",
			token:   viewerToken,
			thingID: createdThing.ID,
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "remove external key as unauthorized user",
			token:   unauthToken,
			thingID: createdThing.ID,
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "remove external key of invalid thing id",
			token:   otherToken,
			thingID: "invalid-id",
			err:     errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveExternalKey(context.Background(), tc.token, tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestBackup(t *testing.T) {
	svc := newService()

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

	var prs []things.Profile
	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		pr := profile
		pr.Name = fmt.Sprintf("name-%d", i)
		prs = append(prs, pr)
	}

	prsc, err := svc.CreateProfiles(context.Background(), token, gr.ID, prs...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prsc[0]

	ths := []things.Thing{}

	for i := 0; i < 10; i++ {
		name := "name_" + fmt.Sprintf("%03d", i+1)
		things, err := svc.CreateThings(context.Background(), token, pr.ID,
			things.Thing{
				Name:     name,
				Metadata: metadata,
			})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th := things[0]

		externalKey := fmt.Sprintf("external_key_%d", i)
		err = svc.UpdateExternalKey(context.Background(), token, externalKey, th.ID)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		ths = append(ths, th)
	}

	backup := things.Backup{
		Groups:   groups,
		Things:   ths,
		Profiles: prsc,
	}

	cases := map[string]struct {
		token  string
		backup things.Backup
		err    error
	}{
		"list backups": {
			token:  adminToken,
			backup: backup,
			err:    nil,
		},
		"list backups with invalid token": {
			token:  wrongValue,
			backup: things.Backup{},
			err:    errors.ErrAuthentication,
		},
		"list backups with empty token": {
			token:  emptyValue,
			backup: things.Backup{},
			err:    errors.ErrAuthentication,
		},
	}

	for desc, tc := range cases {
		backup, err := svc.Backup(context.Background(), tc.token)
		groupSize := len(backup.Groups)
		thingsSize := len(backup.Things)
		profilesSize := len(backup.Profiles)

		assert.Equal(t, len(tc.backup.Groups), groupSize, fmt.Sprintf("%s: expected %v got %d\n", desc, len(tc.backup.Groups), groupSize))
		assert.Equal(t, len(tc.backup.Things), thingsSize, fmt.Sprintf("%s: expected %v got %d\n", desc, len(tc.backup.Things), thingsSize))
		assert.Equal(t, len(tc.backup.Profiles), profilesSize, fmt.Sprintf("%s: expected %v got %d\n", desc, len(tc.backup.Profiles), profilesSize))

		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRestore(t *testing.T) {
	svc := newService()
	idProvider := uuid.New()

	thkey, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	thID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var groups []things.Group
	for i := uint64(0); i < 10; i++ {
		num := strconv.FormatUint(i, 10)
		gr := things.Group{
			ID:          fmt.Sprintf("%s%012d", prefixID, i+1),
			Name:        "test-group-" + num,
			Description: "test group desc",
		}

		groups = append(groups, gr)
	}

	ths := []things.Thing{
		{
			ID:          thID,
			Name:        "testThing",
			Key:         thkey,
			ExternalKey: "abc123",
			Metadata:    map[string]any{},
		},
	}

	var prs []things.Profile
	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		pr := things.Profile{
			ID:       prID,
			Name:     "testProfile",
			Metadata: map[string]any{},
		}
		pr.Name = fmt.Sprintf("name-%d", i)
		prs = append(prs, pr)
	}

	backup := things.Backup{
		Groups:   groups,
		Things:   ths,
		Profiles: prs,
	}

	cases := map[string]struct {
		token  string
		backup things.Backup
		err    error
	}{
		"Restore backup": {
			token:  adminToken,
			backup: backup,
			err:    nil,
		},
		"Restore backup with invalid token": {
			token: wrongValue,
			err:   errors.ErrAuthentication,
		},
		"Restore backup with empty token": {
			token: emptyValue,
			err:   errors.ErrAuthentication,
		},
	}

	for desc, tc := range cases {
		err := svc.Restore(context.Background(), tc.token, tc.backup)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateThingsMetadata(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	thingWithMetadata := things.Thing{
		Name:     "test",
		Metadata: map[string]any{"initial": "data"},
	}
	ths, err := svc.CreateThings(context.Background(), token, prID, thingWithMetadata, thingWithMetadata)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th1, th2 := ths[0], ths[1]

	newMetadata := map[string]any{"updated": "metadata"}
	th1.Metadata = newMetadata
	th2.Metadata = newMetadata

	cases := []struct {
		desc   string
		things []things.Thing
		token  string
		err    error
	}{
		{
			desc:   "update metadata of existing things",
			things: []things.Thing{th1, th2},
			token:  token,
			err:    nil,
		},
		{
			desc:   "update metadata with wrong credentials",
			things: []things.Thing{th1},
			token:  wrongValue,
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "update metadata of non-existing thing",
			things: []things.Thing{{ID: wrongValue, Metadata: newMetadata}},
			token:  token,
			err:    dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateThingsMetadata(context.Background(), tc.token, tc.things...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewGroupInternal(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	cases := []struct {
		desc    string
		groupID string
		err     error
	}{
		{
			desc:    "view group internal with valid id",
			groupID: gr.ID,
			err:     nil,
		},
		{
			desc:    "view group internal with non-existing id",
			groupID: wrongValue,
			err:     dbutil.ErrNotFound,
		},
		{
			desc:    "view group internal with empty id",
			groupID: emptyValue,
			err:     dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := svc.ViewGroupInternal(context.Background(), tc.groupID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestListThingsByGroup(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	var thingCount uint64 = 5
	var ths []things.Thing
	for i := uint64(0); i < thingCount; i++ {
		suffix := i + 1
		th := thing
		th.Name = fmt.Sprintf("thing-%012d", suffix)
		th.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		ths = append(ths, th)
	}
	_, err = svc.CreateThings(context.Background(), token, prID, ths...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc    string
		token   string
		groupID string
		meta    apiutil.PageMetadata
		size    uint64
		err     error
	}{
		{
			desc:    "list things by group",
			token:   token,
			groupID: grID,
			meta:    apiutil.PageMetadata{Offset: 0, Limit: thingCount},
			size:    thingCount,
			err:     nil,
		},
		{
			desc:    "list things by group with wrong credentials",
			token:   wrongValue,
			groupID: grID,
			meta:    apiutil.PageMetadata{},
			size:    0,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "list things by non-existing group",
			token:   token,
			groupID: wrongValue,
			meta:    apiutil.PageMetadata{},
			size:    0,
			err:     dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListThingsByGroup(context.Background(), tc.token, tc.groupID, tc.meta)
		size := uint64(len(page.Things))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestListProfilesByGroup(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	var profileCount uint64 = 5
	var prs []things.Profile
	for i := uint64(0); i < profileCount; i++ {
		suffix := i + 1
		pr := profile
		pr.Name = fmt.Sprintf("profile-%012d", suffix)
		pr.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		prs = append(prs, pr)
	}
	_, err = svc.CreateProfiles(context.Background(), token, grID, prs...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc    string
		token   string
		groupID string
		meta    apiutil.PageMetadata
		size    uint64
		err     error
	}{
		{
			desc:    "list profiles by group",
			token:   token,
			groupID: grID,
			meta:    apiutil.PageMetadata{Offset: 0, Limit: profileCount},
			size:    profileCount,
			err:     nil,
		},
		{
			desc:    "list profiles by group with wrong credentials",
			token:   wrongValue,
			groupID: grID,
			meta:    apiutil.PageMetadata{},
			size:    0,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "list profiles by non-existing group",
			token:   token,
			groupID: wrongValue,
			meta:    apiutil.PageMetadata{},
			size:    0,
			err:     dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListProfilesByGroup(context.Background(), tc.token, tc.groupID, tc.meta)
		size := uint64(len(page.Profiles))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestCreateGroupMembershipsInternal(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	gms := []things.GroupMembership{
		{MemberID: user.ID, GroupID: gr.ID, Email: userEmail, Role: things.Viewer},
		{MemberID: otherUser.ID, GroupID: gr.ID, Email: otherUserEmail, Role: things.Editor},
	}

	cases := []struct {
		desc        string
		memberships []things.GroupMembership
		err         error
	}{
		{
			desc:        "create group memberships internal",
			memberships: gms,
			err:         nil,
		},
		{
			desc:        "create group memberships internal with non-existing group",
			memberships: []things.GroupMembership{{MemberID: user.ID, GroupID: wrongValue, Email: userEmail, Role: things.Viewer}},
			err:         nil,
		},
	}

	for _, tc := range cases {
		err := svc.CreateGroupMembershipsInternal(context.Background(), tc.memberships...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestCanUserAccessThing(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	ths, err := svc.CreateThings(context.Background(), token, prID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	for i := range memberships {
		memberships[i].GroupID = grID
	}
	err = svc.CreateGroupMemberships(context.Background(), token, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc string
		req  things.UserAccessReq
		err  error
	}{
		{
			desc: "check user access to thing as owner",
			req:  things.UserAccessReq{Token: token, ID: th.ID, Action: things.Owner},
			err:  nil,
		},
		{
			desc: "check user access to thing as admin",
			req:  things.UserAccessReq{Token: otherToken, ID: th.ID, Action: things.Admin},
			err:  nil,
		},
		{
			desc: "check user access to thing as editor",
			req:  things.UserAccessReq{Token: editorToken, ID: th.ID, Action: things.Editor},
			err:  nil,
		},
		{
			desc: "check user access to thing as viewer",
			req:  things.UserAccessReq{Token: viewerToken, ID: th.ID, Action: things.Viewer},
			err:  nil,
		},
		{
			desc: "check user access to thing with wrong credentials",
			req:  things.UserAccessReq{Token: wrongValue, ID: th.ID, Action: things.Viewer},
			err:  errors.ErrAuthentication,
		},
		{
			desc: "check user access to non-existing thing",
			req:  things.UserAccessReq{Token: token, ID: wrongValue, Action: things.Viewer},
			err:  dbutil.ErrNotFound,
		},
		{
			desc: "check unauthorized user access to thing",
			req:  things.UserAccessReq{Token: unauthToken, ID: th.ID, Action: things.Viewer},
			err:  dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.CanUserAccessThing(context.Background(), tc.req)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestCanUserAccessProfile(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	for i := range memberships {
		memberships[i].GroupID = grID
	}
	err = svc.CreateGroupMemberships(context.Background(), token, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc string
		req  things.UserAccessReq
		err  error
	}{
		{
			desc: "check user access to profile as owner",
			req:  things.UserAccessReq{Token: token, ID: pr.ID, Action: things.Owner},
			err:  nil,
		},
		{
			desc: "check user access to profile as admin",
			req:  things.UserAccessReq{Token: otherToken, ID: pr.ID, Action: things.Admin},
			err:  nil,
		},
		{
			desc: "check user access to profile as editor",
			req:  things.UserAccessReq{Token: editorToken, ID: pr.ID, Action: things.Editor},
			err:  nil,
		},
		{
			desc: "check user access to profile as viewer",
			req:  things.UserAccessReq{Token: viewerToken, ID: pr.ID, Action: things.Viewer},
			err:  nil,
		},
		{
			desc: "check user access to profile with wrong credentials",
			req:  things.UserAccessReq{Token: wrongValue, ID: pr.ID, Action: things.Viewer},
			err:  errors.ErrAuthentication,
		},
		{
			desc: "check user access to non-existing profile",
			req:  things.UserAccessReq{Token: token, ID: wrongValue, Action: things.Viewer},
			err:  dbutil.ErrNotFound,
		},
		{
			desc: "check unauthorized user access to profile",
			req:  things.UserAccessReq{Token: unauthToken, ID: pr.ID, Action: things.Viewer},
			err:  dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.CanUserAccessProfile(context.Background(), tc.req)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestCanUserAccessGroup(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	for i := range memberships {
		memberships[i].GroupID = gr.ID
	}
	err = svc.CreateGroupMemberships(context.Background(), token, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc string
		req  things.UserAccessReq
		err  error
	}{
		{
			desc: "check user access to group as owner",
			req:  things.UserAccessReq{Token: token, ID: gr.ID, Action: things.Owner},
			err:  nil,
		},
		{
			desc: "check user access to group as admin",
			req:  things.UserAccessReq{Token: otherToken, ID: gr.ID, Action: things.Admin},
			err:  nil,
		},
		{
			desc: "check user access to group as editor",
			req:  things.UserAccessReq{Token: editorToken, ID: gr.ID, Action: things.Editor},
			err:  nil,
		},
		{
			desc: "check user access to group as viewer",
			req:  things.UserAccessReq{Token: viewerToken, ID: gr.ID, Action: things.Viewer},
			err:  nil,
		},
		{
			desc: "check user access to group with wrong credentials",
			req:  things.UserAccessReq{Token: wrongValue, ID: gr.ID, Action: things.Viewer},
			err:  errors.ErrAuthentication,
		},
		{
			desc: "check user access to non-existing group",
			req:  things.UserAccessReq{Token: token, ID: wrongValue, Action: things.Viewer},
			err:  dbutil.ErrNotFound,
		},
		{
			desc: "check unauthorized user access to group",
			req:  things.UserAccessReq{Token: unauthToken, ID: gr.ID, Action: things.Viewer},
			err:  dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.CanUserAccessGroup(context.Background(), tc.req)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestCanThingAccessGroup(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	ths, err := svc.CreateThings(context.Background(), token, prID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	cases := []struct {
		desc string
		req  things.ThingAccessReq
		err  error
	}{
		{
			desc: "check thing access to group",
			req:  things.ThingAccessReq{ThingKey: things.ThingKey{Value: th.Key, Type: things.KeyTypeInternal}, ID: grID},
			err:  nil,
		},
		{
			desc: "check thing access to group with wrong key",
			req:  things.ThingAccessReq{ThingKey: things.ThingKey{Value: wrongValue, Type: things.KeyTypeInternal}, ID: grID},
			err:  dbutil.ErrNotFound,
		},
		{
			desc: "check thing access to non-existing group",
			req:  things.ThingAccessReq{ThingKey: things.ThingKey{Value: th.Key, Type: things.KeyTypeInternal}, ID: wrongValue},
			err:  errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		err := svc.CanThingAccessGroup(context.Background(), tc.req)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestGetConfigByThing(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profileWithConfig := things.Profile{
		Name:   "test",
		Config: map[string]any{"key": "value"},
	}
	prs, err := svc.CreateProfiles(context.Background(), token, grID, profileWithConfig)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	ths, err := svc.CreateThings(context.Background(), token, pr.ID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	cases := []struct {
		desc    string
		thingID string
		err     error
	}{
		{
			desc:    "get config by thing id",
			thingID: th.ID,
			err:     nil,
		},
		{
			desc:    "get config by non-existing thing id",
			thingID: wrongValue,
			err:     dbutil.ErrNotFound,
		},
		{
			desc:    "get config by empty thing id",
			thingID: emptyValue,
			err:     dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		config, err := svc.GetConfigByThing(context.Background(), tc.thingID)
		if err == nil {
			assert.NotNil(t, config, fmt.Sprintf("%s: expected config to be non-nil\n", tc.desc))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestGetGroupIDByThing(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	ths, err := svc.CreateThings(context.Background(), token, prID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	cases := []struct {
		desc    string
		thingID string
		groupID string
		err     error
	}{
		{
			desc:    "get group id by thing id",
			thingID: th.ID,
			groupID: grID,
			err:     nil,
		},
		{
			desc:    "get group id by non-existing thing id",
			thingID: wrongValue,
			groupID: emptyValue,
			err:     dbutil.ErrNotFound,
		},
		{
			desc:    "get group id by empty thing id",
			thingID: emptyValue,
			groupID: emptyValue,
			err:     dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		groupID, err := svc.GetGroupIDByThing(context.Background(), tc.thingID)
		assert.Equal(t, tc.groupID, groupID, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.groupID, groupID))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestGetGroupIDByProfile(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	cases := []struct {
		desc      string
		profileID string
		groupID   string
		err       error
	}{
		{
			desc:      "get group id by profile id",
			profileID: pr.ID,
			groupID:   grID,
			err:       nil,
		},
		{
			desc:      "get group id by non-existing profile id",
			profileID: wrongValue,
			groupID:   emptyValue,
			err:       dbutil.ErrNotFound,
		},
		{
			desc:      "get group id by empty profile id",
			profileID: emptyValue,
			groupID:   emptyValue,
			err:       dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		groupID, err := svc.GetGroupIDByProfile(context.Background(), tc.profileID)
		assert.Equal(t, tc.groupID, groupID, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.groupID, groupID))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestGetGroupIDsByOrg(t *testing.T) {
	svc := newService()

	var groupCount = 5
	for i := 0; i < groupCount; i++ {
		_, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	cases := []struct {
		desc  string
		orgID string
		token string
		size  int
		err   error
	}{
		{
			desc:  "get group ids by org",
			orgID: orgID,
			token: token,
			size:  groupCount,
			err:   nil,
		},
		{
			desc:  "get group ids by org with wrong credentials",
			orgID: orgID,
			token: wrongValue,
			size:  0,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "get group ids by non-existing org",
			orgID: wrongValue,
			token: token,
			size:  0,
			err:   nil,
		},
	}

	for _, tc := range cases {
		groupIDs, err := svc.GetGroupIDsByOrg(context.Background(), tc.orgID, tc.token)
		assert.Equal(t, tc.size, len(groupIDs), fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.size, len(groupIDs)))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestGetThingIDsByProfile(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, orgID, createdGroup)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	var thingCount = 5
	for i := 0; i < thingCount; i++ {
		_, err := svc.CreateThings(context.Background(), token, prID, thing)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	cases := []struct {
		desc      string
		profileID string
		size      int
		err       error
	}{
		{
			desc:      "get thing ids by profile",
			profileID: prID,
			size:      thingCount,
			err:       nil,
		},
		{
			desc:      "get thing ids by non-existing profile",
			profileID: wrongValue,
			size:      0,
			err:       dbutil.ErrNotFound,
		},
		{
			desc:      "get thing ids by empty profile",
			profileID: emptyValue,
			size:      0,
			err:       dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		thingIDs, err := svc.GetThingIDsByProfile(context.Background(), tc.profileID)
		assert.Equal(t, tc.size, len(thingIDs), fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.size, len(thingIDs)))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func testSortEntities[T any](t *testing.T, pm apiutil.PageMetadata, entities []T, getName func(T) string) {
	if len(entities) == 0 {
		return
	}

	currentName := getName(entities[0])
	switch pm.Dir {
	case "asc":
		for _, entity := range entities {
			name := getName(entity)
			assert.GreaterOrEqual(t, name, currentName)
			currentName = name
		}
	case "desc":
		for _, entity := range entities {
			name := getName(entity)
			assert.GreaterOrEqual(t, currentName, name)
			currentName = name
		}
	}
}
