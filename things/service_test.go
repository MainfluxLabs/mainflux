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
	emptyValue     = ""
	wrongValue     = "wrong-value"
	adminEmail     = "admin@example.com"
	userEmail      = "user@example.com"
	otherUserEmail = "other.user@example.com"
	viewerEmail    = "viewer@gmail.com"
	editorEmail    = "editor@gmail.com"
	adminToken     = adminEmail
	viewerToken    = viewerEmail
	editorToken    = editorEmail
	token          = userEmail
	otherToken     = otherUserEmail
	password       = "password"
	n              = uint64(102)
	n2             = uint64(204)
	orgID          = "374106f7-030e-4881-8ab0-151195c29f92"
	prefixID       = "fe6b4e92-cc98-425e-b0aa-"
	prefixName     = "test-"
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
	admin         = users.User{ID: "874106f7-030e-4881-8ab0-151195c29f97", Email: adminEmail, Password: password, Role: auth.RootSub}
	viewer        = users.User{ID: "874106f7-030e-4881-8ab0-151195c29f99", Email: viewerEmail, Password: password, Role: auth.Viewer}
	editor        = users.User{ID: "874106f7-030e-4881-8ab0-151195c29f91", Email: editorEmail, Password: password, Role: auth.Editor}
	usersList     = []users.User{admin, user, otherUser, viewer, editor}
	usersByEmails = map[string]users.User{userEmail: {ID: user.ID, Email: userEmail}, otherUserEmail: {ID: otherUser.ID, Email: otherToken}, viewerEmail: {ID: viewer.ID, Email: viewer.Email},
		editorEmail: {ID: editor.ID, Email: editor.Email}}
	usersByIDs = map[string]users.User{user.ID: {ID: user.ID, Email: userEmail}, otherUser.ID: {ID: otherUser.ID, Email: otherUserEmail}, viewer.ID: {ID: viewer.ID, Email: viewerEmail},
		editor.ID: {ID: editor.ID, Email: editorEmail}}
	members = []things.GroupMember{
		{MemberID: otherUser.ID, Email: otherUser.Email, Role: things.Admin},
		{MemberID: viewer.ID, Email: viewer.Email, Role: things.Viewer},
		{MemberID: editor.ID, Email: editor.Email, Role: things.Editor},
	}
	group    = things.Group{OrgID: orgID, Name: "test-group", Description: "test-group-desc"}
	orgsList = []auth.Org{{ID: orgID, OwnerID: user.ID}}
	metadata = map[string]interface{}{"test": "data"}
)

