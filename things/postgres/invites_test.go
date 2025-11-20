// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	invitesCommon "github.com/MainfluxLabs/mainflux/pkg/invites"
	"github.com/MainfluxLabs/mainflux/things"
	thingspostgres "github.com/MainfluxLabs/mainflux/things/postgres"
	"github.com/stretchr/testify/assert"
)

const inviteExpiryTime = 24 * 7 * time.Hour

func TestSaveGroupInvite(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repoInvites := thingspostgres.NewGroupInviteRepository(dbMiddleware)
	repoGroups := thingspostgres.NewGroupRepository(dbMiddleware)

	// generate org id in-place (assume org exists)
	orgID := generateUUID(t)

	groupID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	group := things.Group{
		ID:    groupID,
		OrgID: orgID,
		Name:  "group",
	}

	_, err = repoGroups.Save(context.Background(), group)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var invites []things.GroupInvite

	m := 5
	for i := 0; i < m; i++ {
		invID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		inviteeID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		inviterID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		invites = append(invites, things.GroupInvite{
			InviteCommon: invitesCommon.InviteCommon{
				ID:          invID,
				InviteeID:   sql.NullString{Valid: true, String: inviteeID},
				InviterID:   inviterID,
				InviteeRole: "viewer",
				CreatedAt:   time.Now(),
				ExpiresAt:   time.Now().Add(inviteExpiryTime),
				State:       invitesCommon.InviteStatePending,
			},
			GroupID: group.ID,
		})
	}

	invID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	inviteeID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	inviterID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	expiredInvite := things.GroupInvite{
		InviteCommon: invitesCommon.InviteCommon{
			ID:          invID,
			InviteeID:   sql.NullString{Valid: true, String: inviteeID},
			InviterID:   inviterID,
			InviteeRole: "editor",
			CreatedAt:   time.Now().Add(-2 * inviteExpiryTime),
			ExpiresAt:   time.Now().Add(-1 * inviteExpiryTime),
			State:       invitesCommon.InviteStatePending,
		},
		GroupID: group.ID,
	}
	invites = append(invites, expiredInvite)

	alreadyInvitedInvite := invites[0]
	invID, err = idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	alreadyInvitedInvite.ID = invID

	var invalidGroupIDInvites []things.GroupInvite
	for _, inv := range invites {
		inv.GroupID = "invalid"
		invalidGroupIDInvites = append(invalidGroupIDInvites, inv)
	}

	var emptyGroupIDInvites []things.GroupInvite
	for _, inv := range invites {
		inv.GroupID = ""
		emptyGroupIDInvites = append(emptyGroupIDInvites, inv)
	}

	invID, err = idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	reinviteExpiredInvite := things.GroupInvite{
		InviteCommon: invitesCommon.InviteCommon{
			ID:          invID,
			InviteeID:   expiredInvite.InviteeID,
			InviterID:   expiredInvite.InviterID,
			InviteeRole: expiredInvite.InviteeRole,
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(inviteExpiryTime),
			State:       invitesCommon.InviteStatePending,
		},
		GroupID: expiredInvite.GroupID,
	}

	cases := []struct {
		desc    string
		invites []things.GroupInvite
		err     error
	}{
		{
			desc:    "save invites",
			invites: invites,
			err:     nil,
		},
		{
			desc:    "save invite that already exists",
			invites: invites,
			err:     dbutil.ErrConflict,
		},
		{
			desc:    "save invite to same invitee by same inviter to same group",
			invites: []things.GroupInvite{alreadyInvitedInvite},
			err:     dbutil.ErrConflict,
		},
		{
			desc:    "save invites with invalid group id",
			invites: invalidGroupIDInvites,
			err:     dbutil.ErrMalformedEntity,
		},
		{
			desc:    "save invites with empty group id",
			invites: emptyGroupIDInvites,
			err:     dbutil.ErrMalformedEntity,
		},
		{
			desc:    "save invite with same properties as existing expired invite",
			invites: []things.GroupInvite{reinviteExpiredInvite},
			err:     nil,
		},
	}

	for _, tc := range cases {
		err := repoInvites.SaveInvites(context.Background(), tc.invites...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s, got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveGroupInviteByID(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repoInvites := thingspostgres.NewGroupInviteRepository(dbMiddleware)
	repoGroups := thingspostgres.NewGroupRepository(dbMiddleware)

	// generate org id in-place (assume org exists)
	orgID := generateUUID(t)

	groupID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	group := things.Group{
		ID:    groupID,
		OrgID: orgID,
		Name:  "group",
	}

	_, err = repoGroups.Save(context.Background(), group)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var invites []things.GroupInvite

	m := 5
	for i := 0; i < m; i++ {
		invID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		inviteeID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		inviterID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		invites = append(invites, things.GroupInvite{
			InviteCommon: invitesCommon.InviteCommon{
				ID:          invID,
				InviteeID:   sql.NullString{Valid: true, String: inviteeID},
				InviterID:   inviterID,
				InviteeRole: "viewer",
				CreatedAt:   time.Now(),
				ExpiresAt:   time.Now(),
				State:       invitesCommon.InviteStatePending,
			},
			GroupID: group.ID,
		})
	}

	err = repoInvites.SaveInvites(context.Background(), invites...)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	nonExistentID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc     string
		inviteID string
		err      error
	}{
		{
			desc:     "retrieve invite",
			inviteID: invites[1].ID,
			err:      nil,
		},
		{
			desc:     "retrieve invite with empty ID",
			inviteID: "",
			err:      dbutil.ErrMalformedEntity,
		},
		{
			desc:     "retrieve invite with invalid ID",
			inviteID: "invalid",
			err:      dbutil.ErrMalformedEntity,
		},
		{
			desc:     "retrieve invite with non-existent ID",
			inviteID: nonExistentID,
			err:      dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := repoInvites.RetrieveInviteByID(context.Background(), tc.inviteID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s, got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveGroupInvite(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repoInvites := thingspostgres.NewGroupInviteRepository(dbMiddleware)
	repoGroups := thingspostgres.NewGroupRepository(dbMiddleware)

	// generate org id in-place (assume org exists)
	orgID := generateUUID(t)

	groupID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	group := things.Group{
		ID:    groupID,
		OrgID: orgID,
		Name:  "group",
	}

	_, err = repoGroups.Save(context.Background(), group)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var invites []things.GroupInvite

	m := 5
	for i := 0; i < m; i++ {
		invID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		inviteeID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		inviterID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		invites = append(invites, things.GroupInvite{
			InviteCommon: invitesCommon.InviteCommon{
				ID:          invID,
				InviteeID:   sql.NullString{Valid: true, String: inviteeID},
				InviterID:   inviterID,
				InviteeRole: "viewer",
				CreatedAt:   time.Now(),
				ExpiresAt:   time.Now(),
				State:       invitesCommon.InviteStatePending,
			},
			GroupID: group.ID,
		})
	}

	err = repoInvites.SaveInvites(context.Background(), invites...)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	nonExistentID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc     string
		inviteID string
		err      error
	}{
		{
			desc:     "remove invite",
			inviteID: invites[1].ID,
			err:      nil,
		},
		{
			desc:     "remove invite with empty ID",
			inviteID: "",
			err:      dbutil.ErrMalformedEntity,
		},
		{
			desc:     "remove invite with invalid ID",
			inviteID: "invalid",
			err:      dbutil.ErrMalformedEntity,
		},
		{
			desc:     "remove invite with non-existent ID",
			inviteID: nonExistentID,
			err:      dbutil.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		err := repoInvites.RemoveInvite(context.Background(), tc.inviteID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s, got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveGroupInvitesByUserID(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repoInvites := thingspostgres.NewGroupInviteRepository(dbMiddleware)
	repoGroups := thingspostgres.NewGroupRepository(dbMiddleware)

	m := 5

	var invites []things.GroupInvite

	inviteeID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	inviterID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	for i := 0; i < m; i++ {
		// generate org id in-place (assume org exists)
		orgID := generateUUID(t)

		groupID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		group := things.Group{
			ID:    groupID,
			OrgID: orgID,
			Name:  fmt.Sprintf("group%d", i),
		}

		_, err = repoGroups.Save(context.Background(), group)
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		invID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		invites = append(invites, things.GroupInvite{
			InviteCommon: invitesCommon.InviteCommon{
				ID:          invID,
				InviteeID:   sql.NullString{Valid: true, String: inviteeID},
				InviterID:   inviterID,
				InviteeRole: "viewer",
				CreatedAt:   time.Now(),
				ExpiresAt:   time.Now().Add(inviteExpiryTime),
				State:       invitesCommon.InviteStatePending,
			},
			GroupID: group.ID,
		})
	}

	err = repoInvites.SaveInvites(context.Background(), invites...)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc     string
		pm       invitesCommon.PageMetadataInvites
		userType string
		userID   string
		size     int
		err      error
	}{
		{
			desc:     "retrieve all pending invites towards invitee",
			pm:       invitesCommon.PageMetadataInvites{PageMetadata: apiutil.PageMetadata{}},
			userID:   inviteeID,
			userType: invitesCommon.UserTypeInvitee,
			size:     m,
			err:      nil,
		},
		{
			desc:     "retrieve 1 pending invite towards invitee",
			pm:       invitesCommon.PageMetadataInvites{PageMetadata: apiutil.PageMetadata{Limit: 1}},
			userID:   inviteeID,
			userType: invitesCommon.UserTypeInvitee,
			size:     1,
			err:      nil,
		},
		{
			desc:     "retrieve pending invites with empty user id",
			pm:       invitesCommon.PageMetadataInvites{PageMetadata: apiutil.PageMetadata{Limit: 1}},
			userID:   "",
			userType: invitesCommon.UserTypeInvitee,
			size:     0,
			err:      dbutil.ErrRetrieveEntity,
		},
		{
			desc:     "retrieve all sent invites by inviter",
			pm:       invitesCommon.PageMetadataInvites{PageMetadata: apiutil.PageMetadata{}},
			userID:   inviterID,
			userType: invitesCommon.UserTypeInviter,
			size:     m,
			err:      nil,
		},
		{
			desc:     "retrieve 1 sent invite by inviter",
			pm:       invitesCommon.PageMetadataInvites{PageMetadata: apiutil.PageMetadata{Limit: 1}},
			userID:   inviterID,
			userType: invitesCommon.UserTypeInviter,
			size:     1,
			err:      nil,
		},
	}

	for _, tc := range cases {
		invPage, err := repoInvites.RetrieveInvitesByUser(context.Background(), tc.userType, tc.userID, tc.pm)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s, got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.size, len(invPage.Invites), fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, len(invPage.Invites)))
	}
}
