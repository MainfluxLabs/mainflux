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

var _ uiconfigs.OrgConfigRepository = (*orgConfigRepository)(nil)

type orgConfigRepository struct {
	db dbutil.Database
}

// NewOrgConfigRepository instantiates a PostgreSQL implementation of UI Config repository.
func NewOrgConfigRepository(db dbutil.Database) uiconfigs.OrgConfigRepository {
	return &orgConfigRepository{
		db: db,
	}
}

func (or orgConfigRepository) Save(ctx context.Context, o uiconfigs.OrgConfig) (uiconfigs.OrgConfig, error) {
	tx, err := or.db.BeginTxx(ctx, nil)
	if err != nil {
		return uiconfigs.OrgConfig{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	q := `INSERT INTO org_configs (org_id, config) 
          VALUES (:org_id, :config);`

	dbOc, err := toDBOrgConfig(o)
	if err != nil {
		return uiconfigs.OrgConfig{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	if _, err := tx.NamedExecContext(ctx, q, dbOc); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return uiconfigs.OrgConfig{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
			case pgerrcode.UniqueViolation:
				return uiconfigs.OrgConfig{}, errors.Wrap(dbutil.ErrConflict, err)
			case pgerrcode.StringDataRightTruncationWarning:
				return uiconfigs.OrgConfig{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}
		return uiconfigs.OrgConfig{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	if err = tx.Commit(); err != nil {
		return uiconfigs.OrgConfig{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return o, nil
}

func (or orgConfigRepository) RetrieveByOrg(ctx context.Context, orgID string) (uiconfigs.OrgConfig, error) {
	q := `SELECT org_id, config
	      FROM org_configs
	      WHERE org_id = $1;`

	dbOc := dbOrgConfig{}
	if err := or.db.QueryRowxContext(ctx, q, orgID).StructScan(&dbOc); err != nil {
		pgErr, ok := err.(*pgconn.PgError)

		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return uiconfigs.OrgConfig{
				OrgID:  orgID,
				Config: make(map[string]any),
			}, nil
		}
		return uiconfigs.OrgConfig{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toOrgConfig(dbOc)
}

func (or orgConfigRepository) RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (uiconfigs.OrgConfigPage, error) {
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	q := `SELECT org_id, config FROM org_configs ORDER BY org_id ` + olq
	cquery := `SELECT COUNT(*) FROM org_configs`

	params := map[string]any{
		"limit":  pm.Limit,
		"offset": pm.Offset,
	}

	rows, err := or.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return uiconfigs.OrgConfigPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var orgConfigs []uiconfigs.OrgConfig
	for rows.Next() {
		var dboc dbOrgConfig
		if err := rows.StructScan(&dboc); err != nil {
			return uiconfigs.OrgConfigPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		oc, err := toOrgConfig(dboc)
		if err != nil {
			return uiconfigs.OrgConfigPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		orgConfigs = append(orgConfigs, oc)
	}

	total, err := dbutil.Total(ctx, or.db, cquery, params)
	if err != nil {
		return uiconfigs.OrgConfigPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return uiconfigs.OrgConfigPage{
		Total:       total,
		OrgsConfigs: orgConfigs,
	}, nil
}

func (or orgConfigRepository) Update(ctx context.Context, o uiconfigs.OrgConfig) (uiconfigs.OrgConfig, error) {
	q := `UPDATE org_configs 
      	  SET config = :config
          WHERE org_id = :org_id`

	dbOc, err := toDBOrgConfig(o)
	if err != nil {
		return uiconfigs.OrgConfig{}, errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	res, errdb := or.db.NamedExecContext(ctx, q, dbOc)
	if errdb != nil {
		pgErr, ok := errdb.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return uiconfigs.OrgConfig{}, errors.Wrap(dbutil.ErrMalformedEntity, errdb)
			case pgerrcode.StringDataRightTruncationWarning:
				return uiconfigs.OrgConfig{}, errors.Wrap(dbutil.ErrMalformedEntity, errdb)
			}
		}
		return uiconfigs.OrgConfig{}, errors.Wrap(dbutil.ErrUpdateEntity, errdb)
	}

	cnt, errdb := res.RowsAffected()
	if errdb != nil {
		return uiconfigs.OrgConfig{}, errors.Wrap(dbutil.ErrUpdateEntity, errdb)
	}

	if cnt == 0 {
		return or.Save(ctx, o)
	}

	qSelect := `SELECT org_id, config
	            FROM org_configs
				WHERE org_id = $1;`

	var dbRes dbOrgConfig
	if err := or.db.GetContext(ctx, &dbRes, qSelect, o.OrgID); err != nil {
		return uiconfigs.OrgConfig{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	updated, err := toOrgConfig(dbRes)
	if err != nil {
		return uiconfigs.OrgConfig{}, errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	return updated, nil
}

func (or orgConfigRepository) Remove(ctx context.Context, orgID string) error {
	q := `DELETE FROM org_configs WHERE org_id = :org_id;`

	args := map[string]any{
		"org_id": orgID,
	}

	if _, err := or.db.NamedExecContext(ctx, q, args); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (or orgConfigRepository) BackupAll(ctx context.Context) (uiconfigs.OrgConfigBackup, error) {
	q := `SELECT org_id, config FROM org_configs`

	var items []dbOrgConfig
	err := or.db.SelectContext(ctx, &items, q)
	if err != nil {
		return uiconfigs.OrgConfigBackup{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	var orgsConfigs []uiconfigs.OrgConfig
	for _, i := range items {
		tc, err := toOrgConfig(i)
		if err != nil {
			return uiconfigs.OrgConfigBackup{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		orgsConfigs = append(orgsConfigs, tc)
	}

	return uiconfigs.OrgConfigBackup{
		OrgsConfigs: orgsConfigs,
	}, nil
}

type dbOrgConfig struct {
	OrgID  string `db:"org_id"`
	Config []byte `db:"config"`
}

func toDBOrgConfig(o uiconfigs.OrgConfig) (dbOrgConfig, error) {
	data := []byte("{}")
	if len(o.Config) > 0 {
		b, err := json.Marshal(o.Config)
		if err != nil {
			return dbOrgConfig{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
		}
		data = b
	}

	return dbOrgConfig{
		OrgID:  o.OrgID,
		Config: data,
	}, nil
}

func toOrgConfig(dbO dbOrgConfig) (uiconfigs.OrgConfig, error) {
	var config map[string]any
	if err := json.Unmarshal([]byte(dbO.Config), &config); err != nil {
		return uiconfigs.OrgConfig{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	return uiconfigs.OrgConfig{
		OrgID:  dbO.OrgID,
		Config: config,
	}, nil
}
