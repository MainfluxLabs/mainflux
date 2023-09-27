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

func (gr groupRepository) RetrieveGroupThings(ctx context.Context, ownerID, groupID string, pm things.PageMetadata) (things.GroupThingsPage, error) {
	_, mq, err := dbutil.GetMetadataQuery("groups", pm.Metadata)
	if err != nil {
		return things.GroupThingsPage{}, errors.Wrap(things.ErrRetrieveGroupThings, err)
	}

	olq := "LIMIT :limit OFFSET :offset"
	if pm.Limit == 0 {
		olq = ""
	}

	var q, qc string
	switch pm.Unassigned {
	case true:
		q = fmt.Sprintf(`SELECT t.id, t.owner, t.name, t.metadata, t.key
			FROM  things t
			WHERE t.owner = :owner_id
			AND t.id NOT IN (SELECT gr.thing_id FROM group_things gr)
			%s %s;`, mq, olq)
		qc = fmt.Sprintf(`SELECT COUNT(*) FROM things t
			WHERE t.owner = :owner_id
			AND t.id NOT IN (SELECT gr.thing_id FROM group_things gr) %s;`, mq)
	default:
		q = fmt.Sprintf(`SELECT t.id, t.owner, t.name, t.metadata, t.key
			FROM group_things gr, things t
			WHERE gr.group_id = :group_id and gr.thing_id = t.id
			%s %s;`, mq, olq)
		qc = fmt.Sprintf(`SELECT COUNT(*) FROM group_things gr WHERE gr.group_id = :group_id %s;`, mq)
	}

	params := map[string]interface{}{
		"owner_id": ownerID,
		"group_id": groupID,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
		"metadata": pm.Metadata,
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.GroupThingsPage{}, errors.Wrap(things.ErrRetrieveGroupThings, err)
	}
	defer rows.Close()

	var items []things.Thing
	for rows.Next() {
		dbmem := dbThing{}
		if err := rows.StructScan(&dbmem); err != nil {
			return things.GroupThingsPage{}, errors.Wrap(things.ErrRetrieveGroupThings, err)
		}

		th, err := toThing(dbmem)
		if err != nil {
			return things.GroupThingsPage{}, err
		}

		items = append(items, th)
	}

	total, err := total(ctx, gr.db, qc, params)
	if err != nil {
		return things.GroupThingsPage{}, errors.Wrap(things.ErrRetrieveGroupThings, err)
	}

	page := things.GroupThingsPage{
		Things: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (gr groupRepository) RetrieveGroupThingsByChannel(ctx context.Context, groupID, channelID string, pm things.PageMetadata) (things.GroupThingsPage, error) {
	_, mq, err := dbutil.GetMetadataQuery("groups", pm.Metadata)
	if err != nil {
		return things.GroupThingsPage{}, errors.Wrap(things.ErrRetrieveGroupThings, err)
	}

	olq := "LIMIT :limit OFFSET :offset"
	if pm.Limit == 0 {
		olq = ""
	}

	q := fmt.Sprintf(`SELECT t.id, t.owner, t.name, t.metadata, t.key
		FROM group_things gr, things t, group_channels gc
		WHERE gr.group_id = :group_id and gr.thing_id = t.id and gc.group_id = gr.group_id and gc.channel_id = :channel_id and t.id
		NOT IN (SELECT c.thing_id FROM connections c)
		%s %s;`, mq, olq)

	qc := fmt.Sprintf(`SELECT COUNT(*) FROM group_things gr, things t, group_channels gc
		WHERE gr.group_id = :group_id and gr.thing_id = t.id and gc.group_id = gr.group_id and gc.channel_id = :channel_id and t.id
		NOT IN (SELECT ct.thing_id FROM connections ct)
		%s;`, mq)

	params := map[string]interface{}{
		"group_id":   groupID,
		"channel_id": channelID,
		"limit":      pm.Limit,
		"offset":     pm.Offset,
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.GroupThingsPage{}, errors.Wrap(things.ErrRetrieveGroupThingsByChannel, err)
	}
	defer rows.Close()

	var items []things.Thing
	for rows.Next() {
		dbt := dbThing{}
		if err := rows.StructScan(&dbt); err != nil {
			return things.GroupThingsPage{}, errors.Wrap(things.ErrRetrieveGroupThingsByChannel, err)
		}

		th, err := toThing(dbt)
		if err != nil {
			return things.GroupThingsPage{}, err
		}

		items = append(items, th)
	}

	total, err := total(ctx, gr.db, qc, params)
	if err != nil {
		return things.GroupThingsPage{}, errors.Wrap(things.ErrRetrieveGroupThingsByChannel, err)
	}

	page := things.GroupThingsPage{
		Things: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (gr groupRepository) RetrieveThingMembership(ctx context.Context, thingID string) (string, error) {
	q := `SELECT group_id FROM group_things WHERE thing_id = :thing_id;`

	params := map[string]interface{}{"thing_id": thingID}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return "", errors.Wrap(things.ErrRetrieveThingMembership, err)
	}
	defer rows.Close()

	var groupID string
	for rows.Next() {
		if err := rows.Scan(&groupID); err != nil {
			return "", errors.Wrap(things.ErrRetrieveThingMembership, err)
		}
	}

	return groupID, nil
}

func (gr groupRepository) RetrieveAllThingRelations(ctx context.Context) ([]things.GroupThingRelation, error) {
	q := `SELECT group_id, thing_id, created_at, updated_at FROM group_things`

	rows, err := gr.db.NamedQueryContext(ctx, q, map[string]interface{}{})
	if err != nil {
		return nil, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	grRel := []things.GroupThingRelation{}
	for rows.Next() {
		dbg := dbThingRelation{}
		if err := rows.StructScan(&dbg); err != nil {
			return nil, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		gr, err := toThingRelation(dbg)
		if err != nil {
			return nil, errors.Wrap(errors.ErrRetrieveEntity, err)
		}
		grRel = append(grRel, gr)
	}

	return grRel, nil
}

func (gr groupRepository) AssignThing(ctx context.Context, groupID string, ids ...string) error {
	tx, err := gr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(things.ErrAssignGroupThing, err)
	}

	qIns := `INSERT INTO group_things (group_id, thing_id, created_at, updated_at)
		VALUES(:group_id, :thing_id, :created_at, :updated_at)`

	for _, id := range ids {
		created := time.Now()

		dbt, err := toDBThingRelation(id, groupID)
		if err != nil {
			return errors.Wrap(things.ErrAssignGroupThing, err)
		}
		dbt.CreatedAt = created
		dbt.UpdatedAt = created

		if _, err := tx.NamedExecContext(ctx, qIns, dbt); err != nil {
			tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.ForeignKeyViolation:
					return errors.Wrap(errors.ErrConflict, errors.New(pgErr.Detail))
				case pgerrcode.UniqueViolation:
					return errors.Wrap(things.ErrThingAlreadyAssigned, errors.New(pgErr.Detail))
				}
			}

			return errors.Wrap(things.ErrAssignGroupThing, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(things.ErrAssignGroupThing, err)
	}

	return nil
}

func (gr groupRepository) UnassignThing(ctx context.Context, groupID string, ids ...string) error {
	tx, err := gr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(things.ErrUnassignGroupThing, err)
	}

	qDel := `DELETE from group_things WHERE group_id = :group_id AND thing_id = :thing_id`

	for _, id := range ids {
		dbt, err := toDBThingRelation(id, groupID)
		if err != nil {
			return errors.Wrap(things.ErrUnassignGroupThing, err)
		}

		if _, err := tx.NamedExecContext(ctx, qDel, dbt); err != nil {
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

			return errors.Wrap(things.ErrUnassignGroupThing, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(things.ErrUnassignGroupThing, err)
	}

	return nil
}

func (gr groupRepository) RetrieveGroupChannels(ctx context.Context, ownerID, groupID string, pm things.PageMetadata) (things.GroupChannelsPage, error) {
	_, mq, err := dbutil.GetMetadataQuery("groups", pm.Metadata)
	if err != nil {
		return things.GroupChannelsPage{}, errors.Wrap(things.ErrRetrieveGroupChannels, err)
	}

	olq := "LIMIT :limit OFFSET :offset"
	if pm.Limit == 0 {
		olq = ""
	}

	var q, qc string
	switch pm.Unassigned {
	case true:
		q = fmt.Sprintf(`SELECT c.id, c.owner, c.name, c.metadata
			FROM channels c
			WHERE c.owner = :owner_id
			AND c.id NOT IN (SELECT gr.channel_id FROM group_channels gr)
			%s %s;`, mq, olq)
		qc = fmt.Sprintf(`SELECT COUNT(*) FROM channels c
			WHERE c.owner = :owner_id
			AND c.id NOT IN (SELECT gr.channel_id FROM group_channels gr) %s;`, mq)
	default:
		q = fmt.Sprintf(`SELECT c.id, c.owner, c.name, c.metadata
			FROM group_channels gr, channels c
			WHERE gr.group_id = :group_id and gr.channel_id = c.id
			%s %s;`, mq, olq)
		qc = fmt.Sprintf(`SELECT COUNT(*) FROM group_channels gr WHERE gr.group_id = :group_id %s;`, mq)
	}

	params := map[string]interface{}{
		"owner_id": ownerID,
		"group_id": groupID,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
		"metadata": pm.Metadata,
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.GroupChannelsPage{}, errors.Wrap(things.ErrRetrieveGroupChannels, err)
	}
	defer rows.Close()

	var items []things.Channel
	for rows.Next() {
		dbch := dbChannel{}
		if err := rows.StructScan(&dbch); err != nil {
			return things.GroupChannelsPage{}, errors.Wrap(things.ErrRetrieveGroupChannels, err)
		}

		ch := toChannel(dbch)

		items = append(items, ch)
	}

	total, err := total(ctx, gr.db, qc, params)
	if err != nil {
		return things.GroupChannelsPage{}, errors.Wrap(things.ErrRetrieveGroupChannels, err)
	}

	page := things.GroupChannelsPage{
		Channels: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (gr groupRepository) RetrieveChannelMembership(ctx context.Context, channelID string) (string, error) {
	q := `SELECT group_id FROM group_channels WHERE channel_id = :channel_id;`

	params := map[string]interface{}{"channel_id": channelID}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return "", errors.Wrap(things.ErrRetrieveChannelMembership, err)
	}
	defer rows.Close()

	var groupID string
	for rows.Next() {
		if err := rows.Scan(&groupID); err != nil {
			return "", errors.Wrap(things.ErrRetrieveChannelMembership, err)
		}
	}

	return groupID, nil
}

func (gr groupRepository) AssignChannel(ctx context.Context, groupID string, ids ...string) error {
	tx, err := gr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(things.ErrAssignGroupChannel, err)
	}

	qIns := `INSERT INTO group_channels (group_id, channel_id, created_at, updated_at)
			VALUES(:group_id, :channel_id, :created_at, :updated_at)`

	for _, id := range ids {
		created := time.Now()

		dgc, err := todbGroupChannels(id, groupID)
		if err != nil {
			return errors.Wrap(things.ErrAssignGroupChannel, err)
		}
		dgc.CreatedAt = created
		dgc.UpdatedAt = created

		if _, err := tx.NamedExecContext(ctx, qIns, dgc); err != nil {
			tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.ForeignKeyViolation:
					return errors.Wrap(errors.ErrConflict, errors.New(pgErr.Detail))
				case pgerrcode.UniqueViolation:
					return errors.Wrap(things.ErrChannelAlreadyAssigned, errors.New(pgErr.Detail))
				}
			}

			return errors.Wrap(things.ErrAssignGroupChannel, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(things.ErrAssignGroupChannel, err)
	}

	return nil
}

func (gr groupRepository) UnassignChannel(ctx context.Context, groupID string, ids ...string) error {
	tx, err := gr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(things.ErrUnassignGroupChannel, err)
	}

	qDel := `DELETE from group_channels WHERE group_id = :group_id AND channel_id = :channel_id`

	for _, id := range ids {
		dgc, err := todbGroupChannels(id, groupID)
		if err != nil {
			return errors.Wrap(things.ErrUnassignGroupChannel, err)
		}

		if _, err := tx.NamedExecContext(ctx, qDel, dgc); err != nil {
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

			return errors.Wrap(things.ErrUnassignGroupChannel, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(things.ErrUnassignGroupChannel, err)
	}

	return nil
}

func (gr groupRepository) retrieve(ctx context.Context, ownerID string, pm things.PageMetadata) (things.GroupPage, error) {
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

type dbThingRelation struct {
	GroupID   sql.NullString `db:"group_id"`
	ThingID   sql.NullString `db:"thing_id"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
}

func toDBThingRelation(thingID, groupID string) (dbThingRelation, error) {
	var grID sql.NullString
	if groupID != "" {
		grID = sql.NullString{String: groupID, Valid: true}
	}

	var tID sql.NullString
	if thingID != "" {
		tID = sql.NullString{String: thingID, Valid: true}
	}

	return dbThingRelation{
		GroupID: grID,
		ThingID: tID,
	}, nil
}

func toThingRelation(dbgr dbThingRelation) (things.GroupThingRelation, error) {
	return things.GroupThingRelation{
		GroupID:   dbgr.GroupID.String,
		ThingID:   dbgr.ThingID.String,
		CreatedAt: dbgr.CreatedAt,
		UpdatedAt: dbgr.UpdatedAt,
	}, nil
}

type dbGroupChannels struct {
	GroupID   sql.NullString `db:"group_id"`
	ChannelID sql.NullString `db:"channel_id"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
}

func todbGroupChannels(channelID, groupID string) (dbGroupChannels, error) {
	var grID sql.NullString
	if groupID != "" {
		grID = sql.NullString{String: groupID, Valid: true}
	}

	var chID sql.NullString
	if channelID != "" {
		chID = sql.NullString{String: channelID, Valid: true}
	}

	return dbGroupChannels{
		GroupID:   grID,
		ChannelID: chID,
	}, nil
}
