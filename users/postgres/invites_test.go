// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/MainfluxLabs/mainflux/users/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const inviteExpiryTime = 24 * time.Hour

func newPlatformInvite(t *testing.T, email string, expiresAt time.Time) users.PlatformInvite {
	id, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	return users.PlatformInvite{
		ID:           id,
		InviteeEmail: email,
		CreatedAt:    time.Now(),
		ExpiresAt:    expiresAt,
		State:        users.InviteStatePending,
	}
}

func TestSavePlatformInvite(t *testing.T) {
	repo := postgres.NewPlatformInvitesRepo(dbutil.NewDatabase(db))

	expiresAt := time.Now().Add(inviteExpiryTime)
	invites := []users.PlatformInvite{
		newPlatformInvite(t, "invitee1@example.com", expiresAt),
		newPlatformInvite(t, "invitee2@example.com", expiresAt),
	}

	expiredInvite := newPlatformInvite(t, "expired@example.com", time.Now().Add(-inviteExpiryTime))
	err := repo.SavePlatformInvite(context.Background(), expiredInvite)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	newInvite := newPlatformInvite(t, "invitee3@example.com", expiresAt)
	invalidInvite := newPlatformInvite(t, "invalid@example.com", expiresAt)
	invalidInvite.ID = "invalid"

	cases := []struct {
		desc       string
		invites    []users.PlatformInvite
		unsavedIDs []string
		err        error
	}{
		{
			desc:    "save platform invites",
			invites: invites,
			err:     nil,
		},
		{
			desc:    "save platform invites to already invited emails",
			invites: []users.PlatformInvite{newPlatformInvite(t, invites[0].InviteeEmail, expiresAt)},
			err:     apiutil.ErrUserAlreadyInvited,
		},
		{
			desc:       "save platform invites with an invalid id",
			invites:    []users.PlatformInvite{newInvite, invalidInvite},
			unsavedIDs: []string{newInvite.ID},
			err:        dbutil.ErrMalformedEntity,
		},
		{
			desc:    "save platform invite to email with an expired invite",
			invites: []users.PlatformInvite{newPlatformInvite(t, expiredInvite.InviteeEmail, expiresAt)},
			err:     nil,
		},
	}

	for _, tc := range cases {
		err := repo.SavePlatformInvite(context.Background(), tc.invites...)
		assert.True(t, errors.Contains(err, tc.err),
			fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		for _, id := range tc.unsavedIDs {
			_, err := repo.RetrievePlatformInviteByID(context.Background(), id)
			assert.True(t, errors.Contains(err, dbutil.ErrNotFound),
				fmt.Sprintf("%s: unexpected error: %s\n", tc.desc, err))
		}
	}

	for _, inv := range invites {
		saved, err := repo.RetrievePlatformInviteByID(context.Background(), inv.ID)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		assert.Equal(t, inv.InviteeEmail, saved.InviteeEmail,
			fmt.Sprintf("expected email %s got %s", inv.InviteeEmail, saved.InviteeEmail))
	}

	expired, err := repo.RetrievePlatformInviteByID(context.Background(), expiredInvite.ID)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	assert.Equal(t, users.InviteStateExpired, expired.State,
		fmt.Sprintf("expected state %s got %s", users.InviteStateExpired, expired.State))
}
