// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pyscripts"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

var _ pyscripts.ScriptRepository = (*scriptRepository)(nil)

type scriptRepository struct {
	db dbutil.Database
}

// NewScriptRepository instantiates a PostgreSQL implementation of the pyscripts repository.
func NewScriptRepository(db dbutil.Database) pyscripts.ScriptRepository {
	return &scriptRepository{db: db}
}

func (sr scriptRepository) SaveScripts(ctx context.Context, scripts ...pyscripts.Script) ([]pyscripts.Script, error) {
	tx, err := sr.db.BeginTxx(ctx, nil)
	if err != nil {
		return []pyscripts.Script{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO scripts (id, group_id, name, description, source, sha256)
		VALUES (:id, :group_id, :name, :description, :source, :sha256)
		RETURNING created_at, updated_at;
	`

	for i, script := range scripts {
		rows, err := sqlx.NamedQueryContext(ctx, tx, query, toDBScript(script))
		if err != nil {
			return []pyscripts.Script{}, wrapWrite(err, dbutil.ErrCreateEntity)
		}

		if !rows.Next() {
			rows.Close()
			return []pyscripts.Script{}, errors.Wrap(dbutil.ErrCreateEntity, rows.Err())
		}

		if err := rows.Scan(&scripts[i].CreatedAt, &scripts[i].UpdatedAt); err != nil {
			rows.Close()
			return []pyscripts.Script{}, errors.Wrap(dbutil.ErrCreateEntity, err)
		}

		if err := rows.Close(); err != nil {
			return []pyscripts.Script{}, errors.Wrap(dbutil.ErrCreateEntity, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return []pyscripts.Script{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return scripts, nil
}

func (sr scriptRepository) RetrieveScriptByID(ctx context.Context, id string) (pyscripts.Script, error) {
	query := `
		SELECT id, group_id, name, description, source, sha256, created_at, updated_at
		FROM scripts
		WHERE id = $1;
	`

	var dbs dbScript
	if err := sr.db.QueryRowxContext(ctx, query, id).StructScan(&dbs); err != nil {
		if err == sql.ErrNoRows {
			return pyscripts.Script{}, errors.Wrap(dbutil.ErrNotFound, err)
		}

		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.InvalidTextRepresentation {
			return pyscripts.Script{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
		}

		return pyscripts.Script{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toScript(dbs), nil
}

func (sr scriptRepository) RetrieveScriptsByGroup(ctx context.Context, groupID string, pm pyscripts.PageMetadata) (pyscripts.ScriptsPage, error) {
	oq := dbutil.GetOrderQuery(pm.Order, pyscripts.ScriptOrderFields)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	gq := "group_id = :group_id"
	nq, name := dbutil.GetNameQuery(pm.Name)

	whereClause := dbutil.BuildWhereClause(gq, nq)

	query := fmt.Sprintf(`
		SELECT id, group_id, name, description, source, sha256, created_at, updated_at
		FROM scripts %s ORDER BY %s %s %s;
	`, whereClause, oq, dq, olq)

	queryCount := fmt.Sprintf(`SELECT COUNT(*) FROM scripts %s;`, whereClause)

	params := map[string]any{
		"group_id": groupID,
		"name":     name,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	rows, err := sr.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return pyscripts.ScriptsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var scripts []pyscripts.Script
	for rows.Next() {
		var dbs dbScript
		if err := rows.StructScan(&dbs); err != nil {
			return pyscripts.ScriptsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		scripts = append(scripts, toScript(dbs))
	}

	total, err := dbutil.Total(ctx, sr.db, queryCount, params)
	if err != nil {
		return pyscripts.ScriptsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return pyscripts.ScriptsPage{Scripts: scripts, Total: total}, nil
}

func (sr scriptRepository) UpdateScript(ctx context.Context, script pyscripts.Script) error {
	query := `
		UPDATE scripts
		SET name = :name, description = :description, source = :source,
		    sha256 = :sha256, updated_at = NOW()
		WHERE id = :id;
	`

	res, err := sr.db.NamedExecContext(ctx, query, toDBScript(script))
	if err != nil {
		return wrapWrite(err, dbutil.ErrUpdateEntity)
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

// RemoveScripts deletes in one transaction. Issued as separate statements, a
// failure part way through would leave the batch half applied while the caller
// got an error, so a retry would act on a different world than the one it was
// authorized against.
func (sr scriptRepository) RemoveScripts(ctx context.Context, ids ...string) error {
	tx, err := sr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}
	defer tx.Rollback()

	query := `DELETE FROM scripts WHERE id = :id;`

	for _, id := range ids {
		if _, err := tx.NamedExecContext(ctx, query, dbScript{ID: id}); err != nil {
			return errors.Wrap(dbutil.ErrRemoveEntity, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (sr scriptRepository) RemoveScriptsByGroup(ctx context.Context, groupID string) error {
	query := `DELETE FROM scripts WHERE group_id = :group_id;`

	params := map[string]any{"group_id": groupID}
	if _, err := sr.db.NamedExecContext(ctx, query, params); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

type dbScript struct {
	ID          string    `db:"id"`
	GroupID     string    `db:"group_id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	Source      string    `db:"source"`
	SHA256      string    `db:"sha256"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func toDBScript(s pyscripts.Script) dbScript {
	return dbScript{
		ID:          s.ID,
		GroupID:     s.GroupID,
		Name:        s.Name,
		Description: s.Description,
		Source:      s.Source,
		SHA256:      s.SHA256,
	}
}

func toScript(dbs dbScript) pyscripts.Script {
	return pyscripts.Script{
		ID:          dbs.ID,
		GroupID:     dbs.GroupID,
		Name:        dbs.Name,
		Description: dbs.Description,
		Source:      dbs.Source,
		SHA256:      dbs.SHA256,
		CreatedAt:   dbs.CreatedAt,
		UpdatedAt:   dbs.UpdatedAt,
	}
}

func wrapWrite(err error, fallback error) error {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		switch pgErr.Code {
		case pgerrcode.InvalidTextRepresentation:
			return errors.Wrap(dbutil.ErrMalformedEntity, err)
		case pgerrcode.UniqueViolation:
			return errors.Wrap(dbutil.ErrConflict, err)
		case pgerrcode.ForeignKeyViolation:
			return errors.Wrap(dbutil.ErrNotFound, err)
		}
	}

	return errors.Wrap(fallback, err)
}
