// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ things.ThingRepository = (*thingRepository)(nil)

type thingRepository struct {
	db Database
}

// NewThingRepository instantiates a PostgreSQL implementation of thing
// repository.
func NewThingRepository(db Database) things.ThingRepository {
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
	q := `UPDATE things SET name = :name, metadata = :metadata WHERE id = :id;`

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

func (tr thingRepository) RetrieveByGroupIDs(ctx context.Context, groupIDs []string, pm things.PageMetadata) (things.ThingsPage, error) {
	if len(groupIDs) == 0 {
		return things.ThingsPage{}, nil
	}

	thPage, err := tr.retrieve(ctx, groupIDs, false, pm)
	if err != nil {
		return things.ThingsPage{}, err
	}

	return thPage, nil
}

func (tr thingRepository) RetrieveAll(ctx context.Context) ([]things.Thing, error) {
	thPage, err := tr.retrieve(ctx, []string{}, true, things.PageMetadata{})
	if err != nil {
		return []things.Thing{}, err
	}

	return thPage.Things, nil
}

func (tr thingRepository) RetrieveByAdmin(ctx context.Context, pm things.PageMetadata) (things.ThingsPage, error) {
	return tr.retrieve(ctx, []string{}, false, pm)
}

func (tr thingRepository) RetrieveByProfile(ctx context.Context, prID string, pm things.PageMetadata) (things.ThingsPage, error) {
	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	// Verify if UUID format is valid to avoid internal Postgres error
	if _, err := uuid.FromString(prID); err != nil {
		return things.ThingsPage{}, errors.Wrap(errors.ErrNotFound, err)
	}

	var q, qc string
	q = fmt.Sprintf(`SELECT id, group_id, name, key, metadata FROM things 
				WHERE profile_id = :profile_id ORDER BY %s %s %s;`, oq, dq, olq)
	qc = `SELECT COUNT(*) FROM things WHERE profile_id = $1;`

	params := map[string]interface{}{
		"profile_id": prID,
		"limit":      pm.Limit,
		"offset":     pm.Offset,
	}

	rows, err := tr.db.NamedQueryContext(ctx, q, params)
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

	var total uint64
	if err := tr.db.GetContext(ctx, &total, qc, prID); err != nil {
		return things.ThingsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	return things.ThingsPage{
		Things: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
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

func (tr thingRepository) retrieve(ctx context.Context, groupIDs []string, allRows bool, pm things.PageMetadata) (things.ThingsPage, error) {
	idsq := getGroupIDsQuery(groupIDs)
	nq, name := dbutil.GetNameQuery(pm.Name)
	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	m, mq, err := dbutil.GetMetadataQuery("", pm.Metadata)
	if err != nil {
		return things.ThingsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	var query []string
	if idsq != "" {
		query = append(query, idsq)
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

	q := fmt.Sprintf(`SELECT id, group_id, profile_id, name, key, metadata FROM things %s ORDER BY %s %s %s;`, whereClause, oq, dq, olq)

	if allRows {
		q = "SELECT id, group_id, profile_id, name, key, metadata FROM things"
	}

	params := map[string]interface{}{
		"limit":    pm.Limit,
		"offset":   pm.Offset,
		"name":     name,
		"metadata": m,
	}

	rows, err := tr.db.NamedQueryContext(ctx, q, params)
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

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM things %s;`, whereClause)

	total, err := total(ctx, tr.db, cq, params)
	if err != nil {
		return things.ThingsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	page := things.ThingsPage{
		Things: items,
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
