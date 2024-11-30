// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ auth.OrgRepository = (*orgRepository)(nil)

type orgRepository struct {
	db Database
}

// NewOrgRepo instantiates a PostgreSQL implementation of org
// repository.
func NewOrgRepo(db Database) auth.OrgRepository {
	return &orgRepository{
		db: db,
	}
}

func (or orgRepository) Save(ctx context.Context, orgs ...auth.Org) error {
	// For root org path is initialized with id
	q := `INSERT INTO orgs (name, description, id, owner_id, metadata, created_at, updated_at)
		  VALUES (:name, :description, :id, :owner_id, :metadata, :created_at, :updated_at)`

	for _, org := range orgs {
		dbo, err := toDBOrg(org)
		if err != nil {
			return err
		}

		_, err = or.db.NamedExecContext(ctx, q, dbo)
		if err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.ForeignKeyViolation:
					return errors.Wrap(errors.ErrCreateEntity, err)
				case pgerrcode.UniqueViolation:
					return errors.Wrap(errors.ErrConflict, err)
				case pgerrcode.StringDataRightTruncationDataException:
					return errors.Wrap(errors.ErrMalformedEntity, err)
				}
			}

			return errors.Wrap(errors.ErrCreateEntity, err)
		}
	}

	return nil
}

func (or orgRepository) Update(ctx context.Context, org auth.Org) error {
	q := `UPDATE orgs SET name = :name, description = :description, metadata = :metadata, updated_at = :updated_at WHERE id = :id
		  RETURNING id, name, owner_id, description, metadata, created_at, updated_at`

	dbo, err := toDBOrg(org)
	if err != nil {
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	row, err := or.db.NamedQueryContext(ctx, q, dbo)
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(errors.ErrMalformedEntity, err)
			case pgerrcode.UniqueViolation:
				return errors.Wrap(errors.ErrConflict, err)
			case pgerrcode.StringDataRightTruncationDataException:
				return errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}
	defer row.Close()

	return nil
}

