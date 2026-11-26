// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"
	"fmt"
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

func (rr ruleRepository) RetrieveRunsByIDs(ctx context.Context, ids ...string) ([]rules.RuleRun, error) {
	query := `
		SELECT id, rule_id, thing_id, started_at, finished_at, status, details
		FROM rule_runs
		WHERE id = ANY(:ids);
	`

	params := map[string]any{
		"ids": ids,
	}

	rows, err := rr.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return []rules.RuleRun{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}
		return []rules.RuleRun{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var runs []rules.RuleRun
	for rows.Next() {
		var dbr dbRuleRun
		if err := rows.StructScan(&dbr); err != nil {
			return []rules.RuleRun{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		run, err := toRun(dbr)
		if err != nil {
			return []rules.RuleRun{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		runs = append(runs, run)
	}

	if err := rows.Err(); err != nil {
		return []rules.RuleRun{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	if len(runs) != countUnique(ids) {
		return []rules.RuleRun{}, dbutil.ErrNotFound
	}

	return runs, nil
}

func (rr ruleRepository) RetrieveRunsByThing(ctx context.Context, thingID string, pm rules.PageMetadata) (rules.RuleRunsPage, error) {
	return rr.retrieveRuns(ctx, "thing_id", thingID, pm)
}

func (rr ruleRepository) RetrieveRunsByRule(ctx context.Context, ruleID string, pm rules.PageMetadata) (rules.RuleRunsPage, error) {
	return rr.retrieveRuns(ctx, "rule_id", ruleID, pm)
}

func (rr ruleRepository) RemoveRuns(ctx context.Context, ids ...string) error {
	query := `
		DELETE FROM rule_runs
		WHERE id = ANY(:ids)
	`

	params := map[string]any{
		"ids": ids,
	}

	if _, err := rr.db.NamedExecContext(ctx, query, params); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (rr ruleRepository) retrieveRuns(ctx context.Context, col, val string, pm rules.PageMetadata) (rules.RuleRunsPage, error) {
	oq := dbutil.GetOrderQuery(pm.Order, rules.RuleRunOrderFields)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	cq := fmt.Sprintf("%s = :%s", col, col)

	statusQuery := ""
	if pm.Status != "" {
		statusQuery = "status = :status"
	}
	fromQuery := ""
	if !pm.From.IsZero() {
		fromQuery = "started_at >= :from"
	}
	toQuery := ""
	if !pm.To.IsZero() {
		toQuery = "started_at < :to"
	}

	whereClause := dbutil.BuildWhereClause(cq, statusQuery, fromQuery, toQuery)

	query := `
		SELECT id, rule_id, thing_id, started_at, finished_at, status, details
		FROM rule_runs %s ORDER BY %s %s %s;
	`

	queryCount := `SELECT COUNT(*) FROM rule_runs %s;`

	query = fmt.Sprintf(query, whereClause, oq, dq, olq)
	queryCount = fmt.Sprintf(queryCount, whereClause)

	params := map[string]any{
		col:      val,
		"limit":  pm.Limit,
		"offset": pm.Offset,
		"status": pm.Status,
		"from":   pm.From,
		"to":     pm.To,
	}

	rows, err := rr.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return rules.RuleRunsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var runs []rules.RuleRun
	for rows.Next() {
		var dbr dbRuleRun
		if err = rows.StructScan(&dbr); err != nil {
			return rules.RuleRunsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		run, err := toRun(dbr)
		if err != nil {
			return rules.RuleRunsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		runs = append(runs, run)
	}

	total, err := dbutil.Total(ctx, rr.db, queryCount, params)
	if err != nil {
		return rules.RuleRunsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	page := rules.RuleRunsPage{
		Runs:  runs,
		Total: total,
	}

	return page, nil
}

func countUnique(ids []string) int {
	unique := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		unique[id] = struct{}{}
	}

	return len(unique)
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

func toRun(dbr dbRuleRun) (rules.RuleRun, error) {
	var details rules.RuleRunDetails
	if err := json.Unmarshal(dbr.Details, &details); err != nil {
		return rules.RuleRun{}, err
	}

	return rules.RuleRun{
		ID:         dbr.ID,
		RuleID:     dbr.RuleID,
		ThingID:    dbr.ThingID,
		StartedAt:  dbr.StartedAt,
		FinishedAt: dbr.FinishedAt,
		Status:     dbr.Status,
		Details:    details,
	}, nil
}
