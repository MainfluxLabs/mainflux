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
	"github.com/MainfluxLabs/mainflux/internal/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ auth.OrgRepository = (*orgRepository)(nil)

type orgRepository struct {
	db Database
}

var membersIDFkey = "member_relations_org_id_fkey"

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

func (or orgRepository) Delete(ctx context.Context, owner, orgID string) error {
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

func (or orgRepository) RetrieveMembersByOrg(ctx context.Context, orgID string, pm auth.PageMetadata) (auth.OrgMembersPage, error) {
	_, mq, err := dbutil.GetMetadataQuery("orgs", pm.Metadata)
	if err != nil {
		return auth.OrgMembersPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembersByOrg, err)
	}

	q := fmt.Sprintf(`SELECT member_id, org_id, created_at, updated_at, role FROM member_relations
					  WHERE org_id = :org_id %s LIMIT :limit OFFSET :offset`, mq)

	dbmp, err := toDBOrgMemberPage("", orgID, pm)
	if err != nil {
		return auth.OrgMembersPage{}, err
	}

	rows, err := or.db.NamedQueryContext(ctx, q, dbmp)
	if err != nil {
		return auth.OrgMembersPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembersByOrg, err)
	}
	defer rows.Close()

	var oms []auth.OrgMember
	for rows.Next() {
		dbm := dbMember{}
		if err := rows.StructScan(&dbm); err != nil {
			return auth.OrgMembersPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembersByOrg, err)
		}

		om, err := toMember(dbm)
		if err != nil {
			return auth.OrgMembersPage{}, err
		}

		oms = append(oms, om)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM orgs o, member_relations ore
					   WHERE ore.org_id = :org_id AND ore.org_id = o.id %s;`, mq)

	total, err := total(ctx, or.db, cq, dbmp)
	if err != nil {
		return auth.OrgMembersPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembersByOrg, err)
	}

	page := auth.OrgMembersPage{
		OrgMembers: oms,
		PageMetadata: auth.PageMetadata{
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

func (or orgRepository) RetrieveOrgsByMember(ctx context.Context, memberID string, pm auth.PageMetadata) (auth.OrgsPage, error) {
	meta, mq, err := dbutil.GetMetadataQuery("o", pm.Metadata)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(auth.ErrFailedToRetrieveOrgsByMember, err)
	}

	nq, name := dbutil.GetNameQuery(pm.Name)

	if mq != "" {
		mq = fmt.Sprintf("AND %s", mq)
	}

	if nq != "" {
		nq = fmt.Sprintf("AND %s", nq)
	}

	q := fmt.Sprintf(`SELECT o.id, o.owner_id, o.name, o.description, o.metadata
		FROM member_relations ore, orgs o
		WHERE ore.org_id = o.id and ore.member_id = :member_id
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
		return auth.OrgsPage{}, errors.Wrap(auth.ErrFailedToRetrieveOrgsByMember, err)
	}
	defer rows.Close()

	var items []auth.Org
	for rows.Next() {
		dbg := dbOrg{}
		if err := rows.StructScan(&dbg); err != nil {
			return auth.OrgsPage{}, errors.Wrap(auth.ErrFailedToRetrieveOrgsByMember, err)
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
		return auth.OrgsPage{}, errors.Wrap(auth.ErrFailedToRetrieveOrgsByMember, err)
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

func (or orgRepository) RetrieveRole(ctx context.Context, memberID, orgID string) (string, error) {
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

func (or orgRepository) AssignMembers(ctx context.Context, oms ...auth.OrgMember) error {
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

func (or orgRepository) UnassignMembers(ctx context.Context, orgID string, ids ...string) error {
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

func (or orgRepository) UpdateMembers(ctx context.Context, oms ...auth.OrgMember) error {
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

func (or orgRepository) RetrieveAllMembersByOrg(ctx context.Context) ([]auth.OrgMember, error) {
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

func (or orgRepository) retrieve(ctx context.Context, ownerID string, pm auth.PageMetadata) (auth.OrgsPage, error) {
	ownq := dbutil.GetOwnerQuery(ownerID)
	nq, name := dbutil.GetNameQuery(pm.Name)
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

	olq := "LIMIT :limit OFFSET :offset"
	if pm.Limit == 0 {
		olq = ""
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

type dbMember struct {
	MemberID  string    `db:"member_id"`
	OrgID     string    `db:"org_id"`
	Role      string    `db:"role"`
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

type dbOrgGroup struct {
	OrgID     string    `db:"org_id"`
	GroupID   string    `db:"group_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func toDBOrgGroup(og auth.OrgGroup) dbOrgGroup {
	return dbOrgGroup{
		OrgID:     og.OrgID,
		GroupID:   og.GroupID,
		CreatedAt: og.CreatedAt,
		UpdatedAt: og.UpdatedAt,
	}
}

func toOrgGroup(og dbOrgGroup) auth.OrgGroup {
	return auth.OrgGroup{
		OrgID:     og.OrgID,
		GroupID:   og.GroupID,
		CreatedAt: og.CreatedAt,
		UpdatedAt: og.UpdatedAt,
	}
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
