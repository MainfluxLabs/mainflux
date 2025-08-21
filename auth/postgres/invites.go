// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type invitesRepository struct {
	db dbutil.Database
}

func NewOrgInvitesRepo(db dbutil.Database) auth.InvitesRepository {
	return &invitesRepository{
		db: db,
	}
}

func (ir invitesRepository) SaveOrgInvite(ctx context.Context, invites ...auth.OrgInvite) error {
	tx, err := ir.db.BeginTxx(ctx, nil)
	if err != nil {
		errors.Wrap(auth.ErrCreateInvite, err)
	}

	qIns := `
		INSERT INTO invites_org (id, invitee_id, inviter_id, org_id, invitee_role, created_at, expires_at, state)	
		VALUES (:id, :invitee_id, :inviter_id, :org_id, :invitee_role, :created_at, :expires_at, :state)
	`

	for _, invite := range invites {
		if err := ir.syncOrgInviteStateByInvite(ctx, invite); err != nil {
			tx.Rollback()
			return err
		}

		dbInvite := toDBOrgInvite(invite)
		if _, err := tx.NamedExecContext(ctx, qIns, dbInvite); err != nil {
			tx.Rollback()

			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					var e = errors.ErrConflict
					if pgErr.ConstraintName == "ux_invites_org_invitee_id_org_id" {
						e = auth.ErrUserAlreadyInvited
					}

					return errors.Wrap(e, errors.New(pgErr.Detail))
				}
			}

			return errors.Wrap(auth.ErrCreateInvite, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(auth.ErrCreateInvite, err)
	}

	return nil
}

