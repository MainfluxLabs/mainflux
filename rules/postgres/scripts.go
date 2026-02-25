package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func (rr ruleRepository) SaveScripts(ctx context.Context, scripts ...rules.LuaScript) ([]rules.LuaScript, error) {
	tx, err := rr.db.BeginTxx(ctx, nil)
	if err != nil {
		return []rules.LuaScript{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO lua_scripts (id, group_id, script, name, description) 
		VALUES (:id, :group_id, :script, :name, :description);
	`

	for _, script := range scripts {
		dbScript := toDBLuaScript(script)
		if _, err := tx.NamedExecContext(ctx, query, dbScript); err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return []rules.LuaScript{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return []rules.LuaScript{}, errors.Wrap(dbutil.ErrConflict, err)
				case pgerrcode.StringDataRightTruncationWarning:
					return []rules.LuaScript{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
				}
			}

			return []rules.LuaScript{}, errors.Wrap(dbutil.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return []rules.LuaScript{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	return scripts, nil
}

func (rr ruleRepository) RetrieveScriptByID(ctx context.Context, id string) (rules.LuaScript, error) {
	query := `
		SELECT id, group_id, script, name, description
		FROM lua_scripts
		WHERE id = $1;
	`

	var dbs dbLuaScript
	if err := rr.db.QueryRowxContext(ctx, query, id).StructScan(&dbs); err != nil {
		if err == sql.ErrNoRows {
			return rules.LuaScript{}, errors.Wrap(dbutil.ErrNotFound, err)
		}

		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return rules.LuaScript{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}
		return rules.LuaScript{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toLuaScript(dbs), nil
}

func (rr ruleRepository) RetrieveScriptsByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (rules.LuaScriptsPage, error) {
	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	thingQuery := "lst.thing_id = :thing_id"
	nameQuery, name := dbutil.GetNameQuery(pm.Name)

	whereClause := dbutil.BuildWhereClause(thingQuery, nameQuery)

	query := `
		SELECT ls.id, ls.group_id, ls.script, ls.name, ls.description
		FROM lua_scripts ls
		INNER JOIN lua_scripts_things lst ON ls.id = lst.lua_script_id
		%s
		ORDER BY %s %s
		%s
	`

	queryCount := `
		SELECT COUNT(*)
		FROM lua_scripts ls
		INNER JOIN lua_scripts_things lst ON ls.id = lst.lua_script_id
		%s
	`

	query = fmt.Sprintf(query, whereClause, oq, dq, olq)
	queryCount = fmt.Sprintf(queryCount, whereClause)

	params := map[string]any{
		"thing_id": thingID,
		"name":     name,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	rows, err := rr.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return rules.LuaScriptsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var scripts []rules.LuaScript
	for rows.Next() {
		var dba dbLuaScript
		if err = rows.StructScan(&dba); err != nil {
			return rules.LuaScriptsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		scripts = append(scripts, toLuaScript(dba))
	}

	total, err := dbutil.Total(ctx, rr.db, queryCount, params)
	if err != nil {
		return rules.LuaScriptsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	page := rules.LuaScriptsPage{
		Scripts: scripts,
		Total:   total,
	}

	return page, nil
}

func (rr ruleRepository) RetrieveScriptsByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (rules.LuaScriptsPage, error) {
	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	gq := "group_id = :group_id"
	nq, name := dbutil.GetNameQuery(pm.Name)

	whereClause := dbutil.BuildWhereClause(gq, nq)

	query := `
		SELECT id, group_id, script, name, description
		FROM lua_scripts %s ORDER BY %s %s %s;
	`

	queryCount := `SELECT COUNT(*) FROM lua_scripts %s;`

	query = fmt.Sprintf(query, whereClause, oq, dq, olq)
	queryCount = fmt.Sprintf(queryCount, whereClause)

	params := map[string]any{
		"group_id": groupID,
		"name":     name,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	rows, err := rr.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return rules.LuaScriptsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var scripts []rules.LuaScript
	for rows.Next() {
		var dba dbLuaScript
		if err = rows.StructScan(&dba); err != nil {
			return rules.LuaScriptsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		scripts = append(scripts, toLuaScript(dba))
	}

	total, err := dbutil.Total(ctx, rr.db, queryCount, params)
	if err != nil {
		return rules.LuaScriptsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	page := rules.LuaScriptsPage{
		Scripts: scripts,
		Total:   total,
	}

	return page, nil
}

func (rr ruleRepository) RetrieveThingIDsByScript(ctx context.Context, scriptID string) ([]string, error) {
	query := `
		SELECT thing_id
		FROM lua_scripts_things
		WHERE lua_script_id = $1
	`

	thingIDs := []string{}
	if err := rr.db.SelectContext(ctx, &thingIDs, query, scriptID); err != nil {
		return nil, err
	}

	return thingIDs, nil
}

func (rr ruleRepository) UpdateScript(ctx context.Context, script rules.LuaScript) error {
	query := `
		UPDATE lua_scripts
		SET script = :script, name = :name, description = :description
		WHERE id = :id;
	`

	dbScript := toDBLuaScript(script)

	res, errdb := rr.db.NamedExecContext(ctx, query, dbScript)
	if errdb != nil {
		pgErr, ok := errdb.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation,
				pgerrcode.StringDataRightTruncationDataException:
				return errors.Wrap(dbutil.ErrMalformedEntity, errdb)
			}
		}

		return errors.Wrap(dbutil.ErrUpdateEntity, errdb)
	}

	cnt, errdb := res.RowsAffected()
	if errdb != nil {
		return errors.Wrap(dbutil.ErrUpdateEntity, errdb)
	}

	if cnt == 0 {
		return dbutil.ErrNotFound
	}

	return nil
}

func (rr ruleRepository) RemoveScripts(ctx context.Context, ids ...string) error {
	query := `
		DELETE FROM lua_scripts
		WHERE id = :id
	`

	for _, id := range ids {
		dbr := dbLuaScript{ID: id}
		if _, err := rr.db.NamedExecContext(ctx, query, dbr); err != nil {
			return errors.Wrap(dbutil.ErrRemoveEntity, err)
		}
	}

	return nil
}

func (rr ruleRepository) AssignScripts(ctx context.Context, thingID string, scriptIDs ...string) error {
	tx, err := rr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO lua_scripts_things (thing_id, lua_script_id)
		VALUES (:thing_id, :lua_script_id)
	`

	for _, scriptID := range scriptIDs {
		params := map[string]any{
			"lua_script_id": scriptID,
			"thing_id":      thingID,
		}

		if _, err := tx.NamedExecContext(ctx, query, params); err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					continue
				}
			}
			return errors.Wrap(dbutil.ErrCreateEntity, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return nil
}

func (rr ruleRepository) UnassignScripts(ctx context.Context, thingID string, scriptIDs ...string) error {
	query := `
		DELETE FROM lua_scripts_things
		WHERE lua_script_id = :lua_script_id AND thing_id = :thing_id
	`

	for _, scriptID := range scriptIDs {
		params := map[string]any{
			"lua_script_id": scriptID,
			"thing_id":      thingID,
		}
		if _, err := rr.db.NamedExecContext(ctx, query, params); err != nil {
			return errors.Wrap(dbutil.ErrRemoveEntity, err)
		}
	}

	return nil
}

func (rr ruleRepository) UnassignScriptsFromThing(ctx context.Context, thingID string) error {
	query := `
		DELETE FROM lua_scripts_things WHERE thing_id = :thing_id;
	`

	params := map[string]any{
		"thing_id": thingID,
	}
	if _, err := rr.db.NamedExecContext(ctx, query, params); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (rr ruleRepository) SaveScriptRuns(ctx context.Context, runs ...rules.ScriptRun) ([]rules.ScriptRun, error) {
	tx, err := rr.db.BeginTxx(ctx, nil)
	if err != nil {
		return []rules.ScriptRun{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO lua_script_runs (id, script_id, thing_id, logs, started_at, finished_at, status, error) 
		VALUES (:id, :script_id, :thing_id, :logs, :started_at, :finished_at, :status, :error);
	`

	for _, run := range runs {
		dbRun, err := toDBScriptRun(run)
		if err != nil {
			return []rules.ScriptRun{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
		}

		if _, err := tx.NamedExecContext(ctx, query, dbRun); err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return []rules.ScriptRun{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return []rules.ScriptRun{}, errors.Wrap(dbutil.ErrConflict, err)
				case pgerrcode.StringDataRightTruncationWarning:
					return []rules.ScriptRun{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
				}
			}

			return []rules.ScriptRun{}, errors.Wrap(dbutil.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return []rules.ScriptRun{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return runs, nil
}

func (rr ruleRepository) RetrieveScriptRunByID(ctx context.Context, id string) (rules.ScriptRun, error) {
	query := `
		SELECT id, script_id, thing_id, logs, started_at, finished_at, status, error
		FROM lua_script_runs
		WHERE id = $1;
	`

	var dbr dbScriptRun
	if err := rr.db.QueryRowxContext(ctx, query, id).StructScan(&dbr); err != nil {
		if err == sql.ErrNoRows {
			return rules.ScriptRun{}, errors.Wrap(dbutil.ErrNotFound, err)
		}

		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return rules.ScriptRun{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}
		return rules.ScriptRun{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	scriptRun, err := toScriptRun(dbr)
	if err != nil {
		return rules.ScriptRun{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return scriptRun, nil
}

func (rr ruleRepository) RemoveScriptRuns(ctx context.Context, ids ...string) error {
	query := `
		DELETE FROM lua_script_runs
		WHERE id = :id
	`

	for _, id := range ids {
		dbRun := dbScriptRun{ID: id}
		if _, err := rr.db.NamedExecContext(ctx, query, dbRun); err != nil {
			return errors.Wrap(dbutil.ErrRemoveEntity, err)
		}
	}

	return nil
}

func (rr ruleRepository) RetrieveScriptRunsByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (rules.ScriptRunsPage, error) {
	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	thingQuery := "thing_id = :thing_id"

	whereClause := dbutil.BuildWhereClause(thingQuery)

	query := `
		SELECT id, script_id, thing_id, logs, started_at, finished_at, status, error
		FROM lua_script_runs %s ORDER BY %s %s %s;
	`

	queryCount := `SELECT COUNT(*) FROM lua_script_runs %s;`

	query = fmt.Sprintf(query, whereClause, oq, dq, olq)
	queryCount = fmt.Sprintf(queryCount, whereClause)

	params := map[string]any{
		"thing_id": thingID,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	rows, err := rr.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return rules.ScriptRunsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var runs []rules.ScriptRun
	for rows.Next() {
		var dbr dbScriptRun
		if err = rows.StructScan(&dbr); err != nil {
			return rules.ScriptRunsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		scriptRun, err := toScriptRun(dbr)
		if err != nil {
			return rules.ScriptRunsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		runs = append(runs, scriptRun)
	}

	total, err := dbutil.Total(ctx, rr.db, queryCount, params)
	if err != nil {
		return rules.ScriptRunsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	page := rules.ScriptRunsPage{
		Runs:  runs,
		Total: total,
	}

	return page, nil
}

type dbLuaScript struct {
	ID          string `db:"id"`
	GroupID     string `db:"group_id"`
	Script      string `db:"script"`
	Name        string `db:"name"`
	Description string `db:"description"`
}

func toDBLuaScript(script rules.LuaScript) dbLuaScript {
	return dbLuaScript{
		ID:          script.ID,
		GroupID:     script.GroupID,
		Script:      script.Script,
		Name:        script.Name,
		Description: script.Description,
	}
}

func toLuaScript(dbScript dbLuaScript) rules.LuaScript {
	return rules.LuaScript{
		ID:          dbScript.ID,
		GroupID:     dbScript.GroupID,
		Script:      dbScript.Script,
		Name:        dbScript.Name,
		Description: dbScript.Description,
	}
}

type dbScriptRun struct {
	ID         string         `db:"id"`
	ScriptID   string         `db:"script_id"`
	ThingID    string         `db:"thing_id"`
	Logs       []byte         `db:"logs"`
	StartedAt  time.Time      `db:"started_at"`
	FinishedAt time.Time      `db:"finished_at"`
	Status     string         `db:"status"`
	Error      sql.NullString `db:"error"`
}

func toDBScriptRun(run rules.ScriptRun) (dbScriptRun, error) {
	logsBytes := []byte("[]")

	if len(run.Logs) > 0 {
		b, err := json.Marshal(run.Logs)
		if err != nil {
			return dbScriptRun{}, err
		}

		logsBytes = b
	}

	errorField := sql.NullString{
		String: run.Error,
		Valid:  run.Error != "",
	}

	return dbScriptRun{
		ID:         run.ID,
		ScriptID:   run.ScriptID,
		ThingID:    run.ThingID,
		Logs:       logsBytes,
		StartedAt:  run.StartedAt,
		FinishedAt: run.FinishedAt,
		Status:     run.Status,
		Error:      errorField,
	}, nil
}

func toScriptRun(dbs dbScriptRun) (rules.ScriptRun, error) {
	var logs []string
	if err := json.Unmarshal(dbs.Logs, &logs); err != nil {
		return rules.ScriptRun{}, err
	}

	errorField := ""
	if dbs.Error.Valid {
		errorField = dbs.Error.String
	}

	return rules.ScriptRun{
		ID:         dbs.ID,
		ScriptID:   dbs.ScriptID,
		ThingID:    dbs.ThingID,
		Logs:       logs,
		StartedAt:  dbs.StartedAt,
		FinishedAt: dbs.FinishedAt,
		Status:     dbs.Status,
		Error:      errorField,
	}, nil
}
