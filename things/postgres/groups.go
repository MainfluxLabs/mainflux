// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
)

var (
	errCreateMetadataQuery = errors.New("failed to create query for metadata")
	groupIDFkeyy           = "group_relations_group_id_fkey"
)

var _ things.GroupRepository = (*groupRepository)(nil)

type groupRepository struct {
	db Database
}

// NewGroupRepo instantiates a PostgreSQL implementation of group
// repository.
func NewGroupRepo(db Database) things.GroupRepository {
	return &groupRepository{
		db: db,
	}
}

func (gr groupRepository) Save(ctx context.Context, g things.Group) (things.Group, error) {
	q := `INSERT INTO groups (name, description, id, owner_id, metadata, created_at, updated_at)
		  VALUES (:name, :description, :id, :owner_id, :metadata, :created_at, :updated_at)
		  RETURNING id, name, owner_id, description, metadata, created_at, updated_at`

	dbg, err := toDBGroup(g)
	if err != nil {
		return things.Group{}, err
	}

	row, err := gr.db.NamedQueryContext(ctx, q, dbg)
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return things.Group{}, errors.Wrap(errors.ErrMalformedEntity, err)
			case pgerrcode.ForeignKeyViolation:
				return things.Group{}, errors.Wrap(errors.ErrCreateEntity, err)
			case pgerrcode.UniqueViolation:
				return things.Group{}, errors.Wrap(errors.ErrConflict, err)
			case pgerrcode.StringDataRightTruncationDataException:
				return things.Group{}, errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}

		return things.Group{}, errors.Wrap(errors.ErrCreateEntity, err)
	}

	defer row.Close()
	row.Next()
	dbg = dbGroup{}
	if err := row.StructScan(&dbg); err != nil {
		return things.Group{}, err
	}

	return toGroup(dbg)
}

func (gr groupRepository) Update(ctx context.Context, g things.Group) (things.Group, error) {
	q := `UPDATE groups SET name = :name, description = :description, metadata = :metadata, updated_at = :updated_at WHERE id = :id
		  RETURNING id, name, owner_id, description, metadata, created_at, updated_at`

	dbu, err := toDBGroup(g)
	if err != nil {
		return things.Group{}, errors.Wrap(errors.ErrUpdateEntity, err)
	}

	row, err := gr.db.NamedQueryContext(ctx, q, dbu)
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return things.Group{}, errors.Wrap(errors.ErrMalformedEntity, err)
			case pgerrcode.UniqueViolation:
				return things.Group{}, errors.Wrap(errors.ErrConflict, err)
			case pgerrcode.StringDataRightTruncationDataException:
				return things.Group{}, errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}
		return things.Group{}, errors.Wrap(errors.ErrUpdateEntity, err)
	}

	defer row.Close()
	row.Next()
	dbu = dbGroup{}
	if err := row.StructScan(&dbu); err != nil {
		return g, errors.Wrap(errors.ErrUpdateEntity, err)
	}

	return toGroup(dbu)
}

