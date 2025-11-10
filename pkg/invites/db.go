package invites

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type DbInvite struct {
	ID            string         `db:"id"`
	DestinationID string         `db:"destination_id"`
	InviteeID     sql.NullString `db:"invitee_id"`
	InviterID     string         `db:"inviter_id"`
	InviteeRole   string         `db:"invitee_role"`
	CreatedAt     time.Time      `db:"created_at"`
	ExpiresAt     time.Time      `db:"expires_at"`
	State         string         `db:"state"`
}

type InviteRepository[T Invitable] interface {
	SaveInvites(ctx context.Context, invites ...T) error
	RetrieveInviteByID(ctx context.Context, inviteID string) (T, error)
	RemoveInvite(ctx context.Context, inviteID string) error
	UpdateInviteState(ctx context.Context, inviteID, state string) error
	RetrieveInvitesByDestination(ctx context.Context, destinationID string, pm PageMetadataInvites) (InvitesPage[T], error)
	RetrieveInvitesByUser(ctx context.Context, userType, userID string, pm PageMetadataInvites) (InvitesPage[T], error)
}

type CommonInviteRepository[T Invitable] struct {
	db dbutil.Database
}

func NewCommonInviteRepository[T Invitable](db dbutil.Database) *CommonInviteRepository[T] {
	return &CommonInviteRepository[T]{
		db: db,
	}
}

