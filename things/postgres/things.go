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
		return []things.Thing{}, errors.Wrap(errors.ErrCreateEntity, err)
	}

	q := `INSERT INTO things (id, group_id, profile_id, name, key, metadata)
		  VALUES (:id, :group_id, :profile_id, :name, :key, :metadata);`

	for _, thing := range ths {
		dbth, err := toDBThing(thing)
		if err != nil {
			return []things.Thing{}, errors.Wrap(errors.ErrCreateEntity, err)
		}

		if _, err := tx.NamedExecContext(ctx, q, dbth); err != nil {
			tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return []things.Thing{}, errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return []things.Thing{}, errors.Wrap(errors.ErrConflict, err)
				case pgerrcode.StringDataRightTruncationDataException:
					return []things.Thing{}, errors.Wrap(errors.ErrMalformedEntity, err)
				}
			}

			return []things.Thing{}, errors.Wrap(errors.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return []things.Thing{}, errors.Wrap(errors.ErrCreateEntity, err)
	}

	return ths, nil
}

func (tr thingRepository) Update(ctx context.Context, t things.Thing) error {
	nq := ""
	if t.Name != "" {
		nq = "name = :name,"
	}
	q := fmt.Sprintf(`UPDATE things SET %s metadata = :metadata WHERE id = :id;`, nq)

	dbth, err := toDBThing(t)
	if err != nil {
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	res, errdb := tr.db.NamedExecContext(ctx, q, dbth)
	if errdb != nil {
		pgErr, ok := errdb.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(errors.ErrMalformedEntity, errdb)
			case pgerrcode.StringDataRightTruncationDataException:
				return errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}

		return errors.Wrap(errors.ErrUpdateEntity, errdb)
	}

	cnt, errdb := res.RowsAffected()
	if errdb != nil {
		return errors.Wrap(errors.ErrUpdateEntity, errdb)
	}

	if cnt == 0 {
		return errors.ErrNotFound
	}

	return nil
}

func (tr thingRepository) UpdateKey(ctx context.Context, id, key string) error {
	q := `UPDATE things SET key = :key WHERE id = :id;`

	dbth := dbThing{
		ID:  id,
		Key: key,
	}

	res, err := tr.db.NamedExecContext(ctx, q, dbth)
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

	cnt, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	if cnt == 0 {
		return errors.ErrNotFound
	}

	return nil
}

func (tr thingRepository) RetrieveByID(ctx context.Context, id string) (things.Thing, error) {
	q := `SELECT group_id, profile_id, name, key, metadata FROM things WHERE id = $1;`

	dbth := dbThing{ID: id}

	if err := tr.db.QueryRowxContext(ctx, q, id).StructScan(&dbth); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		//  If there is no result or ID is in an invalid format, return ErrNotFound.
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return things.Thing{}, errors.Wrap(errors.ErrNotFound, err)
		}
		return things.Thing{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	return toThing(dbth)
}

func (tr thingRepository) RetrieveByKey(ctx context.Context, key string) (string, error) {
	q := `SELECT id FROM things WHERE key = $1;`

	var id string
	if err := tr.db.QueryRowxContext(ctx, q, key).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return "", errors.Wrap(errors.ErrNotFound, err)
		}
		return "", errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	return id, nil
}