func (ir invitesRepository) RetrieveOrgInviteByID(ctx context.Context, inviteID string) (auth.OrgInvite, error) {
	if err := ir.syncOrgInviteStateByID(ctx, inviteID); err != nil {
		return auth.OrgInvite{}, err
	}

	q := `
		SELECT invitee_id, inviter_id, org_id, invitee_role, created_at, expires_at, state
		FROM invites_org
		WHERE id = $1
	`

	dbI := dbOrgInvite{ID: inviteID}

	if err := ir.db.QueryRowxContext(ctx, q, inviteID).StructScan(&dbI); err != nil {
		if err == sql.ErrNoRows {
			return auth.OrgInvite{}, errors.Wrap(errors.ErrNotFound, err)
		}

		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return auth.OrgInvite{}, errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}

		return auth.OrgInvite{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	return toOrgInvite(dbI), nil
}

func (ir invitesRepository) RemoveOrgInvite(ctx context.Context, inviteID string) error {
	qDel := `DELETE FROM invites_org WHERE id = :id`
	invite := dbOrgInvite{
		ID: inviteID,
	}

	res, err := ir.db.NamedExecContext(ctx, qDel, invite)
	if err != nil {
		pqErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pqErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	if cnt != 1 {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	return nil
}

func (ir invitesRepository) UpdateOrgInviteState(ctx context.Context, inviteID string, state string) error {
	query := `
		UPDATE invites_org
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
				return errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	return nil
}

func (ir invitesRepository) RetrieveOrgInvitesByOrgID(ctx context.Context, orgID string, pm apiutil.PageMetadata) (auth.OrgInvitesPage, error) {
	query := `
		SELECT id, invitee_id, inviter_id, org_id, invitee_role, created_at, expires_at, state
		FROM invites_org
		WHERE org_id = :orgID
	`

	queryCount := `SELECT COUNT(*) FROM invites_org WHERE org_id = :orgID`

	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	query = fmt.Sprintf("%s %s", query, olq)

	params := map[string]any{
		"orgID":  orgID,
		"limit":  pm.Limit,
		"offset": pm.Offset,
	}

	rows, err := ir.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return auth.OrgInvitesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var invites []auth.OrgInvite

	for rows.Next() {
		dbInv := dbOrgInvite{}

		if err := rows.StructScan(&dbInv); err != nil {
			return auth.OrgInvitesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		inv := toOrgInvite(dbInv)
		invites = append(invites, inv)
	}

	total, err := dbutil.Total(ctx, ir.db, queryCount, params)
	if err != nil {
		return auth.OrgInvitesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	page := auth.OrgInvitesPage{
		Invites: invites,
		PageMetadata: apiutil.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (ir invitesRepository) RetrieveOrgInvitesByUserID(ctx context.Context, userType string, userID string, pm apiutil.PageMetadata) (auth.OrgInvitesPage, error) {
	query := `
		SELECT id, invitee_id, inviter_id, org_id, invitee_role, created_at, expires_at, state
		FROM invites_org
		WHERE %s = :userID
	`

	queryCount := `SELECT COUNT(*) FROM invites_org WHERE %s = :userID`

	switch userType {
	case auth.UserTypeInvitee:
		query = fmt.Sprintf(query, "invitee_id")
		queryCount = fmt.Sprintf(queryCount, "invitee_id")
	case auth.UserTypeInviter:
		query = fmt.Sprintf(query, "inviter_id")
		queryCount = fmt.Sprintf(queryCount, "inviter_id")
	default:
		return auth.OrgInvitesPage{}, errors.New("invalid invite user type")
	}

	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	query = fmt.Sprintf("%s %s", query, olq)

	params := map[string]any{
		"userID": userID,
		"limit":  pm.Limit,
		"offset": pm.Offset,
	}

	if err := ir.syncOrgInviteStateByUserID(ctx, userType, userID); err != nil {
		return auth.OrgInvitesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	rows, err := ir.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return auth.OrgInvitesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var invites []auth.OrgInvite

	for rows.Next() {
		dbInv := dbOrgInvite{}

		if err := rows.StructScan(&dbInv); err != nil {
			return auth.OrgInvitesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		inv := toOrgInvite(dbInv)
		invites = append(invites, inv)
	}

	total, err := dbutil.Total(ctx, ir.db, queryCount, params)
	if err != nil {
		return auth.OrgInvitesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	page := auth.OrgInvitesPage{
		Invites: invites,
		PageMetadata: apiutil.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

// Syncs the state of all invites either sent by or sent to the user denoted by `userID`, depending on the value
// of userType, which may be one of `inviter` or `invitee`. That is, sets state='expired' for all matching invites where
// state='pending' and expires_at < now().
func (ir invitesRepository) syncOrgInviteStateByUserID(ctx context.Context, userType string, userID string) error {
	query := `
		UPDATE invites_org
		SET state='expired'
		WHERE %s=:userID AND state='pending' AND expires_at < NOW()
	`

	var col string
	switch userType {
	case auth.UserTypeInvitee:
		col = "invitee_id"
	case auth.UserTypeInviter:
		col = "inviter_id"
	default:
		return errors.New("invalid invite user type")
	}

	query = fmt.Sprintf(query, col)

	_, err := ir.db.NamedExecContext(ctx, query, map[string]any{"userID": userID})
	if err != nil {
		pqErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pqErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	return nil
}

// Syncs the state of all invites in the database that would prevent the passed `invite`
// from being preserved. That is, sets state='expired' for invites where state='pending' and expires_at < now().
func (ir invitesRepository) syncOrgInviteStateByInvite(ctx context.Context, invite auth.OrgInvite) error {
	query := `
		UPDATE invites_org
		SET state='expired'
		WHERE invitee_id=:invitee_id AND org_id=:org_id AND inviter_id=:inviter_id AND state='pending' AND expires_at < NOW()
	`

	dbInvite := toDBOrgInvite(invite)

	_, err := ir.db.NamedExecContext(ctx, query, dbInvite)
	if err != nil {
		pqErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pqErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	return nil
}

// Syncs the state of the Invite with the passed inviteID. That is, sets state='expired' if state='pending' and expires_at < now().
func (ir invitesRepository) syncOrgInviteStateByID(ctx context.Context, inviteID string) error {
	query := `
		UPDATE invites_org
		SET state='expired'
		WHERE id=:inviteID AND state='pending' AND expires_at < NOW()
	`

	_, err := ir.db.NamedExecContext(ctx, query, map[string]any{"inviteID": inviteID})
	if err != nil {
		pqErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pqErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	return nil
}

func (ir invitesRepository) SavePlatformInvite(ctx context.Context, invites ...auth.PlatformInvite) error {
	tx, err := ir.db.BeginTxx(ctx, nil)
	if err != nil {
		errors.Wrap(auth.ErrCreateInvite, err)
	}

	qIns := `
		INSERT INTO invites_platform (id, invitee_email, created_at, expires_at, state)	
		VALUES (:id, :invitee_email, :created_at, :expires_at, :state)
	`

	for _, invite := range invites {
		if err := ir.syncPlatformInviteStateByEmail(ctx, invite.InviteeEmail); err != nil {
			tx.Rollback()
			return err
		}

		dbInvite := toDBPlatformInvite(invite)
		if _, err := tx.NamedExecContext(ctx, qIns, dbInvite); err != nil {
			tx.Rollback()

			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					var e = errors.ErrConflict
					if pgErr.ConstraintName == "ux_invites_platform_invitee_email" {
						e = auth.ErrUserAlreadyInvited
					}

					return errors.Wrap(e, errors.New(pgErr.Detail))
				}
			}

			return errors.Wrap(auth.ErrCreateInvite, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(auth.ErrCreateInvite, err)
	}

	return nil
}

func (ir invitesRepository) RetrievePlatformInviteByID(ctx context.Context, inviteID string) (auth.PlatformInvite, error) {
	if err := ir.syncPlatformInviteStateByID(ctx, inviteID); err != nil {
		return auth.PlatformInvite{}, err
	}

	q := `
		SELECT id, invitee_email, created_at, expires_at, state
		FROM invites_platform
		WHERE id = $1
	`

	dbI := dbPlatformInvite{ID: inviteID}

	if err := ir.db.QueryRowxContext(ctx, q, inviteID).StructScan(&dbI); err != nil {
		if err == sql.ErrNoRows {
			return auth.PlatformInvite{}, errors.Wrap(errors.ErrNotFound, err)
		}

		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return auth.PlatformInvite{}, errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}

		return auth.PlatformInvite{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	return toPlatformInvite(dbI), nil
}

func (ir invitesRepository) RetrievePlatformInvites(ctx context.Context, pm apiutil.PageMetadata) (auth.PlatformInvitesPage, error) {
	query := `
		SELECT id, invitee_email, created_at, expires_at, state
		FROM invites_platform
	`

	queryCount := `SELECT COUNT(*) FROM invites_platform`

	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	query = fmt.Sprintf("%s %s", query, olq)

	params := map[string]any{
		"limit":  pm.Limit,
		"offset": pm.Offset,
	}

	if err := ir.syncPlatformInviteState(ctx); err != nil {
		return auth.PlatformInvitesPage{}, err
	}

	rows, err := ir.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return auth.PlatformInvitesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var invites []auth.PlatformInvite

	for rows.Next() {
		dbInv := dbPlatformInvite{}

		if err := rows.StructScan(&dbInv); err != nil {
			return auth.PlatformInvitesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		inv := toPlatformInvite(dbInv)
		invites = append(invites, inv)
	}

	total, err := dbutil.Total(ctx, ir.db, queryCount, params)
	if err != nil {
		return auth.PlatformInvitesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	page := auth.PlatformInvitesPage{
		Invites: invites,
		PageMetadata: apiutil.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (ir invitesRepository) UpdatePlatformInviteState(ctx context.Context, inviteID string, state string) error {
	query := `
		UPDATE invites_platform
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
				return errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	return nil
}

// Syncs the states of all platform invites with the passed invitee email. That is, sets
// state='expired' where state='pending' and expires_at < now().
func (ir invitesRepository) syncPlatformInviteStateByEmail(ctx context.Context, email string) error {
	query := `
		UPDATE invites_platform
		SET state='expired'
		WHERE invitee_email=:email AND state='pending' AND expires_at < NOW()
	`

	_, err := ir.db.NamedExecContext(ctx, query, map[string]any{"email": email})
	if err != nil {
		pqErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pqErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	return nil
}

// Syncs the states of a specific invite denoted by its ID. That is, sets state='expired' where state='pending'
// and expires_at < now().
func (ir invitesRepository) syncPlatformInviteStateByID(ctx context.Context, inviteID string) error {
	query := `
		UPDATE invites_platform
		SET state='expired'
		WHERE id=:inviteID AND state='pending' AND expires_at < NOW()
	`

	_, err := ir.db.NamedExecContext(ctx, query, map[string]any{"inviteID": inviteID})
	if err != nil {
		pqErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pqErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	return nil
}

// Syncs the state of all platform invites in the database. That is, sets state='expired' where state='pending'
// and expires_at < now().
func (ir invitesRepository) syncPlatformInviteState(ctx context.Context) error {
	query := `
		UPDATE invites_platform
		SET state='expired'
		WHERE state='pending' AND expires_at < NOW()
	`

	_, err := ir.db.NamedExecContext(ctx, query, map[string]any{})
	if err != nil {
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	return nil
}

func toDBOrgInvite(invite auth.OrgInvite) dbOrgInvite {
	return dbOrgInvite{
		ID:          invite.ID,
		InviteeID:   invite.InviteeID,
		InviterID:   invite.InviterID,
		OrgID:       invite.OrgID,
		InviteeRole: invite.InviteeRole,
		CreatedAt:   invite.CreatedAt,
		ExpiresAt:   invite.ExpiresAt,
		State:       invite.State,
	}
}

func toOrgInvite(dbI dbOrgInvite) auth.OrgInvite {
	return auth.OrgInvite{
		ID:          dbI.ID,
		InviteeID:   dbI.InviteeID,
		InviterID:   dbI.InviterID,
		OrgID:       dbI.OrgID,
		InviteeRole: dbI.InviteeRole,
		CreatedAt:   dbI.CreatedAt,
		ExpiresAt:   dbI.ExpiresAt,
		State:       dbI.State,
	}
}

func toDBPlatformInvite(invite auth.PlatformInvite) dbPlatformInvite {
	return dbPlatformInvite{
		ID:           invite.ID,
		InviteeEmail: invite.InviteeEmail,
		CreatedAt:    invite.CreatedAt,
		ExpiresAt:    invite.ExpiresAt,
		State:        invite.State,
	}
}

func toPlatformInvite(dbI dbPlatformInvite) auth.PlatformInvite {
	return auth.PlatformInvite{
		ID:           dbI.ID,
		InviteeEmail: dbI.InviteeEmail,
		CreatedAt:    dbI.CreatedAt,
		ExpiresAt:    dbI.ExpiresAt,
		State:        dbI.State,
	}
}

type dbOrgInvite struct {
	ID          string    `db:"id"`
	InviteeID   string    `db:"invitee_id"`
	InviterID   string    `db:"inviter_id"`
	OrgID       string    `db:"org_id"`
	InviteeRole string    `db:"invitee_role"`
	CreatedAt   time.Time `db:"created_at"`
	ExpiresAt   time.Time `db:"expires_at"`
	State       string    `db:"state"`
}

type dbPlatformInvite struct {
	ID           string    `db:"id"`
	InviteeEmail string    `db:"invitee_email"`
	CreatedAt    time.Time `db:"created_at"`
	ExpiresAt    time.Time `db:"expires_at"`
	State        string    `db:"state"`
}
