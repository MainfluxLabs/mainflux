// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ auth.OrgRepository = (*orgRepository)(nil)

type orgRepository struct {
	db dbutil.Database
}

// NewOrgRepo instantiates a PostgreSQL implementation of org
// repository.
func NewOrgRepo(db dbutil.Database) auth.OrgRepository {
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
					return errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.ForeignKeyViolation:
					return errors.Wrap(dbutil.ErrCreateEntity, err)
				case pgerrcode.UniqueViolation:
					return errors.Wrap(dbutil.ErrConflict, err)
				case pgerrcode.StringDataRightTruncationDataException:
					return errors.Wrap(dbutil.ErrMalformedEntity, err)
				}
			}

			return errors.Wrap(dbutil.ErrCreateEntity, err)
		}
	}

	return nil
}

func (or orgRepository) Update(ctx context.Context, org auth.Org) error {
	q := `UPDATE orgs SET name = :name, description = :description, metadata = :metadata, updated_at = :updated_at WHERE id = :id
		  RETURNING id, name, owner_id, description, metadata, created_at, updated_at`

	dbo, err := toDBOrg(org)
	if err != nil {
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	row, err := or.db.NamedQueryContext(ctx, q, dbo)
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(dbutil.ErrMalformedEntity, err)
			case pgerrcode.UniqueViolation:
				return errors.Wrap(dbutil.ErrConflict, err)
			case pgerrcode.StringDataRightTruncationDataException:
				return errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}
	defer row.Close()

	return nil
}

func (or orgRepository) Remove(ctx context.Context, ownerID string, orgIDs ...string) error {
	qd := `DELETE FROM orgs WHERE id = :id AND owner_id = :owner_id;`
	for _, orgID := range orgIDs {
		org := auth.Org{
			ID:      orgID,
			OwnerID: ownerID,
		}
		dbo, err := toDBOrg(org)
		if err != nil {
			return errors.Wrap(dbutil.ErrUpdateEntity, err)
		}

		res, err := or.db.NamedExecContext(ctx, qd, dbo)
		if err != nil {
			pqErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pqErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.ForeignKeyViolation:
					switch pqErr.ConstraintName {
					case membershipsIDFkey:
						return errors.Wrap(auth.ErrOrgNotEmpty, err)
					}
					return errors.Wrap(dbutil.ErrConflict, err)
				}
			}
			return errors.Wrap(dbutil.ErrUpdateEntity, err)
		}

		cnt, err := res.RowsAffected()
		if err != nil {
			return errors.Wrap(dbutil.ErrRemoveEntity, err)
		}

		if cnt != 1 {
			return errors.Wrap(dbutil.ErrRemoveEntity, err)
		}
	}
	return nil
}

func (or orgRepository) RetrieveByID(ctx context.Context, id string) (auth.Org, error) {
	dbo := dbOrg{
		ID: id,
	}
	q := `SELECT id, name, owner_id, description, metadata, created_at, updated_at FROM orgs WHERE id = $1`
	if err := or.db.QueryRowxContext(ctx, q, id).StructScan(&dbo); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		//  If there is no result or ID is in an invalid format, return ErrNotFound.
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return auth.Org{}, errors.Wrap(dbutil.ErrNotFound, err)
		}
		return auth.Org{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	return toOrg(dbo)
}

func (or orgRepository) RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (auth.OrgsPage, error) {
	oq := dbutil.GetOrderQuery(pm.Order, auth.AllowedOrders)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	nq, name := dbutil.GetNameQuery(pm.Name)
	m, mq, err := dbutil.GetMetadataQuery(pm.Metadata)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	whereClause := dbutil.BuildWhereClause(nq, mq)
	query := fmt.Sprintf(`SELECT id, owner_id, name, description, metadata, created_at, updated_at FROM orgs %s ORDER BY %s %s %s;`, whereClause, oq, dq, olq)
	cquery := fmt.Sprintf(`SELECT COUNT(*) FROM orgs %s`, whereClause)

	params := map[string]any{
		"name":     name,
		"metadata": m,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	return or.retrieve(ctx, query, cquery, params)
}

func (or orgRepository) BackupAll(ctx context.Context) ([]auth.Org, error) {
	query := "SELECT id, owner_id, name, description, metadata, created_at, updated_at FROM orgs"

	var items []dbOrg
	err := or.db.SelectContext(ctx, &items, query)
	if err != nil {
		return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	var orgs []auth.Org
	for _, i := range items {
		org, err := toOrg(i)
		if err != nil {
			return []auth.Org{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		orgs = append(orgs, org)
	}

	return orgs, nil
}

func (or orgRepository) RetrieveByMember(ctx context.Context, memberID string, pm apiutil.PageMetadata) (auth.OrgsPage, error) {
	oq := dbutil.GetOrderQuery(pm.Order, auth.AllowedOrders)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	nq, name := dbutil.GetNameQuery(pm.Name)
	meta, mq, err := dbutil.GetMetadataQuery(pm.Metadata)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(auth.ErrRetrieveOrgsByMembership, err)
	}

	if mq != "" {
		mq = "o." + mq
	}

	moq, miq := "om.org_id = o.id", "om.member_id = :member_id"
	whereClause := dbutil.BuildWhereClause(moq, miq, nq, mq)

	query := fmt.Sprintf(`SELECT o.id, o.owner_id, o.name, o.description, o.metadata, o.created_at, o.updated_at
				FROM org_memberships om, orgs o %s ORDER BY %s %s %s;`, whereClause, oq, dq, olq)
	cquery := fmt.Sprintf(`SELECT COUNT(*) FROM org_memberships om, orgs o %s`, whereClause)

	params := map[string]any{
		"member_id": memberID,
		"name":      name,
		"limit":     pm.Limit,
		"offset":    pm.Offset,
		"metadata":  meta,
	}

	return or.retrieve(ctx, query, cquery, params)
}

func (or orgRepository) retrieve(ctx context.Context, query, cquery string, params map[string]any) (auth.OrgsPage, error) {
	rows, err := or.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []auth.Org
	for rows.Next() {
		dbor := dbOrg{}
		if err := rows.StructScan(&dbor); err != nil {
			return auth.OrgsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		or, err := toOrg(dbor)
		if err != nil {
			return auth.OrgsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		items = append(items, or)
	}

	total, err := dbutil.Total(ctx, or.db, cquery, params)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	page := auth.OrgsPage{
		Orgs:  items,
		Total: total,
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

// dbOrgMetadata type for handling metadata properly in database/sql
type dbOrgMetadata map[string]any

// Scan - Implement the database/sql scanner interface
func (m *dbOrgMetadata) Scan(value any) error {
	if value == nil {
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return dbutil.ErrScanMetadata
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