func (tr thingRepository) RetrieveByGroupIDs(ctx context.Context, groupIDs []string, pm apiutil.PageMetadata) (things.ThingsPage, error) {
	if len(groupIDs) == 0 {
		return things.ThingsPage{}, nil
	}

	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	giq := getGroupIDsQuery(groupIDs)
	nq, name := dbutil.GetNameQuery(pm.Name)
	m, mq, err := dbutil.GetMetadataQuery("", pm.Metadata)
	if err != nil {
		return things.ThingsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	whereClause := dbutil.BuildWhereClause(giq, nq, mq)
	query := fmt.Sprintf(`SELECT id, group_id, profile_id, name, key, metadata FROM things %s ORDER BY %s %s %s`, whereClause, pm.Order, strings.ToUpper(pm.Dir), olq)
	cquery := fmt.Sprintf(`SELECT COUNT(*) FROM things %s;`, whereClause)

	params := map[string]interface{}{
		"name":     name,
		"metadata": m,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	return tr.retrieve(ctx, query, cquery, params)
}

func (tr thingRepository) RetrieveAll(ctx context.Context) ([]things.Thing, error) {
	query := "SELECT id, group_id, profile_id, name, key, metadata FROM things"

	var items []dbThing
	err := tr.db.SelectContext(ctx, &items, query)
	if err != nil {
		return nil, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	var ths []things.Thing
	for _, i := range items {
		th, err := toThing(i)
		if err != nil {
			return []things.Thing{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		ths = append(ths, th)
	}

	return ths, nil
}

func (tr thingRepository) RetrieveByAdmin(ctx context.Context, pm apiutil.PageMetadata) (things.ThingsPage, error) {
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	nq, name := dbutil.GetNameQuery(pm.Name)
	m, mq, err := dbutil.GetMetadataQuery("", pm.Metadata)
	if err != nil {
		return things.ThingsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	whereClause := dbutil.BuildWhereClause(nq, mq)
	query := fmt.Sprintf(`SELECT id, group_id, profile_id, name, key, metadata FROM things %s ORDER BY %s %s %s`, whereClause, pm.Order, strings.ToUpper(pm.Dir), olq)
	cquery := fmt.Sprintf(`SELECT COUNT(*) FROM things %s;`, whereClause)

	params := map[string]interface{}{
		"name":     name,
		"metadata": m,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	return tr.retrieve(ctx, query, cquery, params)
}

func (tr thingRepository) RetrieveByProfile(ctx context.Context, prID string, pm apiutil.PageMetadata) (things.ThingsPage, error) {
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	// Verify if UUID format is valid to avoid internal Postgres error
	if _, err := uuid.FromString(prID); err != nil {
		return things.ThingsPage{}, errors.Wrap(errors.ErrNotFound, err)
	}

	query := fmt.Sprintf(`SELECT id, group_id, name, key, metadata FROM things 
				WHERE profile_id = :profile_id ORDER BY %s %s %s;`, pm.Order, strings.ToUpper(pm.Dir), olq)
	cquery := `SELECT COUNT(*) FROM things WHERE profile_id = :profile_id;`

	params := map[string]interface{}{
		"profile_id": prID,
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
			return errors.Wrap(errors.ErrRemoveEntity, err)
		}
	}

	return nil
}

func (tr thingRepository) retrieve(ctx context.Context, query, cquery string, params map[string]interface{}) (things.ThingsPage, error) {
	rows, err := tr.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return things.ThingsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []things.Thing
	for rows.Next() {
		dbth := dbThing{}
		if err := rows.StructScan(&dbth); err != nil {
			return things.ThingsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		th, err := toThing(dbth)
		if err != nil {
			return things.ThingsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		items = append(items, th)
	}

	total, err := dbutil.Total(ctx, tr.db, cquery, params)
	if err != nil {
		return things.ThingsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	page := things.ThingsPage{
		Things: items,
		PageMetadata: apiutil.PageMetadata{
			Total:  total,
			Offset: params["offset"].(uint64),
			Limit:  params["limit"].(uint64),
		},
	}

	return page, nil
}

type dbThing struct {
	ID        string `db:"id"`
	GroupID   string `db:"group_id"`
	ProfileID string `db:"profile_id"`
	Name      string `db:"name"`
	Key       string `db:"key"`
	Metadata  []byte `db:"metadata"`
}

func toDBThing(th things.Thing) (dbThing, error) {
	data := []byte("{}")
	if len(th.Metadata) > 0 {
		b, err := json.Marshal(th.Metadata)
		if err != nil {
			return dbThing{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
		data = b
	}

	return dbThing{
		ID:        th.ID,
		GroupID:   th.GroupID,
		ProfileID: th.ProfileID,
		Name:      th.Name,
		Key:       th.Key,
		Metadata:  data,
	}, nil
}

func toThing(dbth dbThing) (things.Thing, error) {
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(dbth.Metadata), &metadata); err != nil {
		return things.Thing{}, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return things.Thing{
		ID:        dbth.ID,
		GroupID:   dbth.GroupID,
		ProfileID: dbth.ProfileID,
		Name:      dbth.Name,
		Key:       dbth.Key,
		Metadata:  metadata,
	}, nil
}
