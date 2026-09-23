// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package uiconfigs_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/uiconfigs"
	"github.com/MainfluxLabs/mainflux/uiconfigs/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	wrongValue = "wrong-value"

	adminID    = "admin-id"
	adminEmail = "admin@example.com"
	adminToken = adminEmail

	editorID    = "editor-id"
	editorEmail = "editor@example.com"
	editorToken = editorEmail

	viewerID    = "viewer-id"
	viewerEmail = "viewer@example.com"
	viewerToken = viewerEmail

	unauthID    = "unauth-id"
	unauthEmail = "unauth@example.com"
	unauthToken = unauthEmail

	orgID = "9325aef3-5a2b-448c-bae1-5d45f86ba2aa"

	thingID      = "fe6b4e92-cc98-425e-b0aa-000000000001"
	otherThingID = "fe6b4e92-cc98-425e-b0aa-000000000002"

	groupID      = "574106f7-030e-4881-8ab0-151195c29f94"
	otherGroupID = "574106f7-030e-4881-8ab0-151195c29f95"
)

var testConfig = uiconfigs.Config{"key": "value"}

func newService() uiconfigs.Service {
	orgConfigs := mocks.NewOrgConfigRepository()
	thingConfigs := mocks.NewThingConfigRepository()
	groupConfigs := mocks.NewGroupConfigRepository()

	usersList := []domain.User{
		{ID: adminID, Email: adminEmail},
		{ID: editorID, Email: editorEmail, Role: domain.OrgEditor},
		{ID: viewerID, Email: viewerEmail, Role: domain.OrgViewer},
		{ID: unauthID, Email: unauthEmail},
	}
	authClient := pkgmocks.NewAuthService(adminID, usersList, nil)

	things := map[string]domain.Thing{
		editorToken:  {ID: thingID, GroupID: groupID},
		thingID:      {ID: thingID, GroupID: groupID},
		viewerToken:  {ID: otherThingID, GroupID: otherGroupID},
		otherThingID: {ID: otherThingID, GroupID: otherGroupID},
	}
	groupsByToken := map[string]domain.Group{
		editorToken: {ID: groupID, OrgID: orgID},
		viewerToken: {ID: otherGroupID, OrgID: orgID},
	}
	thingsClient := pkgmocks.NewThingsServiceClient(map[string]domain.Profile{}, things, groupsByToken)

	return uiconfigs.New(orgConfigs, thingConfigs, groupConfigs, thingsClient, authClient, uuid.NewMock(), logger.NewMock())
}

// -- Org config --

