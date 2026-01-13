// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

type invitesRepository struct {
	db dbutil.Database
}

func NewOrgInvitesRepo(db dbutil.Database) auth.OrgInvitesRepository {
	return &invitesRepository{
		db: db,
	}
}

func (ir invitesRepository) SaveOrgInvite(ctx context.Context, invites ...auth.OrgInvite) error {
	tx, err := ir.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	qIns := `
		INSERT INTO org_invites (id, invitee_id, inviter_id, org_id, invitee_role, created_at, expires_at, state)	
		VALUES (:id, :invitee_id, :inviter_id, :org_id, :invitee_role, :created_at, :expires_at, :state)
	`

	for _, invite := range invites {
		if err := ir.syncOrgInviteStateByInvite(ctx, invite); err != nil {
			return err
		}

		dbInvite := toDBOrgInvite(invite)
		if _, err := tx.NamedExecContext(ctx, qIns, dbInvite); err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return errors.Wrap(dbutil.ErrConflict, err)
				}
			}

			return errors.Wrap(dbutil.ErrCreateEntity, err)
		}

		if err := ir.saveOrgInviteGroups(ctx, tx, invite); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return nil
}

func (ir invitesRepository) SaveDormantInviteRelation(ctx context.Context, orgInviteID, platformInviteID string) error {
	qIns := `
		INSERT INTO dormant_org_invites (org_invite_id, platform_invite_id)	
		VALUES (:org_invite_id, :platform_invite_id)
	`

	params := map[string]any{
		"org_invite_id":      orgInviteID,
		"platform_invite_id": platformInviteID,
	}

	if _, err := ir.db.NamedExecContext(ctx, qIns, params); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(dbutil.ErrMalformedEntity, err)
			case pgerrcode.UniqueViolation:
				return errors.Wrap(dbutil.ErrConflict, err)
			}
		}

		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return nil
}

