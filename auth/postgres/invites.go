// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/invites"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type orgInviteRepository struct {
	*invites.CommonInviteRepository[auth.OrgInvite]
}

func NewOrgInviteRepository(db dbutil.Database) auth.OrgInviteRepository {
	return &orgInviteRepository{
		CommonInviteRepository: invites.NewCommonInviteRepository[auth.OrgInvite](db),
	}
}

func (ir orgInviteRepository) SaveDormantInviteRelation(ctx context.Context, orgInviteID, platformInviteID string) error {
	qIns := `
		INSERT INTO dormant_org_invites (org_invite_id, platform_invite_id)	
		VALUES (:org_invite_id, :platform_invite_id)
	`

	params := map[string]any{
		"org_invite_id":      orgInviteID,
		"platform_invite_id": platformInviteID,
	}

	if _, err := ir.Db.NamedExecContext(ctx, qIns, params); err != nil {
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

func (ir orgInviteRepository) ActivateOrgInvite(ctx context.Context, platformInviteID, userID string, expiresAt time.Time) ([]auth.OrgInvite, error) {
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

	rows, err := ir.Db.NamedQueryContext(ctx, queryUpdate, params)
	if err != nil {
		return nil, errors.Wrap(dbutil.ErrUpdateEntity, err)
	}
	defer rows.Close()

	var invites []auth.OrgInvite

	for rows.Next() {
		var inv auth.OrgInvite

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
