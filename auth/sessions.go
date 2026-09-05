// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

type Session = domain.Session

// A row is written for every login token that is issued, and is never deleted
// on revocation: the revoked row is what lets RefreshToken tell a replayed token
// apart from an unknown one. Rows are removed only once they are old enough
// that no session could still depend on them, see RemoveExpired.
type SessionRepository interface {
	// Save persists a newly issued session token.
	Save(ctx context.Context, session Session) error

	// Retrieve returns the session token identified by the provided JTI.
	Retrieve(ctx context.Context, jti string) (Session, error)

	// Rotate replaces one session token with its successor. It revokes the
	// presented token, but only if it was still live, and inserts the
	// successor into the same family and session.
	Rotate(ctx context.Context, jti, nextJTI string, at time.Time) (Session, bool, error)

	// RevokeFamily marks every token descending from one login dead. Used by
	// reuse detection and by logging out a single session.
	RevokeFamily(ctx context.Context, familyID string, at time.Time) error

	// RevokeByUser marks every token belonging to a user dead, across all of
	// their sessions. Used by "log out everywhere".
	RevokeByUser(ctx context.Context, userID string, at time.Time) error

	// RemoveExpired deletes rows whose session began before the provided
	// cutoff. Such rows can no longer belong to a live session, so dropping
	// them cannot weaken reuse detection.
	RemoveExpired(ctx context.Context, before time.Time) error
}

type Sessions interface {
	// RefreshToken rotates a login token: the presented token is revoked and a
	// replacement is issued into the same session. Replaying an already
	// rotated token is treated as theft and kills the whole session.
	RefreshToken(ctx context.Context, token string) (string, error)

	// Logout ends the session the provided token belongs to, leaving the
	// user's other sessions untouched.
	Logout(ctx context.Context, token string) error

	// LogoutAll ends every session belonging to the owner of the provided
	// token.
	LogoutAll(ctx context.Context, token string) error

	// RevokeSessions ends every session belonging to the identified user.
	// It is the path another service takes after a credential change, so
	// unlike LogoutAll it works from a user ID alone, with no token of that
	// user's in hand.
	RevokeSessions(ctx context.Context, userID string) error
}
