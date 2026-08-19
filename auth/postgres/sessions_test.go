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
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSessionRepo() auth.SessionRepository {
	return postgres.NewSessionRepository(dbutil.NewDatabase(db))
}

// newSession builds a live session row for a fresh user and family.
func newSession(t *testing.T) auth.Session {
	jti, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	userID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	familyID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	return auth.Session{
		JTI:            jti,
		UserID:         userID,
		FamilyID:       familyID,
		SessionStartAt: time.Now().UTC().Round(time.Millisecond),
	}
}

// rotate adds a successor into an existing session, the way Refresh does.
func rotate(t *testing.T, s auth.Session) auth.Session {
	jti, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	next := s
	next.JTI = jti
	next.RevokedAt = time.Time{}

	return next
}

func TestSessionSaveAndRetrieve(t *testing.T) {
	repo := newSessionRepo()
	session := newSession(t)

	err := repo.Save(context.Background(), session)
	require.Nil(t, err, fmt.Sprintf("saving session expected to succeed: %s", err))

	err = repo.Save(context.Background(), session)
	assert.True(t, errors.Contains(err, dbutil.ErrConflict), fmt.Sprintf("expected a conflict on a duplicate jti, got %s", err))

	retrieved, err := repo.Retrieve(context.Background(), session.JTI)
	require.Nil(t, err, fmt.Sprintf("retrieving session expected to succeed: %s", err))
	assert.Equal(t, session.JTI, retrieved.JTI, "expected the same jti")
	assert.Equal(t, session.FamilyID, retrieved.FamilyID, "expected the same family")
	assert.False(t, retrieved.Revoked(), "a freshly saved session must not be revoked")

	_, err = repo.Retrieve(context.Background(), "non-existent")
	assert.True(t, errors.Contains(err, dbutil.ErrNotFound), fmt.Sprintf("expected not found, got %s", err))
}

func TestSessionRevoke(t *testing.T) {
	repo := newSessionRepo()
	session := newSession(t)
	require.Nil(t, repo.Save(context.Background(), session), "saving session expected to succeed")

	revokedAt := time.Now().UTC().Round(time.Millisecond)
	require.Nil(t, repo.Revoke(context.Background(), session.JTI, revokedAt), "revoking session expected to succeed")

	retrieved, err := repo.Retrieve(context.Background(), session.JTI)
	require.Nil(t, err, fmt.Sprintf("retrieving session expected to succeed: %s", err))
	assert.True(t, retrieved.Revoked(), "expected the session to be revoked")

	// The first revocation timestamp is the one reuse detection reasons about,
	// so a second revoke must not move it.
	first := retrieved.RevokedAt
	require.Nil(t, repo.Revoke(context.Background(), session.JTI, revokedAt.Add(time.Hour)), "re-revoking expected to succeed")

	retrieved, err = repo.Retrieve(context.Background(), session.JTI)
	require.Nil(t, err, fmt.Sprintf("retrieving session expected to succeed: %s", err))
	assert.True(t, first.Equal(retrieved.RevokedAt), "expected the original revocation time to be kept")
}

func TestSessionRevokeFamily(t *testing.T) {
	repo := newSessionRepo()

	session := newSession(t)
	successor := rotate(t, session)
	other := newSession(t)

	for _, s := range []auth.Session{session, successor, other} {
		require.Nil(t, repo.Save(context.Background(), s), "saving session expected to succeed")
	}

	at := time.Now().UTC().Round(time.Millisecond)
	require.Nil(t, repo.RevokeFamily(context.Background(), session.FamilyID, at), "revoking family expected to succeed")

	for _, jti := range []string{session.JTI, successor.JTI} {
		retrieved, err := repo.Retrieve(context.Background(), jti)
		require.Nil(t, err, fmt.Sprintf("retrieving session expected to succeed: %s", err))
		assert.True(t, retrieved.Revoked(), "expected every token in the family to be revoked")
	}

	// A different login is a different family and must survive.
	retrieved, err := repo.Retrieve(context.Background(), other.JTI)
	require.Nil(t, err, fmt.Sprintf("retrieving session expected to succeed: %s", err))
	assert.False(t, retrieved.Revoked(), "expected another family to be untouched")
}

func TestSessionRevokeByUser(t *testing.T) {
	repo := newSessionRepo()

	first := newSession(t)
	second := newSession(t)
	second.UserID = first.UserID
	other := newSession(t)

	for _, s := range []auth.Session{first, second, other} {
		require.Nil(t, repo.Save(context.Background(), s), "saving session expected to succeed")
	}

	at := time.Now().UTC().Round(time.Millisecond)
	require.Nil(t, repo.RevokeByUser(context.Background(), first.UserID, at), "revoking by user expected to succeed")

	for _, jti := range []string{first.JTI, second.JTI} {
		retrieved, err := repo.Retrieve(context.Background(), jti)
		require.Nil(t, err, fmt.Sprintf("retrieving session expected to succeed: %s", err))
		assert.True(t, retrieved.Revoked(), "expected every session of the user to be revoked")
	}

	retrieved, err := repo.Retrieve(context.Background(), other.JTI)
	require.Nil(t, err, fmt.Sprintf("retrieving session expected to succeed: %s", err))
	assert.False(t, retrieved.Revoked(), "expected another user's session to be untouched")
}

func TestSessionRemoveExpired(t *testing.T) {
	repo := newSessionRepo()

	old := newSession(t)
	old.SessionStartAt = time.Now().UTC().Add(-30 * 24 * time.Hour)
	recent := newSession(t)

	for _, s := range []auth.Session{old, recent} {
		require.Nil(t, repo.Save(context.Background(), s), "saving session expected to succeed")
	}

	cutoff := time.Now().UTC().Add(-7 * 24 * time.Hour)
	require.Nil(t, repo.RemoveExpired(context.Background(), cutoff), "purging expected to succeed")

	_, err := repo.Retrieve(context.Background(), old.JTI)
	assert.True(t, errors.Contains(err, dbutil.ErrNotFound), fmt.Sprintf("expected the stale row to be purged, got %s", err))

	_, err = repo.Retrieve(context.Background(), recent.JTI)
	assert.Nil(t, err, fmt.Sprintf("expected a live session to survive the purge: %s", err))
}
