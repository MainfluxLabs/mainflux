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

var _ auth.OrgMembershipsRepository = (*orgMembershipsRepository)(nil)

type orgMembershipsRepository struct {
	db dbutil.Database
}

var membershipsIDFkey = "org_memberships_org_id_fkey"

// NewOrgMembershipsRepo instantiates a PostgreSQL implementation of membership repository.
func NewOrgMembershipsRepo(db dbutil.Database) auth.OrgMembershipsRepository {
	return &orgMembershipsRepository{
		db: db,
	}
}

func (omr orgMembershipsRepository) RetrieveByOrg(ctx context.Context, orgID string, pm apiutil.PageMetadata) (auth.OrgMembershipsPage, error) {
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	q := fmt.Sprintf(`SELECT member_id, org_id, created_at, updated_at, role FROM org_memberships 
					  WHERE org_id = :org_id %s`, olq)

	params := map[string]any{
		"org_id": orgID,
		"limit":  pm.Limit,
		"offset": pm.Offset,
	}

	rows, err := omr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return auth.OrgMembershipsPage{}, errors.Wrap(auth.ErrRetrieveMembershipsByOrg, err)
	}
	defer rows.Close()

	var oms []auth.OrgMembership
	for rows.Next() {
		dbm := dbOrgMembership{}
		if err := rows.StructScan(&dbm); err != nil {
			return auth.OrgMembershipsPage{}, errors.Wrap(auth.ErrRetrieveMembershipsByOrg, err)
		}

		oms = append(oms, toOrgMembership(dbm))
	}

	cq := `SELECT COUNT(*) FROM org_memberships WHERE org_id = :org_id;`

	total, err := dbutil.Total(ctx, omr.db, cq, params)
	if err != nil {
		return auth.OrgMembershipsPage{}, errors.Wrap(auth.ErrRetrieveMembershipsByOrg, err)
	}

	page := auth.OrgMembershipsPage{
		OrgMemberships: oms,
		Total:          total,
	}

	return page, nil
}

func (omr orgMembershipsRepository) RetrieveRole(ctx context.Context, memberID, orgID string) (string, error) {
	q := `SELECT role FROM org_memberships WHERE member_id = $1 AND org_id = $2`

	membership := auth.OrgMembership{}
	if err := omr.db.QueryRowxContext(ctx, q, memberID, orgID).StructScan(&membership); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return "", errors.Wrap(dbutil.ErrNotFound, err)
		}

		return "", errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return membership.Role, nil
}

func (omr orgMembershipsRepository) Save(ctx context.Context, oms ...auth.OrgMembership) error {
	tx, err := omr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(auth.ErrCreateOrgMembership, err)
	}
	defer tx.Rollback()

	qIns := `INSERT INTO org_memberships (org_id, member_id, role, created_at, updated_at)
			 VALUES(:org_id, :member_id, :role, :created_at, :updated_at)`

	for _, om := range oms {
		dbom := toDBOrgMembership(om)

		if _, err := tx.NamedExecContext(ctx, qIns, dbom); err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.ForeignKeyViolation:
					return errors.Wrap(dbutil.ErrConflict, errors.New(pgErr.Detail))
				case pgerrcode.UniqueViolation:
					return errors.Wrap(auth.ErrOrgMembershipExists, errors.New(pgErr.Detail))
				}
			}

			return errors.Wrap(auth.ErrCreateOrgMembership, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(auth.ErrCreateOrgMembership, err)
	}

	return nil
}

func (omr orgMembershipsRepository) Remove(ctx context.Context, orgID string, ids ...string) error {
	tx, err := omr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(auth.ErrRemoveOrgMembership, err)
	}

	qDel := `DELETE from org_memberships WHERE org_id = :org_id AND member_id = :member_id`

	for _, id := range ids {
		om := auth.OrgMembership{
			OrgID:    orgID,
			MemberID: id,
		}
		dbom := toDBOrgMembership(om)

		if _, err := tx.NamedExecContext(ctx, qDel, dbom); err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return errors.Wrap(dbutil.ErrConflict, err)
				}
			}

			return errors.Wrap(auth.ErrRemoveOrgMembership, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(auth.ErrRemoveOrgMembership, err)
	}

	return nil
}

func (omr orgMembershipsRepository) Update(ctx context.Context, oms ...auth.OrgMembership) error {
	qUpd := `UPDATE org_memberships SET role = :role, updated_at = :updated_at
			 WHERE org_id = :org_id AND member_id = :member_id`

	for _, om := range oms {
		dbm := toDBOrgMembership(om)

		row, err := omr.db.NamedExecContext(ctx, qUpd, dbm)
		if err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.StringDataRightTruncationDataException:
					return errors.Wrap(dbutil.ErrMalformedEntity, err)
				}
			}

			return errors.Wrap(dbutil.ErrUpdateEntity, err)
		}

		cnt, errdb := row.RowsAffected()
		if errdb != nil {
			return errors.Wrap(dbutil.ErrUpdateEntity, errdb)
		}

		if cnt != 1 {
			return errors.Wrap(dbutil.ErrNotFound, err)
		}
	}

	return nil
}

func (omr orgMembershipsRepository) BackupAll(ctx context.Context) ([]auth.OrgMembership, error) {
	q := `SELECT org_id, member_id, role, created_at, updated_at FROM org_memberships;`

	rows, err := omr.db.NamedQueryContext(ctx, q, map[string]any{})
	if err != nil {
		return []auth.OrgMembership{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var oms []auth.OrgMembership
	for rows.Next() {
		dbom := dbOrgMembership{}
		if err := rows.StructScan(&dbom); err != nil {
			return []auth.OrgMembership{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		oms = append(oms, toOrgMembership(dbom))
	}

	return oms, nil
}

func (omr orgMembershipsRepository) BackupByOrg(ctx context.Context, orgID string) ([]auth.OrgMembership, error) {
	q := `SELECT org_id, member_id, role, created_at, updated_at FROM org_memberships WHERE org_id = :org_id;`

	rows, err := omr.db.NamedQueryContext(ctx, q, map[string]any{
		"org_id": orgID,
	})
	if err != nil {
		return []auth.OrgMembership{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var oms []auth.OrgMembership
	for rows.Next() {
		dbom := dbOrgMembership{}
		if err := rows.StructScan(&dbom); err != nil {
			return []auth.OrgMembership{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		oms = append(oms, toOrgMembership(dbom))
	}

	return oms, nil
}

type dbOrgMembership struct {
	MemberID  string    `db:"member_id"`
	OrgID     string    `db:"org_id"`
	Role      string    `db:"role"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func toDBOrgMembership(om auth.OrgMembership) dbOrgMembership {
	return dbOrgMembership{
		OrgID:     om.OrgID,
		MemberID:  om.MemberID,
		Role:      om.Role,
		CreatedAt: om.CreatedAt,
		UpdatedAt: om.UpdatedAt,
	}
}

func toOrgMembership(dbom dbOrgMembership) auth.OrgMembership {
	return auth.OrgMembership{
		OrgID:     dbom.OrgID,
		MemberID:  dbom.MemberID,
		Role:      dbom.Role,
		CreatedAt: dbom.CreatedAt,
		UpdatedAt: dbom.UpdatedAt,
	}
}
