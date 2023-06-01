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
	membersIDFkey          = "member_relations_org_id_fkey"
)

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

func (or orgRepository) RetrieveMembers(ctx context.Context, orgID string, pm auth.PageMetadata) (auth.OrgMembersPage, error) {
	_, mq, err := getOrgsMetadataQuery("orgs", pm.Metadata)
	if err != nil {
		return auth.OrgMembersPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembers, err)
	}

	q := fmt.Sprintf(`SELECT member_id, org_id, created_at, updated_at, role FROM member_relations
					  WHERE org_id = :org_id %s`, mq)

	dbmp, err := toDBOrgMemberPage("", orgID, pm)
	if err != nil {
		return auth.OrgMembersPage{}, err
	}

	rows, err := or.db.NamedQueryContext(ctx, q, dbmp)
	if err != nil {
		return auth.OrgMembersPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembers, err)
	}
	defer rows.Close()

	var items []auth.Member
	for rows.Next() {
		dbmb := dbMember{}
		if err := rows.StructScan(&dbmb); err != nil {
			return auth.OrgMembersPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembers, err)
		}

		mb, err := toMember(dbmb)
		if err != nil {
			return auth.OrgMembersPage{}, err
		}

		items = append(items, mb)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM orgs o, member_relations ore
					   WHERE ore.org_id = :org_id AND ore.org_id = o.id %s;`, mq)

	total, err := total(ctx, or.db, cq, dbmp)
	if err != nil {
		return auth.OrgMembersPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembers, err)
	}

	page := auth.OrgMembersPage{
		Members: items,
		PageMetadata: auth.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func toMember(dbmb dbMember) (auth.Member, error) {
	return auth.Member{
		ID:   dbmb.MemberID,
		Role: dbmb.Role,
	}, nil
}

func (or orgRepository) RetrieveMemberships(ctx context.Context, memberID string, pm auth.PageMetadata) (auth.OrgsPage, error) {
	_, mq, err := getOrgsMetadataQuery("o", pm.Metadata)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembership, err)
	}

	nq := getNameQuery(pm.Name)

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

	dbmp, err := toDBOrgMemberPage(memberID, "", pm)
	if err != nil {
		return auth.OrgsPage{}, err
	}

	rows, err := or.db.NamedQueryContext(ctx, q, dbmp)
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

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM member_relations ore, orgs o
		WHERE ore.org_id = o.id and ore.member_id = :member_id %s %s`, mq, nq)

	total, err := total(ctx, or.db, cq, dbmp)
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

func (or orgRepository) RetrieveRole(ctx context.Context, memberID, orgID string) (string, error) {
	q := `SELECT role FROM member_relations WHERE member_id = $1 AND org_id = $2`

	member := auth.Member{}
	if err := or.db.QueryRowxContext(ctx, q, memberID, orgID).StructScan(&member); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return "", errors.Wrap(errors.ErrNotFound, err)
		}

		return "", errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	return member.Role, nil
}

