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

func NewInvitesRepo(db dbutil.Database) auth.InvitesRepository {
	return &invitesRepository{
		db: db,
	}
}

func (ir invitesRepository) Save(ctx context.Context, invites ...auth.Invite) error {
	tx, err := ir.db.BeginTxx(ctx, nil)
	if err != nil {
		errors.Wrap(auth.ErrCreateInvite, err)
	}

	qIns := `
		INSERT INTO invites (id, invitee_id, invitee_email, inviter_id, org_id, invitee_role, created_at, expires_at)	
		VALUES (:id, :invitee_id, :invitee_email, :inviter_id, :org_id, :invitee_role, :created_at, :expires_at)
	`

	for _, invite := range invites {
		if err := ir.purgeExpiredInvites(ctx, invite); err != nil {
			tx.Rollback()
			return err
		}

		dbInvite := toDBInvite(invite)
		if _, err := tx.NamedExecContext(ctx, qIns, dbInvite); err != nil {
			tx.Rollback()

			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					var e = errors.ErrConflict
					if pgErr.ConstraintName == "invites_invitee_email_inviter_id_org_id_key" || pgErr.ConstraintName == "invites_invitee_id_inviter_id_org_id_key" {
						e = auth.ErrUserAlreadyInvited
					}

					return errors.Wrap(e, errors.New(pgErr.Detail))
				}
			}

			return errors.Wrap(auth.ErrCreateInvite, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(auth.ErrCreateOrgMembership, err)
	}

	return nil
}

func (ir invitesRepository) RetrieveByID(ctx context.Context, inviteID string) (auth.Invite, error) {
	q := `
		SELECT invitee_id, invitee_email, inviter_id, org_id, invitee_role, created_at, expires_at
		FROM invites
		WHERE id = $1
	`

	dbI := dbInvite{ID: inviteID}

	if err := ir.db.QueryRowxContext(ctx, q, inviteID).StructScan(&dbI); err != nil {
		if err == sql.ErrNoRows {
			return auth.Invite{}, errors.Wrap(errors.ErrNotFound, err)
		}

		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return auth.Invite{}, errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}

		return auth.Invite{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	return toInvite(dbI), nil
}

func (ir invitesRepository) Remove(ctx context.Context, inviteID string) error {
	qDel := `DELETE FROM invites WHERE id = :id`
	invite := dbInvite{
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

func (ir invitesRepository) RetrieveByUserID(ctx context.Context, userType string, userID string, pm apiutil.PageMetadata) (auth.InvitesPage, error) {
	query := `
		SELECT id, invitee_id, invitee_email, inviter_id, org_id, invitee_role, created_at, expires_at
		FROM invites
		WHERE %s = :userID AND expires_at > NOW()
	`

	queryCount := `SELECT COUNT(*) FROM invites WHERE %s = :userID AND expires_at > NOW()`

	switch userType {
	case auth.UserTypeInvitee:
		query = fmt.Sprintf(query, "invitee_id")
		queryCount = fmt.Sprintf(queryCount, "invitee_id")
	case auth.UserTypeInviter:
		query = fmt.Sprintf(query, "inviter_id")
		queryCount = fmt.Sprintf(queryCount, "inviter_id")
	default:
		return auth.InvitesPage{}, errors.New("invalid invite user type")
	}

	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	query = fmt.Sprintf("%s %s", query, olq)

	params := map[string]any{
		"userID": userID,
		"limit":  pm.Limit,
		"offset": pm.Offset,
	}

	rows, err := ir.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return auth.InvitesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var invites []auth.Invite

	for rows.Next() {
		dbInv := dbInvite{}

		if err := rows.StructScan(&dbInv); err != nil {
			return auth.InvitesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		inv := toInvite(dbInv)
		invites = append(invites, inv)
	}

	total, err := dbutil.Total(ctx, ir.db, queryCount, params)
	if err != nil {
		return auth.InvitesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	page := auth.InvitesPage{
		Invites: invites,
		PageMetadata: apiutil.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (ir invitesRepository) FlipInactiveInvites(ctx context.Context, email string, inviteeID string) (uint32, error) {
	query := `
		UPDATE invites
		SET invitee_email = NULL, invitee_id = :invitee_id
		WHERE invitee_email = :email AND expires_at > NOW()
	`

	res, err := ir.db.NamedExecContext(ctx, query, map[string]any{
		"invitee_id": inviteeID,
		"email":      email,
	})

	if err != nil {
		pqErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pqErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return 0, errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}
		return 0, errors.Wrap(errors.ErrUpdateEntity, err)
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(errors.ErrUpdateEntity, err)
	}

	return uint32(cnt), nil
}

// Purges all expired invites that would prevent the `invite` from being saved due to the UNIQUE
// constraints in the 'invites' database table. In other words, removes all expired invites that match
// the passed `invite`'s (invitee_id, inviter_id, org_id) or (invitee_email, inviter_id, org_id) triplet.
func (ir invitesRepository) purgeExpiredInvites(ctx context.Context, invite auth.Invite) error {
	query := `
		DELETE FROM invites
		WHERE (:invitee_id = invitee_id OR :invitee_email = invitee_email) AND inviter_id = :inviter_id AND org_id = :org_id AND expires_at < NOW()
	`

	dbInvite := toDBInvite(invite)

	_, err := ir.db.NamedExecContext(ctx, query, dbInvite)
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

	return nil
}

func toDBInvite(invite auth.Invite) dbInvite {
	return dbInvite{
		ID:           invite.ID,
		InviteeID:    sql.NullString{String: invite.InviteeID, Valid: len(invite.InviteeID) != 0},
		InviteeEmail: sql.NullString{String: invite.InviteeEmail, Valid: len(invite.InviteeEmail) != 0},
		InviterID:    invite.InviterID,
		OrgID:        invite.OrgID,
		InviteeRole:  invite.InviteeRole,
		CreatedAt:    invite.CreatedAt,
		ExpiresAt:    invite.ExpiresAt,
	}
}

func toInvite(dbI dbInvite) auth.Invite {
	inviteeID := ""
	if dbI.InviteeID.Valid {
		inviteeID = dbI.InviteeID.String
	}

	inviteeEmail := ""
	if dbI.InviteeEmail.Valid {
		inviteeEmail = dbI.InviteeEmail.String
	}

	return auth.Invite{
		ID:           dbI.ID,
		InviteeID:    inviteeID,
		InviteeEmail: inviteeEmail,
		InviterID:    dbI.InviterID,
		OrgID:        dbI.OrgID,
		InviteeRole:  dbI.InviteeRole,
		CreatedAt:    dbI.CreatedAt,
		ExpiresAt:    dbI.ExpiresAt,
	}
}

type dbInvite struct {
	ID           string         `db:"id"`
	InviteeID    sql.NullString `db:"invitee_id"`
	InviteeEmail sql.NullString `db:"invitee_email"`
	InviterID    string         `db:"inviter_id"`
	OrgID        string         `db:"org_id"`
	InviteeRole  string         `db:"invitee_role"`
	CreatedAt    time.Time      `db:"created_at"`
	ExpiresAt    time.Time      `db:"expires_at"`
}
