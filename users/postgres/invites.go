// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type invitesRepository struct {
	db dbutil.Database
}

func NewPlatformInvitesRepo(db dbutil.Database) users.PlatformInvitesRepository {
	return &invitesRepository{
		db: db,
	}
}

func (ir invitesRepository) SavePlatformInvite(ctx context.Context, invites ...users.PlatformInvite) error {
	tx, err := ir.db.BeginTxx(ctx, nil)
	if err != nil {
		errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	qIns := `
		INSERT INTO platform_invites (id, invitee_email, created_at, expires_at, state)	
		VALUES (:id, :invitee_email, :created_at, :expires_at, :state)
	`

	for _, invite := range invites {
		if err := ir.syncPlatformInviteStateByEmail(ctx, invite.InviteeEmail); err != nil {
			return err
		}

		dbInvite := toDBPlatformInvite(invite)
		if _, err := tx.NamedExecContext(ctx, qIns, dbInvite); err != nil {

			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return errors.Wrap(apiutil.ErrUserAlreadyInvited, err)
				}
			}

			return errors.Wrap(dbutil.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return nil
}

func (ir invitesRepository) RetrievePlatformInviteByID(ctx context.Context, inviteID string) (users.PlatformInvite, error) {
	if err := ir.syncPlatformInviteStateByID(ctx, inviteID); err != nil {
		return users.PlatformInvite{}, err
	}

	q := `
		SELECT id, invitee_email, created_at, expires_at, state
		FROM platform_invites
		WHERE id = $1
	`

	dbI := dbPlatformInvite{ID: inviteID}

	if err := ir.db.QueryRowxContext(ctx, q, inviteID).StructScan(&dbI); err != nil {
		if err == sql.ErrNoRows {
			return users.PlatformInvite{}, errors.Wrap(dbutil.ErrNotFound, err)
		}

		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return users.PlatformInvite{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}

		return users.PlatformInvite{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toPlatformInvite(dbI), nil
}

func (ir invitesRepository) RetrievePlatformInvites(ctx context.Context, pm users.PageMetadataInvites) (users.PlatformInvitesPage, error) {
	query := `
		SELECT id, invitee_email, created_at, expires_at, state
		FROM platform_invites %s ORDER BY %s %s %s
	`

	queryCount := `SELECT COUNT(*) FROM platform_invites %s`

	filterState := ``
	if pm.State != "" {
		filterState = `state = :state`
	}

	whereClause := dbutil.BuildWhereClause(filterState)
	oq := dbutil.GetOrderQuery(pm.Order, users.AllowedOrders)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	query = fmt.Sprintf(query, whereClause, oq, dq, olq)
	queryCount = fmt.Sprintf(queryCount, whereClause)

	params := map[string]any{
		"limit":  pm.Limit,
		"offset": pm.Offset,
		"state":  pm.State,
	}

	if err := ir.syncPlatformInviteState(ctx); err != nil {
		return users.PlatformInvitesPage{}, err
	}

	rows, err := ir.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return users.PlatformInvitesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var invites []users.PlatformInvite

	for rows.Next() {
		dbInv := dbPlatformInvite{}

		if err := rows.StructScan(&dbInv); err != nil {
			return users.PlatformInvitesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		inv := toPlatformInvite(dbInv)
		invites = append(invites, inv)
	}

	total, err := dbutil.Total(ctx, ir.db, queryCount, params)
	if err != nil {
		return users.PlatformInvitesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	page := users.PlatformInvitesPage{
		Invites: invites,
		PageMetadata: apiutil.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (ir invitesRepository) UpdatePlatformInviteState(ctx context.Context, inviteID, state string) error {
	query := `
		UPDATE platform_invites
		SET state=:state
		WHERE id=:inviteID
	`
	_, err := ir.db.NamedExecContext(ctx, query, map[string]any{
		"inviteID": inviteID,
		"state":    state,
	})

	if err != nil {
		pqErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pqErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	return nil
}

// Syncs the states of all platform invites with the passed invitee email. That is, sets
// state='expired' where state='pending' and expires_at < now().
func (ir invitesRepository) syncPlatformInviteStateByEmail(ctx context.Context, email string) error {
	query := `
		UPDATE platform_invites
		SET state='expired'
		WHERE invitee_email=:email AND state='pending' AND expires_at < NOW()
	`

	_, err := ir.db.NamedExecContext(ctx, query, map[string]any{"email": email})
	if err != nil {
		pqErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pqErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	return nil
}

// Syncs the states of a specific invite denoted by its ID. That is, sets state='expired' where state='pending'
// and expires_at < now().
func (ir invitesRepository) syncPlatformInviteStateByID(ctx context.Context, inviteID string) error {
	query := `
		UPDATE platform_invites
		SET state='expired'
		WHERE id=:inviteID AND state='pending' AND expires_at < NOW()
	`

	_, err := ir.db.NamedExecContext(ctx, query, map[string]any{"inviteID": inviteID})
	if err != nil {
		pqErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pqErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	return nil
}

// Syncs the state of all platform invites in the database. That is, sets state='expired' where state='pending'
// and expires_at < now().
func (ir invitesRepository) syncPlatformInviteState(ctx context.Context) error {
	query := `
		UPDATE platform_invites
		SET state='expired'
		WHERE state='pending' AND expires_at < NOW()
	`

	_, err := ir.db.NamedExecContext(ctx, query, map[string]any{})
	if err != nil {
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	return nil
}

func toDBPlatformInvite(invite users.PlatformInvite) dbPlatformInvite {
	return dbPlatformInvite{
		ID:           invite.ID,
		InviteeEmail: invite.InviteeEmail,
		CreatedAt:    invite.CreatedAt,
		ExpiresAt:    invite.ExpiresAt,
		State:        invite.State,
	}
}

func toPlatformInvite(dbI dbPlatformInvite) users.PlatformInvite {
	return users.PlatformInvite{
		ID:           dbI.ID,
		InviteeEmail: dbI.InviteeEmail,
		CreatedAt:    dbI.CreatedAt,
		ExpiresAt:    dbI.ExpiresAt,
		State:        dbI.State,
	}
}

type dbPlatformInvite struct {
	ID           string    `db:"id"`
	InviteeEmail string    `db:"invitee_email"`
	CreatedAt    time.Time `db:"created_at"`
	ExpiresAt    time.Time `db:"expires_at"`
	State        string    `db:"state"`
}
