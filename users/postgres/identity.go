// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ users.IdentityRepository = (*identityRepository)(nil)

type identityRepository struct {
	db dbutil.Database
}

// NewIdentityRepo instantiates a PostgreSQL implementation of identity repository.
// This repository manages external user identities, such as Google or GitHub accounts,
// and maps them to internal users for SSO/OAuth login.
func NewIdentityRepo(db dbutil.Database) users.IdentityRepository {
	return &identityRepository{
		db: db,
	}
}
func (ir identityRepository) Save(ctx context.Context, identity users.Identity) error {
	q := `INSERT INTO user_identities (user_id, provider, provider_user_id) VALUES (:user_id, :provider, :provider_user_id)`

	_, err := ir.db.NamedExecContext(ctx, q, identity)
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok && pgErr.Code == pgerrcode.UniqueViolation {
			return errors.Wrap(dbutil.ErrConflict, err)
		}
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return nil
}

func (ur identityRepository) Retrieve(ctx context.Context, provider, providerUserID string) (users.Identity, error) {
	q := `SELECT user_id, provider, provider_user_id, email FROM user_identities 
	      WHERE provider=$1 AND provider_user_id=$2`

	var dbID dbIdentity
	if err := ur.db.QueryRowxContext(ctx, q, provider, providerUserID).StructScan(&dbID); err != nil {
		if err == sql.ErrNoRows {
			return users.Identity{}, errors.Wrap(dbutil.ErrNotFound, err)
		}
		return users.Identity{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toUserIdentity(dbID), nil
}

type dbIdentity struct {
	UserID         string `db:"user_id"`
	Provider       string `db:"provider"`
	ProviderUserID string `db:"provider_user_id"`
	Email          string `db:"email"`
}

func toUserIdentity(dbID dbIdentity) users.Identity {
	return users.Identity{
		UserID:         dbID.UserID,
		Provider:       dbID.Provider,
		ProviderUserID: dbID.ProviderUserID,
	}
}
