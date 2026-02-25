// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ things.ThingRepository = (*thingRepository)(nil)

type thingRepository struct {
	db dbutil.Database
}

// NewThingRepository instantiates a PostgreSQL implementation of thing
// repository.
func NewThingRepository(db dbutil.Database) things.ThingRepository {
	return &thingRepository{
		db: db,
	}
}

func (tr thingRepository) Save(ctx context.Context, ths ...things.Thing) ([]things.Thing, error) {
	tx, err := tr.db.BeginTxx(ctx, nil)
	if err != nil {
		return []things.Thing{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	q := `INSERT INTO things (id, group_id, profile_id, name, key, external_key, metadata)
		  VALUES (:id, :group_id, :profile_id, :name, :key, :external_key, :metadata);`

	for _, thing := range ths {
		dbth, err := toDBThing(thing)
		if err != nil {
			return []things.Thing{}, errors.Wrap(dbutil.ErrCreateEntity, err)
		}

		if _, err := tx.NamedExecContext(ctx, q, dbth); err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return []things.Thing{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return []things.Thing{}, errors.Wrap(dbutil.ErrConflict, err)
				case pgerrcode.StringDataRightTruncationDataException:
					return []things.Thing{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
				}
			}

			return []things.Thing{}, errors.Wrap(dbutil.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return []things.Thing{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return ths, nil
}

func (tr thingRepository) Update(ctx context.Context, t things.Thing) error {
	return tr.update(ctx, t)
}

func (tr thingRepository) UpdateGroupAndProfile(ctx context.Context, t things.Thing) error {
	return tr.update(ctx, t)
}

func (tr thingRepository) RetrieveByID(ctx context.Context, id string) (things.Thing, error) {
	q := `SELECT group_id, profile_id, name, key, external_key, metadata FROM things WHERE id = $1;`

	dbth := dbThing{ID: id}

	if err := tr.db.QueryRowxContext(ctx, q, id).StructScan(&dbth); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		//  If there is no result or ID is in an invalid format, return ErrNotFound.
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return things.Thing{}, errors.Wrap(dbutil.ErrNotFound, err)
		}
		return things.Thing{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toThing(dbth)
}

func (tr thingRepository) RetrieveByKey(ctx context.Context, key things.ThingKey) (string, error) {
	query := `
		SELECT id FROM things WHERE %s = $1;	
	`
	switch key.Type {
	case things.KeyTypeInternal:
		query = fmt.Sprintf(query, "key")
	case things.KeyTypeExternal:
		query = fmt.Sprintf(query, "external_key")
	default:
		return "", apiutil.ErrInvalidThingKeyType
	}

	var id string
	if err := tr.db.QueryRowxContext(ctx, query, key.Value).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return "", errors.Wrap(dbutil.ErrNotFound, err)
		}

		return "", errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return id, nil
}

func (tr thingRepository) RetrieveByGroups(ctx context.Context, groupIDs []string, pm apiutil.PageMetadata) (things.ThingsPage, error) {
	if len(groupIDs) == 0 {
		return things.ThingsPage{}, nil
	}

	oq := dbutil.GetOrderQuery(pm.Order, things.AllowedOrders)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	giq := dbutil.GetGroupIDsQuery(groupIDs)
	nq, name := dbutil.GetNameQuery(pm.Name)
	m, mq, err := dbutil.GetMetadataQuery(pm.Metadata)
	if err != nil {
		return things.ThingsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	whereClause := dbutil.BuildWhereClause(giq, nq, mq)
	query := fmt.Sprintf(`SELECT id, group_id, profile_id, name, key, external_key, metadata FROM things %s ORDER BY %s %s %s`, whereClause, oq, dq, olq)
	cquery := fmt.Sprintf(`SELECT COUNT(*) FROM things %s;`, whereClause)

	params := map[string]any{
		"name":     name,
		"metadata": m,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	return tr.retrieve(ctx, query, cquery, params)
}

func (tr thingRepository) BackupAll(ctx context.Context) ([]things.Thing, error) {
	query := "SELECT id, group_id, profile_id, name, key, external_key, metadata FROM things"

	var items []dbThing
	err := tr.db.SelectContext(ctx, &items, query)
	if err != nil {
		return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	var ths []things.Thing
	for _, i := range items {
		th, err := toThing(i)
		if err != nil {
			return []things.Thing{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		ths = append(ths, th)
	}

	return ths, nil
}

func (tr thingRepository) RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (things.ThingsPage, error) {
	oq := dbutil.GetOrderQuery(pm.Order, things.AllowedOrders)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	nq, name := dbutil.GetNameQuery(pm.Name)
	m, mq, err := dbutil.GetMetadataQuery(pm.Metadata)
	if err != nil {
		return things.ThingsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	whereClause := dbutil.BuildWhereClause(nq, mq)
	query := fmt.Sprintf(`SELECT id, group_id, profile_id, name, key, external_key, metadata FROM things %s ORDER BY %s %s %s`, whereClause, oq, dq, olq)
	cquery := fmt.Sprintf(`SELECT COUNT(*) FROM things %s;`, whereClause)

	params := map[string]any{
		"name":     name,
		"metadata": m,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	return tr.retrieve(ctx, query, cquery, params)
}

func (tr thingRepository) RetrieveByProfile(ctx context.Context, prID string, pm apiutil.PageMetadata) (things.ThingsPage, error) {
	oq := dbutil.GetOrderQuery(pm.Order, things.AllowedOrders)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	// Verify if UUID format is valid to avoid internal Postgres error
	if _, err := uuid.FromString(prID); err != nil {
		return things.ThingsPage{}, errors.Wrap(dbutil.ErrNotFound, err)
	}

	baseCondition := "profile_id = :profile_id"
	m, mq, err := dbutil.GetMetadataQuery(pm.Metadata)
	if err != nil {
		return things.ThingsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	whereClause := dbutil.BuildWhereClause(baseCondition, mq)
	query := fmt.Sprintf(`SELECT id, group_id, name, key, external_key, metadata FROM things %s ORDER BY %s %s %s;`, whereClause, oq, dq, olq)
	cquery := fmt.Sprintf(`SELECT COUNT(*) FROM things %s;`, whereClause)

	params := map[string]any{
		"profile_id": prID,
		"metadata":   m,
		"limit":      pm.Limit,
		"offset":     pm.Offset,
	}

	return tr.retrieve(ctx, query, cquery, params)
}

func (tr thingRepository) Remove(ctx context.Context, ids ...string) error {
	for _, id := range ids {
		dbth := dbThing{
			ID: id,
		}
		q := `DELETE FROM things WHERE id = :id;`
		_, err := tr.db.NamedExecContext(ctx, q, dbth)
		if err != nil {
			return errors.Wrap(dbutil.ErrRemoveEntity, err)
		}
	}

	return nil
}

func (tr thingRepository) UpdateExternalKey(ctx context.Context, key, thingID string) error {
	query := `
		UPDATE things
		SET external_key = :key
		WHERE id = :thingID
	`

	params := map[string]any{
		"thingID": thingID,
		"key":     key,
	}

	res, err := tr.db.NamedExecContext(ctx, query, params)
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(dbutil.ErrMalformedEntity, err)
			case pgerrcode.UniqueViolation:
				return errors.Wrap(dbutil.ErrConflict, err)
			}
		}

		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	if cnt == 0 {
		return dbutil.ErrNotFound
	}

	return nil
}

func (tr thingRepository) RemoveExternalKey(ctx context.Context, thingID string) error {
	query := `
		UPDATE things
		SET external_key = NULL
		WHERE id = :thingID
	`

	params := map[string]any{
		"thingID": thingID,
	}

	_, err := tr.db.NamedExecContext(ctx, query, params)
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}

		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (tr thingRepository) retrieve(ctx context.Context, query, cquery string, params map[string]any) (things.ThingsPage, error) {
	rows, err := tr.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return things.ThingsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []things.Thing
	for rows.Next() {
		dbth := dbThing{}
		if err := rows.StructScan(&dbth); err != nil {
			return things.ThingsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		th, err := toThing(dbth)
		if err != nil {
			return things.ThingsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		items = append(items, th)
	}

	total, err := dbutil.Total(ctx, tr.db, cquery, params)
	if err != nil {
		return things.ThingsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	page := things.ThingsPage{
		Things: items,
		Total:  total,
	}

	return page, nil
}

func (tr thingRepository) update(ctx context.Context, t things.Thing) error {
	var fields []string

	if t.Name != "" {
		fields = append(fields, "name = :name")
	}

	if t.Metadata != nil {
		fields = append(fields, "metadata = :metadata")
	}

	if t.ProfileID != "" {
		fields = append(fields, "profile_id = :profile_id")
	}

	if t.GroupID != "" {
		fields = append(fields, "group_id = :group_id")
	}

	if t.Key != "" {
		fields = append(fields, "key = :key")
	}

	columns := strings.Join(fields, ",")
	q := fmt.Sprintf(`UPDATE things SET %s WHERE id = :id;`, columns)

	dbth, err := toDBThing(t)
	if err != nil {
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	res, errdb := tr.db.NamedExecContext(ctx, q, dbth)
	if errdb != nil {
		pgErr, ok := errdb.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(dbutil.ErrMalformedEntity, errdb)
			case pgerrcode.StringDataRightTruncationDataException:
				return errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}

		return errors.Wrap(dbutil.ErrUpdateEntity, errdb)
	}

	cnt, errdb := res.RowsAffected()
	if errdb != nil {
		return errors.Wrap(dbutil.ErrUpdateEntity, errdb)
	}

	if cnt == 0 {
		return dbutil.ErrNotFound
	}

	return nil
}

type dbThing struct {
	ID          string         `db:"id"`
	GroupID     string         `db:"group_id"`
	ProfileID   string         `db:"profile_id"`
	Name        string         `db:"name"`
	Key         string         `db:"key"`
	ExternalKey sql.NullString `db:"external_key"`
	Metadata    []byte         `db:"metadata"`
}

func toDBThing(th things.Thing) (dbThing, error) {
	data := []byte("{}")
	if len(th.Metadata) > 0 {
		b, err := json.Marshal(th.Metadata)
		if err != nil {
			return dbThing{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
		}
		data = b
	}

	return dbThing{
		ID:          th.ID,
		GroupID:     th.GroupID,
		ProfileID:   th.ProfileID,
		Name:        th.Name,
		Key:         th.Key,
		ExternalKey: sql.NullString{String: th.ExternalKey, Valid: len(th.ExternalKey) > 0},
		Metadata:    data,
	}, nil
}

func toThing(dbth dbThing) (things.Thing, error) {
	var metadata map[string]any
	if err := json.Unmarshal([]byte(dbth.Metadata), &metadata); err != nil {
		return things.Thing{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	return things.Thing{
		ID:          dbth.ID,
		GroupID:     dbth.GroupID,
		ProfileID:   dbth.ProfileID,
		Name:        dbth.Name,
		Key:         dbth.Key,
		ExternalKey: dbth.ExternalKey.String,
		Metadata:    metadata,
	}, nil
}
