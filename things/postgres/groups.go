// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ things.GroupRepository = (*groupRepository)(nil)

type groupRepository struct {
	db dbutil.Database
}

// NewGroupRepository instantiates a PostgreSQL implementation of group
// repository.
func NewGroupRepository(db dbutil.Database) things.GroupRepository {
	return &groupRepository{
		db: db,
	}
}

func (gr groupRepository) Save(ctx context.Context, gs ...things.Group) ([]things.Group, error) {
	tx, err := gr.db.BeginTxx(ctx, nil)
	if err != nil {
		return []things.Group{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	q := `INSERT INTO groups (name, description, id, org_id, metadata, created_at, updated_at)
		  VALUES (:name, :description, :id, :org_id, :metadata, :created_at, :updated_at)`

	for _, group := range gs {
		dbg, err := toDBGroup(group)
		if err != nil {
			return []things.Group{}, errors.Wrap(dbutil.ErrCreateEntity, err)
		}

		if _, err := tx.NamedExecContext(ctx, q, dbg); err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return []things.Group{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.ForeignKeyViolation:
					return []things.Group{}, errors.Wrap(dbutil.ErrCreateEntity, err)
				case pgerrcode.UniqueViolation:
					return []things.Group{}, errors.Wrap(dbutil.ErrConflict, err)
				case pgerrcode.StringDataRightTruncationDataException:
					return []things.Group{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
				}
			}

			return []things.Group{}, errors.Wrap(dbutil.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return []things.Group{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return gs, nil
}

func (gr groupRepository) Update(ctx context.Context, g things.Group) (things.Group, error) {
	q := `UPDATE groups SET name = :name, description = :description, metadata = :metadata, updated_at = :updated_at WHERE id = :id
		  RETURNING id, name, description, metadata, created_at, updated_at`

	dbu, err := toDBGroup(g)
	if err != nil {
		return things.Group{}, errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	row, err := gr.db.NamedQueryContext(ctx, q, dbu)
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return things.Group{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
			case pgerrcode.UniqueViolation:
				return things.Group{}, errors.Wrap(dbutil.ErrConflict, err)
			case pgerrcode.StringDataRightTruncationDataException:
				return things.Group{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}
		return things.Group{}, errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	defer row.Close()
	row.Next()
	dbu = dbGroup{}
	if err := row.StructScan(&dbu); err != nil {
		return g, errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	return toGroup(dbu)
}

func (gr groupRepository) Remove(ctx context.Context, groupIDs ...string) error {
	qd := `DELETE FROM groups WHERE id = :id`

	for _, groupID := range groupIDs {
		dbGr := dbGroup{
			ID: groupID,
		}

		res, err := gr.db.NamedExecContext(ctx, qd, dbGr)
		if err != nil {
			pqErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pqErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.ForeignKeyViolation:
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

func (gr groupRepository) RemoveByOrg(ctx context.Context, orgID string) error {
	q := `DELETE FROM groups WHERE org_id = :org_id;`

	dbg := dbGroup{OrgID: orgID}
	if _, err := gr.db.NamedExecContext(ctx, q, dbg); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (gr groupRepository) BackupAll(ctx context.Context) ([]things.Group, error) {
	query := "SELECT id, name, org_id, description, metadata, created_at, updated_at FROM groups"

	var items []dbGroup
	err := gr.db.SelectContext(ctx, &items, query)
	if err != nil {
		return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	var groups []things.Group
	for _, i := range items {
		gr, err := toGroup(i)
		if err != nil {
			return []things.Group{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		groups = append(groups, gr)
	}

	return groups, nil
}

func (gr groupRepository) BackupByOrg(ctx context.Context, orgID string) ([]things.Group, error) {
	query := "SELECT id, name, org_id, description, metadata, created_at, updated_at FROM groups WHERE org_id = $1"

	var items []dbGroup
	err := gr.db.SelectContext(ctx, &items, query, orgID)
	if err != nil {
		return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	var groups []things.Group
	for _, i := range items {
		gr, err := toGroup(i)
		if err != nil {
			return []things.Group{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		groups = append(groups, gr)
	}

	return groups, nil
}

func (gr groupRepository) RetrieveIDsByOrg(ctx context.Context, orgID string) ([]string, error) {
	q := `SELECT id FROM groups WHERE org_id = :org_id`
	params := map[string]any{
		"org_id": orgID,
	}

	return gr.retrieveIDs(ctx, q, params)
}

func (gr groupRepository) RetrieveIDsByOrgMembership(ctx context.Context, orgID, memberID string) ([]string, error) {
	q := `SELECT g.id FROM groups g
          JOIN group_memberships gm ON g.id = gm.group_id
          WHERE g.org_id = :org_id AND gm.member_id = :member_id`

	params := map[string]any{
		"org_id":    orgID,
		"member_id": memberID,
	}

	return gr.retrieveIDs(ctx, q, params)
}

func (gr groupRepository) RetrieveByID(ctx context.Context, id string) (things.Group, error) {
	dbu := dbGroup{
		ID: id,
	}
	q := `SELECT id, name, org_id, description, metadata, created_at, updated_at FROM groups WHERE id = $1`
	if err := gr.db.QueryRowxContext(ctx, q, id).StructScan(&dbu); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		//  If there is no result or ID is in an invalid format, return ErrNotFound.
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return things.Group{}, errors.Wrap(dbutil.ErrNotFound, err)
		}
		return things.Group{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	return toGroup(dbu)
}

func (gr groupRepository) RetrieveByIDs(ctx context.Context, groupIDs []string, pm apiutil.PageMetadata) (things.GroupPage, error) {
	if len(groupIDs) == 0 {
		return things.GroupPage{}, nil
	}

	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	iq := getIDsQuery(groupIDs)
	nq, name := dbutil.GetNameQuery(pm.Name)
	m, mq, err := dbutil.GetMetadataQuery(pm.Metadata)
	if err != nil {
		return things.GroupPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	whereClause := dbutil.BuildWhereClause(iq, nq, mq)
	query := fmt.Sprintf(`SELECT id, name, org_id, description, metadata, created_at, updated_at FROM groups %s ORDER BY %s %s %s;`, whereClause, oq, dq, olq)
	cquery := fmt.Sprintf(`SELECT COUNT(*) FROM groups %s;`, whereClause)

	params := map[string]any{
		"name":     name,
		"metadata": m,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	return gr.retrieve(ctx, query, cquery, params)
}

func (gr groupRepository) RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (things.GroupPage, error) {
	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	nq, name := dbutil.GetNameQuery(pm.Name)
	m, mq, err := dbutil.GetMetadataQuery(pm.Metadata)
	if err != nil {
		return things.GroupPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	whereClause := dbutil.BuildWhereClause(nq, mq)
	query := fmt.Sprintf(`SELECT id, name, org_id, description, metadata, created_at, updated_at FROM groups %s ORDER BY %s %s %s;`, whereClause, oq, dq, olq)
	cquery := fmt.Sprintf(`SELECT COUNT(*) FROM groups %s;`, whereClause)

	params := map[string]any{
		"name":     name,
		"metadata": m,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	return gr.retrieve(ctx, query, cquery, params)
}

func (gr groupRepository) retrieveIDs(ctx context.Context, query string, params map[string]any) ([]string, error) {
	var ids []string
	rows, err := gr.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (gr groupRepository) retrieve(ctx context.Context, query, cquery string, params map[string]any) (things.GroupPage, error) {
	rows, err := gr.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return things.GroupPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	items := []things.Group{}
	for rows.Next() {
		dbg := dbGroup{}
		if err := rows.StructScan(&dbg); err != nil {
			return things.GroupPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		gr, err := toGroup(dbg)
		if err != nil {
			return things.GroupPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		items = append(items, gr)
	}

	total, err := dbutil.Total(ctx, gr.db, cquery, params)
	if err != nil {
		return things.GroupPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	page := things.GroupPage{
		Groups: items,
		Total:  total,
	}

	return page, nil
}

type dbGroup struct {
	ID          string    `db:"id"`
	OrgID       string    `db:"org_id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	Metadata    dbJSONB   `db:"metadata"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func toDBGroup(g things.Group) (dbGroup, error) {
	return dbGroup{
		ID:          g.ID,
		Name:        g.Name,
		OrgID:       g.OrgID,
		Description: g.Description,
		Metadata:    dbJSONB(g.Metadata),
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}, nil
}

func toGroup(dbu dbGroup) (things.Group, error) {
	return things.Group{
		ID:          dbu.ID,
		Name:        dbu.Name,
		OrgID:       dbu.OrgID,
		Description: dbu.Description,
		Metadata:    things.Metadata(dbu.Metadata),
		UpdatedAt:   dbu.UpdatedAt,
		CreatedAt:   dbu.CreatedAt,
	}, nil
}
