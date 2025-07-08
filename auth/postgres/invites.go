// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
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
		INSERT INTO invites (id, invitee_id, inviter_id, org_id, invitee_role, created_at, expires_at)	
		VALUES (:id, :invitee_id, :inviter_id, :org_id, :invitee_role, :created_at, :expires_at)
	`

	for _, invite := range invites {
		dbInvite := toDBInvite(invite)
		log.Printf("dbInvite: %+v\n", dbInvite)
		if _, err := tx.NamedExecContext(ctx, qIns, dbInvite); err != nil {
			tx.Rollback()

			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return errors.Wrap(auth.ErrUserAlreadyInvited, errors.New(pgErr.Detail))
				}
			}

			return errors.Wrap(auth.ErrCreateInvite, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(auth.ErrAssignMember, err)
	}

	return nil
}

func (ir invitesRepository) RetrieveByID(ctx context.Context, inviteID string) (auth.Invite, error) {
	q := `
		SELECT invitee_id, inviter_id, org_id, invitee_role, created_at, expires_at
		FROM invites
		WHERE id = $1
	`

	dbI := dbInvite{ID: inviteID}

	if err := ir.db.QueryRowxContext(ctx, q, inviteID).StructScan(&dbI); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if err == sql.ErrNoRows || ok && pgErr.Code == pgerrcode.InvalidTextRepresentation {
			return auth.Invite{}, errors.Wrap(errors.ErrNotFound, err)
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

func toDBInvite(invite auth.Invite) dbInvite {
	return dbInvite{
		ID:          invite.ID,
		InviteeID:   invite.InviteeID,
		InviterID:   invite.InviterID,
		OrgID:       invite.OrgID,
		InviteeRole: invite.InviteeRole,
		CreatedAt:   invite.CreatedAt,
		ExpiresAt:   invite.ExpiresAt,
	}
}

func toInvite(dbI dbInvite) auth.Invite {
	return auth.Invite{
		ID:          dbI.ID,
		InviteeID:   dbI.InviteeID,
		InviterID:   dbI.InviterID,
		OrgID:       dbI.OrgID,
		InviteeRole: dbI.InviteeRole,
		CreatedAt:   dbI.CreatedAt,
		ExpiresAt:   dbI.ExpiresAt,
	}
}

type dbInvite struct {
	ID          string    `db:"id"`
	InviteeID   string    `db:"invitee_id"`
	InviterID   string    `db:"inviter_id"`
	OrgID       string    `db:"org_id"`
	InviteeRole string    `db:"invitee_role"`
	CreatedAt   time.Time `db:"created_at"`
	ExpiresAt   time.Time `db:"expires_at"`
}