func (ir invitesRepository) ActivateOrgInvite(ctx context.Context, platformInviteID, userID string, expiresAt time.Time) ([]auth.OrgInvite, error) {
	queryUpdate := `
		UPDATE org_invites AS oi
		SET invitee_id = :invitee_id,
		    expires_at = :expires_at			
		FROM dormant_org_invites AS doi
		WHERE oi.id = doi.org_invite_id
	      AND doi.platform_invite_id = :platform_invite_id
		RETURNING oi.id, oi.invitee_id, oi.inviter_id, oi.org_id, oi.invitee_role,
		          oi.created_at, oi.expires_at, oi.state
	`

	queryDelete := `
		DELETE FROM dormant_org_invites
		WHERE platform_invite_id = :platform_invite_id	
	`

	params := map[string]any{
		"platform_invite_id": platformInviteID,
		"invitee_id":         userID,
		"expires_at":         expiresAt,
	}

	rows, err := ir.db.NamedQueryContext(ctx, queryUpdate, params)
	if err != nil {
		return nil, errors.Wrap(dbutil.ErrUpdateEntity, err)
	}
	defer rows.Close()

	var invites []auth.OrgInvite

	for rows.Next() {
		dbInv := dbOrgInvite{}

		if err := rows.StructScan(&dbInv); err != nil {
			return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		inv := toOrgInvite(dbInv)
		invites = append(invites, inv)
	}

	_, err = ir.db.NamedExecContext(ctx, queryDelete, params)
	if err != nil {
		pqErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pqErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return nil, errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}
		return nil, errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return invites, nil
}

func (ir invitesRepository) RetrieveOrgInviteByID(ctx context.Context, inviteID string) (auth.OrgInvite, error) {
	if err := ir.syncOrgInviteStateByID(ctx, inviteID); err != nil {
		return auth.OrgInvite{}, err
	}

	q := `
		SELECT invitee_id, inviter_id, org_id, invitee_role, created_at, expires_at, state
		FROM org_invites
		WHERE id = $1
	`

	dbI := dbOrgInvite{ID: inviteID}

	if err := ir.db.QueryRowxContext(ctx, q, inviteID).StructScan(&dbI); err != nil {
		if err == sql.ErrNoRows {
			return auth.OrgInvite{}, errors.Wrap(dbutil.ErrNotFound, err)
		}

		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return auth.OrgInvite{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}

		return auth.OrgInvite{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	invite := toOrgInvite(dbI)

	groups, err := ir.retrieveOrgInviteGroups(ctx, inviteID)
	if err != nil {
		return auth.OrgInvite{}, err
	}

	invite.Groups = groups

	return invite, nil
}

func (ir invitesRepository) RemoveOrgInvite(ctx context.Context, inviteID string) error {
	qDel := `DELETE FROM org_invites WHERE id = :id`
	invite := dbOrgInvite{
		ID: inviteID,
	}

	res, err := ir.db.NamedExecContext(ctx, qDel, invite)
	if err != nil {
		pqErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pqErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	if cnt != 1 {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	if err := ir.removeOrgInviteGroupsByID(ctx, inviteID); err != nil {
		return err
	}

	return nil
}

func (ir invitesRepository) UpdateOrgInviteState(ctx context.Context, inviteID, state string) error {
	query := `
		UPDATE org_invites
		SET state = :state
		WHERE id = :inviteID
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

func (ir invitesRepository) RetrieveOrgInvitesByOrg(ctx context.Context, orgID string, pm auth.PageMetadataInvites) (auth.OrgInvitesPage, error) {
	query := `
		SELECT id, invitee_id, inviter_id, org_id, invitee_role, created_at, expires_at, state
		FROM org_invites %s ORDER BY %s %s %s
	`

	queryCount := `SELECT COUNT(*) FROM org_invites %s`

	filterOrgID := `org_id = :orgID`
	filterState := ``
	if pm.State != "" {
		filterState = "state = :state"
	}

	whereClause := dbutil.BuildWhereClause(filterOrgID, filterState)
	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	query = fmt.Sprintf(query, whereClause, oq, dq, olq)
	queryCount = fmt.Sprintf(queryCount, whereClause)

	params := map[string]any{
		"orgID":  orgID,
		"limit":  pm.Limit,
		"offset": pm.Offset,
		"state":  pm.State,
	}

	rows, err := ir.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return auth.OrgInvitesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var invites []auth.OrgInvite

	for rows.Next() {
		dbInv := dbOrgInvite{}

		if err := rows.StructScan(&dbInv); err != nil {
			return auth.OrgInvitesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		groupIDs, err := ir.retrieveOrgInviteGroups(ctx, dbInv.ID)
		if err != nil {
			return auth.OrgInvitesPage{}, err
		}

		inv := toOrgInvite(dbInv)
		inv.Groups = groupIDs
		invites = append(invites, inv)
	}

	total, err := dbutil.Total(ctx, ir.db, queryCount, params)
	if err != nil {
		return auth.OrgInvitesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	page := auth.OrgInvitesPage{
		Invites: invites,
		Total:   total,
	}

	return page, nil
}

func (ir invitesRepository) RetrieveOrgInvitesByUser(ctx context.Context, userType, userID string, pm auth.PageMetadataInvites) (auth.OrgInvitesPage, error) {
	query := `
		SELECT id, invitee_id, inviter_id, org_id, invitee_role, created_at, expires_at, state
		FROM org_invites %s ORDER BY %s %s %s
	`

	queryCount := `SELECT COUNT(*) FROM org_invites %s`

	filterUserType := `%s = :userID`
	switch userType {
	case auth.UserTypeInvitee:
		filterUserType = fmt.Sprintf(filterUserType, "invitee_id")
	case auth.UserTypeInviter:
		filterUserType = fmt.Sprintf(filterUserType, "inviter_id")
	default:
		return auth.OrgInvitesPage{}, errors.New("invalid invite user type")
	}

	filterState := ``
	if pm.State != "" {
		filterState = "state = :state"
	}

	whereClause := dbutil.BuildWhereClause(filterUserType, filterState)
	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	query = fmt.Sprintf(query, whereClause, oq, dq, olq)
	queryCount = fmt.Sprintf(queryCount, whereClause)

	params := map[string]any{
		"userID": userID,
		"limit":  pm.Limit,
		"offset": pm.Offset,
		"state":  pm.State,
	}

	if err := ir.syncOrgInviteStateByUserID(ctx, userType, userID); err != nil {
		return auth.OrgInvitesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	rows, err := ir.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return auth.OrgInvitesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var invites []auth.OrgInvite

	for rows.Next() {
		dbInv := dbOrgInvite{}

		if err := rows.StructScan(&dbInv); err != nil {
			return auth.OrgInvitesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		groupIDs, err := ir.retrieveOrgInviteGroups(ctx, dbInv.ID)
		if err != nil {
			return auth.OrgInvitesPage{}, err
		}

		inv := toOrgInvite(dbInv)
		inv.Groups = groupIDs

		invites = append(invites, inv)
	}

	total, err := dbutil.Total(ctx, ir.db, queryCount, params)
	if err != nil {
		return auth.OrgInvitesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	page := auth.OrgInvitesPage{
		Invites: invites,
		Total:   total,
	}

	return page, nil
}

// Syncs the state of all invites either sent by or sent to the user denoted by `userID`, depending on the value
// of userType, which may be one of `inviter` or `invitee`. That is, sets state='expired' for all matching invites where
// state='pending' and expires_at < now().
func (ir invitesRepository) syncOrgInviteStateByUserID(ctx context.Context, userType, userID string) error {
	query := `
		UPDATE org_invites
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
				return errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	return nil
}

// Syncs the state of all invites in the database that would prevent the passed `invite`
// from being preserved. That is, sets state='expired' for invites where state='pending' and expires_at < now().
func (ir invitesRepository) syncOrgInviteStateByInvite(ctx context.Context, invite auth.OrgInvite) error {
	query := `
		UPDATE org_invites
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
				return errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	return nil
}

// Syncs the state of the Invite with the passed inviteID. That is, sets state='expired' if state='pending' and expires_at < now().
func (ir invitesRepository) syncOrgInviteStateByID(ctx context.Context, inviteID string) error {
	query := `
		UPDATE org_invites
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

func (ir invitesRepository) saveOrgInviteGroups(ctx context.Context, tx *sqlx.Tx, invite auth.OrgInvite) error {
	qIns := `
		INSERT INTO org_invites_groups (org_invite_id, group_id, member_role)
		VALUES (:org_invite_id, :group_id, :member_role)
	`

	for _, group := range invite.Groups {
		values := map[string]any{
			"org_invite_id": invite.ID,
			"group_id":      group.GroupID,
			"member_role":   group.MemberRole,
		}

		if _, err := tx.NamedExecContext(ctx, qIns, values); err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return errors.Wrap(dbutil.ErrConflict, err)
				}
			}

			return errors.Wrap(dbutil.ErrCreateEntity, err)
		}
	}

	return nil
}

func (ir invitesRepository) retrieveOrgInviteGroups(ctx context.Context, inviteID string) ([]auth.GroupInvite, error) {
	query := `
		SELECT group_id, member_role
		FROM org_invites_groups
		WHERE org_invite_id = :org_invite_id
	`

	groups := []auth.GroupInvite{}

	rows, err := ir.db.NamedQueryContext(ctx, query, map[string]any{"org_invite_id": inviteID})
	if err != nil {
		return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	defer rows.Close()

	for rows.Next() {
		var groupID, memberRole string
		if err := rows.Scan(&groupID, &memberRole); err != nil {
			return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		groups = append(groups, auth.GroupInvite{
			GroupID:    groupID,
			MemberRole: memberRole,
		})
	}

	return groups, nil
}

func (ir invitesRepository) removeOrgInviteGroupsByID(ctx context.Context, inviteID string) error {
	query := `
		DELETE FROM org_invites_groups
		WHERE org_invite_id = :id
	`

	_, err := ir.db.NamedExecContext(ctx, query, map[string]any{"id": inviteID})
	if err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func toDBOrgInvite(invite auth.OrgInvite) dbOrgInvite {
	return dbOrgInvite{
		ID:          invite.ID,
		InviteeID:   sql.NullString{String: invite.InviteeID, Valid: len(invite.InviteeID) > 0},
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
		InviteeID:   dbI.InviteeID.String,
		InviterID:   dbI.InviterID,
		OrgID:       dbI.OrgID,
		InviteeRole: dbI.InviteeRole,
		CreatedAt:   dbI.CreatedAt,
		ExpiresAt:   dbI.ExpiresAt,
		State:       dbI.State,
	}
}

type dbOrgInvite struct {
	ID          string         `db:"id"`
	InviteeID   sql.NullString `db:"invitee_id"`
	InviterID   string         `db:"inviter_id"`
	OrgID       string         `db:"org_id"`
	InviteeRole string         `db:"invitee_role"`
	CreatedAt   time.Time      `db:"created_at"`
	ExpiresAt   time.Time      `db:"expires_at"`
	State       string         `db:"state"`
}
