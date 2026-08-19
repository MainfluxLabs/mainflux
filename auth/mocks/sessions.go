// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
)

var _ auth.SessionRepository = (*sessionRepositoryMock)(nil)

type sessionRepositoryMock struct {
	mu       sync.Mutex
	sessions map[string]auth.Session
}

// NewSessionRepository creates an in-memory session repository.
func NewSessionRepository() auth.SessionRepository {
	return &sessionRepositoryMock{
		sessions: make(map[string]auth.Session),
	}
}

func (srm *sessionRepositoryMock) Save(ctx context.Context, session auth.Session) error {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	if _, ok := srm.sessions[session.JTI]; ok {
		return dbutil.ErrConflict
	}

	srm.sessions[session.JTI] = session
	return nil
}

func (srm *sessionRepositoryMock) Retrieve(ctx context.Context, jti string) (auth.Session, error) {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	session, ok := srm.sessions[jti]
	if !ok {
		return auth.Session{}, dbutil.ErrNotFound
	}

	return session, nil
}

func (srm *sessionRepositoryMock) Revoke(ctx context.Context, jti string, at time.Time) error {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	session, ok := srm.sessions[jti]
	if !ok {
		return dbutil.ErrNotFound
	}
	if session.Revoked() {
		return nil
	}

	session.RevokedAt = at
	srm.sessions[jti] = session
	return nil
}

func (srm *sessionRepositoryMock) RevokeFamily(ctx context.Context, familyID string, at time.Time) error {
	return srm.revokeMatching(func(s auth.Session) bool { return s.FamilyID == familyID }, at)
}

func (srm *sessionRepositoryMock) RevokeByUser(ctx context.Context, userID string, at time.Time) error {
	return srm.revokeMatching(func(s auth.Session) bool { return s.UserID == userID }, at)
}

func (srm *sessionRepositoryMock) revokeMatching(match func(auth.Session) bool, at time.Time) error {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	for jti, session := range srm.sessions {
		if !match(session) || session.Revoked() {
			continue
		}
		session.RevokedAt = at
		srm.sessions[jti] = session
	}

	return nil
}

func (srm *sessionRepositoryMock) RemoveExpired(ctx context.Context, before time.Time) error {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	for jti, session := range srm.sessions {
		if session.SessionStartAt.Before(before) {
			delete(srm.sessions, jti)
		}
	}

	return nil
}
