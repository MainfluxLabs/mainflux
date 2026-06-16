package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/shadows"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var _ shadows.ShadowRepository = (*shadowRepository)(nil)

type shadowRepository struct {
	db dbutil.Database
}

// NewShadowRepository instantiates a PostgreSQL implementation of shadow repository.
func NewShadowRepository(db dbutil.Database) shadows.ShadowRepository {
	return &shadowRepository{
		db: db,
	}
}

func (sr shadowRepository) Upsert(ctx context.Context, shadow shadows.Shadow) (shadows.Shadow, error) {
	dbSh, err := toDBShadow(shadow)
	if err != nil {
		return shadows.Shadow{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	q := `INSERT INTO shadows (thing_id, desired, reported, metadata, version, updated_at)
	      VALUES (:thing_id, :desired, :reported, :metadata, 1, :updated_at)
	      ON CONFLICT (thing_id) DO UPDATE SET
	          desired    = EXCLUDED.desired,
	          reported   = EXCLUDED.reported,
	          metadata   = EXCLUDED.metadata,
	          version    = shadows.version + 1,
	          updated_at = EXCLUDED.updated_at
	      RETURNING version;`

	rows, err := sr.db.NamedQueryContext(ctx, q, dbSh)
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return shadows.Shadow{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
			case pgerrcode.UniqueViolation:
				return shadows.Shadow{}, errors.Wrap(dbutil.ErrConflict, err)
			}
		}
		return shadows.Shadow{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&shadow.Version); err != nil {
			return shadows.Shadow{}, errors.Wrap(dbutil.ErrCreateEntity, err)
		}
	}

	return shadow, nil
}

func (sr shadowRepository) RetrieveByThing(ctx context.Context, thingID string) (shadows.Shadow, error) {
	q := `SELECT thing_id, desired, reported, metadata, version, updated_at
	      FROM shadows WHERE thing_id = $1;`

	dbSh := dbShadow{}
	if err := sr.db.QueryRowxContext(ctx, q, thingID).StructScan(&dbSh); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if err == sql.ErrNoRows || (ok && pgerrcode.InvalidTextRepresentation == pgErr.Code) {
			return shadows.Shadow{}, errors.Wrap(shadows.ErrShadowNotFound, err)
		}
		return shadows.Shadow{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toShadow(dbSh)
}

func (sr shadowRepository) Remove(ctx context.Context, thingID string) error {
	q := `DELETE FROM shadows WHERE thing_id = :thing_id;`
	if _, err := sr.db.NamedExecContext(ctx, q, dbShadow{ThingID: thingID}); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}
	return nil
}

func (sr shadowRepository) RetrieveAll(ctx context.Context) ([]shadows.Shadow, error) {
	q := `SELECT thing_id, desired, reported, metadata, version, updated_at FROM shadows;`

	var items []dbShadow
	if err := sr.db.SelectContext(ctx, &items, q); err != nil {
		return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	shs := make([]shadows.Shadow, 0, len(items))
	for _, i := range items {
		sh, err := toShadow(i)
		if err != nil {
			return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		shs = append(shs, sh)
	}

	return shs, nil
}

type dbShadow struct {
	ThingID   string `db:"thing_id"`
	Desired   []byte `db:"desired"`
	Reported  []byte `db:"reported"`
	Metadata  []byte `db:"metadata"`
	Version   uint64 `db:"version"`
	UpdatedAt int64  `db:"updated_at"`
}

func marshalState(s map[string]any) ([]byte, error) {
	if len(s) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(s)
}

func toDBShadow(sh shadows.Shadow) (dbShadow, error) {
	desired, err := marshalState(sh.Desired)
	if err != nil {
		return dbShadow{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	reported, err := marshalState(sh.Reported)
	if err != nil {
		return dbShadow{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	metadata, err := marshalState(sh.Metadata)
	if err != nil {
		return dbShadow{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	return dbShadow{
		ThingID:   sh.ThingID,
		Desired:   desired,
		Reported:  reported,
		Metadata:  metadata,
		Version:   sh.Version,
		UpdatedAt: sh.Timestamp,
	}, nil
}

func toShadow(dbSh dbShadow) (shadows.Shadow, error) {
	var desired shadows.State
	if err := json.Unmarshal(dbSh.Desired, &desired); err != nil {
		return shadows.Shadow{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	var reported shadows.State
	if err := json.Unmarshal(dbSh.Reported, &reported); err != nil {
		return shadows.Shadow{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	var metadata map[string]any
	if err := json.Unmarshal(dbSh.Metadata, &metadata); err != nil {
		return shadows.Shadow{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	return shadows.Shadow{
		ThingID:   dbSh.ThingID,
		Desired:   desired,
		Reported:  reported,
		Metadata:  metadata,
		Version:   dbSh.Version,
		Timestamp: dbSh.UpdatedAt,
	}, nil
}
