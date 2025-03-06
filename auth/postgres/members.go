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

var _ auth.MembersRepository = (*membersRepository)(nil)

type membersRepository struct {
	db dbutil.Database
}

var membersIDFkey = "member_relations_org_id_fkey"

// NewMembersRepo instantiates a PostgreSQL implementation of members repository.
func NewMembersRepo(db dbutil.Database) auth.MembersRepository {
	return &membersRepository{
		db: db,
	}
}

func (or membersRepository) RetrieveByOrgID(ctx context.Context, orgID string, pm apiutil.PageMetadata) (auth.OrgMembersPage, error) {
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	q := fmt.Sprintf(`SELECT member_id, org_id, created_at, updated_at, role FROM member_relations 
					  WHERE org_id = :org_id %s`, olq)

	params := map[string]interface{}{
		"org_id": orgID,
		"limit":  pm.Limit,
		"offset": pm.Offset,
	}

	rows, err := or.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return auth.OrgMembersPage{}, errors.Wrap(auth.ErrRetrieveMembersByOrg, err)
	}
	defer rows.Close()

	var oms []auth.OrgMember
	for rows.Next() {
		dbm := dbMember{}
		if err := rows.StructScan(&dbm); err != nil {
			return auth.OrgMembersPage{}, errors.Wrap(auth.ErrRetrieveMembersByOrg, err)
		}

		om, err := toMember(dbm)
		if err != nil {
			return auth.OrgMembersPage{}, err
		}

		oms = append(oms, om)
	}

	cq := `SELECT COUNT(*) FROM member_relations WHERE org_id = :org_id;`

	total, err := dbutil.Total(ctx, or.db, cq, params)
	if err != nil {
		return auth.OrgMembersPage{}, errors.Wrap(auth.ErrRetrieveMembersByOrg, err)
	}

	page := auth.OrgMembersPage{
		OrgMembers: oms,
		PageMetadata: apiutil.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func toMember(dbmb dbMember) (auth.OrgMember, error) {
	return auth.OrgMember{
		MemberID: dbmb.MemberID,
		Role:     dbmb.Role,
	}, nil
}

func (or membersRepository) RetrieveRole(ctx context.Context, memberID, orgID string) (string, error) {
	q := `SELECT role FROM member_relations WHERE member_id = $1 AND org_id = $2`

	member := auth.OrgMember{}
	if err := or.db.QueryRowxContext(ctx, q, memberID, orgID).StructScan(&member); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return "", errors.Wrap(errors.ErrNotFound, err)
		}

		return "", errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	return member.Role, nil
}

func (or membersRepository) Save(ctx context.Context, oms ...auth.OrgMember) error {
	tx, err := or.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(auth.ErrAssignMember, err)
	}

	qIns := `INSERT INTO member_relations (org_id, member_id, role, created_at, updated_at)
			 VALUES(:org_id, :member_id, :role, :created_at, :updated_at)`

	for _, om := range oms {
		dbom := toDBOrgMember(om)

		if _, err := tx.NamedExecContext(ctx, qIns, dbom); err != nil {
			tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.ForeignKeyViolation:
					return errors.Wrap(errors.ErrConflict, errors.New(pgErr.Detail))
				case pgerrcode.UniqueViolation:
					return errors.Wrap(auth.ErrOrgMemberAlreadyAssigned, errors.New(pgErr.Detail))
				}
			}

			return errors.Wrap(auth.ErrAssignMember, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(auth.ErrAssignMember, err)
	}

	return nil
}

func (or membersRepository) Remove(ctx context.Context, orgID string, ids ...string) error {
	tx, err := or.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(auth.ErrUnassignMember, err)
	}

	qDel := `DELETE from member_relations WHERE org_id = :org_id AND member_id = :member_id`

	for _, id := range ids {
		om := auth.OrgMember{
			OrgID:    orgID,
			MemberID: id,
		}
		dbom := toDBOrgMember(om)

		if _, err := tx.NamedExecContext(ctx, qDel, dbom); err != nil {
			tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return errors.Wrap(errors.ErrConflict, err)
				}
			}

			return errors.Wrap(auth.ErrUnassignMember, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(auth.ErrUnassignMember, err)
	}

	return nil
}

func (or membersRepository) Update(ctx context.Context, oms ...auth.OrgMember) error {
	qUpd := `UPDATE member_relations SET role = :role, updated_at = :updated_at
			 WHERE org_id = :org_id AND member_id = :member_id`

	for _, om := range oms {
		dbom := toDBOrgMember(om)

		row, err := or.db.NamedExecContext(ctx, qUpd, dbom)
		if err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.StringDataRightTruncationDataException:
					return errors.Wrap(errors.ErrMalformedEntity, err)
				}
			}

			return errors.Wrap(errors.ErrUpdateEntity, err)
		}

		cnt, errdb := row.RowsAffected()
		if errdb != nil {
			return errors.Wrap(errors.ErrUpdateEntity, errdb)
		}

		if cnt != 1 {
			return errors.Wrap(errors.ErrNotFound, err)
		}
	}

	return nil
}

func (or membersRepository) RetrieveAll(ctx context.Context) ([]auth.OrgMember, error) {
	q := `SELECT org_id, member_id, role, created_at, updated_at FROM member_relations;`

	rows, err := or.db.NamedQueryContext(ctx, q, map[string]interface{}{})
	if err != nil {
		return []auth.OrgMember{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var oms []auth.OrgMember
	for rows.Next() {
		dbom := dbOrgMember{}
		if err := rows.StructScan(&dbom); err != nil {
			return []auth.OrgMember{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		oms = append(oms, toMemberRelation(dbom))
	}

	return oms, nil
}

type dbMember struct {
	MemberID  string    `db:"member_id"`
	OrgID     string    `db:"org_id"`
	Role      string    `db:"role"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type dbOrgMember struct {
	OrgID     string    `db:"org_id"`
	MemberID  string    `db:"member_id"`
	Role      string    `db:"role"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func toDBOrgMember(om auth.OrgMember) dbOrgMember {
	return dbOrgMember{
		OrgID:     om.OrgID,
		MemberID:  om.MemberID,
		Role:      om.Role,
		CreatedAt: om.CreatedAt,
		UpdatedAt: om.UpdatedAt,
	}
}

func toMemberRelation(dbom dbOrgMember) auth.OrgMember {
	return auth.OrgMember{
		OrgID:     dbom.OrgID,
		MemberID:  dbom.MemberID,
		Role:      dbom.Role,
		CreatedAt: dbom.CreatedAt,
		UpdatedAt: dbom.UpdatedAt,
	}
}
