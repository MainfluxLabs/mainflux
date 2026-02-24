package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/uiconfigs"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var _ uiconfigs.ThingConfigRepository = (*thingConfigRepository)(nil)

type thingConfigRepository struct {
	db dbutil.Database
}

// NewThingConfigRepository instantiates a PostgreSQL implementation of UI Config repository.
func NewThingConfigRepository(db dbutil.Database) uiconfigs.ThingConfigRepository {
	return &thingConfigRepository{
		db: db,
	}
}

func (tr thingConfigRepository) Save(ctx context.Context, t uiconfigs.ThingConfig) (uiconfigs.ThingConfig, error) {
	tx, err := tr.db.BeginTxx(ctx, nil)
	if err != nil {
		return uiconfigs.ThingConfig{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	q := `INSERT INTO thing_configs (thing_id, group_id, config) 
          VALUES (:thing_id, :group_id, :config);`

	dbTc, err := toDBThingConfig(t)
	if err != nil {
		return uiconfigs.ThingConfig{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	if _, err := tx.NamedExecContext(ctx, q, dbTc); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return uiconfigs.ThingConfig{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
			case pgerrcode.UniqueViolation:
				return uiconfigs.ThingConfig{}, errors.Wrap(dbutil.ErrConflict, err)
			case pgerrcode.StringDataRightTruncationWarning:
				return uiconfigs.ThingConfig{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}
		return uiconfigs.ThingConfig{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	if err = tx.Commit(); err != nil {
		return uiconfigs.ThingConfig{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return t, nil
}

func (tr thingConfigRepository) RetrieveByThing(ctx context.Context, thingID string) (uiconfigs.ThingConfig, error) {
	q := `SELECT thing_id, config
	      FROM thing_configs
	      WHERE thing_id = $1;`

	dbTc := dbThingConfig{}
	if err := tr.db.QueryRowxContext(ctx, q, thingID).StructScan(&dbTc); err != nil {
		pgErr, ok := err.(*pgconn.PgError)

		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return uiconfigs.ThingConfig{
				ThingID: thingID,
				Config:  make(map[string]any),
			}, nil
		}
		return uiconfigs.ThingConfig{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toThingConfig(dbTc)
}

func (tr thingConfigRepository) RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (uiconfigs.ThingConfigPage, error) {
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	q := `SELECT thing_id, config FROM thing_configs ORDER BY thing_id ` + olq
	cquery := `SELECT COUNT(*) FROM thing_configs`

	params := map[string]any{
		"limit":  pm.Limit,
		"offset": pm.Offset,
	}

	rows, err := tr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return uiconfigs.ThingConfigPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var thingsConfigs []uiconfigs.ThingConfig
	for rows.Next() {
		var dbtc dbThingConfig
		if err := rows.StructScan(&dbtc); err != nil {
			return uiconfigs.ThingConfigPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		tc, err := toThingConfig(dbtc)
		if err != nil {
			return uiconfigs.ThingConfigPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		thingsConfigs = append(thingsConfigs, tc)
	}

	total, err := dbutil.Total(ctx, tr.db, cquery, params)
	if err != nil {
		return uiconfigs.ThingConfigPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return uiconfigs.ThingConfigPage{
		Total:         total,
		ThingsConfigs: thingsConfigs,
	}, nil
}

func (tr thingConfigRepository) Update(ctx context.Context, t uiconfigs.ThingConfig) (uiconfigs.ThingConfig, error) {
	q := `UPDATE thing_configs 
      	  SET config = :config
          WHERE thing_id = :thing_id`

	dbTc, err := toDBThingConfig(t)
	if err != nil {
		return uiconfigs.ThingConfig{}, errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	res, errdb := tr.db.NamedExecContext(ctx, q, dbTc)
	if errdb != nil {
		pgErr, ok := errdb.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return uiconfigs.ThingConfig{}, errors.Wrap(dbutil.ErrMalformedEntity, errdb)
			case pgerrcode.StringDataRightTruncationWarning:
				return uiconfigs.ThingConfig{}, errors.Wrap(dbutil.ErrMalformedEntity, errdb)
			}
		}
		return uiconfigs.ThingConfig{}, errors.Wrap(dbutil.ErrUpdateEntity, errdb)
	}

	cnt, errdb := res.RowsAffected()
	if errdb != nil {
		return uiconfigs.ThingConfig{}, errors.Wrap(dbutil.ErrUpdateEntity, errdb)
	}

	if cnt == 0 {
		return tr.Save(ctx, t)
	}

	qSelect := `SELECT thing_id,config
	            FROM thing_configs
				WHERE thing_id = $1`

	var dbRes dbThingConfig
	if err := tr.db.GetContext(ctx, &dbRes, qSelect, t.ThingID); err != nil {
		return uiconfigs.ThingConfig{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	updated, err := toThingConfig(dbRes)
	if err != nil {
		return uiconfigs.ThingConfig{}, errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	return updated, nil
}

func (tr thingConfigRepository) Remove(ctx context.Context, thingID string) error {
	q := `DELETE FROM thing_configs WHERE thing_id = :thing_id`

	args := map[string]any{
		"thing_id": thingID,
	}

	if _, err := tr.db.NamedExecContext(ctx, q, args); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (tr thingConfigRepository) RemoveByGroup(ctx context.Context, groupID string) error {
	q := `DELETE FROM thing_configs WHERE group_id = :group_id`

	args := map[string]any{
		"group_id": groupID,
	}

	if _, err := tr.db.NamedExecContext(ctx, q, args); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (tr thingConfigRepository) BackupAll(ctx context.Context) (uiconfigs.ThingConfigBackup, error) {
	q := `SELECT thing_id, group_id, config FROM thing_configs`

	var items []dbThingConfig
	err := tr.db.SelectContext(ctx, &items, q)
	if err != nil {
		return uiconfigs.ThingConfigBackup{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	var thingsConfigs []uiconfigs.ThingConfig
	for _, i := range items {
		tc, err := toThingConfig(i)
		if err != nil {
			return uiconfigs.ThingConfigBackup{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		thingsConfigs = append(thingsConfigs, tc)
	}

	return uiconfigs.ThingConfigBackup{
		ThingsConfigs: thingsConfigs,
	}, nil
}

type dbThingConfig struct {
	ThingID string `db:"thing_id"`
	GroupID string `db:"group_id"`
	Config  []byte `db:"config"`
}

func toDBThingConfig(t uiconfigs.ThingConfig) (dbThingConfig, error) {
	data := []byte("{}")
	if len(t.Config) > 0 {
		b, err := json.Marshal(t.Config)
		if err != nil {
			return dbThingConfig{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
		}
		data = b
	}

	return dbThingConfig{
		ThingID: t.ThingID,
		GroupID: t.GroupID,
		Config:  data,
	}, nil
}

func toThingConfig(dbT dbThingConfig) (uiconfigs.ThingConfig, error) {
	var config map[string]any
	if err := json.Unmarshal([]byte(dbT.Config), &config); err != nil {
		return uiconfigs.ThingConfig{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	return uiconfigs.ThingConfig{
		ThingID: dbT.ThingID,
		GroupID: dbT.GroupID,
		Config:  config,
	}, nil
}
