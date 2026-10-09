// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pyscripts"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func (sr scriptRepository) SaveRuns(ctx context.Context, runs ...pyscripts.ScriptRun) ([]pyscripts.ScriptRun, error) {
	tx, err := sr.db.BeginTxx(ctx, nil)
	if err != nil {
		return []pyscripts.ScriptRun{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO script_runs (id, script_id, script_sha256, return_value, logs,
		                         started_at, finished_at, status, error)
		VALUES (:id, :script_id, :script_sha256, :return_value, :logs,
		        :started_at, :finished_at, :status, :error);
	`

	for _, run := range runs {
		dbr, err := toDBRun(run)
		if err != nil {
			return []pyscripts.ScriptRun{}, errors.Wrap(dbutil.ErrCreateEntity, err)
		}

		if _, err := tx.NamedExecContext(ctx, query, dbr); err != nil {
			return []pyscripts.ScriptRun{}, wrapWrite(err, dbutil.ErrCreateEntity)
		}
	}

	if err := tx.Commit(); err != nil {
		return []pyscripts.ScriptRun{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return runs, nil
}

func (sr scriptRepository) RetrieveRunByID(ctx context.Context, id string) (pyscripts.ScriptRun, error) {
	query := `
		SELECT id, script_id, script_sha256, return_value, logs,
		       started_at, finished_at, status, error
		FROM script_runs
		WHERE id = $1;
	`

	var dbr dbScriptRun
	if err := sr.db.QueryRowxContext(ctx, query, id).StructScan(&dbr); err != nil {
		if err == sql.ErrNoRows {
			return pyscripts.ScriptRun{}, errors.Wrap(dbutil.ErrNotFound, err)
		}

		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.InvalidTextRepresentation {
			return pyscripts.ScriptRun{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
		}

		return pyscripts.ScriptRun{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	run, err := toRun(dbr)
	if err != nil {
		return pyscripts.ScriptRun{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return run, nil
}

func (sr scriptRepository) RetrieveRunsByScript(ctx context.Context, scriptID string, pm pyscripts.PageMetadata) (pyscripts.ScriptRunsPage, error) {
	oq := dbutil.GetOrderQuery(pm.Order, pyscripts.ScriptRunOrderFields)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	clauses := []string{"script_id = :script_id"}
	params := map[string]any{
		"script_id": scriptID,
		"limit":     pm.Limit,
		"offset":    pm.Offset,
	}

	if pm.Status != "" {
		clauses = append(clauses, "status = :status")
		params["status"] = pm.Status
	}
	if !pm.From.IsZero() {
		clauses = append(clauses, "started_at >= :from")
		params["from"] = pm.From
	}
	if !pm.To.IsZero() {
		clauses = append(clauses, "started_at < :to")
		params["to"] = pm.To
	}

	whereClause := dbutil.BuildWhereClause(clauses...)

	query := fmt.Sprintf(`
		SELECT id, script_id, script_sha256, return_value, logs,
		       started_at, finished_at, status, error
		FROM script_runs %s ORDER BY %s %s %s;
	`, whereClause, oq, dq, olq)

	queryCount := fmt.Sprintf(`SELECT COUNT(*) FROM script_runs %s;`, whereClause)

	rows, err := sr.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return pyscripts.ScriptRunsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var runs []pyscripts.ScriptRun
	for rows.Next() {
		var dbr dbScriptRun
		if err := rows.StructScan(&dbr); err != nil {
			return pyscripts.ScriptRunsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		run, err := toRun(dbr)
		if err != nil {
			return pyscripts.ScriptRunsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		runs = append(runs, run)
	}

	total, err := dbutil.Total(ctx, sr.db, queryCount, params)
	if err != nil {
		return pyscripts.ScriptRunsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return pyscripts.ScriptRunsPage{Runs: runs, Total: total}, nil
}

func (sr scriptRepository) RemoveRuns(ctx context.Context, ids ...string) error {
	tx, err := sr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}
	defer tx.Rollback()

	query := `DELETE FROM script_runs WHERE id = :id;`

	for _, id := range ids {
		if _, err := tx.NamedExecContext(ctx, query, dbScriptRun{ID: id}); err != nil {
			return errors.Wrap(dbutil.ErrRemoveEntity, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

type dbScriptRun struct {
	ID           string         `db:"id"`
	ScriptID     string         `db:"script_id"`
	ScriptSHA256 string         `db:"script_sha256"`
	Value        []byte         `db:"return_value"`
	Logs         []byte         `db:"logs"`
	StartedAt    time.Time      `db:"started_at"`
	FinishedAt   time.Time      `db:"finished_at"`
	Status       string         `db:"status"`
	Error        sql.NullString `db:"error"`
}

func toDBRun(r pyscripts.ScriptRun) (dbScriptRun, error) {
	logs := r.Logs
	if logs == nil {
		logs = []string{}
	}

	lb, err := json.Marshal(logs)
	if err != nil {
		return dbScriptRun{}, err
	}

	var vb []byte
	if r.Value != nil {
		vb, err = json.Marshal(r.Value)
		if err != nil {
			return dbScriptRun{}, err
		}
	}

	return dbScriptRun{
		ID:           r.ID,
		ScriptID:     r.ScriptID,
		ScriptSHA256: r.ScriptSHA256,
		Value:        vb,
		Logs:         lb,
		StartedAt:    r.StartedAt,
		FinishedAt:   r.FinishedAt,
		Status:       r.Status,
		Error:        sql.NullString{String: r.Error, Valid: r.Error != ""},
	}, nil
}

func toRun(dbr dbScriptRun) (pyscripts.ScriptRun, error) {
	var logs []string
	if len(dbr.Logs) > 0 {
		if err := json.Unmarshal(dbr.Logs, &logs); err != nil {
			return pyscripts.ScriptRun{}, err
		}
	}

	var value any
	if len(dbr.Value) > 0 {
		if err := json.Unmarshal(dbr.Value, &value); err != nil {
			return pyscripts.ScriptRun{}, err
		}
	}

	return pyscripts.ScriptRun{
		ID:           dbr.ID,
		ScriptID:     dbr.ScriptID,
		ScriptSHA256: dbr.ScriptSHA256,
		Value:        value,
		Logs:         logs,
		StartedAt:    dbr.StartedAt,
		FinishedAt:   dbr.FinishedAt,
		Status:       dbr.Status,
		Error:        dbr.Error.String,
	}, nil
}