func (ir CommonInviteRepository[T]) SaveInvites(ctx context.Context, invites ...T) error {
	tx, err := ir.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	qIns := `
		INSERT INTO %s (id, invitee_id, inviter_id, %s, invitee_role, created_at, expires_at, state)	
		VALUES (:id, :invitee_id, :inviter_id, :destination_id, :invitee_role, :created_at, :expires_at, :state)
	`

	qIns = fmt.Sprintf(qIns, invites[0].TableName(), invites[0].ColumnDestinationID())

	for _, invite := range invites {
		if err := ir.syncInviteStateByInvite(ctx, invite); err != nil {
			return err
		}

		dbInvite := invite.ToDBInvite()
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
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return nil
}

func (ir CommonInviteRepository[T]) RetrieveInviteByID(ctx context.Context, inviteID string) (T, error) {
	if err := ir.syncInviteStateByID(ctx, inviteID); err != nil {
		return *new(T), err
	}

	var invite T

	query := `
		SELECT invitee_id, inviter_id, %s, invitee_role, created_at, expires_at, state
		FROM %s
		WHERE id = $1
	`

	query = fmt.Sprintf(query, invite.ColumnDestinationID(), invite.TableName())

	if err := ir.db.QueryRowxContext(ctx, query, inviteID).StructScan(&invite); err != nil {
		if err == sql.ErrNoRows {
			return *new(T), errors.Wrap(dbutil.ErrNotFound, err)
		}

		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return *new(T), errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}

		return *new(T), errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return invite, nil
}

func (ir CommonInviteRepository[T]) RemoveInvite(ctx context.Context, inviteID string) error {
	query := `DELETE FROM %s WHERE id = :id`

	var invite T
	query = fmt.Sprintf(query, invite.TableName())

	res, err := ir.db.NamedExecContext(ctx, query, map[string]any{
		"id": inviteID,
	})

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

	return nil
}

func (ir CommonInviteRepository[T]) UpdateInviteState(ctx context.Context, inviteID, state string) error {
	query := `
		UPDATE %s
		SET state = :state
		WHERE id = :inviteID
	`

	var invite T
	query = fmt.Sprintf(query, invite.TableName())

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

func (ir CommonInviteRepository[T]) RetrieveInvitesByDestination(ctx context.Context, destinationID string, pm PageMetadataInvites) (InvitesPage[T], error) {
	var i T

	query := `
		SELECT id, invitee_id, inviter_id, %s, invitee_role, created_at, expires_at, state
		FROM %s %s ORDER BY %s %s %s
	`

	queryCount := `SELECT COUNT(*) FROM %s %s`

	filterDestinationID := fmt.Sprintf(`%s = :destination_id`, i.ColumnDestinationID())
	filterState := ``
	if pm.State != "" {
		filterState = "state = :state"
	}

	whereClause := dbutil.BuildWhereClause(filterDestinationID, filterState)
	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	query = fmt.Sprintf(query, i.ColumnDestinationID(), i.TableName(), whereClause, oq, dq, olq)
	queryCount = fmt.Sprintf(queryCount, i.TableName(), whereClause)

	params := map[string]any{
		"destination_id": destinationID,
		"limit":          pm.Limit,
		"offset":         pm.Offset,
		"state":          pm.State,
	}

	rows, err := ir.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return *new(InvitesPage[T]), errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var invites []T

	for rows.Next() {
		var invite T

		if err := rows.StructScan(&invite); err != nil {
			return *new(InvitesPage[T]), errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		invites = append(invites, invite)
	}

	total, err := dbutil.Total(ctx, ir.db, queryCount, params)
	if err != nil {
		return *new(InvitesPage[T]), errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	page := InvitesPage[T]{
		Invites: invites,
		Total:   total,
	}

	return page, nil
}

func (ir CommonInviteRepository[T]) RetrieveInvitesByUser(ctx context.Context, userType, userID string, pm PageMetadataInvites) (InvitesPage[T], error) {
	var i T

	query := `
		SELECT id, invitee_id, inviter_id, %s, invitee_role, created_at, expires_at, state
		FROM %s %s ORDER BY %s %s %s
	`

	queryCount := `SELECT COUNT(*) FROM %s %s`

	filterUserType := `%s = :userID`
	switch userType {
	case UserTypeInvitee:
		filterUserType = fmt.Sprintf(filterUserType, "invitee_id")
	case UserTypeInviter:
		filterUserType = fmt.Sprintf(filterUserType, "inviter_id")
	default:
		return *new(InvitesPage[T]), errors.New("invalid invite user type")
	}

	filterState := ``
	if pm.State != "" {
		filterState = "state = :state"
	}

	whereClause := dbutil.BuildWhereClause(filterUserType, filterState)
	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	query = fmt.Sprintf(query, i.ColumnDestinationID(), i.TableName(), whereClause, oq, dq, olq)
	queryCount = fmt.Sprintf(queryCount, i.TableName(), whereClause)

	params := map[string]any{
		"userID": userID,
		"limit":  pm.Limit,
		"offset": pm.Offset,
		"state":  pm.State,
	}

	if err := ir.syncInviteStateByUserID(ctx, userType, userID); err != nil {
		return *new(InvitesPage[T]), errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	rows, err := ir.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return *new(InvitesPage[T]), errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var invites []T

	for rows.Next() {
		var invite T

		if err := rows.StructScan(&invite); err != nil {
			return *new(InvitesPage[T]), errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		invites = append(invites, invite)
	}

	total, err := dbutil.Total(ctx, ir.db, queryCount, params)
	if err != nil {
		return *new(InvitesPage[T]), errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	page := InvitesPage[T]{
		Invites: invites,
		Total:   total,
	}

	return page, nil
}

// Syncs the state of all invites either sent by or sent to the user denoted by `userID`, depending on the value
// of userType, which may be one of `inviter` or `invitee`. That is, sets state='expired' for all matching invites where
// state='pending' and expires_at < now().
func (ir CommonInviteRepository[T]) syncInviteStateByUserID(ctx context.Context, userType, userID string) error {
	var invite T

	query := `
		UPDATE %s
		SET state='expired'
		WHERE %s=:userID AND state='pending' AND expires_at < NOW()
	`

	var col string
	switch userType {
	case UserTypeInvitee:
		col = "invitee_id"
	case UserTypeInviter:
		col = "inviter_id"
	default:
		return errors.New("invalid invite user type")
	}

	query = fmt.Sprintf(query, invite.TableName(), col)

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
func (ir CommonInviteRepository[T]) syncInviteStateByInvite(ctx context.Context, invite T) error {
	query := `
		UPDATE %s
		SET state='expired'
		WHERE invitee_id=:invitee_id AND %s=:destination_id AND inviter_id=:inviter_id AND state='pending' AND expires_at < NOW()
	`

	query = fmt.Sprintf(query, invite.TableName(), invite.ColumnDestinationID())

	common := invite.GetCommon()

	_, err := ir.db.NamedExecContext(ctx, query, map[string]any{
		"invitee_id":     common.InviteeID,
		"inviter_id":     common.InviterID,
		"destination_id": invite.GetDestinationID(),
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

// Syncs the state of the Invite with the passed inviteID. That is, sets state='expired' if state='pending' and expires_at < now().
func (ir CommonInviteRepository[T]) syncInviteStateByID(ctx context.Context, inviteID string) error {
	query := `
		UPDATE %s
		SET state='expired'
		WHERE id=:inviteID AND state='pending' AND expires_at < NOW()
	`

	var invite T

	query = fmt.Sprintf(query, invite.TableName())

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
