package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jackc/pgerrcode"
)

var _ rules.RuleRepository = (*ruleRepository)(nil)

type ruleRepository struct {
	db dbutil.Database
}

// NewRuleRepository instantiates a PostgreSQL implementation of rule repository.
func NewRuleRepository(db dbutil.Database) rules.RuleRepository {
	return &ruleRepository{
		db: db,
	}
}

func (rr ruleRepository) Save(ctx context.Context, rls ...rules.Rule) ([]rules.Rule, error) {
	tx, err := rr.db.BeginTxx(ctx, nil)
	if err != nil {
		return []rules.Rule{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	q := `INSERT INTO rules (id, group_id, name, description, conditions, operator, actions) VALUES (:id, :group_id, :name, :description, :conditions, :operator, :actions);`

	for _, rule := range rls {
		dbr, err := toDBRule(rule)
		if err != nil {
			return []rules.Rule{}, errors.Wrap(dbutil.ErrCreateEntity, err)
		}

		if _, err := tx.NamedExecContext(ctx, q, dbr); err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return []rules.Rule{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return []rules.Rule{}, errors.Wrap(dbutil.ErrConflict, err)
				case pgerrcode.StringDataRightTruncationWarning:
					return []rules.Rule{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
				}
			}

			return []rules.Rule{}, errors.Wrap(dbutil.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return []rules.Rule{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	return rls, nil
}

func (rr ruleRepository) RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (rules.RulesPage, error) {
	if _, err := uuid.FromString(groupID); err != nil {
		return rules.RulesPage{}, errors.Wrap(dbutil.ErrNotFound, err)
	}

	gq := "group_id = :group_id"
	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	nq, name := dbutil.GetNameQuery(pm.Name)
	whereClause := dbutil.BuildWhereClause(gq, nq)

	q := fmt.Sprintf(`SELECT id, group_id, name, description, conditions, operator, actions FROM rules %s ORDER BY %s %s %s;`, whereClause, oq, dq, olq)
	qc := fmt.Sprintf(`SELECT COUNT(*) FROM rules WHERE %s;`, gq)

	params := map[string]any{
		"group_id": groupID,
		"name":     name,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	return rr.retrieve(ctx, q, qc, params)
}

func (rr ruleRepository) RetrieveByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (rules.RulesPage, error) {
	if _, err := uuid.FromString(thingID); err != nil {
		return rules.RulesPage{}, errors.Wrap(dbutil.ErrNotFound, err)
	}

	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	tq := "rt.thing_id = :thing_id"
	nq, name := dbutil.GetNameQuery(pm.Name)
	whereClause := dbutil.BuildWhereClause(tq, nq)

	q := fmt.Sprintf(`
		SELECT r.id, r.group_id, r.name, r.description, r.conditions, r.operator, r.actions
		FROM rules r
		INNER JOIN rules_things rt ON r.id = rt.rule_id
		%s
		ORDER BY %s %s
		%s;`,
		whereClause, oq, dq, olq)

	qc := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM rules r
		INNER JOIN rules_things rt ON r.id = rt.rule_id
		%s;`, whereClause)

	params := map[string]any{
		"thing_id": thingID,
		"name":     name,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	return rr.retrieve(ctx, q, qc, params)
}

func (rr ruleRepository) RetrieveByID(ctx context.Context, id string) (rules.Rule, error) {
	q := `SELECT id, group_id, name, description, conditions, operator, actions FROM rules WHERE id = $1;`

	var dbr dbRule
	if err := rr.db.QueryRowxContext(ctx, q, id).StructScan(&dbr); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		//  If there is no result or ID is in an invalid format, return ErrNotFound.
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return rules.Rule{}, errors.Wrap(dbutil.ErrNotFound, err)
		}
		return rules.Rule{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toRule(dbr)
}

func (rr ruleRepository) RetrieveThingIDsByRule(ctx context.Context, ruleID string) ([]string, error) {
	q := `SELECT thing_id FROM rules_things WHERE rule_id = $1;`

	thingIDs := []string{}
	if err := rr.db.SelectContext(ctx, &thingIDs, q, ruleID); err != nil {
		return nil, err
	}

	return thingIDs, nil
}

func (rr ruleRepository) Update(ctx context.Context, r rules.Rule) error {
	q := `UPDATE rules SET name = :name, description = :description, conditions = :conditions, operator = :operator, actions = :actions WHERE id = :id;`

	dbr, err := toDBRule(r)
	if err != nil {
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	res, errdb := rr.db.NamedExecContext(ctx, q, dbr)
	if errdb != nil {
		pgErr, ok := errdb.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(dbutil.ErrMalformedEntity, errdb)
			case pgerrcode.StringDataRightTruncationDataException:
				return errors.Wrap(dbutil.ErrMalformedEntity, err)
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

func (rr ruleRepository) Remove(ctx context.Context, ids ...string) error {
	q := `DELETE FROM rules WHERE id = :id;`

	for _, id := range ids {
		dbr := dbRule{ID: id}
		if _, err := rr.db.NamedExecContext(ctx, q, dbr); err != nil {
			return errors.Wrap(dbutil.ErrRemoveEntity, err)
		}
	}

	return nil
}

func (rr ruleRepository) RemoveByGroup(ctx context.Context, groupID string) error {
	q := `DELETE FROM rules WHERE group_id = :group_id;`

	dbr := dbRule{GroupID: groupID}
	if _, err := rr.db.NamedExecContext(ctx, q, dbr); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (rr ruleRepository) Assign(ctx context.Context, thingID string, ruleIDs ...string) error {
	tx, err := rr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	q := `INSERT INTO rules_things (rule_id, thing_id) VALUES (:rule_id, :thing_id);`

	for _, ruleID := range ruleIDs {
		params := map[string]any{
			"rule_id":  ruleID,
			"thing_id": thingID,
		}

		if _, err := tx.NamedExecContext(ctx, q, params); err != nil {
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

func (rr ruleRepository) Unassign(ctx context.Context, thingID string, ruleIDs ...string) error {
	q := `DELETE FROM rules_things WHERE rule_id = :rule_id AND thing_id = :thing_id;`

	for _, ruleID := range ruleIDs {
		params := map[string]any{
			"rule_id":  ruleID,
			"thing_id": thingID,
		}
		if _, err := rr.db.NamedExecContext(ctx, q, params); err != nil {
			return errors.Wrap(dbutil.ErrRemoveEntity, err)
		}
	}

	return nil
}

func (rr ruleRepository) UnassignByThing(ctx context.Context, thingID string) error {
	q := `DELETE FROM rules_things WHERE thing_id = :thing_id;`

	params := map[string]any{
		"thing_id": thingID,
	}
	if _, err := rr.db.NamedExecContext(ctx, q, params); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (rr ruleRepository) retrieve(ctx context.Context, query, cquery string, params map[string]any) (rules.RulesPage, error) {
	rows, err := rr.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return rules.RulesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []rules.Rule
	for rows.Next() {
		var dbr dbRule
		if err = rows.StructScan(&dbr); err != nil {
			return rules.RulesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		rule, err := toRule(dbr)
		if err != nil {
			return rules.RulesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		items = append(items, rule)
	}

	total, err := dbutil.Total(ctx, rr.db, cquery, params)
	if err != nil {
		return rules.RulesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	page := rules.RulesPage{
		Rules: items,
		Total: total,
	}

	return page, nil
}

type dbRule struct {
	ID          string `db:"id"`
	GroupID     string `db:"group_id"`
	Name        string `db:"name"`
	Description string `db:"description"`
	Conditions  []byte `db:"conditions"`
	Operator    string `db:"operator"`
	Actions     []byte `db:"actions"`
}

func toDBRule(r rules.Rule) (dbRule, error) {
	conditions, err := json.Marshal(r.Conditions)
	if err != nil {
		return dbRule{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	actions, err := json.Marshal(r.Actions)
	if err != nil {
		return dbRule{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	return dbRule{
		ID:          r.ID,
		GroupID:     r.GroupID,
		Name:        r.Name,
		Description: r.Description,
		Conditions:  conditions,
		Operator:    r.Operator,
		Actions:     actions,
	}, nil
}

func toRule(dbr dbRule) (rules.Rule, error) {
	var conditions []rules.Condition
	if err := json.Unmarshal(dbr.Conditions, &conditions); err != nil {
		return rules.Rule{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	var actions []rules.Action
	if err := json.Unmarshal(dbr.Actions, &actions); err != nil {
		return rules.Rule{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	return rules.Rule{
		ID:          dbr.ID,
		GroupID:     dbr.GroupID,
		Name:        dbr.Name,
		Description: dbr.Description,
		Conditions:  conditions,
		Operator:    dbr.Operator,
		Actions:     actions,
	}, nil
}
