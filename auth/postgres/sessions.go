// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ auth.SessionRepository = (*sessionRepo)(nil)

type sessionRepo struct {
	db dbutil.Database
}

func NewSessionRepository(db dbutil.Database) auth.SessionRepository {
	return &sessionRepo{
		db: db,
	}
}

func (sr sessionRepo) Save(ctx context.Context, session auth.Session) error {
	q := `INSERT INTO sessions (jti, user_id, family_id, session_start_at, revoked_at)
	      VALUES (:jti, :user_id, :family_id, :session_start_at, :revoked_at)`

	if _, err := sr.db.NamedExecContext(ctx, q, toDBSession(session)); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.UniqueViolation {
			return errors.Wrap(dbutil.ErrConflict, err)
		}

		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return nil
}

func (sr sessionRepo) Retrieve(ctx context.Context, jti string) (auth.Session, error) {
	q := `SELECT jti, user_id, family_id, session_start_at, revoked_at FROM sessions WHERE jti = $1`

	dbs := dbSession{}
	if err := sr.db.QueryRowxContext(ctx, q, jti).StructScan(&dbs); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return auth.Session{}, errors.Wrap(dbutil.ErrNotFound, err)
		}

		return auth.Session{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toSession(dbs), nil
}

func (sr sessionRepo) RevokeIfLive(ctx context.Context, jti string, at time.Time) (auth.Session, bool, error) {
	q := `UPDATE sessions SET revoked_at = $1
	      WHERE jti = $2 AND revoked_at IS NULL
	      RETURNING jti, user_id, family_id, session_start_at, revoked_at`

	dbs := dbSession{}
	if err := sr.db.QueryRowxContext(ctx, q, at, jti).StructScan(&dbs); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return auth.Session{}, false, nil
		}

		return auth.Session{}, false, errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	return toSession(dbs), true, nil
}

func (sr sessionRepo) RevokeFamily(ctx context.Context, familyID string, at time.Time) error {
	q := `UPDATE sessions SET revoked_at = :revoked_at WHERE family_id = :family_id AND revoked_at IS NULL`
	return sr.revoke(ctx, q, dbSession{FamilyID: familyID, RevokedAt: sql.NullTime{Time: at, Valid: true}})
}

func (sr sessionRepo) RevokeByUser(ctx context.Context, userID string, at time.Time) error {
	q := `UPDATE sessions SET revoked_at = :revoked_at WHERE user_id = :user_id AND revoked_at IS NULL`
	return sr.revoke(ctx, q, dbSession{UserID: userID, RevokedAt: sql.NullTime{Time: at, Valid: true}})
}

func (sr sessionRepo) revoke(ctx context.Context, q string, dbs dbSession) error {
	if _, err := sr.db.NamedExecContext(ctx, q, dbs); err != nil {
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	return nil
}

func (sr sessionRepo) RemoveExpired(ctx context.Context, before time.Time) error {
	q := `DELETE FROM sessions WHERE session_start_at < :session_start_at`

	if _, err := sr.db.NamedExecContext(ctx, q, dbSession{SessionStartAt: before}); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

type dbSession struct {
	JTI            string       `db:"jti"`
	UserID         string       `db:"user_id"`
	FamilyID       string       `db:"family_id"`
	SessionStartAt time.Time    `db:"session_start_at"`
	RevokedAt      sql.NullTime `db:"revoked_at"`
}

func toDBSession(s auth.Session) dbSession {
	return dbSession{
		JTI:            s.JTI,
		UserID:         s.UserID,
		FamilyID:       s.FamilyID,
		SessionStartAt: s.SessionStartAt,
		RevokedAt:      sql.NullTime{Time: s.RevokedAt, Valid: !s.RevokedAt.IsZero()},
	}
}

func toSession(dbs dbSession) auth.Session {
	s := auth.Session{
		JTI:            dbs.JTI,
		UserID:         dbs.UserID,
		FamilyID:       dbs.FamilyID,
		SessionStartAt: dbs.SessionStartAt,
	}
	if dbs.RevokedAt.Valid {
		s.RevokedAt = dbs.RevokedAt.Time
	}

	return s
}