func (gr groupRepository) Remove(ctx context.Context, groupID string) error {
	qd := `DELETE FROM groups WHERE id = :id`
	group := things.Group{
		ID: groupID,
	}
	dbg, err := toDBGroup(group)
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
				case groupIDFkeyy:
					return errors.Wrap(things.ErrGroupNotEmpty, err)
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

func (gr groupRepository) RetrieveAll(ctx context.Context) ([]things.Group, error) {
	gp, err := gr.retrieve(ctx, "", things.PageMetadata{})
	if err != nil {
		return nil, err
	}

	return gp.Groups, nil
}

func (gr groupRepository) RetrieveByID(ctx context.Context, id string) (things.Group, error) {
	dbu := dbGroup{
		ID: id,
	}
	q := `SELECT id, name, owner_id, description, metadata, created_at, updated_at FROM groups WHERE id = $1`
	if err := gr.db.QueryRowxContext(ctx, q, id).StructScan(&dbu); err != nil {
		if err == sql.ErrNoRows {
			return things.Group{}, errors.Wrap(errors.ErrNotFound, err)

		}
		return things.Group{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	return toGroup(dbu)
}

func (gr groupRepository) RetrieveByIDs(ctx context.Context, groupIDs []string) (things.GroupPage, error) {
	if len(groupIDs) == 0 {
		return things.GroupPage{}, nil
	}

	idq := fmt.Sprintf("WHERE id IN ('%s') ", strings.Join(groupIDs, "','"))
	q := fmt.Sprintf(`SELECT id, name, owner_id, description, metadata, created_at, updated_at FROM groups %s;`, idq)

	rows, err := gr.db.NamedQueryContext(ctx, q, map[string]interface{}{})
	if err != nil {
		return things.GroupPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	items := []things.Group{}
	for rows.Next() {
		dbg := dbGroup{}
		if err := rows.StructScan(&dbg); err != nil {
			return things.GroupPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		gr, err := toGroup(dbg)
		if err != nil {
			return things.GroupPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}
		items = append(items, gr)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM groups %s`, idq)

	total, err := total(ctx, gr.db, cq, map[string]interface{}{})
	if err != nil {
		return things.GroupPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	page := things.GroupPage{
		Groups: items,
		PageMetadata: things.PageMetadata{
			Total: total,
		},
	}

	return page, nil
}

func (gr groupRepository) RetrieveByOwner(ctx context.Context, ownerID string, pm things.PageMetadata) (things.GroupPage, error) {
	if ownerID == "" {
		return things.GroupPage{}, errors.ErrRetrieveEntity
	}

	return gr.retrieve(ctx, ownerID, pm)
}

func (gr groupRepository) RetrieveByAdmin(ctx context.Context, pm things.PageMetadata) (things.GroupPage, error) {
	return gr.retrieve(ctx, "", pm)
}

func (gr groupRepository) RetrieveMembers(ctx context.Context, groupID string, pm things.PageMetadata) (things.MemberPage, error) {
	_, mq, err := getGroupsMetadataQuery("groups", pm.Metadata)
	if err != nil {
		return things.MemberPage{}, errors.Wrap(things.ErrFailedToRetrieveMembers, err)
	}

	olq := "LIMIT :limit OFFSET :offset"
	if pm.Limit == 0 {
		olq = ""
	}

	q := fmt.Sprintf(`SELECT t.id, t.owner, t.name, t.metadata, t.key
	FROM group_relations gr, things t
	WHERE gr.group_id = :group_id and gr.member_id = t.id
	%s %s;`, mq, olq)

	params, err := toDBMemberPage("", groupID, pm)
	if err != nil {
		return things.MemberPage{}, err
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.MemberPage{}, errors.Wrap(things.ErrFailedToRetrieveMembers, err)
	}
	defer rows.Close()

	var items []things.Thing
	for rows.Next() {
		dbmem := dbThing{}
		if err := rows.StructScan(&dbmem); err != nil {
			return things.MemberPage{}, errors.Wrap(things.ErrFailedToRetrieveMembers, err)
		}

		th, err := toThing(dbmem)
		if err != nil {
			return things.MemberPage{}, err
		}

		items = append(items, th)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM group_relations gr WHERE gr.group_id = :group_id %s;`, mq)

	total, err := total(ctx, gr.db, cq, params)
	if err != nil {
		return things.MemberPage{}, errors.Wrap(things.ErrFailedToRetrieveMembers, err)
	}

	page := things.MemberPage{
		Members: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (gr groupRepository) RetrieveMemberships(ctx context.Context, memberID string, pm things.PageMetadata) (things.GroupPage, error) {
	_, mq, err := getGroupsMetadataQuery("groups", pm.Metadata)
	if err != nil {
		return things.GroupPage{}, errors.Wrap(things.ErrFailedToRetrieveMembership, err)
	}

	if mq != "" {
		mq = fmt.Sprintf("AND %s", mq)
	}

	olq := "LIMIT :limit OFFSET :offset"
	if pm.Limit == 0 {
		olq = ""
	}

	q := fmt.Sprintf(`SELECT g.id, g.owner_id, g.name, g.description, g.metadata
		FROM group_relations gr, groups g
		WHERE gr.group_id = g.id and gr.member_id = :member_id
		%s ORDER BY id %s;`, mq, olq)

	params, err := toDBMemberPage(memberID, "", pm)
	if err != nil {
		return things.GroupPage{}, err
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.GroupPage{}, errors.Wrap(things.ErrFailedToRetrieveMembership, err)
	}
	defer rows.Close()

	var items []things.Group
	for rows.Next() {
		dbg := dbGroup{}
		if err := rows.StructScan(&dbg); err != nil {
			return things.GroupPage{}, errors.Wrap(things.ErrFailedToRetrieveMembership, err)
		}
		gr, err := toGroup(dbg)
		if err != nil {
			return things.GroupPage{}, err
		}
		items = append(items, gr)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM group_relations gr, groups g
		WHERE gr.group_id = g.id and gr.member_id = :member_id %s `, mq)

	total, err := total(ctx, gr.db, cq, params)
	if err != nil {
		return things.GroupPage{}, errors.Wrap(things.ErrFailedToRetrieveMembership, err)
	}

	page := things.GroupPage{
		Groups: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (gr groupRepository) RetrieveAllGroupRelations(ctx context.Context) ([]things.GroupRelation, error) {
	q := `SELECT group_id, member_id, created_at, updated_at FROM group_relations`

	rows, err := gr.db.NamedQueryContext(ctx, q, map[string]interface{}{})
	if err != nil {
		return nil, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	grRel := []things.GroupRelation{}
	for rows.Next() {
		dbg := dbGroupRelation{}
		if err := rows.StructScan(&dbg); err != nil {
			return nil, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		gr, err := toGroupRelation(dbg)
		if err != nil {
			return nil, errors.Wrap(errors.ErrRetrieveEntity, err)
		}
		grRel = append(grRel, gr)
	}

	return grRel, nil
}

func (gr groupRepository) AssignMember(ctx context.Context, groupID string, ids ...string) error {
	tx, err := gr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(things.ErrAssignToGroup, err)
	}

	qIns := `INSERT INTO group_relations (group_id, member_id, created_at, updated_at)
		VALUES(:group_id, :member_id, :created_at, :updated_at)`

	for _, id := range ids {
		dbg, err := toDBGroupRelation(id, groupID)
		if err != nil {
			return errors.Wrap(things.ErrAssignToGroup, err)
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
					return errors.Wrap(things.ErrMemberAlreadyAssigned, errors.New(pgErr.Detail))
				}
			}

			return errors.Wrap(things.ErrAssignToGroup, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(things.ErrAssignToGroup, err)
	}

	return nil
}

func (gr groupRepository) UnassignMember(ctx context.Context, groupID string, ids ...string) error {
	tx, err := gr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(things.ErrAssignToGroup, err)
	}

	qDel := `DELETE from group_relations WHERE group_id = :group_id AND member_id = :member_id`

	for _, id := range ids {
		dbg, err := toDBGroupRelation(id, groupID)
		if err != nil {
			return errors.Wrap(things.ErrAssignToGroup, err)
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

			return errors.Wrap(things.ErrAssignToGroup, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(things.ErrAssignToGroup, err)
	}

	return nil
}

func (gr groupRepository) retrieve(ctx context.Context, ownerID string, pm things.PageMetadata) (things.GroupPage, error) {
	var ownq string
	if ownerID != "" {
		ownq = "owner_id = :owner_id"
	}

	nq, name := getNameQuery(pm.Name)

	meta, mq, err := getMetadataQuery(pm.Metadata)
	if err != nil {
		return things.GroupPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	var whereClause string
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

	if len(query) > 0 {
		whereClause = fmt.Sprintf(" WHERE %s", strings.Join(query, " AND "))
	}

	olq := "LIMIT :limit OFFSET :offset"
	if pm.Limit == 0 {
		olq = ""
	}

	q := fmt.Sprintf(`SELECT id, owner_id, name, description, metadata, created_at, updated_at FROM groups %s %s;`, whereClause, olq)

	params := map[string]interface{}{
		"owner_id": ownerID,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
		"name":     name,
		"metadata": meta,
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.GroupPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	items := []things.Group{}
	for rows.Next() {
		dbg := dbGroup{}
		if err := rows.StructScan(&dbg); err != nil {
			return things.GroupPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}
		gr, err := toGroup(dbg)
		if err != nil {
			return things.GroupPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}
		items = append(items, gr)
	}

	cq := fmt.Sprintf("SELECT COUNT(*) FROM groups %s;", whereClause)

	total, err := total(ctx, gr.db, cq, params)
	if err != nil {
		return things.GroupPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	page := things.GroupPage{
		Groups: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Order:  pm.Order,
			Dir:    pm.Dir,
		},
	}

	return page, nil
}

type dbGroup struct {
	ID          string     `db:"id"`
	OwnerID     string     `db:"owner_id"`
	Name        string     `db:"name"`
	Description string     `db:"description"`
	Metadata    dbMetadata `db:"metadata"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

type dbMemberPage struct {
	GroupID  string     `db:"group_id"`
	MemberID string     `db:"member_id"`
	Metadata dbMetadata `db:"metadata"`
	Limit    uint64     `db:"limit"`
	Offset   uint64     `db:"offset"`
}

func toDBGroup(g things.Group) (dbGroup, error) {
	return dbGroup{
		ID:          g.ID,
		Name:        g.Name,
		OwnerID:     g.OwnerID,
		Description: g.Description,
		Metadata:    dbMetadata(g.Metadata),
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}, nil
}

func toDBMemberPage(memberID, groupID string, pm things.PageMetadata) (dbMemberPage, error) {
	return dbMemberPage{
		GroupID:  groupID,
		MemberID: memberID,
		Metadata: dbMetadata(pm.Metadata),
		Offset:   pm.Offset,
		Limit:    pm.Limit,
	}, nil
}

func toGroup(dbu dbGroup) (things.Group, error) {
	return things.Group{
		ID:          dbu.ID,
		Name:        dbu.Name,
		OwnerID:     dbu.OwnerID,
		Description: dbu.Description,
		Metadata:    things.GroupMetadata(dbu.Metadata),
		UpdatedAt:   dbu.UpdatedAt,
		CreatedAt:   dbu.CreatedAt,
	}, nil
}

type dbGroupRelation struct {
	GroupID   sql.NullString `db:"group_id"`
	MemberID  sql.NullString `db:"member_id"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
}

func toDBGroupRelation(memberID, groupID string) (dbGroupRelation, error) {
	var grID sql.NullString
	if groupID != "" {
		grID = sql.NullString{String: groupID, Valid: true}
	}

	var mID sql.NullString
	if memberID != "" {
		mID = sql.NullString{String: memberID, Valid: true}
	}

	return dbGroupRelation{
		GroupID:  grID,
		MemberID: mID,
	}, nil
}

func toGroupRelation(dbgr dbGroupRelation) (things.GroupRelation, error) {
	return things.GroupRelation{
		GroupID:   dbgr.GroupID.String,
		MemberID:  dbgr.MemberID.String,
		CreatedAt: dbgr.CreatedAt,
		UpdatedAt: dbgr.UpdatedAt,
	}, nil
}

func getGroupsMetadataQuery(db string, m things.GroupMetadata) (mb []byte, mq string, err error) {
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
