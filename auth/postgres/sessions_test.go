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

func TestSessionRotate(t *testing.T) {
	repo := newSessionRepo()
	session := newSession(t)
	require.Nil(t, repo.Save(context.Background(), session), "saving session expected to succeed")

	successor := rotate(t, session)
	revokedAt := time.Now().UTC().Round(time.Millisecond)
	won, ok, err := repo.Rotate(context.Background(), session.JTI, successor.JTI, revokedAt)
	require.Nil(t, err, fmt.Sprintf("rotating session expected to succeed: %s", err))
	assert.True(t, ok, "expected to win the rotation on a live session")
	assert.Equal(t, session.FamilyID, won.FamilyID, "expected the row to be returned so the caller can reach its family")
	assert.True(t, won.Revoked(), "expected the returned row to carry the revocation")

	next, err := repo.Retrieve(context.Background(), successor.JTI)
	require.Nil(t, err, fmt.Sprintf("expected the successor to be written by the same rotation: %s", err))
	assert.Equal(t, session.FamilyID, next.FamilyID, "expected the successor to stay in the family")
	assert.Equal(t, session.UserID, next.UserID, "expected the successor to belong to the same user")
	assert.True(t, session.SessionStartAt.Equal(next.SessionStartAt), "expected the session start to carry over")
	assert.False(t, next.Revoked(), "expected the successor to be live")

	lost, ok, err := repo.Rotate(context.Background(), session.JTI, rotate(t, session).JTI, revokedAt.Add(time.Hour))
	require.Nil(t, err, fmt.Sprintf("replaying a rotated token expected to return cleanly: %s", err))
	assert.False(t, ok, "expected the second rotation to lose")
	assert.Equal(t, session.FamilyID, lost.FamilyID, "expected the family of a replayed token so that it can be killed")

	retrieved, err := repo.Retrieve(context.Background(), session.JTI)
	require.Nil(t, err, fmt.Sprintf("retrieving session expected to succeed: %s", err))
	assert.True(t, revokedAt.Equal(retrieved.RevokedAt), "expected the original revocation time to be kept")

	unknown := rotate(t, session)
	_, _, err = repo.Rotate(context.Background(), "00000000-0000-0000-0000-000000000000", unknown.JTI, revokedAt)
	assert.True(t, errors.Contains(err, dbutil.ErrNotFound), fmt.Sprintf("expected an unknown jti to be reported as not found, got %s", err))

	_, err = repo.Retrieve(context.Background(), unknown.JTI)
	assert.True(t, errors.Contains(err, dbutil.ErrNotFound), fmt.Sprintf("expected no successor for an unknown jti, got %s", err))

	_, _, err = repo.Rotate(context.Background(), "not-a-uuid", unknown.JTI, revokedAt)
	assert.True(t, errors.Contains(err, dbutil.ErrNotFound), fmt.Sprintf("expected a malformed jti to be reported as not found, got %s", err))
}

// A rotation that cannot write its successor must not revoke the token it was
// handed, or the caller is left with no way back into the session.
func TestSessionRotateIsAtomic(t *testing.T) {
	repo := newSessionRepo()
	session := newSession(t)
	taken := newSession(t)

	for _, s := range []auth.Session{session, taken} {
		require.Nil(t, repo.Save(context.Background(), s), "saving session expected to succeed")
	}

	_, _, err := repo.Rotate(context.Background(), session.JTI, taken.JTI, time.Now().UTC())
	assert.True(t, errors.Contains(err, dbutil.ErrConflict), fmt.Sprintf("expected a conflict on a duplicate successor jti, got %s", err))

	retrieved, err := repo.Retrieve(context.Background(), session.JTI)
	require.Nil(t, err, fmt.Sprintf("retrieving session expected to succeed: %s", err))
	assert.False(t, retrieved.Revoked(), "expected a failed rotation to leave the presented token live")
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
