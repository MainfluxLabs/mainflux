// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func (rr ruleRepository) SaveRuns(ctx context.Context, runs ...rules.RuleRun) ([]rules.RuleRun, error) {
	tx, err := rr.db.BeginTxx(ctx, nil)
	if err != nil {
		return []rules.RuleRun{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO rule_runs (id, rule_id, thing_id, started_at, finished_at, status, details)
		VALUES (:id, :rule_id, :thing_id, :started_at, :finished_at, :status, :details);
	`

	dbRuns := make([]dbRuleRun, 0, len(runs))
	for _, run := range runs {
		dbr, err := toDBRun(run)
		if err != nil {
			return []rules.RuleRun{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
		}
		dbRuns = append(dbRuns, dbr)
	}

	if err := dbutil.NamedExecBatch(ctx, tx, query, dbRuns); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return []rules.RuleRun{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
			case pgerrcode.UniqueViolation:
				return []rules.RuleRun{}, errors.Wrap(dbutil.ErrConflict, err)
			}
		}

		return []rules.RuleRun{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	if err = tx.Commit(); err != nil {
		return []rules.RuleRun{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return runs, nil
}

type dbRuleRun struct {
	ID         string    `db:"id"`
	RuleID     string    `db:"rule_id"`
	ThingID    string    `db:"thing_id"`
	StartedAt  time.Time `db:"started_at"`
	FinishedAt time.Time `db:"finished_at"`
	Status     string    `db:"status"`
	Details    []byte    `db:"details"`
}

func toDBRun(run rules.RuleRun) (dbRuleRun, error) {
	details, err := json.Marshal(run.Details)
	if err != nil {
		return dbRuleRun{}, err
	}

	return dbRuleRun{
		ID:         run.ID,
		RuleID:     run.RuleID,
		ThingID:    run.ThingID,
		StartedAt:  run.StartedAt,
		FinishedAt: run.FinishedAt,
		Status:     run.Status,
		Details:    details,
	}, nil
}
