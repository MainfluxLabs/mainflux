// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/auth/postgres"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const inviteExpiryTime = 24 * 7 * time.Hour

func TestSaveInvite(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repoInvites := postgres.NewInvitesRepo(dbMiddleware)
	repoOrgs := postgres.NewOrgRepo(dbMiddleware)

	orgID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	ownerID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	org := auth.Org{
		ID:          orgID,
		OwnerID:     ownerID,
		Name:        "org",
		Description: "org",
	}

	err = repoOrgs.Save(context.Background(), org)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var invites []auth.Invite

	m := 5
	for i := 0; i < m; i++ {
		invID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		inviteeID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		inviterID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		invites = append(invites, auth.Invite{
			ID:           invID,
			InviteeID:    inviteeID,
			InviteeEmail: fmt.Sprintf("invitee%d@test.com", i),
			InviterID:    inviterID,
			OrgID:        org.ID,
			InviteeRole:  auth.Viewer,
			CreatedAt:    time.Now(),
			ExpiresAt:    time.Now().Add(inviteExpiryTime),
		})
	}

	alreadyInvitedInvites := []auth.Invite{}
	alreadyInvitedInvites = append(alreadyInvitedInvites, invites[0])
	invID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	alreadyInvitedInvites[0].ID = invID

	var invalidOrgIDInvites []auth.Invite
	for _, inv := range invites {
		inv.OrgID = "invalid"
		invalidOrgIDInvites = append(invalidOrgIDInvites, inv)
	}

	var emptyOrgIDInvites []auth.Invite
	for _, inv := range invites {
		inv.OrgID = ""
		emptyOrgIDInvites = append(emptyOrgIDInvites, inv)
	}

	cases := []struct {
		desc    string
		invites []auth.Invite
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
			err:     errors.ErrConflict,
		},
		{
			desc:    "save invite to user with existing pending invite to same org",
			invites: alreadyInvitedInvites,
			err:     auth.ErrUserAlreadyInvited,
		},
		{
			desc:    "save invites with invalid org id",
			invites: invalidOrgIDInvites,
			err:     errors.ErrMalformedEntity,
		},
		{
			desc:    "save invites with empty org id",
			invites: emptyOrgIDInvites,
			err:     errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		err := repoInvites.Save(context.Background(), tc.invites...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s, got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveInviteByID(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repoInvites := postgres.NewInvitesRepo(dbMiddleware)
	repoOrgs := postgres.NewOrgRepo(dbMiddleware)

	orgID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	ownerID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	org := auth.Org{
		ID:          orgID,
		OwnerID:     ownerID,
		Name:        "org",
		Description: "org",
	}

	err = repoOrgs.Save(context.Background(), org)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var invites []auth.Invite

	m := 5
	for i := 0; i < m; i++ {
		invID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		inviteeID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		inviterID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		invites = append(invites, auth.Invite{
			ID:           invID,
			InviteeID:    inviteeID,
			InviteeEmail: fmt.Sprintf("invitee%d@test.com", i),
			InviterID:    inviterID,
			OrgID:        org.ID,
			InviteeRole:  auth.Viewer,
			CreatedAt:    time.Now(),
			ExpiresAt:    time.Now(),
		})
	}

	err = repoInvites.Save(context.Background(), invites...)
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
			err:      errors.ErrMalformedEntity,
		},
		{
			desc:     "retrieve invite with invalid ID",
			inviteID: "invalid",
			err:      errors.ErrMalformedEntity,
		},
		{
			desc:     "retrieve invite with non-existent ID",
			inviteID: nonExistentID,
			err:      errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := repoInvites.RetrieveByID(context.Background(), tc.inviteID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s, got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveInvite(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repoInvites := postgres.NewInvitesRepo(dbMiddleware)
	repoOrgs := postgres.NewOrgRepo(dbMiddleware)

	orgID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	ownerID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	org := auth.Org{
		ID:          orgID,
		OwnerID:     ownerID,
		Name:        "org",
		Description: "org",
	}

	err = repoOrgs.Save(context.Background(), org)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var invites []auth.Invite

	m := 5
	for i := 0; i < m; i++ {
		invID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		inviteeID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		inviterID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		invites = append(invites, auth.Invite{
			ID:           invID,
			InviteeID:    inviteeID,
			InviteeEmail: fmt.Sprintf("invitee%d@test.com", i),
			InviterID:    inviterID,
			OrgID:        org.ID,
			InviteeRole:  auth.Viewer,
			CreatedAt:    time.Now(),
			ExpiresAt:    time.Now(),
		})
	}

	err = repoInvites.Save(context.Background(), invites...)
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
			err:      errors.ErrMalformedEntity,
		},
		{
			desc:     "remove invite with invalid ID",
			inviteID: "invalid",
			err:      errors.ErrMalformedEntity,
		},
		{
			desc:     "remove invite with non-existent ID",
			inviteID: nonExistentID,
			err:      errors.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		err := repoInvites.Remove(context.Background(), tc.inviteID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s, got %s\n", tc.desc, tc.err, err))
	}
}

func TestListInivtesByInvitee(t *testing.T) {
	dbMiddleware := dbutil.NewDatabase(db)
	repoInvites := postgres.NewInvitesRepo(dbMiddleware)
	repoOrgs := postgres.NewOrgRepo(dbMiddleware)

	m := 5

	var invites []auth.Invite

	inviteeID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	for i := range m {
		orgID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		ownerID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		org := auth.Org{
			ID:          orgID,
			OwnerID:     ownerID,
			Name:        fmt.Sprintf("org%d", i),
			Description: fmt.Sprintf("org%d", i),
		}

		err = repoOrgs.Save(context.Background(), org)
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		invID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		inviterID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		invites = append(invites, auth.Invite{
			ID:           invID,
			InviteeID:    inviteeID,
			InviteeEmail: fmt.Sprintf("invitee%d@test.com", i),
			InviterID:    inviterID,
			OrgID:        org.ID,
			InviteeRole:  auth.Viewer,
			CreatedAt:    time.Now(),
			ExpiresAt:    time.Now().Add(inviteExpiryTime),
		})
	}

	err = repoInvites.Save(context.Background(), invites...)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc      string
		pm        apiutil.PageMetadata
		inviteeID string
		size      int
		err       error
	}{
		{
			desc:      "retrieve all pending invites towards invitee",
			pm:        apiutil.PageMetadata{},
			inviteeID: inviteeID,
			size:      m,
			err:       nil,
		},
		{
			desc:      "retrieve 1 pending invite towards invitee",
			pm:        apiutil.PageMetadata{Limit: 1},
			inviteeID: inviteeID,
			size:      1,
			err:       nil,
		},
		{
			desc:      "retrieve pending invites with empty invitee id",
			pm:        apiutil.PageMetadata{Limit: 1},
			inviteeID: "",
			size:      0,
			err:       errors.ErrRetrieveEntity,
		},
	}

	for _, tc := range cases {
		invPage, err := repoInvites.RetrieveByUserID(context.Background(), auth.UserTypeInvitee, tc.inviteeID, tc.pm)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s, got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.size, len(invPage.Invites), fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, len(invPage.Invites)))
	}
}