func TestViewOrgConfig(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc  string
		token string
		orgID string
		err   error
	}{
		{
			desc:  "view org config with valid token",
			token: editorToken,
			orgID: orgID,
			err:   nil,
		},
		{
			desc:  "view org config with invalid token",
			token: wrongValue,
			orgID: orgID,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "view org config as unauthorized user",
			token: unauthToken,
			orgID: orgID,
			err:   errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		_, err := svc.ViewOrgConfig(context.Background(), tc.token, tc.orgID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestUpdateOrgConfig(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc      string
		token     string
		orgConfig uiconfigs.OrgConfig
		err       error
	}{
		{
			desc:      "create org config with valid token",
			token:     editorToken,
			orgConfig: uiconfigs.OrgConfig{OrgID: orgID, Config: testConfig},
			err:       nil,
		},
		{
			desc:      "update existing org config with valid token",
			token:     editorToken,
			orgConfig: uiconfigs.OrgConfig{OrgID: orgID, Config: uiconfigs.Config{"key": "updated"}},
			err:       nil,
		},
		{
			desc:      "update org config with invalid token",
			token:     wrongValue,
			orgConfig: uiconfigs.OrgConfig{OrgID: orgID, Config: testConfig},
			err:       errors.ErrAuthentication,
		},
		{
			desc:      "update org config with viewer-only role",
			token:     viewerToken,
			orgConfig: uiconfigs.OrgConfig{OrgID: orgID, Config: testConfig},
			err:       errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateOrgConfig(context.Background(), tc.token, tc.orgConfig)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}

	updated, err := svc.ViewOrgConfig(context.Background(), editorToken, orgID)
	require.Nil(t, err)
	assert.Equal(t, "updated", updated.Config["key"])
}

func TestListOrgsConfigs(t *testing.T) {
	svc := newService()

	secondOrgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	for _, id := range []string{orgID, secondOrgID} {
		err := svc.UpdateOrgConfig(context.Background(), editorToken, uiconfigs.OrgConfig{OrgID: id, Config: testConfig})
		require.Nil(t, err)
	}

	cases := []struct {
		desc  string
		token string
		total uint64
	}{
		{
			desc:  "admin lists all org configs",
			token: adminToken,
			total: 2,
		},
		{
			desc:  "member with an org role lists all org configs",
			token: editorToken,
			total: 2,
		},
		{
			desc:  "unauthorized user sees no org configs",
			token: unauthToken,
			total: 0,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListOrgsConfigs(context.Background(), tc.token, apiutil.PageMetadata{Limit: 10})
		require.Nil(t, err, tc.desc)
		assert.Equal(t, tc.total, page.Total, tc.desc)
	}

	_, err := svc.ListOrgsConfigs(context.Background(), wrongValue, apiutil.PageMetadata{Limit: 10})
	assert.True(t, errors.Contains(err, errors.ErrAuthentication))
}

func TestRemoveOrgConfig(t *testing.T) {
	svc := newService()

	err := svc.UpdateOrgConfig(context.Background(), editorToken, uiconfigs.OrgConfig{OrgID: orgID, Config: testConfig})
	require.Nil(t, err)

	err = svc.RemoveOrgConfig(context.Background(), orgID)
	assert.Nil(t, err)

	oc, err := svc.ViewOrgConfig(context.Background(), editorToken, orgID)
	require.Nil(t, err)
	assert.Empty(t, oc.Config)
}

// -- Thing config --

func TestViewThingConfig(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc    string
		token   string
		thingID string
		err     error
	}{
		{
			desc:    "view thing config with valid token",
			token:   editorToken,
			thingID: thingID,
			err:     nil,
		},
		{
			desc:    "view thing config with invalid token",
			token:   wrongValue,
			thingID: thingID,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "view thing config without access to the thing",
			token:   viewerToken,
			thingID: thingID,
			err:     errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		_, err := svc.ViewThingConfig(context.Background(), tc.token, tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestUpdateThingConfig(t *testing.T) {
	svc := newService()

	err := svc.UpdateThingConfig(context.Background(), editorToken, uiconfigs.ThingConfig{ThingID: thingID, Config: testConfig})
	require.Nil(t, err)

	updated, err := svc.ViewThingConfig(context.Background(), editorToken, thingID)
	require.Nil(t, err)
	assert.Equal(t, groupID, updated.GroupID, "group ID should be auto-populated via GetGroupIDByThing")

	err = svc.UpdateThingConfig(context.Background(), editorToken, uiconfigs.ThingConfig{ThingID: thingID, Config: uiconfigs.Config{"key": "updated"}})
	require.Nil(t, err)

	updated, err = svc.ViewThingConfig(context.Background(), editorToken, thingID)
	require.Nil(t, err)
	assert.Equal(t, groupID, updated.GroupID)
	assert.Equal(t, "updated", updated.Config["key"])

	cases := []struct {
		desc        string
		token       string
		thingConfig uiconfigs.ThingConfig
		err         error
	}{
		{
			desc:        "update thing config with invalid token",
			token:       wrongValue,
			thingConfig: uiconfigs.ThingConfig{ThingID: thingID, Config: testConfig},
			err:         errors.ErrAuthentication,
		},
		{
			desc:        "update thing config without access to the thing",
			token:       viewerToken,
			thingConfig: uiconfigs.ThingConfig{ThingID: thingID, Config: testConfig},
			err:         errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateThingConfig(context.Background(), tc.token, tc.thingConfig)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestListThingsConfigs(t *testing.T) {
	svc := newService()

	err := svc.UpdateThingConfig(context.Background(), editorToken, uiconfigs.ThingConfig{ThingID: thingID, Config: testConfig})
	require.Nil(t, err)
	err = svc.UpdateThingConfig(context.Background(), viewerToken, uiconfigs.ThingConfig{ThingID: otherThingID, Config: testConfig})
	require.Nil(t, err)

	cases := []struct {
		desc  string
		token string
		total uint64
	}{
		{
			desc:  "admin lists all thing configs",
			token: adminToken,
			total: 2,
		},
		{
			desc:  "user only sees things it can access",
			token: editorToken,
			total: 1,
		},
		{
			desc:  "unauthorized user sees no thing configs",
			token: unauthToken,
			total: 0,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListThingsConfigs(context.Background(), tc.token, apiutil.PageMetadata{Limit: 10})
		require.Nil(t, err, tc.desc)
		assert.Equal(t, tc.total, page.Total, tc.desc)
	}
}

func TestRemoveThingConfig(t *testing.T) {
	svc := newService()

	err := svc.UpdateThingConfig(context.Background(), editorToken, uiconfigs.ThingConfig{ThingID: thingID, Config: testConfig})
	require.Nil(t, err)

	err = svc.RemoveThingConfig(context.Background(), thingID)
	assert.Nil(t, err)

	tc, err := svc.ViewThingConfig(context.Background(), editorToken, thingID)
	require.Nil(t, err)
	assert.Empty(t, tc.Config)
}

func TestRemoveThingConfigByGroup(t *testing.T) {
	svc := newService()

	err := svc.UpdateThingConfig(context.Background(), editorToken, uiconfigs.ThingConfig{ThingID: thingID, Config: testConfig})
	require.Nil(t, err)
	err = svc.UpdateThingConfig(context.Background(), viewerToken, uiconfigs.ThingConfig{ThingID: otherThingID, Config: testConfig})
	require.Nil(t, err)

	err = svc.RemoveThingConfigByGroup(context.Background(), groupID)
	assert.Nil(t, err)

	tc, err := svc.ViewThingConfig(context.Background(), editorToken, thingID)
	require.Nil(t, err)
	assert.Empty(t, tc.Config, "config for a thing in the removed group should be gone")

	otherTc, err := svc.ViewThingConfig(context.Background(), viewerToken, otherThingID)
	require.Nil(t, err)
	assert.Equal(t, testConfig, otherTc.Config, "config for a thing in a different group should remain")
}

// -- Group config --

func TestViewGroupConfig(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc    string
		token   string
		groupID string
		err     error
	}{
		{
			desc:    "view group config with valid token",
			token:   editorToken,
			groupID: groupID,
			err:     nil,
		},
		{
			desc:    "view group config with invalid token",
			token:   wrongValue,
			groupID: groupID,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "view group config without access to the group",
			token:   viewerToken,
			groupID: groupID,
			err:     errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		_, err := svc.ViewGroupConfig(context.Background(), tc.token, tc.groupID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestUpdateGroupConfig(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc        string
		token       string
		groupConfig uiconfigs.GroupConfig
		err         error
	}{
		{
			desc:        "create group config with valid token",
			token:       editorToken,
			groupConfig: uiconfigs.GroupConfig{GroupID: groupID, Config: testConfig},
			err:         nil,
		},
		{
			desc:        "update existing group config with valid token",
			token:       editorToken,
			groupConfig: uiconfigs.GroupConfig{GroupID: groupID, Config: uiconfigs.Config{"key": "updated"}},
			err:         nil,
		},
		{
			desc:        "update group config with invalid token",
			token:       wrongValue,
			groupConfig: uiconfigs.GroupConfig{GroupID: groupID, Config: testConfig},
			err:         errors.ErrAuthentication,
		},
		{
			desc:        "update group config without access to the group",
			token:       viewerToken,
			groupConfig: uiconfigs.GroupConfig{GroupID: groupID, Config: testConfig},
			err:         errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateGroupConfig(context.Background(), tc.token, tc.groupConfig)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}

	updated, err := svc.ViewGroupConfig(context.Background(), editorToken, groupID)
	require.Nil(t, err)
	assert.Equal(t, "updated", updated.Config["key"])
}

func TestListGroupsConfigs(t *testing.T) {
	svc := newService()

	err := svc.UpdateGroupConfig(context.Background(), editorToken, uiconfigs.GroupConfig{GroupID: groupID, Config: testConfig})
	require.Nil(t, err)
	err = svc.UpdateGroupConfig(context.Background(), viewerToken, uiconfigs.GroupConfig{GroupID: otherGroupID, Config: testConfig})
	require.Nil(t, err)

	cases := []struct {
		desc  string
		token string
		pm    apiutil.PageMetadata
		total uint64
		size  int
	}{
		{
			desc:  "admin lists all group configs",
			token: adminToken,
			pm:    apiutil.PageMetadata{Limit: 10},
			total: 2,
			size:  2,
		},
		{
			desc:  "user only sees groups it can access",
			token: editorToken,
			pm:    apiutil.PageMetadata{Limit: 10},
			total: 1,
			size:  1,
		},
		{
			desc:  "unauthorized user sees no group configs",
			token: unauthToken,
			pm:    apiutil.PageMetadata{Limit: 10},
			total: 0,
			size:  0,
		},
		{
			desc:  "admin lists group configs with offset beyond available",
			token: adminToken,
			pm:    apiutil.PageMetadata{Limit: 20, Offset: 10},
			total: 2,
			size:  0,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListGroupsConfigs(context.Background(), tc.token, tc.pm)
		require.Nil(t, err, tc.desc)
		assert.Equal(t, tc.total, page.Total, tc.desc)
		assert.Equal(t, tc.size, len(page.GroupsConfigs), tc.desc)
	}
}

func TestRemoveGroupConfig(t *testing.T) {
	svc := newService()

	err := svc.UpdateGroupConfig(context.Background(), editorToken, uiconfigs.GroupConfig{GroupID: groupID, Config: testConfig})
	require.Nil(t, err)

	err = svc.RemoveGroupConfig(context.Background(), groupID)
	assert.Nil(t, err)

	gc, err := svc.ViewGroupConfig(context.Background(), editorToken, groupID)
	require.Nil(t, err)
	assert.Empty(t, gc.Config)
}

// -- Combined backup / restore --

func TestBackup(t *testing.T) {
	svc := newService()

	err := svc.UpdateOrgConfig(context.Background(), editorToken, uiconfigs.OrgConfig{OrgID: orgID, Config: testConfig})
	require.Nil(t, err)
	err = svc.UpdateThingConfig(context.Background(), editorToken, uiconfigs.ThingConfig{ThingID: thingID, Config: testConfig})
	require.Nil(t, err)
	err = svc.UpdateGroupConfig(context.Background(), editorToken, uiconfigs.GroupConfig{GroupID: groupID, Config: testConfig})
	require.Nil(t, err)

	backup, err := svc.Backup(context.Background(), adminToken)
	require.Nil(t, err)
	assert.Len(t, backup.OrgsConfigs, 1)
	assert.Len(t, backup.ThingsConfigs, 1)
	assert.Len(t, backup.GroupsConfigs, 1)

	backup, err = svc.Backup(context.Background(), unauthToken)
	require.Nil(t, err)
	assert.Empty(t, backup.OrgsConfigs)
	assert.Empty(t, backup.ThingsConfigs)
	assert.Empty(t, backup.GroupsConfigs)
}

func TestRestore(t *testing.T) {
	svc := newService()

	backup := uiconfigs.Backup{
		OrgsConfigs:   []uiconfigs.OrgConfig{{OrgID: orgID, Config: testConfig}},
		ThingsConfigs: []uiconfigs.ThingConfig{{ThingID: thingID, GroupID: groupID, Config: testConfig}},
		GroupsConfigs: []uiconfigs.GroupConfig{{GroupID: groupID, Config: testConfig}},
	}

	cases := []struct {
		desc  string
		token string
		err   error
	}{
		{
			desc:  "restore with non-admin token",
			token: editorToken,
			err:   errors.ErrAuthorization,
		},
		{
			desc:  "restore with invalid token",
			token: wrongValue,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "restore with admin token",
			token: adminToken,
			err:   nil,
		},
	}

	for _, tc := range cases {
		err := svc.Restore(context.Background(), tc.token, backup)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}

	oc, err := svc.ViewOrgConfig(context.Background(), editorToken, orgID)
	require.Nil(t, err)
	assert.Equal(t, testConfig, oc.Config)

	tc, err := svc.ViewThingConfig(context.Background(), editorToken, thingID)
	require.Nil(t, err)
	assert.Equal(t, testConfig, tc.Config)

	gc, err := svc.ViewGroupConfig(context.Background(), editorToken, groupID)
	require.Nil(t, err)
	assert.Equal(t, testConfig, gc.Config)

	err = svc.Restore(context.Background(), adminToken, backup)
	assert.True(t, errors.Contains(err, dbutil.ErrConflict), fmt.Sprintf("expected %s got %s", dbutil.ErrConflict, err))
}
