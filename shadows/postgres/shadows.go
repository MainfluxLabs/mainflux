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

func (sr shadowRepository) UpsertDesiredState(ctx context.Context, thingID string, desired shadows.State, updatedAt int64) (shadows.Shadow, error) {
	desiredB, err := marshalState(desired)
	if err != nil {
		return shadows.Shadow{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	q := `INSERT INTO shadows (thing_id, desired, updated_at)
	      VALUES (:thing_id, :desired, :updated_at)
	      ON CONFLICT (thing_id) DO UPDATE SET
	          desired    = EXCLUDED.desired,
	          updated_at = EXCLUDED.updated_at
	      RETURNING thing_id, desired, reported, reported_at, updated_at;`

	row, err := sr.db.NamedQueryContext(ctx, q, dbShadow{
		ThingID:   thingID,
		Desired:   desiredB,
		UpdatedAt: updatedAt,
	})
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.InvalidTextRepresentation {
			return shadows.Shadow{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
		}
		return shadows.Shadow{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer row.Close()

	row.Next()
	dbSh := dbShadow{}
	if err := row.StructScan(&dbSh); err != nil {
		return shadows.Shadow{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return toShadow(dbSh)
}

func (sr shadowRepository) UpsertReportedState(ctx context.Context, thingID string, reported shadows.State, reportedAt int64) error {
	reportedB, err := marshalState(reported)
	if err != nil {
		return errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	q := `INSERT INTO shadows (thing_id, reported, reported_at)
	      VALUES (:thing_id, :reported, :reported_at)
	      ON CONFLICT (thing_id) DO UPDATE SET
	          reported    = EXCLUDED.reported,
	          reported_at = EXCLUDED.reported_at;`

	if _, err := sr.db.NamedExecContext(ctx, q, dbShadow{
		ThingID:    thingID,
		Reported:   reportedB,
		ReportedAt: reportedAt,
	}); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.InvalidTextRepresentation {
			return errors.Wrap(dbutil.ErrMalformedEntity, err)
		}
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	return nil
}

func (sr shadowRepository) RetrieveByThing(ctx context.Context, thingID string) (shadows.Shadow, error) {
	q := `SELECT thing_id, desired, reported, reported_at, updated_at
	      FROM shadows WHERE thing_id = $1;`

	dbSh := dbShadow{}
	if err := sr.db.QueryRowxContext(ctx, q, thingID).StructScan(&dbSh); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		// A thing's shadow always exists conceptually: a missing row
		// reads back as an empty shadow rather than an error.
		if err == sql.ErrNoRows || (ok && pgerrcode.InvalidTextRepresentation == pgErr.Code) {
			return shadows.Shadow{
				ThingID:  thingID,
				Desired:  shadows.State{},
				Reported: shadows.State{},
			}, nil
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

type dbShadow struct {
	ThingID    string `db:"thing_id"`
	Desired    []byte `db:"desired"`
	Reported   []byte `db:"reported"`
	ReportedAt int64  `db:"reported_at"`
	UpdatedAt  int64  `db:"updated_at"`
}

func marshalState(s map[string]any) ([]byte, error) {
	if len(s) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(s)
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

	return shadows.Shadow{
		ThingID:    dbSh.ThingID,
		Desired:    desired,
		Reported:   reported,
		ReportedAt: dbSh.ReportedAt,
		UpdatedAt:  dbSh.UpdatedAt,
	}, nil
}
