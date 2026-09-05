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
		JTI:       jti,
		UserID:    userID,
		FamilyID:  familyID,
		StartedAt: time.Now().UTC().Round(time.Millisecond),
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
	replayed := rotate(t, session)
	unknown := rotate(t, session)
	revokedAt := time.Now().UTC().Round(time.Millisecond)

	cases := []struct {
		desc    string
		jti     string
		nextJTI string
		at      time.Time
		rotated auth.Session
		ok      bool
		err     error
	}{
		{
			desc:    "rotate a live session",
			jti:     session.JTI,
			nextJTI: successor.JTI,
			at:      revokedAt,
			rotated: auth.Session{
				JTI:       session.JTI,
				UserID:    session.UserID,
				FamilyID:  session.FamilyID,
				StartedAt: session.StartedAt,
				RevokedAt: revokedAt,
			},
			ok:  true,
			err: nil,
		},
		{
			desc:    "replay a rotated token",
			jti:     session.JTI,
			nextJTI: replayed.JTI,
			at:      revokedAt.Add(time.Hour),
			rotated: auth.Session{
				JTI:       session.JTI,
				UserID:    session.UserID,
				FamilyID:  session.FamilyID,
				StartedAt: session.StartedAt,
				RevokedAt: revokedAt,
			},
			ok:  false,
			err: nil,
		},
		{
			desc:    "rotate an unknown jti",
			jti:     "00000000-0000-0000-0000-000000000000",
			nextJTI: unknown.JTI,
			at:      revokedAt,
			rotated: auth.Session{},
			ok:      false,
			err:     dbutil.ErrNotFound,
		},
		{
			desc:    "rotate a malformed jti",
			jti:     "not-a-uuid",
			nextJTI: unknown.JTI,
			at:      revokedAt,
			rotated: auth.Session{},
			ok:      false,
			err:     dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		rotated, ok, err := repo.Rotate(context.Background(), tc.jti, tc.nextJTI, tc.at)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.ok, ok, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.ok, ok))
		assert.Equal(t, tc.rotated.JTI, rotated.JTI, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.rotated.JTI, rotated.JTI))
		assert.Equal(t, tc.rotated.FamilyID, rotated.FamilyID, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.rotated.FamilyID, rotated.FamilyID))
		assert.True(t, tc.rotated.RevokedAt.Equal(rotated.RevokedAt), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.rotated.RevokedAt, rotated.RevokedAt))
	}

	retrieveCases := []struct {
		desc    string
		jti     string
		session auth.Session
		err     error
	}{
		{
			desc: "retrieve the successor written by the rotation",
			jti:  successor.JTI,
			session: auth.Session{
				JTI:       successor.JTI,
				UserID:    session.UserID,
				FamilyID:  session.FamilyID,
				StartedAt: session.StartedAt,
			},
			err: nil,
		},
		{
			desc: "retrieve the rotated session",
			jti:  session.JTI,
			session: auth.Session{
				JTI:       session.JTI,
				UserID:    session.UserID,
				FamilyID:  session.FamilyID,
				StartedAt: session.StartedAt,
				RevokedAt: revokedAt,
			},
			err: nil,
		},
		{
			desc:    "retrieve the successor of a rotation that never happened",
			jti:     unknown.JTI,
			session: auth.Session{},
			err:     dbutil.ErrNotFound,
		},
	}

	for _, tc := range retrieveCases {
		retrieved, err := repo.Retrieve(context.Background(), tc.jti)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.session.JTI, retrieved.JTI, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.session.JTI, retrieved.JTI))
		assert.Equal(t, tc.session.UserID, retrieved.UserID, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.session.UserID, retrieved.UserID))
		assert.Equal(t, tc.session.FamilyID, retrieved.FamilyID, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.session.FamilyID, retrieved.FamilyID))
		assert.True(t, tc.session.StartedAt.Equal(retrieved.StartedAt), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.session.StartedAt, retrieved.StartedAt))
		assert.True(t, tc.session.RevokedAt.Equal(retrieved.RevokedAt), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.session.RevokedAt, retrieved.RevokedAt))
	}
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
	old.StartedAt = time.Now().UTC().Add(-30 * 24 * time.Hour)
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
