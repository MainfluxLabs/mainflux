package postgres

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/invites"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type groupInviteRepository struct {
	*invites.CommonInviteRepository[things.GroupInvite]
}

func NewGroupInviteRepository(db dbutil.Database) things.GroupInviteRepository {
	return &groupInviteRepository{
		CommonInviteRepository: invites.NewCommonInviteRepository[things.GroupInvite](db),
	}
}

func (ir groupInviteRepository) SaveDormantInviteRelations(ctx context.Context, orgInviteID string, groupInviteIDs ...string) error {
	query := `
		INSERT INTO dormant_group_invites
		(org_invite_id, group_invite_id)	
		VALUES (:org_invite_id, :group_invite_id)
	`

	tx, err := ir.Db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	for _, groupInviteID := range groupInviteIDs {
		args := map[string]any{
			"org_invite_id":   orgInviteID,
			"group_invite_id": groupInviteID,
		}

		if _, err := tx.NamedExecContext(ctx, query, args); err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(dbutil.ErrMalformedEntity, err)
				}
			}

			return errors.Wrap(dbutil.ErrCreateEntity, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return nil
}

func (ir groupInviteRepository) ActivateGroupInvites(ctx context.Context, orgInviteID, userID string, expiresAt time.Time) ([]things.GroupInvite, error) {
	queryUpdate := `
		UPDATE group_invites AS gi
		SET invitee_id = :invitee_id,
		    expires_at = :expires_at			
		FROM dormant_group_invites AS dgi
		WHERE gi.id = dgi.group_invite_id
	      AND dgi.org_invite_id = :org_invite_id
		RETURNING gi.id, gi.invitee_id, gi.inviter_id, gi.group_id, gi.invitee_role,
		          gi.created_at, gi.expires_at, gi.state
	`

	queryDelete := `
		DELETE FROM dormant_group_invites
		WHERE org_invite_id = :org_invite_id	
	`

	params := map[string]any{
		"org_invite_id": orgInviteID,
		"invitee_id":    userID,
		"expires_at":    expiresAt,
	}

	rows, err := ir.Db.NamedQueryContext(ctx, queryUpdate, params)
	if err != nil {
		return nil, errors.Wrap(dbutil.ErrUpdateEntity, err)
	}
	defer rows.Close()

	var invites []things.GroupInvite

	for rows.Next() {
		var inv things.GroupInvite

		if err := rows.StructScan(&inv); err != nil {
			return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		invites = append(invites, inv)
	}

	_, err = ir.Db.NamedExecContext(ctx, queryDelete, params)
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