func newService() things.Service {
	auth := authmock.NewAuthService(admin.ID, usersList, orgsList)
	uc := mocks.NewUsersService(usersByIDs, usersByEmails)
	thingsRepo := mocks.NewThingRepository()
	profilesRepo := mocks.NewProfileRepository(thingsRepo)
	groupMembersRepo := mocks.NewGroupMembersRepository()
	groupsRepo := mocks.NewGroupRepository(groupMembersRepo)
	profileCache := mocks.NewProfileCache()
	thingCache := mocks.NewThingCache()
	groupCache := mocks.NewGroupCache()
	idProvider := uuid.NewMock()

	return things.New(auth, uc, thingsRepo, profilesRepo, groupsRepo, groupMembersRepo, profileCache, thingCache, groupCache, idProvider)
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
	grs, err := svc.CreateGroups(context.Background(), token, group, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID, grID1 := grs[0].ID, grs[1].ID

	profile.GroupID = grID
	profile1 := profile
	profile1.GroupID = grID1
	prs, err := svc.CreateProfiles(context.Background(), token, profile, profile1)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID, prID1 := prs[0].ID, prs[1].ID

	thsExtID[0].GroupID = grID
	thsExtID[0].ProfileID = prID
	thsExtID[1].GroupID = grID
	thsExtID[1].ProfileID = prID

	cases := []struct {
		desc   string
		things []things.Thing
		token  string
		err    error
	}{
		{
			desc:   "create new things",
			things: []things.Thing{{Name: "a", GroupID: grID, ProfileID: prID}, {Name: "b", GroupID: grID, ProfileID: prID}, {Name: "c", GroupID: grID, ProfileID: prID}, {Name: "d", GroupID: grID, ProfileID: prID}},
			token:  token,
			err:    nil,
		},
		{
			desc:   "create new thing with wrong group id",
			things: []things.Thing{{Name: "e", GroupID: wrongValue, ProfileID: prID}},
			token:  token,
			err:    errors.ErrNotFound,
		},
		{
			desc:   "create thing with wrong credentials",
			things: []things.Thing{{Name: "f", GroupID: grID, ProfileID: prID}},
			token:  wrongValue,
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "create new things with external UUID",
			things: thsExtID,
			token:  token,
			err:    nil,
		},
		{
			desc:   "create new things with external wrong UUID",
			things: []things.Thing{{ID: "b0aa-000000000001", Name: "a", GroupID: grID, ProfileID: prID}, {ID: "b0aa-000000000002", Name: "b", GroupID: grID, ProfileID: prID}},
			token:  token,
			err:    nil,
		},
		{
			desc:   "create thing with profile from different group",
			things: []things.Thing{{Name: "test", GroupID: grID, ProfileID: prID1}},
			token:  token,
			err:    errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		_, err := svc.CreateThings(context.Background(), tc.token, tc.things...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateThing(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID, grID1 := grs[0].ID, grs[1].ID

	profile.GroupID = grID
	profile1 := profile
	profile1.GroupID = grID1
	prs, err := svc.CreateProfiles(context.Background(), token, profile, profile1)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID, prID1 := prs[0].ID, prs[1].ID

	thing.GroupID = grID
	thing.ProfileID = prID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]
	other := things.Thing{ID: emptyValue, Key: "x", ProfileID: prID}
	invalidPrGr := th
	invalidPrGr.ProfileID = prID1

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
			err:   errors.ErrNotFound,
		},
		{
			desc:  "update thing with profile from different group",
			thing: invalidPrGr,
			token: token,
			err:   errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateThing(context.Background(), tc.token, tc.thing)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateKey(t *testing.T) {
	key := "new-key"
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	thing.GroupID = grID
	thing.ProfileID = prID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	cases := []struct {
		desc  string
		token string
		id    string
		key   string
		err   error
	}{
		{
			desc:  "update key of an existing thing",
			token: token,
			id:    th.ID,
			key:   key,
			err:   nil,
		},
		{
			desc:  "update key with invalid credentials",
			token: wrongValue,
			id:    th.ID,
			key:   key,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "update key of non-existing thing",
			token: token,
			id:    emptyValue,
			key:   wrongValue,
			err:   errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateKey(context.Background(), tc.token, tc.id, tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewThing(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	thing.GroupID = grID
	thing.ProfileID = prID
	ths, err := svc.CreateThings(context.Background(), token, thing)
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
			err:   errors.ErrNotFound,
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

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	thing := things.Thing{
		GroupID:   grID,
		ProfileID: prID,
		Name:      "test-meta",
		Key:       key,
		Metadata:  metadata,
	}
	_, err = svc.CreateThings(context.Background(), token, thing)
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
			err: errors.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := svc.ViewMetadataByKey(context.Background(), tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListThings(t *testing.T) {
	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	grs2, err := svc.CreateGroups(context.Background(), otherToken, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID2 := grs2[0].ID

	profile.GroupID = grID
	profile2 := profile
	profile2.GroupID = grID2
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prs2, err := svc.CreateProfiles(context.Background(), otherToken, profile2)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID
	prID2 := prs2[0].ID
	thingList[0].Metadata = metadata

	var ths1 []things.Thing
	var suffix uint64
	for i := uint64(0); i < n; i++ {
		suffix = i + 1
		th := thingList[i]
		th.GroupID = grID
		th.ProfileID = prID
		th.Name = fmt.Sprintf("%s%d", prefixName, suffix)
		th.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		th.Key = fmt.Sprintf("%s%d", prefixID, suffix)

		ths1 = append(ths1, th)
	}

	var ths2 []things.Thing
	for i := uint64(0); i < n; i++ {
		suffix = n + i + 1
		th := thingList[i]
		th.GroupID = grID2
		th.ProfileID = prID2
		th.Name = fmt.Sprintf("%s%d", prefixName, suffix)
		th.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		th.Key = fmt.Sprintf("%s%d", prefixID, suffix)

		ths2 = append(ths2, th)
	}

	_, err = svc.CreateThings(context.Background(), token, ths1...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	_, err = svc.CreateThings(context.Background(), otherToken, ths2...)
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

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profile.GroupID = gr.ID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	var ths []things.Thing
	for i := uint64(0); i < n; i++ {
		suffix := i + 1
		th := thingList[i]
		th.GroupID = gr.ID
		th.ProfileID = pr.ID
		th.Name = fmt.Sprintf("%s%012d", prefixName, suffix)
		th.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		ths = append(ths, th)
	}

	_, err = svc.CreateThings(context.Background(), token, ths...)
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
			err:  errors.ErrNotFound,
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

	grs, err := svc.CreateGroups(context.Background(), adminToken, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grs2, err := svc.CreateGroups(context.Background(), otherToken, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr, gr2 := grs[0], grs2[0]

	profile.GroupID = gr.ID
	prs, err := svc.CreateProfiles(context.Background(), adminToken, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	profile.GroupID = gr2.ID
	prs2, err := svc.CreateProfiles(context.Background(), otherToken, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr, pr2 := prs[0], prs2[0]

	var ths []things.Thing
	for i := uint64(0); i < n; i++ {
		suffix := i + 1
		th := thing
		th.GroupID = gr.ID
		th.ProfileID = pr.ID
		th.Name = fmt.Sprintf("%s%012d", prefixName, suffix)
		th.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		ths = append(ths, th)
	}

	var ths2 []things.Thing
	for i := uint64(0); i < n; i++ {
		suffix := n + i + 1
		th2 := thing
		th2.GroupID = gr2.ID
		th2.ProfileID = pr2.ID
		th2.Name = fmt.Sprintf("%s%012d", prefixName, suffix)
		th2.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		ths2 = append(ths2, th2)
	}

	_, err = svc.CreateThings(context.Background(), adminToken, ths...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	_, err = svc.CreateThings(context.Background(), otherToken, ths2...)
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
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	thing.GroupID = grID
	thing.ProfileID = prID
	ths, err := svc.CreateThings(context.Background(), token, thing)
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
			err:   errors.ErrNotFound,
		},
		{
			desc:  "remove non-existing thing",
			id:    emptyValue,
			token: token,
			err:   errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveThings(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestCreateProfiles(t *testing.T) {
	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID
	prsExtID[0].GroupID = grID
	prsExtID[1].GroupID = grID
	cases := []struct {
		desc     string
		profiles []things.Profile
		token    string
		err      error
	}{
		{
			desc:     "create new profiles",
			profiles: []things.Profile{{Name: "a", GroupID: grID}, {Name: "b", GroupID: grID}, {Name: "c", GroupID: grID}, {Name: "d", GroupID: grID}},
			token:    token,
			err:      nil,
		},
		{
			desc:     "create new profile with wrong group id",
			profiles: []things.Profile{{Name: "e", GroupID: wrongValue}},
			token:    token,
			err:      errors.ErrNotFound,
		},
		{
			desc:     "create profile with wrong credentials",
			profiles: []things.Profile{{Name: "f", GroupID: grID}},
			token:    wrongValue,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "create new profiles with external UUID",
			profiles: prsExtID,
			token:    token,
			err:      nil,
		},
		{
			desc:     "create new profiles with invalid external UUID",
			profiles: []things.Profile{{ID: "b0aa-000000000001", Name: "a", GroupID: grID}, {ID: "b0aa-000000000002", Name: "b", GroupID: grID}},
			token:    token,
			err:      nil,
		},
	}

	for _, cc := range cases {
		_, err := svc.CreateProfiles(context.Background(), cc.token, cc.profiles...)
		assert.True(t, errors.Contains(err, cc.err), fmt.Sprintf("%s: expected %s got %s\n", cc.desc, cc.err, err))
	}
}

func TestUpdateProfile(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profile.GroupID = gr.ID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
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
			err:     errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateProfile(context.Background(), tc.token, tc.profile)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewProfile(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profile.GroupID = gr.ID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	pr := prs[0]

	cases := map[string]struct {
		id       string
		token    string
		err      error
		metadata map[string]interface{}
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
			err:   errors.ErrNotFound,
		},
		"view profile with metadata": {
			id:    emptyValue,
			token: token,
			err:   errors.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := svc.ViewProfile(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListProfiles(t *testing.T) {
	svc := newService()
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	grs2, err := svc.CreateGroups(context.Background(), otherToken, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr2 := grs2[0]
	profileList[0].Metadata = metadata

	var prs1 []things.Profile
	var suffix uint64
	for i := uint64(0); i < n; i++ {
		suffix = i + 1
		pr := profileList[i]
		pr.GroupID = gr.ID
		pr.Name = fmt.Sprintf("%s%d", prefixName, suffix)
		pr.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		prs1 = append(prs1, pr)
	}

	var prs2 []things.Profile
	for i := uint64(0); i < n; i++ {
		suffix = n + i + 1
		pr := profileList[i]
		pr.GroupID = gr2.ID
		pr.Name = fmt.Sprintf("%s%d", prefixName, suffix)
		pr.ID = fmt.Sprintf("%s%012d", prefixID, suffix)

		prs2 = append(prs2, pr)
	}

	_, err = svc.CreateProfiles(context.Background(), token, prs1...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	_, err = svc.CreateProfiles(context.Background(), otherToken, prs2...)
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

	grs, err := svc.CreateGroups(context.Background(), adminToken, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grs2, err := svc.CreateGroups(context.Background(), otherToken, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	gr, gr2 := grs[0], grs2[0]
	var prs1 []things.Profile
	for i := uint64(0); i < n; i++ {
		suffix := i + 1
		pr := profile
		pr.GroupID = gr.ID
		pr.Name = fmt.Sprintf("%s%d", prefixName, suffix)
		pr.ID = fmt.Sprintf("%s%012d", prefixID, suffix)
		prs1 = append(prs1, pr)
	}

	var prs2 []things.Profile
	for i := uint64(0); i < n; i++ {
		suffix := n + i + 1
		pr2 := profile
		pr2.GroupID = gr2.ID
		pr2.Name = fmt.Sprintf("%s%d", prefixName, suffix)
		pr2.ID = fmt.Sprintf("%s%012d", prefixID, suffix)

		prs2 = append(prs2, pr2)
	}

	_, err = svc.CreateProfiles(context.Background(), adminToken, prs1...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	_, err = svc.CreateProfiles(context.Background(), otherToken, prs2...)
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

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	p := profile
	p.Name = "test-profile"
	p.GroupID = gr.ID

	prs, err := svc.CreateProfiles(context.Background(), token, p)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	thing.GroupID = gr.ID
	thing.ProfileID = pr.ID
	ths, err := svc.CreateThings(context.Background(), token, thing)
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
			err:     errors.ErrNotFound,
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
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	profile.GroupID = grID
	prs, err := svc.CreateProfiles(context.Background(), token, profile, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	prID, prID1 := prs[0].ID, prs[1].ID

	thing.GroupID = grID
	thing.ProfileID = prID1
	_, err = svc.CreateThings(context.Background(), token, thing)
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
			err:   errors.ErrNotFound,
		},
		{
			desc:  "remove non-existing profile",
			id:    emptyValue,
			token: token,
			err:   errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveProfiles(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestGetPubConfByKey(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profile.GroupID = gr.ID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	thing.GroupID = gr.ID
	thing.ProfileID = pr.ID
	ths, err := svc.CreateThings(context.Background(), token, thing)
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
			err: errors.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := svc.GetPubConfByKey(context.Background(), tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected '%s' got '%s'\n", desc, tc.err, err))
	}
}

func TestIdentify(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	profile.GroupID = gr.ID
	prs, err := svc.CreateProfiles(context.Background(), token, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prs[0]

	thing.GroupID = gr.ID
	thing.ProfileID = pr.ID
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	cases := map[string]struct {
		token string
		id    string
		err   error
	}{
		"identify existing thing": {
			token: th.Key,
			id:    th.ID,
			err:   nil,
		},
		"identify non-existing thing": {
			token: wrongValue,
			id:    emptyValue,
			err:   errors.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		id, err := svc.Identify(context.Background(), tc.token)
		assert.Equal(t, tc.id, id, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.id, id))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestCreateGroups(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc  string
		token string
		group things.Group
		err   error
	}{
		{
			desc:  "create group",
			token: token,
			group: group,
			err:   nil,
		},
		{
			desc:  "create group with wrong credentials",
			token: wrongValue,
			group: group,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "create group without credentials",
			token: emptyValue,
			group: group,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "create group without org",
			token: token,
			group: things.Group{Name: "test-group", Description: "test"},
			err:   errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := svc.CreateGroups(context.Background(), tc.token, tc.group)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}

}

func TestListGroups(t *testing.T) {
	svc := newService()
	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		group.Name = fmt.Sprintf("group-%d", i)
		group.Description = fmt.Sprintf("desc-%d", i)
		_, err := svc.CreateGroups(context.Background(), token, group)
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
		group.Name = fmt.Sprintf("group-%d", i)
		group.Description = fmt.Sprintf("desc-%d", i)
		_, err := svc.CreateGroups(context.Background(), token, group)
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

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	grID := grs[0].ID

	for i := range members {
		members[i].GroupID = grID
	}
	err = svc.CreateGroupMembers(context.Background(), token, members...)
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
			err:   errors.ErrNotFound,
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
			err:   errors.ErrNotFound,
		},
		{
			desc:  "remove non-existing group",
			token: token,
			id:    wrongValue,
			err:   errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveGroups(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateGroup(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	grID := grs[0].ID

	for i := range members {
		members[i].GroupID = grID
	}
	err = svc.CreateGroupMembers(context.Background(), token, members...)
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
			err:   errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := svc.UpdateGroup(context.Background(), tc.token, tc.group)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewGroup(t *testing.T) {
	svc := newService()

	group.Metadata = map[string]interface{}{"test": "meta"}
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	gr := grs[0]

	for i := range members {
		members[i].GroupID = gr.ID
	}
	err = svc.CreateGroupMembers(context.Background(), token, members...)
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
			err:   errors.ErrNotFound,
		},
		{
			desc:  "view non-existing group",
			token: token,
			grID:  wrongValue,
			res:   things.Group{},
			err:   errors.ErrNotFound,
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

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	gr := grs[0]

	profile.GroupID = gr.ID
	prs, err := svc.CreateProfiles(context.Background(), adminToken, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	prID := prs[0].ID

	thing.ProfileID = prID
	thing.GroupID = gr.ID
	ths, err := svc.CreateThings(context.Background(), token, thing)
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
			err:   errors.ErrNotFound,
		},
		{
			desc:  "view group by non-existing thing",
			token: token,
			thID:  wrongValue,
			res:   things.Group{},
			err:   errors.ErrNotFound,
		},
		{
			desc:  "view group by thing without user rights",
			token: otherToken,
			thID:  th.ID,
			res:   things.Group{},
			err:   errors.ErrNotFound,
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

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	gr := grs[0]

	profile.GroupID = gr.ID
	prs, err := svc.CreateProfiles(context.Background(), adminToken, profile)
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
			err:   errors.ErrNotFound,
		},
		{
			desc:  "view group by profile with non-existing profile",
			token: token,
			prID:  wrongValue,
			res:   things.Group{},
			err:   errors.ErrNotFound,
		},
		{
			desc:  "view group by profile without user rights",
			token: otherToken,
			prID:  prID,
			res:   things.Group{},
			err:   errors.ErrNotFound,
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

func TestCreateGroupMembers(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	for i := range members {
		members[i].GroupID = gr.ID
	}
	mbs := []things.GroupMember{members[1], members[2]}
	mb := things.GroupMember{MemberID: "1", GroupID: gr.ID, Email: "member@gmail.com", Role: things.Viewer}

	cases := []struct {
		desc   string
		token  string
		member []things.GroupMember
		err    error
	}{
		{
			desc:   "create group members as owner",
			token:  token,
			member: []things.GroupMember{{MemberID: otherUser.ID, GroupID: gr.ID, Email: otherUserEmail, Role: things.Admin}},
			err:    nil,
		},
		{
			desc:   "create group members as admin",
			token:  otherToken,
			member: mbs,
			err:    nil,
		},
		{
			desc:   "create group members as editor",
			token:  editorToken,
			member: []things.GroupMember{mb},
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "create group members as viewer",
			token:  viewerToken,
			member: []things.GroupMember{mb},
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "create group members with wrong credentials",
			token:  wrongValue,
			member: []things.GroupMember{mb},
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "create group members without credentials",
			token:  emptyValue,
			member: []things.GroupMember{mb},
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "create group members without group id",
			token:  token,
			member: []things.GroupMember{{MemberID: "2", Email: "member2@gmail.com", Role: things.Viewer}},
			err:    errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.CreateGroupMembers(context.Background(), tc.token, tc.member...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestListGroupMembers(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	for i := range members {
		members[i].GroupID = gr.ID
	}
	err = svc.CreateGroupMembers(context.Background(), token, members...)
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
			desc:    "list group members as owner",
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
			desc:    "list group members as admin",
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
			desc:    "list group members as editor",
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
			desc:    "list group members as viewer",
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
			desc:    "list group members as system admin",
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
			desc:    "list half group members",
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
			desc:    "list last group member",
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
			desc:    "list group members with wrong credentials",
			token:   wrongValue,
			groupID: gr.ID,
			meta:    apiutil.PageMetadata{},
			size:    0,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "list group members without credentials",
			token:   emptyValue,
			groupID: gr.ID,
			meta:    apiutil.PageMetadata{},
			size:    0,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "list members from non-existing group",
			token:   token,
			groupID: wrongValue,
			meta:    apiutil.PageMetadata{},
			size:    0,
			err:     errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListGroupMembers(context.Background(), tc.token, tc.groupID, tc.meta)
		size := uint64(len(page.GroupMembers))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s expected %d got %d\n", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateMembers(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	for i := range members {
		members[i].GroupID = gr.ID
	}
	err = svc.CreateGroupMembers(context.Background(), token, members...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	grOwner := things.GroupMember{GroupID: gr.ID, MemberID: user.ID, Email: user.Email, Role: things.Owner}

	cases := []struct {
		desc   string
		token  string
		member things.GroupMember
		err    error
	}{
		{
			desc:   "update group member role as viewer",
			token:  viewerToken,
			member: members[1],
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "update group member role as editor",
			token:  editorToken,
			member: members[2],
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "update group member role as admin",
			token:  otherToken,
			member: members[2],
			err:    nil,
		},
		{
			desc:   "update group member role as owner",
			token:  token,
			member: members[1],
			err:    nil,
		},
		{
			desc:   "update group owner role as owner",
			token:  token,
			member: grOwner,
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "update group owner role as admin",
			token:  otherToken,
			member: grOwner,
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "update group member role with wrong credentials",
			token:  wrongValue,
			member: members[1],
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "update group member role without credentials",
			token:  emptyValue,
			member: members[1],
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "update group member role with non-existing group",
			token:  token,
			member: things.GroupMember{MemberID: editor.ID, GroupID: wrongValue, Email: editor.Email, Role: things.Editor},
			err:    errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateGroupMembers(context.Background(), tc.token, tc.member)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveGroupMembers(t *testing.T) {
	svc := newService()

	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	for i := range members {
		members[i].GroupID = gr.ID
	}
	err = svc.CreateGroupMembers(context.Background(), token, members...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc     string
		token    string
		groupID  string
		memberID string
		err      error
	}{
		{
			desc:     "remove member from group as viewer",
			token:    viewerToken,
			groupID:  gr.ID,
			memberID: editor.ID,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "remove member from group as editor",
			token:    editorToken,
			groupID:  gr.ID,
			memberID: viewer.ID,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "remove member from group as admin",
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
			desc:     "remove member from group as owner",
			token:    token,
			groupID:  gr.ID,
			memberID: editor.ID,
			err:      nil,
		},
		{
			desc:     "remove member with wrong credentials",
			token:    wrongValue,
			groupID:  gr.ID,
			memberID: editor.ID,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "remove member without credentials",
			token:    emptyValue,
			groupID:  gr.ID,
			memberID: editor.ID,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "remove member from non-existing group",
			token:    token,
			groupID:  wrongValue,
			memberID: editor.ID,
			err:      errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveGroupMembers(context.Background(), tc.token, tc.groupID, tc.memberID)
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

		grs, err := svc.CreateGroups(context.Background(), token, group)
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
		pr.GroupID = gr.ID
		prs = append(prs, pr)
	}

	prsc, err := svc.CreateProfiles(context.Background(), token, prs...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	pr := prsc[0]

	ths := []things.Thing{}
	for i := 0; i < 10; i++ {
		name := "name_" + fmt.Sprintf("%03d", i+1)
		things, err := svc.CreateThings(context.Background(), token,
			things.Thing{
				Name:      name,
				GroupID:   gr.ID,
				ProfileID: pr.ID,
				Metadata:  metadata,
			})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th := things[0]

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
			ID:       thID,
			Name:     "testThing",
			Key:      thkey,
			Metadata: map[string]interface{}{},
		},
	}

	var prs []things.Profile
	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		pr := things.Profile{
			ID:       prID,
			Name:     "testProfile",
			Metadata: map[string]interface{}{},
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
