// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package downlinks_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/downlinks"
	dlmocks "github.com/MainfluxLabs/mainflux/downlinks/mocks"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/cron"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token      = "admin@example.com"
	userToken  = "user@example.com"
	wrongToken = "wrong-token"
	thingID    = "5384fb1c-d0ae-4cbe-be52-c54223150fe0"
	groupID    = "574106f7-030e-4881-8ab0-151195c29f94"
	wrongID    = "wrong-id"
)

var (
	adminUser   = users.User{ID: "874106f7-030e-4881-8ab0-151195c29f97", Email: token, Role: auth.RootSub}
	regularUser = users.User{ID: "974106f7-030e-4881-8ab0-151195c29f98", Email: userToken}
	usersList   = []users.User{adminUser, regularUser}

	downlink = downlinks.Downlink{
		Name:   "test-downlink",
		Url:    "https://example.com/data",
		Method: "GET",
		Scheduler: cron.Scheduler{
			Frequency: cron.MinutelyFreq,
			Minute:    5,
			TimeZone:  "UTC",
		},
	}
)

func newService() downlinks.Service {
	authSvc := pkgmocks.NewAuthService(adminUser.ID, usersList, nil)
	thingsSvc := pkgmocks.NewThingsServiceClient(
		nil,
		map[string]things.Thing{
			token:   {ID: thingID, GroupID: groupID},
			thingID: {ID: thingID, GroupID: groupID},
		},
		map[string]things.Group{token: {ID: groupID}},
	)
	repo := dlmocks.NewDownlinkRepository()
	pub := pkgmocks.NewPublisher()
	idp := uuid.NewMock()
	log := logger.NewMock()

	return downlinks.New(thingsSvc, authSvc, pub, repo, idp, log)
}

func TestCreateDownlinks(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc      string
		token     string
		thingID   string
		downlinks []downlinks.Downlink
		err       error
	}{
		{
			desc:      "create downlinks with valid token",
			token:     token,
			thingID:   thingID,
			downlinks: []downlinks.Downlink{downlink},
			err:       nil,
		},
		{
			desc:      "create downlinks with invalid token",
			token:     wrongToken,
			thingID:   thingID,
			downlinks: []downlinks.Downlink{downlink},
			err:       errors.ErrAuthorization,
		},
		{
			desc:      "create downlinks for wrong thing ID",
			token:     token,
			thingID:   wrongID,
			downlinks: []downlinks.Downlink{downlink},
			err:       errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		dls, err := svc.CreateDownlinks(context.Background(), tc.token, tc.thingID, tc.downlinks...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Equal(t, len(tc.downlinks), len(dls), fmt.Sprintf("%s: expected %d downlinks got %d", tc.desc, len(tc.downlinks), len(dls)))
		}
	}
}

func TestListDownlinksByThing(t *testing.T) {
	svc := newService()

	dls, err := svc.CreateDownlinks(context.Background(), token, thingID, downlink)
	require.Nil(t, err, fmt.Sprintf("unexpected error creating downlinks: %s", err))
	require.Equal(t, 1, len(dls))

	cases := []struct {
		desc    string
		token   string
		thingID string
		pm      apiutil.PageMetadata
		size    uint64
		err     error
	}{
		{
			desc:    "list downlinks by thing with valid token",
			token:   token,
			thingID: thingID,
			pm:      apiutil.PageMetadata{},
			size:    1,
			err:     nil,
		},
		{
			desc:    "list downlinks by thing with invalid token",
			token:   wrongToken,
			thingID: thingID,
			pm:      apiutil.PageMetadata{},
			size:    0,
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "list downlinks by thing for wrong thing ID",
			token:   token,
			thingID: wrongID,
			pm:      apiutil.PageMetadata{},
			size:    0,
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "list downlinks by thing with limit",
			token:   token,
			thingID: thingID,
			pm:      apiutil.PageMetadata{Limit: 1, Offset: 0},
			size:    1,
			err:     nil,
		},
		{
			desc:    "list downlinks by thing with offset beyond available",
			token:   token,
			thingID: thingID,
			pm:      apiutil.PageMetadata{Limit: 1, Offset: 1},
			size:    0,
			err:     nil,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListDownlinksByThing(context.Background(), tc.token, tc.thingID, tc.pm)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Equal(t, tc.size, uint64(len(page.Downlinks)), fmt.Sprintf("%s: expected %d downlinks got %d", tc.desc, tc.size, len(page.Downlinks)))
		}
	}
}

