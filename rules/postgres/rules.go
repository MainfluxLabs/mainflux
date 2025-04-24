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
		return []rules.Rule{}, errors.Wrap(errors.ErrCreateEntity, err)
	}

	q := `INSERT INTO rules (id, profile_id, group_id, name, description, condition, actions) VALUES (:id, :profile_id, :group_id, :name, :description, :condition, :actions);`

	for _, rule := range rls {
		dbr, err := toDBRule(rule)
		if err != nil {
			return []rules.Rule{}, errors.Wrap(errors.ErrCreateEntity, err)
		}

		if _, err := tx.NamedExecContext(ctx, q, dbr); err != nil {
			tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return []rules.Rule{}, errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return []rules.Rule{}, errors.Wrap(errors.ErrConflict, err)
				case pgerrcode.StringDataRightTruncationWarning:
					return []rules.Rule{}, errors.Wrap(errors.ErrMalformedEntity, err)
				}
			}

			return []rules.Rule{}, errors.Wrap(errors.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return []rules.Rule{}, errors.Wrap(errors.ErrCreateEntity, err)
	}
	return rls, nil
}

func (rr ruleRepository) RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (rules.RulesPage, error) {
	if _, err := uuid.FromString(groupID); err != nil {
		return rules.RulesPage{}, errors.Wrap(errors.ErrNotFound, err)
	}

	gq := "group_id = :group_id"
	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	nq, name := dbutil.GetNameQuery(pm.Name)
	whereClause := dbutil.BuildWhereClause(gq, nq)

	q := fmt.Sprintf(`SELECT id, profile_id, group_id, name, description, condition, actions FROM rules %s ORDER BY %s %s %s;`, whereClause, oq, dq, olq)
	qc := fmt.Sprintf(`SELECT COUNT(*) FROM rules WHERE %s;`, gq)

	params := map[string]interface{}{
		"group_id": groupID,
		"name":     name,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	return rr.retrieve(ctx, q, qc, params)
}

func (rr ruleRepository) RetrieveByProfile(ctx context.Context, profileID string, pm apiutil.PageMetadata) (rules.RulesPage, error) {
	if _, err := uuid.FromString(profileID); err != nil {
		return rules.RulesPage{}, errors.Wrap(errors.ErrNotFound, err)
	}

	pq := "profile_id = :profile_id"
	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	nq, name := dbutil.GetNameQuery(pm.Name)
	whereClause := dbutil.BuildWhereClause(pq, nq)

	q := fmt.Sprintf(`SELECT id, profile_id, group_id, name, description, condition, actions FROM rules %s ORDER BY %s %s %s;`, whereClause, oq, dq, olq)
	qc := fmt.Sprintf(`SELECT COUNT(*) FROM rules WHERE %s;`, pq)

	params := map[string]interface{}{
		"profile_id": profileID,
		"name":       name,
		"limit":      pm.Limit,
		"offset":     pm.Offset,
	}

	return rr.retrieve(ctx, q, qc, params)
}

func (rr ruleRepository) RetrieveByID(ctx context.Context, id string) (rules.Rule, error) {
	q := `SELECT id, profile_id, group_id, name, description, condition, actions FROM rules WHERE id = $1;`

	var dbr dbRule
	if err := rr.db.QueryRowxContext(ctx, q, id).StructScan(&dbr); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		//  If there is no result or ID is in an invalid format, return ErrNotFound.
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return rules.Rule{}, errors.Wrap(errors.ErrNotFound, err)
		}
		return rules.Rule{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	return toRule(dbr)
}

func (rr ruleRepository) Update(ctx context.Context, r rules.Rule) error {
	q := `UPDATE rules SET name = :name, description = :description, condition = :condition, actions = :actions WHERE id = :id;`

	dbr, err := toDBRule(r)
	if err != nil {
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	res, errdb := rr.db.NamedExecContext(ctx, q, dbr)
	if errdb != nil {
		pgErr, ok := errdb.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(errors.ErrMalformedEntity, errdb)
			case pgerrcode.StringDataRightTruncationDataException:
				return errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}

		return errors.Wrap(errors.ErrUpdateEntity, errdb)
	}

	cnt, errdb := res.RowsAffected()
	if errdb != nil {
		return errors.Wrap(errors.ErrUpdateEntity, errdb)
	}

	if cnt == 0 {
		return errors.ErrNotFound
	}

	return nil
}

func (rr ruleRepository) Remove(ctx context.Context, ids ...string) error {
	for _, id := range ids {
		dbr := dbRule{ID: id}
		q := `DELETE FROM rules WHERE id = :id;`

		_, err := rr.db.NamedExecContext(ctx, q, dbr)
		if err != nil {
			return errors.Wrap(errors.ErrRemoveEntity, err)
		}
	}

	return nil
}

func (rr ruleRepository) retrieve(ctx context.Context, query, cquery string, params map[string]interface{}) (rules.RulesPage, error) {
	rows, err := rr.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return rules.RulesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []rules.Rule
	for rows.Next() {
		var dbr dbRule
		if err = rows.StructScan(&dbr); err != nil {
			return rules.RulesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		rule, err := toRule(dbr)
		if err != nil {
			return rules.RulesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		items = append(items, rule)
	}

	total, err := dbutil.Total(ctx, rr.db, cquery, params)
	if err != nil {
		return rules.RulesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	page := rules.RulesPage{
		Rules: items,
		PageMetadata: apiutil.PageMetadata{
			Total:  total,
			Offset: params["offset"].(uint64),
			Limit:  params["limit"].(uint64),
		},
	}

	return page, nil
}

type dbRule struct {
	ID          string `db:"id"`
	ProfileID   string `db:"profile_id"`
	GroupID     string `db:"group_id"`
	Name        string `db:"name"`
	Description string `db:"description"`
	Condition   []byte `db:"condition"`
	Actions     []byte `db:"actions"`
}

func toDBRule(r rules.Rule) (dbRule, error) {
	condition, err := json.Marshal(r.Condition)
	if err != nil {
		return dbRule{}, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	actions, err := json.Marshal(r.Actions)
	if err != nil {
		return dbRule{}, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return dbRule{
		ID:          r.ID,
		ProfileID:   r.ProfileID,
		GroupID:     r.GroupID,
		Name:        r.Name,
		Description: r.Description,
		Condition:   condition,
		Actions:     actions,
	}, nil
}

func toRule(dbr dbRule) (rules.Rule, error) {
	var condition rules.Condition
	if err := json.Unmarshal(dbr.Condition, &condition); err != nil {
		return rules.Rule{}, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	var actions []rules.Action
	if err := json.Unmarshal(dbr.Actions, &actions); err != nil {
		return rules.Rule{}, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return rules.Rule{
		ID:          dbr.ID,
		ProfileID:   dbr.ProfileID,
		GroupID:     dbr.GroupID,
		Name:        dbr.Name,
		Description: dbr.Description,
		Condition:   condition,
		Actions:     actions,
	}, nil
}