func (or orgRepository) Remove(ctx context.Context, owner, orgID string) error {
	qd := `DELETE FROM orgs WHERE id = :id AND owner_id = :owner_id;`
	org := auth.Org{
		ID:      orgID,
		OwnerID: owner,
	}
	dbo, err := toDBOrg(org)
	if err != nil {
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	res, err := or.db.NamedExecContext(ctx, qd, dbo)
	if err != nil {
		pqErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pqErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(errors.ErrMalformedEntity, err)
			case pgerrcode.ForeignKeyViolation:
				switch pqErr.ConstraintName {
				case membersIDFkey:
					return errors.Wrap(auth.ErrOrgNotEmpty, err)
				}
				return errors.Wrap(errors.ErrConflict, err)
			}
		}
		return errors.Wrap(errors.ErrUpdateEntity, err)
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

func (or orgRepository) RetrieveByID(ctx context.Context, id string) (auth.Org, error) {
	dbo := dbOrg{
		ID: id,
	}
	q := `SELECT id, name, owner_id, description, metadata, created_at, updated_at FROM orgs WHERE id = $1`
	if err := or.db.QueryRowxContext(ctx, q, id).StructScan(&dbo); err != nil {
		if err == sql.ErrNoRows {
			return auth.Org{}, errors.Wrap(errors.ErrNotFound, err)

		}
		return auth.Org{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	return toOrg(dbo)
}

func (or orgRepository) RetrieveByOwner(ctx context.Context, ownerID string, pm auth.PageMetadata) (auth.OrgsPage, error) {
	if ownerID == "" {
		return auth.OrgsPage{}, errors.ErrRetrieveEntity
	}

	return or.retrieve(ctx, ownerID, pm)
}

func (or orgRepository) RetrieveByAdmin(ctx context.Context, pm auth.PageMetadata) (auth.OrgsPage, error) {
	return or.retrieve(ctx, "", pm)
}

func (or orgRepository) RetrieveAll(ctx context.Context) ([]auth.Org, error) {
	orPage, err := or.retrieve(ctx, "", auth.PageMetadata{})
	if err != nil {
		return nil, err
	}

	return orPage.Orgs, nil
}

func (or orgRepository) RetrieveByMemberID(ctx context.Context, memberID string, pm auth.PageMetadata) (auth.OrgsPage, error) {
	meta, mq, err := dbutil.GetMetadataQuery("o", pm.Metadata)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(auth.ErrRetrieveOrgsByMember, err)
	}

	nq, name := dbutil.GetNameQuery(pm.Name)

	if mq != "" {
		mq = fmt.Sprintf("AND %s", mq)
	}

	if nq != "" {
		nq = fmt.Sprintf("AND %s", nq)
	}

	q := fmt.Sprintf(`SELECT o.id, o.owner_id, o.name, o.description, o.metadata, o.created_at, o.updated_at
		FROM member_relations mr, orgs o
		WHERE mr.org_id = o.id and mr.member_id = :member_id
		%s %s ORDER BY id LIMIT :limit OFFSET :offset;`, mq, nq)

	params := map[string]interface{}{
		"member_id": memberID,
		"name":      name,
		"limit":     pm.Limit,
		"offset":    pm.Offset,
		"metadata":  meta,
	}

	rows, err := or.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(auth.ErrRetrieveOrgsByMember, err)
	}
	defer rows.Close()

	var items []auth.Org
	for rows.Next() {
		dbg := dbOrg{}
		if err := rows.StructScan(&dbg); err != nil {
			return auth.OrgsPage{}, errors.Wrap(auth.ErrRetrieveOrgsByMember, err)
		}
		og, err := toOrg(dbg)
		if err != nil {
			return auth.OrgsPage{}, err
		}
		items = append(items, og)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM member_relations ore, orgs o
		WHERE ore.org_id = o.id and ore.member_id = :member_id %s %s`, mq, nq)

	total, err := total(ctx, or.db, cq, params)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(auth.ErrRetrieveOrgsByMember, err)
	}

	page := auth.OrgsPage{
		Orgs: items,
		PageMetadata: auth.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (or orgRepository) retrieve(ctx context.Context, ownerID string, pm auth.PageMetadata) (auth.OrgsPage, error) {
	ownq := dbutil.GetOwnerQuery(ownerID)
	nq, name := dbutil.GetNameQuery(pm.Name)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	meta, mq, err := dbutil.GetMetadataQuery("orgs", pm.Metadata)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	var query []string
	if ownq != "" {
		query = append(query, ownq)
	}

	if mq != "" {
		query = append(query, mq)
	}

	if nq != "" {
		query = append(query, nq)
	}

	var whereClause string
	if len(query) > 0 {
		whereClause = fmt.Sprintf(" WHERE %s", strings.Join(query, " AND "))
	}

	q := fmt.Sprintf(`SELECT id, owner_id, name, description, metadata, created_at, updated_at FROM orgs %s %s;`, whereClause, olq)

	params := map[string]interface{}{
		"owner_id": ownerID,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
		"name":     name,
		"metadata": meta,
	}

	rows, err := or.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []auth.Org
	for rows.Next() {
		dbor := dbOrg{}
		if err := rows.StructScan(&dbor); err != nil {
			return auth.OrgsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		or, err := toOrg(dbor)
		if err != nil {
			return auth.OrgsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		items = append(items, or)
	}

	cq := fmt.Sprintf("SELECT COUNT(*) FROM orgs %s;", whereClause)

	total, err := total(ctx, or.db, cq, params)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	page := auth.OrgsPage{
		Orgs: items,
		PageMetadata: auth.PageMetadata{
			Total:  total,
			Limit:  pm.Limit,
			Offset: pm.Offset,
		},
	}

	return page, nil
}

type dbOrg struct {
	ID          string        `db:"id"`
	OwnerID     string        `db:"owner_id"`
	Name        string        `db:"name"`
	Description string        `db:"description"`
	Metadata    dbOrgMetadata `db:"metadata"`
	CreatedAt   time.Time     `db:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at"`
}

func toDBOrg(o auth.Org) (dbOrg, error) {
	return dbOrg{
		ID:          o.ID,
		Name:        o.Name,
		OwnerID:     o.OwnerID,
		Description: o.Description,
		Metadata:    dbOrgMetadata(o.Metadata),
		CreatedAt:   o.CreatedAt,
		UpdatedAt:   o.UpdatedAt,
	}, nil
}

func toOrg(dbo dbOrg) (auth.Org, error) {
	return auth.Org{
		ID:          dbo.ID,
		Name:        dbo.Name,
		OwnerID:     dbo.OwnerID,
		Description: dbo.Description,
		Metadata:    auth.OrgMetadata(dbo.Metadata),
		UpdatedAt:   dbo.UpdatedAt,
		CreatedAt:   dbo.CreatedAt,
	}, nil
}

func total(ctx context.Context, db Database, query string, params interface{}) (uint64, error) {
	rows, err := db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	total := uint64(0)
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, err
		}
	}
	return total, nil
}

// dbOrgMetadata type for handling metadata properly in database/sql
type dbOrgMetadata map[string]interface{}

// Scan - Implement the database/sql scanner interface
func (m *dbOrgMetadata) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return errors.ErrScanMetadata
	}

	if err := json.Unmarshal(b, m); err != nil {
		return err
	}

	return nil
}

// Value Implements valuer
func (m dbOrgMetadata) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}

	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return b, err
}