func TestListDownlinksByGroup(t *testing.T) {
	svc := newService()

	_, err := svc.CreateDownlinks(context.Background(), token, thingID, downlink)
	require.Nil(t, err, fmt.Sprintf("unexpected error creating downlinks: %s", err))

	cases := []struct {
		desc    string
		token   string
		groupID string
		pm      apiutil.PageMetadata
		size    uint64
		err     error
	}{
		{
			desc:    "list downlinks by group with valid token",
			token:   token,
			groupID: groupID,
			pm:      apiutil.PageMetadata{},
			size:    1,
			err:     nil,
		},
		{
			desc:    "list downlinks by group with invalid token",
			token:   wrongToken,
			groupID: groupID,
			pm:      apiutil.PageMetadata{},
			size:    0,
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "list downlinks by group for wrong group ID",
			token:   token,
			groupID: wrongID,
			pm:      apiutil.PageMetadata{},
			size:    0,
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "list downlinks by group with limit",
			token:   token,
			groupID: groupID,
			pm:      apiutil.PageMetadata{Limit: 1, Offset: 0},
			size:    1,
			err:     nil,
		},
		{
			desc:    "list downlinks by group with offset beyond available",
			token:   token,
			groupID: groupID,
			pm:      apiutil.PageMetadata{Limit: 1, Offset: 1},
			size:    0,
			err:     nil,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListDownlinksByGroup(context.Background(), tc.token, tc.groupID, tc.pm)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Equal(t, tc.size, uint64(len(page.Downlinks)), fmt.Sprintf("%s: expected %d downlinks got %d", tc.desc, tc.size, len(page.Downlinks)))
		}
	}
}

