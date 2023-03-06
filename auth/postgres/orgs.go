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
	errCreateMetadataQuery = errors.New("failed to create query for metadata")
	errGetTotal            = errors.New("failed to get total number of groups")
	orgIDFkey              = "org_relations_org_id_fkey"
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

func (gr orgRepository) Delete(ctx context.Context, owner, orgID string) error {
	qd := `DELETE FROM orgs WHERE id = :id AND owner_id = :owner_id;`
	org := auth.Org{
		ID:      orgID,
		OwnerID: owner,
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
		return auth.Org{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	return toOrg(dbu)
}

func (gr orgRepository) RetrieveByOwner(ctx context.Context, ownerID string, pm auth.PageMetadata) (auth.OrgsPage, error) {
	whereq := "WHERE owner_id = :owner_id"

	_, mq, err := getOrgsMetadataQuery("orgs", pm.Metadata)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	var query []string
	if mq != "" {
		query = append(query, mq)
	}
	if len(query) > 0 {
		whereq = fmt.Sprintf("%s AND %s", whereq, strings.Join(query, " AND "))
	}

	q := fmt.Sprintf(`SELECT id, owner_id, name, description, metadata, created_at, updated_at FROM orgs %s;`, whereq)

	dbPage, err := toDBOrgsPage(ownerID, pm)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	items, err := gr.processRows(rows)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	cq := fmt.Sprintf("SELECT COUNT(*) FROM orgs %s;", whereq)

	total, err := total(ctx, gr.db, cq, dbPage)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	page := auth.OrgsPage{
		Orgs: items,
		PageMetadata: auth.PageMetadata{
			Total: total,
		},
	}

	return page, nil
}

func (gr orgRepository) RetrieveMembers(ctx context.Context, orgID string, pm auth.PageMetadata) (auth.OrgMembersPage, error) {
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

	var items []string
	for rows.Next() {
		member := dbOrgMember{}
		if err := rows.StructScan(&member); err != nil {
			return auth.OrgMembersPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembers, err)
		}

		if err != nil {
			return auth.OrgMembersPage{}, err
		}

		items = append(items, member.MemberID)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM orgs o, org_relations ore
					   WHERE ore.org_id = :org_id AND ore.org_id = o.id %s;`, mq)

	total, err := total(ctx, gr.db, cq, params)
	if err != nil {
		return auth.OrgMembersPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembers, err)
	}

	page := auth.OrgMembersPage{
		MemberIDs: items,
		PageMetadata: auth.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (gr orgRepository) RetrieveMemberships(ctx context.Context, memberID string, pm auth.PageMetadata) (auth.OrgsPage, error) {
	_, mq, err := getOrgsMetadataQuery("orgs", pm.Metadata)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembership, err)
	}

	if mq != "" {
		mq = fmt.Sprintf("AND %s", mq)
	}
	q := fmt.Sprintf(`SELECT o.id, o.owner_id, o.name, o.description, o.metadata
		FROM org_relations ore, orgs o
		WHERE ore.org_id = o.id and ore.member_id = :member_id
		%s ORDER BY id LIMIT :limit OFFSET :offset;`, mq)

	params, err := toDBOrgMemberPage(memberID, "", pm)
	if err != nil {
		return auth.OrgsPage{}, err
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembership, err)
	}
	defer rows.Close()

	var items []auth.Org
	for rows.Next() {
		dbg := dbOrg{}
		if err := rows.StructScan(&dbg); err != nil {
			return auth.OrgsPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembership, err)
		}
		gr, err := toOrg(dbg)
		if err != nil {
			return auth.OrgsPage{}, err
		}
		items = append(items, gr)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM org_relations ore, orgs o
		WHERE ore.org_id = o.id and ore.member_id = :member_id %s `, mq)

	total, err := total(ctx, gr.db, cq, params)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembership, err)
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

func (gr orgRepository) AssignMembers(ctx context.Context, orgID string, ids ...string) error {
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
					return errors.Wrap(auth.ErrOrgMemberAlreadyAssigned, errors.New(pgErr.Detail))
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

func (gr orgRepository) UnassignMembers(ctx context.Context, orgID string, ids ...string) error {
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

func (gr orgRepository) AssignGroups(ctx context.Context, orgID string, groupIDs ...string) error {
	tx, err := gr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(auth.ErrAssignToOrg, err)
	}

	qIns := `INSERT INTO group_relations (org_id, group_id, created_at, updated_at)
			 VALUES(:org_id, :group_id, :created_at, :updated_at)`

	for _, id := range groupIDs {
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
					return errors.Wrap(auth.ErrOrgMemberAlreadyAssigned, errors.New(pgErr.Detail))
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

func (gr orgRepository) UnassignGroups(ctx context.Context, orgID string, groupIDs ...string) error {
	tx, err := gr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(auth.ErrAssignToOrg, err)
	}

	qDel := `DELETE from group_relations WHERE org_id = :org_id AND group_id = :group_id`

	for _, id := range groupIDs {
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

func (gr orgRepository) RetrieveGroups(ctx context.Context, orgID string, pm auth.PageMetadata) (auth.OrgGroupsPage, error) {
	_, mq, err := getOrgsMetadataQuery("orgs", pm.Metadata)
	if err != nil {
		return auth.OrgGroupsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	q := fmt.Sprintf(`SELECT gre.group_id, gre.org_id, gre.created_at, oge.updated_at FROM group_relations gre
					  WHERE gre.org_id = :org_id %s`, mq)

	params, err := toDBOrgMemberPage("", orgID, pm)
	if err != nil {
		return auth.OrgGroupsPage{}, err
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return auth.OrgGroupsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []string
	for rows.Next() {
		member := dbOrgMember{}
		if err := rows.StructScan(&member); err != nil {
			return auth.OrgGroupsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		if err != nil {
			return auth.OrgGroupsPage{}, err
		}

		items = append(items, member.MemberID)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM orgs o, group_relations gre
					   WHERE gre.org_id = :org_id AND gre.org_id = o.id %s;`, mq)

	total, err := total(ctx, gr.db, cq, params)
	if err != nil {
		return auth.OrgGroupsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	page := auth.OrgGroupsPage{
		GroupIDs: items,
		PageMetadata: auth.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
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

type dbOrgsPage struct {
	ID       string        `db:"id"`
	OwnerID  string        `db:"owner_id"`
	Metadata dbOrgMetadata `db:"metadata"`
	Total    uint64        `db:"total"`
	Limit    uint64        `db:"limit"`
	Offset   uint64        `db:"offset"`
}

type dbOrgMemberPage struct {
	OrgID    string        `db:"org_id"`
	MemberID string        `db:"member_id"`
	Metadata dbOrgMetadata `db:"metadata"`
	Limit    uint64        `db:"limit"`
	Offset   uint64        `db:"offset"`
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

func toDBOrgsPage(ownerID string, pm auth.PageMetadata) (dbOrgsPage, error) {
	return dbOrgsPage{
		Metadata: dbOrgMetadata(pm.Metadata),
		OwnerID:  ownerID,
		Total:    pm.Total,
		Offset:   pm.Offset,
		Limit:    pm.Limit,
	}, nil
}

func toDBOrgMemberPage(memberID, orgID string, pm auth.PageMetadata) (dbOrgMemberPage, error) {
	return dbOrgMemberPage{
		OrgID:    orgID,
		MemberID: memberID,
		Metadata: dbOrgMetadata(pm.Metadata),
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
	OrgID     string    `db:"org_id"`
	MemberID  string    `db:"member_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func toDBOrgRelation(memberID, orgID string) (dbOrgRelation, error) {
	return dbOrgRelation{
		OrgID:    orgID,
		MemberID: memberID,
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

func total(ctx context.Context, db Database, query string, params interface{}) (uint64, error) {
	rows, err := db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return 0, errors.Wrap(errGetTotal, err)
	}
	defer rows.Close()
	total := uint64(0)
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, errors.Wrap(errGetTotal, err)
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
