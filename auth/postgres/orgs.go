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

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var _ auth.OrgRepository = (*orgRepository)(nil)

type orgRepository struct {
	db Database
}

var (
	orgIDFkey = "org_relations_org_id_fkey"
)

// NewOrgRepo instantiates a PostgreSQL implementation of org
// repository.
func NewOrgRepo(db Database) auth.OrgRepository {
	return &orgRepository{
		db: db,
	}
}

func (gr orgRepository) Save(ctx context.Context, g auth.Org) error {
	// For root org path is initialized with id
	q := `INSERT INTO orgs (name, description, id, owner_id, metadata, created_at, updated_at)
		  VALUES (:name, :description, :id, :owner_id, :metadata, :created_at, :updated_at)`

	dbg, err := toDBOrg(g)
	if err != nil {
		return err
	}

	row, err := gr.db.NamedQueryContext(ctx, q, dbg)
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
	defer row.Close()

	return nil
}

func (gr orgRepository) Update(ctx context.Context, g auth.Org) error {
	q := `UPDATE orgs SET name = :name, description = :description, metadata = :metadata, updated_at = :updated_at WHERE id = :id
		  RETURNING id, name, owner_id, description, metadata, created_at, updated_at`

	dbu, err := toDBOrg(g)
	if err != nil {
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	row, err := gr.db.NamedQueryContext(ctx, q, dbu)
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

func (gr orgRepository) Delete(ctx context.Context, orgID string) error {
	qd := `DELETE FROM orgs WHERE id = :id`
	org := auth.Org{
		ID: orgID,
	}
	dbg, err := toDBOrg(org)
	if err != nil {
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	res, err := gr.db.NamedExecContext(ctx, qd, dbg)
	if err != nil {
		pqErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pqErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(errors.ErrMalformedEntity, err)
			case pgerrcode.ForeignKeyViolation:
				switch pqErr.ConstraintName {
				case orgIDFkey:
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

func (gr orgRepository) RetrieveByID(ctx context.Context, id string) (auth.Org, error) {
	dbu := dbOrg{
		ID: id,
	}
	q := `SELECT id, name, owner_id, description, metadata, created_at, updated_at FROM orgs WHERE id = $1`
	if err := gr.db.QueryRowxContext(ctx, q, id).StructScan(&dbu); err != nil {
		if err == sql.ErrNoRows {
			return auth.Org{}, errors.Wrap(errors.ErrNotFound, err)

		}
		return auth.Org{}, errors.Wrap(errors.ErrViewEntity, err)
	}
	return toOrg(dbu)
}

func (gr orgRepository) RetrieveAll(ctx context.Context, ownerID string, pm auth.OrgPageMetadata) (auth.OrgPage, error) {
	_, metaQuery, err := getOrgsMetadataQuery("orgs", pm.Metadata)
	if err != nil {
		return auth.OrgPage{}, errors.Wrap(auth.ErrFailedToRetrieveAll, err)
	}

	var mq string
	if metaQuery != "" {
		mq = fmt.Sprintf(" AND %s", metaQuery)
	}

	q := fmt.Sprintf(`SELECT id, owner_id, name, description, metadata, created_at, updated_at FROM orgs
					  WHERE owner_id = :owner_id %s`, mq)

	dbPage, err := toDBOrgPage(ownerID, pm)
	if err != nil {
		return auth.OrgPage{}, errors.Wrap(auth.ErrFailedToRetrieveAll, err)
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return auth.OrgPage{}, errors.Wrap(auth.ErrFailedToRetrieveAll, err)
	}
	defer rows.Close()

	items, err := gr.processRows(rows)
	if err != nil {
		return auth.OrgPage{}, errors.Wrap(auth.ErrFailedToRetrieveAll, err)
	}

	cq := "SELECT COUNT(*) FROM orgs"
	if metaQuery != "" {
		cq = fmt.Sprintf(" %s WHERE %s", cq, metaQuery)
	}

	total, err := total(ctx, gr.db, cq, dbPage)
	if err != nil {
		return auth.OrgPage{}, errors.Wrap(auth.ErrFailedToRetrieveAll, err)
	}

	page := auth.OrgPage{
		Orgs: items,
		OrgPageMetadata: auth.OrgPageMetadata{
			Total: total,
		},
	}

	return page, nil
}

func (gr orgRepository) Members(ctx context.Context, orgID string, pm auth.OrgPageMetadata) (auth.OrgMembersPage, error) {
	_, mq, err := getOrgsMetadataQuery("orgs", pm.Metadata)
	if err != nil {
		return auth.OrgMembersPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembers, err)
	}

	q := fmt.Sprintf(`SELECT ore.member_id, ore.org_id, ore.created_at, ore.updated_at FROM org_relations ore
					  WHERE ore.org_id = :org_id %s`, mq)

	params, err := toDBOrgMemberPage("", orgID, pm)
	if err != nil {
		return auth.OrgMembersPage{}, err
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return auth.OrgMembersPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembers, err)
	}
	defer rows.Close()

	var items []auth.OrgMember
	for rows.Next() {
		member := dbOrgMember{}
		if err := rows.StructScan(&member); err != nil {
			return auth.OrgMembersPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembers, err)
		}

		if err != nil {
			return auth.OrgMembersPage{}, err
		}

		items = append(items, auth.OrgMember{ID: member.MemberID})
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM orgs o, org_relations ore
					   WHERE ore.org_id = :org_id AND ore.org_id = o.id %s;`, mq)

	total, err := total(ctx, gr.db, cq, params)
	if err != nil {
		return auth.OrgMembersPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembers, err)
	}

	page := auth.OrgMembersPage{
		Members: items,
		OrgPageMetadata: auth.OrgPageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (gr orgRepository) Memberships(ctx context.Context, memberID string, pm auth.OrgPageMetadata) (auth.OrgPage, error) {
	_, mq, err := getOrgsMetadataQuery("orgs", pm.Metadata)
	if err != nil {
		return auth.OrgPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembership, err)
	}

	if mq != "" {
		mq = fmt.Sprintf("AND %s", mq)
	}
	q := fmt.Sprintf(`SELECT g.id, g.owner_id, g.name, g.description, g.metadata
					  FROM org_relations gr, orgs g
					  WHERE gr.org_id = g.id and gr.member_id = :member_id
		  			  %s ORDER BY id LIMIT :limit OFFSET :offset;`, mq)

	params, err := toDBOrgMemberPage(memberID, "", pm)
	if err != nil {
		return auth.OrgPage{}, err
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return auth.OrgPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembership, err)
	}
	defer rows.Close()

	var items []auth.Org
	for rows.Next() {
		dbg := dbOrg{}
		if err := rows.StructScan(&dbg); err != nil {
			return auth.OrgPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembership, err)
		}
		gr, err := toOrg(dbg)
		if err != nil {
			return auth.OrgPage{}, err
		}
		items = append(items, gr)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM org_relations gr, orgs g
					   WHERE gr.org_id = g.id and gr.member_id = :member_id %s `, mq)

	total, err := total(ctx, gr.db, cq, params)
	if err != nil {
		return auth.OrgPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembership, err)
	}

	page := auth.OrgPage{
		Orgs: items,
		OrgPageMetadata: auth.OrgPageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (gr orgRepository) Assign(ctx context.Context, orgID string, ids ...string) error {
	tx, err := gr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(auth.ErrAssignToOrg, err)
	}

	qIns := `INSERT INTO org_relations (org_id, member_id, created_at, updated_at)
			 VALUES(:org_id, :member_id, :created_at, :updated_at)`

	for _, id := range ids {
		dbg, err := toDBOrgRelation(id, orgID)
		if err != nil {
			return errors.Wrap(auth.ErrAssignToOrg, err)
		}
		created := time.Now()
		dbg.CreatedAt = created
		dbg.UpdatedAt = created

		if _, err := tx.NamedExecContext(ctx, qIns, dbg); err != nil {
			tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.ForeignKeyViolation:
					return errors.Wrap(errors.ErrConflict, errors.New(pgErr.Detail))
				case pgerrcode.UniqueViolation:
					return errors.Wrap(auth.ErrMemberAlreadyAssigned, errors.New(pgErr.Detail))
				}
			}

			return errors.Wrap(auth.ErrAssignToOrg, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(auth.ErrAssignToOrg, err)
	}

	return nil
}

func (gr orgRepository) Unassign(ctx context.Context, orgID string, ids ...string) error {
	tx, err := gr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(auth.ErrAssignToOrg, err)
	}

	qDel := `DELETE from org_relations WHERE org_id = :org_id AND member_id = :member_id`

	for _, id := range ids {
		dbg, err := toDBOrgRelation(id, orgID)
		if err != nil {
			return errors.Wrap(auth.ErrAssignToOrg, err)
		}

		if _, err := tx.NamedExecContext(ctx, qDel, dbg); err != nil {
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

			return errors.Wrap(auth.ErrAssignToOrg, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(auth.ErrAssignToOrg, err)
	}

	return nil
}

type dbOrgMember struct {
	MemberID  string    `db:"member_id"`
	OrgID     string    `db:"org_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
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

type dbOrgPage struct {
	ID       string        `db:"id"`
	OwnerID  string        `db:"owner_id"`
	Metadata dbOrgMetadata `db:"metadata"`
	Total    uint64        `db:"total"`
	Limit    uint64        `db:"limit"`
	Offset   uint64        `db:"offset"`
}

type dbOrgMemberPage struct {
	OrgID    string     `db:"org_id"`
	MemberID string     `db:"member_id"`
	Metadata dbMetadata `db:"metadata"`
	Limit    uint64     `db:"limit"`
	Offset   uint64     `db:"offset"`
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

func toDBOrgPage(ownerID string, pm auth.OrgPageMetadata) (dbOrgPage, error) {
	return dbOrgPage{
		Metadata: dbOrgMetadata(pm.Metadata),
		OwnerID:  ownerID,
		Total:    pm.Total,
		Offset:   pm.Offset,
		Limit:    pm.Limit,
	}, nil
}

func toDBOrgMemberPage(memberID, orgID string, pm auth.OrgPageMetadata) (dbOrgMemberPage, error) {
	return dbOrgMemberPage{
		OrgID:    orgID,
		MemberID: memberID,
		Metadata: dbMetadata(pm.Metadata),
		Offset:   pm.Offset,
		Limit:    pm.Limit,
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

type dbOrgRelation struct {
	OrgID     sql.NullString `db:"org_id"`
	MemberID  sql.NullString `db:"member_id"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
}

func toDBOrgRelation(memberID, orgID string) (dbOrgRelation, error) {
	var grID sql.NullString
	if orgID != "" {
		grID = sql.NullString{String: orgID, Valid: true}
	}

	var mID sql.NullString
	if memberID != "" {
		mID = sql.NullString{String: memberID, Valid: true}
	}

	return dbOrgRelation{
		OrgID:    grID,
		MemberID: mID,
	}, nil
}

func getOrgsMetadataQuery(db string, m auth.OrgMetadata) (mb []byte, mq string, err error) {
	if len(m) > 0 {
		mq = `metadata @> :metadata`
		if db != "" {
			mq = db + "." + mq
		}

		b, err := json.Marshal(m)
		if err != nil {
			return nil, "", errors.Wrap(err, errCreateMetadataQuery)
		}
		mb = b
	}
	return mb, mq, nil
}

func (gr orgRepository) processRows(rows *sqlx.Rows) ([]auth.Org, error) {
	var items []auth.Org
	for rows.Next() {
		dbg := dbOrg{}
		if err := rows.StructScan(&dbg); err != nil {
			return items, err
		}
		org, err := toOrg(dbg)
		if err != nil {
			return items, err
		}
		items = append(items, org)
	}
	return items, nil
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