func TestViewDownlink(t *testing.T) {
	svc := newService()

	dls, err := svc.CreateDownlinks(context.Background(), token, thingID, downlink)
	require.Nil(t, err, fmt.Sprintf("unexpected error creating downlinks: %s", err))
	require.Equal(t, 1, len(dls))
	dlID := dls[0].ID

	cases := []struct {
		desc  string
		token string
		id    string
		err   error
	}{
		{
			desc:  "view downlink with valid token",
			token: token,
			id:    dlID,
			err:   nil,
		},
		{
			desc:  "view downlink with invalid ID",
			token: token,
			id:    wrongID,
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "view downlink with invalid token",
			token: wrongToken,
			id:    dlID,
			err:   errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		dl, err := svc.ViewDownlink(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Equal(t, tc.id, dl.ID, fmt.Sprintf("%s: expected ID %s got %s", tc.desc, tc.id, dl.ID))
		}
	}
}

func TestUpdateDownlink(t *testing.T) {
	svc := newService()

	dls, err := svc.CreateDownlinks(context.Background(), token, thingID, downlink)
	require.Nil(t, err, fmt.Sprintf("unexpected error creating downlinks: %s", err))
	require.Equal(t, 1, len(dls))
	dlID := dls[0].ID

	updated := downlinks.Downlink{
		ID:     dlID,
		Name:   "updated-downlink",
		Url:    "https://example.com/updated",
		Method: "POST",
		Scheduler: cron.Scheduler{
			Frequency: cron.HourlyFreq,
			Hour:      1,
			TimeZone:  "UTC",
		},
	}

	cases := []struct {
		desc     string
		token    string
		downlink downlinks.Downlink
		err      error
	}{
		{
			desc:     "update downlink with valid token",
			token:    token,
			downlink: updated,
			err:      nil,
		},
		{
			desc:     "update downlink with invalid ID",
			token:    token,
			downlink: downlinks.Downlink{ID: wrongID, Name: "x", Url: "https://x.com", Method: "GET"},
			err:      dbutil.ErrNotFound,
		},
		{
			desc:     "update downlink with invalid token",
			token:    wrongToken,
			downlink: updated,
			err:      errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateDownlink(context.Background(), tc.token, tc.downlink)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestRemoveDownlinks(t *testing.T) {
	svc := newService()

	dls, err := svc.CreateDownlinks(context.Background(), token, thingID, downlink)
	require.Nil(t, err, fmt.Sprintf("unexpected error creating downlinks: %s", err))
	require.Equal(t, 1, len(dls))
	dlID := dls[0].ID

	cases := []struct {
		desc  string
		token string
		ids   []string
		err   error
	}{
		{
			desc:  "remove downlinks with invalid ID",
			token: token,
			ids:   []string{wrongID},
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "remove downlinks with invalid token",
			token: wrongToken,
			ids:   []string{dlID},
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "remove downlinks with valid token",
			token: token,
			ids:   []string{dlID},
			err:   nil,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveDownlinks(context.Background(), tc.token, tc.ids...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestRemoveDownlinksByThing(t *testing.T) {
	svc := newService()

	_, err := svc.CreateDownlinks(context.Background(), token, thingID, downlink)
	require.Nil(t, err, fmt.Sprintf("unexpected error creating downlinks: %s", err))

	cases := []struct {
		desc    string
		thingID string
		err     error
	}{
		{
			desc:    "remove downlinks by thing with valid thing ID",
			thingID: thingID,
			err:     nil,
		},
		{
			desc:    "remove downlinks by thing with unknown thing ID",
			thingID: wrongID,
			err:     nil,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveDownlinksByThing(context.Background(), tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestRemoveDownlinksByGroup(t *testing.T) {
	svc := newService()

	_, err := svc.CreateDownlinks(context.Background(), token, thingID, downlink)
	require.Nil(t, err, fmt.Sprintf("unexpected error creating downlinks: %s", err))

	cases := []struct {
		desc    string
		groupID string
		err     error
	}{
		{
			desc:    "remove downlinks by group with valid group ID",
			groupID: groupID,
			err:     nil,
		},
		{
			desc:    "remove downlinks by group with unknown group ID",
			groupID: wrongID,
			err:     nil,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveDownlinksByGroup(context.Background(), tc.groupID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestBackup(t *testing.T) {
	svc := newService()

	_, err := svc.CreateDownlinks(context.Background(), token, thingID, downlink)
	require.Nil(t, err, fmt.Sprintf("unexpected error creating downlinks: %s", err))

	cases := []struct {
		desc  string
		token string
		size  int
		err   error
	}{
		{
			desc:  "backup with admin token",
			token: token,
			size:  1,
			err:   nil,
		},
		{
			desc:  "backup with non-admin token",
			token: userToken,
			size:  0,
			err:   errors.ErrAuthorization,
		},
		{
			desc:  "backup with invalid token",
			token: wrongToken,
			size:  0,
			err:   errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		dls, err := svc.Backup(context.Background(), tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Equal(t, tc.size, len(dls), fmt.Sprintf("%s: expected %d downlinks got %d", tc.desc, tc.size, len(dls)))
		}
	}
}

func TestRestore(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc      string
		token     string
		downlinks []downlinks.Downlink
		err       error
	}{
		{
			desc:      "restore with admin token",
			token:     token,
			downlinks: []downlinks.Downlink{downlink},
			err:       nil,
		},
		{
			desc:      "restore with non-admin token",
			token:     userToken,
			downlinks: []downlinks.Downlink{downlink},
			err:       errors.ErrAuthorization,
		},
		{
			desc:      "restore with invalid token",
			token:     wrongToken,
			downlinks: []downlinks.Downlink{downlink},
			err:       errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		err := svc.Restore(context.Background(), tc.token, tc.downlinks)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}
