// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux/internal/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ things.GroupRepository = (*groupRepository)(nil)

type groupRepository struct {
	db Database
}

// NewGroupRepository instantiates a PostgreSQL implementation of group
// repository.
func NewGroupRepository(db Database) things.GroupRepository {
	return &groupRepository{
		db: db,
	}
}

func (gr groupRepository) Save(ctx context.Context, g things.Group) (things.Group, error) {
	q := `INSERT INTO groups (name, description, id, owner_id, org_id, metadata, created_at, updated_at)
		  VALUES (:name, :description, :id, :owner_id, :org_id, :metadata, :created_at, :updated_at)
		  RETURNING id, name, owner_id, org_id, description, metadata, created_at, updated_at`

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

func (gr groupRepository) Remove(ctx context.Context, groupIDs ...string) error {
	qd := `DELETE FROM groups WHERE id = :id`

	for _, groupID := range groupIDs {
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
	}

	return nil
}

func (gr groupRepository) RetrieveAll(ctx context.Context) ([]things.Group, error) {
	gp, err := gr.retrieve(ctx, "", "", things.PageMetadata{})
	if err != nil {
		return nil, err
	}

	return gp.Groups, nil
}

func (gr groupRepository) RetrieveByID(ctx context.Context, id string) (things.Group, error) {
	dbu := dbGroup{
		ID: id,
	}
	q := `SELECT id, name, owner_id, org_id, description, metadata, created_at, updated_at FROM groups WHERE id = $1`
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

func (gr groupRepository) RetrieveByOwner(ctx context.Context, ownerID, orgID string, pm things.PageMetadata) (things.GroupPage, error) {
	if ownerID == "" {
		return things.GroupPage{}, errors.ErrRetrieveEntity
	}

	return gr.retrieve(ctx, ownerID, orgID, pm)
}

func (gr groupRepository) RetrieveByAdmin(ctx context.Context, orgID string, pm things.PageMetadata) (things.GroupPage, error) {
	return gr.retrieve(ctx, "", orgID, pm)
}

func (gr groupRepository) RetrieveThingsByGroup(ctx context.Context, groupID string, pm things.PageMetadata) (things.ThingsPage, error) {
	_, mq, err := dbutil.GetMetadataQuery("groups", pm.Metadata)
	if err != nil {
		return things.ThingsPage{}, errors.Wrap(things.ErrRetrieveGroupThings, err)
	}

	olq := "LIMIT :limit OFFSET :offset"
	if pm.Limit == 0 {
		olq = ""
	}

	q := fmt.Sprintf(`SELECT id, owner_id, group_id, name, metadata, key FROM things
			WHERE group_id = :group_id %s %s;`, mq, olq)
	qc := fmt.Sprintf(`SELECT COUNT(*) FROM things WHERE group_id = :group_id %s;`, mq)

	params := map[string]interface{}{
		"group_id": groupID,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
		"metadata": pm.Metadata,
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.ThingsPage{}, errors.Wrap(things.ErrRetrieveGroupThings, err)
	}
	defer rows.Close()

	var items []things.Thing
	for rows.Next() {
		dbmem := dbThing{}
		if err := rows.StructScan(&dbmem); err != nil {
			return things.ThingsPage{}, errors.Wrap(things.ErrRetrieveGroupThings, err)
		}

		th, err := toThing(dbmem)
		if err != nil {
			return things.ThingsPage{}, err
		}

		items = append(items, th)
	}

	total, err := total(ctx, gr.db, qc, params)
	if err != nil {
		return things.ThingsPage{}, errors.Wrap(things.ErrRetrieveGroupThings, err)
	}

	page := things.ThingsPage{
		Things: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (gr groupRepository) RetrieveChannelsByGroup(ctx context.Context, groupID string, pm things.PageMetadata) (things.ChannelsPage, error) {
	_, mq, err := dbutil.GetMetadataQuery("groups", pm.Metadata)
	if err != nil {
		return things.ChannelsPage{}, errors.Wrap(things.ErrRetrieveGroupChannels, err)
	}

	olq := "LIMIT :limit OFFSET :offset"
	if pm.Limit == 0 {
		olq = ""
	}

	q := fmt.Sprintf(`SELECT id, owner_id, group_id, name, metadata FROM channels
			WHERE group_id = :group_id %s %s;`, mq, olq)
	qc := fmt.Sprintf(`SELECT COUNT(*) FROM channels WHERE group_id = :group_id %s;`, mq)

	params := map[string]interface{}{
		"group_id": groupID,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
		"metadata": pm.Metadata,
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.ChannelsPage{}, errors.Wrap(things.ErrRetrieveGroupChannels, err)
	}
	defer rows.Close()

	var items []things.Channel
	for rows.Next() {
		dbch := dbChannel{}
		if err := rows.StructScan(&dbch); err != nil {
			return things.ChannelsPage{}, errors.Wrap(things.ErrRetrieveGroupChannels, err)
		}

		ch := toChannel(dbch)

		items = append(items, ch)
	}

	total, err := total(ctx, gr.db, qc, params)
	if err != nil {
		return things.ChannelsPage{}, errors.Wrap(things.ErrRetrieveGroupChannels, err)
	}

	page := things.ChannelsPage{
		Channels: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (gr groupRepository) retrieve(ctx context.Context, ownerID, orgID string, pm things.PageMetadata) (things.GroupPage, error) {
	var ownq string
	if ownerID != "" {
		ownq = "owner_id = :owner_id"
	}

	nq, name := dbutil.GetNameQuery(pm.Name)

	meta, mq, err := dbutil.GetMetadataQuery("", pm.Metadata)
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

	if orgID != "" {
		orgq := "org_id = :org_id"
		query = append(query, orgq)
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
		"org_id":   orgID,
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
	ID          string    `db:"id"`
	OwnerID     string    `db:"owner_id"`
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
		OwnerID:     g.OwnerID,
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
		OwnerID:     dbu.OwnerID,
		OrgID:       dbu.OrgID,
		Description: dbu.Description,
		Metadata:    things.GroupMetadata(dbu.Metadata),
		UpdatedAt:   dbu.UpdatedAt,
		CreatedAt:   dbu.CreatedAt,
	}, nil
}