func (or orgRepository) AssignMembers(ctx context.Context, mrs ...auth.MemberRelation) error {
	tx, err := or.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(auth.ErrAssignToOrg, err)
	}

	qIns := `INSERT INTO member_relations (org_id, member_id, role, created_at, updated_at)
			 VALUES(:org_id, :member_id, :role, :created_at, :updated_at)`

	for _, mr := range mrs {
		dbmr, err := toDBMemberRelation(mr)
		if err != nil {
			return errors.Wrap(auth.ErrAssignToOrg, err)
		}

		if _, err := tx.NamedExecContext(ctx, qIns, dbmr); err != nil {
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

func (or orgRepository) UnassignMembers(ctx context.Context, orgID string, ids ...string) error {
	tx, err := or.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(auth.ErrAssignToOrg, err)
	}

	qDel := `DELETE from member_relations WHERE org_id = :org_id AND member_id = :member_id`

	for _, id := range ids {
		mr := auth.MemberRelation{
			OrgID:    orgID,
			MemberID: id,
		}

		dbmr, err := toDBMemberRelation(mr)
		if err != nil {
			return errors.Wrap(auth.ErrAssignToOrg, err)
		}

		if _, err := tx.NamedExecContext(ctx, qDel, dbmr); err != nil {
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

func (or orgRepository) UpdateMembers(ctx context.Context, mrs ...auth.MemberRelation) error {
	qUpd := `UPDATE member_relations SET role = :role, updated_at = :updated_at
			 WHERE org_id = :org_id AND member_id = :member_id`

	for _, mr := range mrs {
		dbmr, err := toDBMemberRelation(mr)
		if err != nil {
			return errors.Wrap(errors.ErrUpdateEntity, err)
		}

		row, err := or.db.NamedExecContext(ctx, qUpd, dbmr)
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

func (or orgRepository) AssignGroups(ctx context.Context, grs ...auth.GroupRelation) error {
	tx, err := or.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(auth.ErrAssignToOrg, err)
	}

	qIns := `INSERT INTO group_relations (org_id, group_id, created_at, updated_at)
			 VALUES(:org_id, :group_id, :created_at, :updated_at)`

	for _, gr := range grs {
		dbgr, err := toDBGroupRelation(gr)
		if err != nil {
			return errors.Wrap(auth.ErrAssignToOrg, err)
		}

		if _, err := tx.NamedExecContext(ctx, qIns, dbgr); err != nil {
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

func (or orgRepository) UnassignGroups(ctx context.Context, orgID string, groupIDs ...string) error {
	tx, err := or.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(auth.ErrAssignToOrg, err)
	}

	qDel := `DELETE from group_relations WHERE org_id = :org_id AND group_id = :group_id`

	for _, id := range groupIDs {
		gr := auth.GroupRelation{
			OrgID:   orgID,
			GroupID: id,
		}

		dbgr, err := toDBGroupRelation(gr)
		if err != nil {
			return errors.Wrap(auth.ErrAssignToOrg, err)
		}

		if _, err := tx.NamedExecContext(ctx, qDel, dbgr); err != nil {
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

func (or orgRepository) RetrieveGroups(ctx context.Context, orgID string, pm auth.PageMetadata) (auth.OrgGroupsPage, error) {
	_, mq, err := getOrgsMetadataQuery("orgs", pm.Metadata)
	if err != nil {
		return auth.OrgGroupsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	q := fmt.Sprintf(`SELECT gre.group_id, gre.org_id, gre.created_at, gre.updated_at FROM group_relations gre
					  WHERE gre.org_id = :org_id %s`, mq)

	dbmp, err := toDBOrgMemberPage("", orgID, pm)
	if err != nil {
		return auth.OrgGroupsPage{}, err
	}

	rows, err := or.db.NamedQueryContext(ctx, q, dbmp)
	if err != nil {
		return auth.OrgGroupsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []string
	for rows.Next() {
		group := dbGroup{}
		if err := rows.StructScan(&group); err != nil {
			return auth.OrgGroupsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		if err != nil {
			return auth.OrgGroupsPage{}, err
		}

		items = append(items, group.GroupID)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM orgs o, group_relations gre
					   WHERE gre.org_id = :org_id AND gre.org_id = o.id %s;`, mq)

	total, err := total(ctx, or.db, cq, dbmp)
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

func (or orgRepository) RetrieveByGroupID(ctx context.Context, groupID string) (auth.OrgsPage, error) {
	q := `SELECT o.id, o.owner_id, o.name, o.description, o.metadata
		FROM group_relations gre, orgs o
		WHERE gre.org_id = o.id and gre.group_id = :group_id;`

	gr := auth.GroupRelation{GroupID: groupID}
	dbgr, err := toDBGroupRelation(gr)
	if err != nil {
		return auth.OrgsPage{}, err
	}

	rows, err := or.db.NamedQueryContext(ctx, q, dbgr)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []auth.Org
	for rows.Next() {
		dbo := dbOrg{}
		if err := rows.StructScan(&dbo); err != nil {
			return auth.OrgsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}
		or, err := toOrg(dbo)
		if err != nil {
			return auth.OrgsPage{}, err
		}
		items = append(items, or)
	}

	cq := `SELECT COUNT(*) FROM group_relations gre, orgs o
		WHERE gre.org_id = o.id and gre.group_id = :group_id;`

	total, err := total(ctx, or.db, cq, dbgr)
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

func (or orgRepository) RetrieveAllMemberRelations(ctx context.Context) ([]auth.MemberRelation, error) {
	q := `SELECT org_id, member_id, role, created_at, updated_at FROM member_relations;`

	rows, err := or.db.NamedQueryContext(ctx, q, map[string]interface{}{})
	if err != nil {
		return []auth.MemberRelation{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var mrs []auth.MemberRelation
	for rows.Next() {
		dbmr := dbMemberRelation{}
		if err := rows.StructScan(&dbmr); err != nil {
			return []auth.MemberRelation{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		mrs = append(mrs, toMemberRelation(dbmr))
	}

	return mrs, nil
}

func (or orgRepository) RetrieveAllGroupRelations(ctx context.Context) ([]auth.GroupRelation, error) {
	q := `SELECT org_id, group_id, created_at, updated_at FROM group_relations;`

	rows, err := or.db.NamedQueryContext(ctx, q, map[string]interface{}{})
	if err != nil {
		return []auth.GroupRelation{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var grs []auth.GroupRelation
	for rows.Next() {
		dbgr := dbGroupRelation{}
		if err := rows.StructScan(&dbgr); err != nil {
			return []auth.GroupRelation{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		grs = append(grs, toGroupRelation(dbgr))
	}

	return grs, nil
}

func (or orgRepository) retrieve(ctx context.Context, ownerID string, pm auth.PageMetadata) (auth.OrgsPage, error) {
	var whereq string
	if ownerID != "" {
		whereq = "WHERE owner_id = :owner_id"
	}

	_, mq, err := getOrgsMetadataQuery("orgs", pm.Metadata)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	nq := getNameQuery(pm.Name)

	var query []string
	if mq != "" {
		query = append(query, mq)
	}

	if nq != "" {
		query = append(query, nq)
	}

	if len(query) > 0 {
		whereq = fmt.Sprintf("%s AND %s", whereq, strings.Join(query, " AND "))
	}

	q := fmt.Sprintf(`SELECT id, owner_id, name, description, metadata, created_at, updated_at FROM orgs %s;`, whereq)

	dbop, err := toDBOrgsPage(ownerID, pm)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	rows, err := or.db.NamedQueryContext(ctx, q, dbop)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	items, err := or.processRows(rows)
	if err != nil {
		return auth.OrgsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	cq := fmt.Sprintf("SELECT COUNT(*) FROM orgs %s;", whereq)

	total, err := total(ctx, or.db, cq, dbop)
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

type dbMember struct {
	MemberID  string    `db:"member_id"`
	OrgID     string    `db:"org_id"`
	Role      string    `db:"role"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type dbGroup struct {
	GroupID   string    `db:"group_id"`
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

type dbMemberRelation struct {
	OrgID     string    `db:"org_id"`
	MemberID  string    `db:"member_id"`
	Role      string    `db:"role"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func toDBMemberRelation(mr auth.MemberRelation) (dbMemberRelation, error) {
	return dbMemberRelation{
		OrgID:     mr.OrgID,
		MemberID:  mr.MemberID,
		Role:      mr.Role,
		CreatedAt: mr.CreatedAt,
		UpdatedAt: mr.UpdatedAt,
	}, nil
}

func toMemberRelation(mr dbMemberRelation) auth.MemberRelation {
	return auth.MemberRelation{
		OrgID:     mr.OrgID,
		MemberID:  mr.MemberID,
		Role:      mr.Role,
		CreatedAt: mr.CreatedAt,
		UpdatedAt: mr.UpdatedAt,
	}
}

type dbGroupRelation struct {
	OrgID     string    `db:"org_id"`
	GroupID   string    `db:"group_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func toDBGroupRelation(gr auth.GroupRelation) (dbGroupRelation, error) {
	return dbGroupRelation{
		OrgID:     gr.OrgID,
		GroupID:   gr.GroupID,
		CreatedAt: gr.CreatedAt,
		UpdatedAt: gr.UpdatedAt,
	}, nil
}

func toGroupRelation(gr dbGroupRelation) auth.GroupRelation {
	return auth.GroupRelation{
		OrgID:     gr.OrgID,
		GroupID:   gr.GroupID,
		CreatedAt: gr.CreatedAt,
		UpdatedAt: gr.UpdatedAt,
	}
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

func getNameQuery(name string) string {
	if name == "" {
		return ""
	}

	name = fmt.Sprintf(`%%%s%%`, strings.ToLower(name))
	nq := fmt.Sprintf("LOWER(name) LIKE '%s'", name)

	return nq
}

func (or orgRepository) processRows(rows *sqlx.Rows) ([]auth.Org, error) {
	var items []auth.Org
	for rows.Next() {
		dbo := dbOrg{}
		if err := rows.StructScan(&dbo); err != nil {
			return items, err
		}
		org, err := toOrg(dbo)
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
