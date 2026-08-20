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

var _ uiconfigs.GroupConfigRepository = (*groupConfigRepository)(nil)

type groupConfigRepository struct {
	db dbutil.Database
}

// NewGroupConfigRepository instantiates a PostgreSQL implementation of UI Config repository.
func NewGroupConfigRepository(db dbutil.Database) uiconfigs.GroupConfigRepository {
	return &groupConfigRepository{
		db: db,
	}
}

func (gr groupConfigRepository) Save(ctx context.Context, g uiconfigs.GroupConfig) (uiconfigs.GroupConfig, error) {
	tx, err := gr.db.BeginTxx(ctx, nil)
	if err != nil {
		return uiconfigs.GroupConfig{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	q := `INSERT INTO group_configs (group_id, config)
          VALUES (:group_id, :config);`

	dbGc, err := toDBGroupConfig(g)
	if err != nil {
		return uiconfigs.GroupConfig{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	if _, err := tx.NamedExecContext(ctx, q, dbGc); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return uiconfigs.GroupConfig{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
			case pgerrcode.UniqueViolation:
				return uiconfigs.GroupConfig{}, errors.Wrap(dbutil.ErrConflict, err)
			case pgerrcode.StringDataRightTruncationWarning:
				return uiconfigs.GroupConfig{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}
		return uiconfigs.GroupConfig{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	if err = tx.Commit(); err != nil {
		return uiconfigs.GroupConfig{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return g, nil
}

func (gr groupConfigRepository) RetrieveByGroup(ctx context.Context, groupID string) (uiconfigs.GroupConfig, error) {
	q := `SELECT group_id, config
	      FROM group_configs
	      WHERE group_id = $1;`

	dbGc := dbGroupConfig{}
	if err := gr.db.QueryRowxContext(ctx, q, groupID).StructScan(&dbGc); err != nil {
		pgErr, ok := err.(*pgconn.PgError)

		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return uiconfigs.GroupConfig{
				GroupID: groupID,
				Config:  make(map[string]any),
			}, nil
		}
		return uiconfigs.GroupConfig{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toGroupConfig(dbGc)
}

func (gr groupConfigRepository) RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (uiconfigs.GroupConfigPage, error) {
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	q := `SELECT group_id, config FROM group_configs ORDER BY group_id ` + olq
	cquery := `SELECT COUNT(*) FROM group_configs`

	params := map[string]any{
		"limit":  pm.Limit,
		"offset": pm.Offset,
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return uiconfigs.GroupConfigPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var groupConfigs []uiconfigs.GroupConfig
	for rows.Next() {
		var dbgc dbGroupConfig
		if err := rows.StructScan(&dbgc); err != nil {
			return uiconfigs.GroupConfigPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		gc, err := toGroupConfig(dbgc)
		if err != nil {
			return uiconfigs.GroupConfigPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		groupConfigs = append(groupConfigs, gc)
	}

	total, err := dbutil.Total(ctx, gr.db, cquery, params)
	if err != nil {
		return uiconfigs.GroupConfigPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return uiconfigs.GroupConfigPage{
		Total:         total,
		GroupsConfigs: groupConfigs,
	}, nil
}

func (gr groupConfigRepository) Update(ctx context.Context, g uiconfigs.GroupConfig) (uiconfigs.GroupConfig, error) {
	q := `UPDATE group_configs
      	  SET config = :config
          WHERE group_id = :group_id`

	dbGc, err := toDBGroupConfig(g)
	if err != nil {
		return uiconfigs.GroupConfig{}, errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	res, errdb := gr.db.NamedExecContext(ctx, q, dbGc)
	if errdb != nil {
		pgErr, ok := errdb.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return uiconfigs.GroupConfig{}, errors.Wrap(dbutil.ErrMalformedEntity, errdb)
			case pgerrcode.StringDataRightTruncationWarning:
				return uiconfigs.GroupConfig{}, errors.Wrap(dbutil.ErrMalformedEntity, errdb)
			}
		}
		return uiconfigs.GroupConfig{}, errors.Wrap(dbutil.ErrUpdateEntity, errdb)
	}

	cnt, errdb := res.RowsAffected()
	if errdb != nil {
		return uiconfigs.GroupConfig{}, errors.Wrap(dbutil.ErrUpdateEntity, errdb)
	}

	if cnt == 0 {
		return gr.Save(ctx, g)
	}

	qSelect := `SELECT group_id, config
	            FROM group_configs
				WHERE group_id = $1;`

	var dbRes dbGroupConfig
	if err := gr.db.GetContext(ctx, &dbRes, qSelect, g.GroupID); err != nil {
		return uiconfigs.GroupConfig{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	updated, err := toGroupConfig(dbRes)
	if err != nil {
		return uiconfigs.GroupConfig{}, errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	return updated, nil
}

func (gr groupConfigRepository) Remove(ctx context.Context, groupID string) error {
	q := `DELETE FROM group_configs WHERE group_id = :group_id;`

	args := map[string]any{
		"group_id": groupID,
	}

	if _, err := gr.db.NamedExecContext(ctx, q, args); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (gr groupConfigRepository) BackupAll(ctx context.Context) (uiconfigs.GroupConfigBackup, error) {
	q := `SELECT group_id, config FROM group_configs`

	var items []dbGroupConfig
	err := gr.db.SelectContext(ctx, &items, q)
	if err != nil {
		return uiconfigs.GroupConfigBackup{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	var groupsConfigs []uiconfigs.GroupConfig
	for _, i := range items {
		gc, err := toGroupConfig(i)
		if err != nil {
			return uiconfigs.GroupConfigBackup{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		groupsConfigs = append(groupsConfigs, gc)
	}

	return uiconfigs.GroupConfigBackup{
		GroupsConfigs: groupsConfigs,
	}, nil
}

type dbGroupConfig struct {
	GroupID string `db:"group_id"`
	Config  []byte `db:"config"`
}

func toDBGroupConfig(g uiconfigs.GroupConfig) (dbGroupConfig, error) {
	data := []byte("{}")
	if len(g.Config) > 0 {
		b, err := json.Marshal(g.Config)
		if err != nil {
			return dbGroupConfig{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
		}
		data = b
	}

	return dbGroupConfig{
		GroupID: g.GroupID,
		Config:  data,
	}, nil
}

func toGroupConfig(dbG dbGroupConfig) (uiconfigs.GroupConfig, error) {
	var config map[string]any
	if err := json.Unmarshal([]byte(dbG.Config), &config); err != nil {
		return uiconfigs.GroupConfig{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	return uiconfigs.GroupConfig{
		GroupID: dbG.GroupID,
		Config:  config,
	}, nil
}
