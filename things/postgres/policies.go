package postgres

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ things.PoliciesRepository = (*policiesRepository)(nil)

type policiesRepository struct {
	db Database
}

// NewPoliciesRepository instantiates a PostgreSQL implementation of policies repository.
func NewPoliciesRepository(db Database) things.PoliciesRepository {
	return &policiesRepository{
		db: db,
	}
}

func (pr policiesRepository) SaveGroupPolicies(ctx context.Context, groupID string, gps ...things.GroupPolicyByID) error {
	tx, err := pr.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	q := `INSERT INTO group_policies (member_id, group_id, policy) VALUES (:member_id, :group_id, :policy);`

	for _, g := range gps {
		gp := things.GroupPolicy{
			MemberID: g.MemberID,
			GroupID:  groupID,
			Policy:   g.Policy,
		}
		dbgp := toDBGroupPolicy(gp)

		if _, err := pr.db.NamedExecContext(ctx, q, dbgp); err != nil {
			tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.ForeignKeyViolation:
					return errors.Wrap(errors.ErrConflict, errors.New(pgErr.Detail))
				case pgerrcode.UniqueViolation:
					return errors.Wrap(errors.ErrConflict, errors.New(pgErr.Detail))
				}
			}
			return errors.Wrap(errors.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (pr policiesRepository) RetrieveGroupPolicy(ctc context.Context, gp things.GroupPolicy) (string, error) {
	q := `SELECT policy FROM group_policies WHERE member_id = :member_id AND group_id = :group_id;`

	params := map[string]interface{}{
		"member_id": gp.MemberID,
		"group_id":  gp.GroupID,
	}

	rows, err := pr.db.NamedQueryContext(ctc, q, params)
	if err != nil {
		return "", errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var policy string
	for rows.Next() {
		if err := rows.Scan(&policy); err != nil {
			return "", errors.Wrap(errors.ErrRetrieveEntity, err)
		}
	}

	return policy, nil
}

func (pr policiesRepository) RetrieveGroupPolicies(ctx context.Context, groupID string, pm things.PageMetadata) (things.GroupPoliciesPage, error) {
	q := `SELECT member_id, policy FROM group_policies WHERE group_id = :group_id LIMIT :limit OFFSET :offset;`

	params := map[string]interface{}{
		"group_id": groupID,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	rows, err := pr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.GroupPoliciesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []things.GroupPolicy
	for rows.Next() {
		dbgp := dbGroupPolicy{}
		if err := rows.StructScan(&dbgp); err != nil {
			return things.GroupPoliciesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		gp := toGroupPolicy(dbgp)
		items = append(items, gp)
	}

	cq := `SELECT COUNT(*) FROM group_policies WHERE group_id = :group_id;`

	total, err := total(ctx, pr.db, cq, params)
	if err != nil {
		return things.GroupPoliciesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	page := things.GroupPoliciesPage{
		GroupPolicies: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (pr policiesRepository) RetrieveAllGroupPolicies(ctx context.Context) ([]things.GroupPolicy, error) {
	q := `SELECT member_id, group_id, policy FROM group_policies;`

	rows, err := pr.db.NamedQueryContext(ctx, q, map[string]interface{}{})
	if err != nil {
		return nil, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []things.GroupPolicy
	for rows.Next() {
		dbgp := dbGroupPolicy{}
		if err := rows.StructScan(&dbgp); err != nil {
			return nil, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		gp := toGroupPolicy(dbgp)
		items = append(items, gp)
	}

	return items, nil
}

func (pr policiesRepository) RemoveGroupPolicies(ctx context.Context, groupID string, memberIDs ...string) error {
	q := `DELETE FROM group_policies WHERE member_id = :member_id AND group_id = :group_id;`

	for _, memberID := range memberIDs {
		dbgp := dbGroupPolicy{
			MemberID: memberID,
			GroupID:  groupID,
		}

		if _, err := pr.db.NamedExecContext(ctx, q, dbgp); err != nil {
			return errors.Wrap(errors.ErrRemoveEntity, err)
		}
	}
	return nil
}

func (pr policiesRepository) UpdateGroupPolicies(ctx context.Context, groupID string, gps ...things.GroupPolicyByID) error {
	q := `UPDATE group_policies SET policy = :policy WHERE member_id = :member_id AND group_id = :group_id;`

	for _, g := range gps {
		gp := things.GroupPolicy{
			MemberID: g.MemberID,
			GroupID:  groupID,
			Policy:   g.Policy,
		}
		dbgp := toDBGroupPolicy(gp)

		row, err := pr.db.NamedExecContext(ctx, q, dbgp)
		if err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.StringDataRightTruncationDataException:
					return errors.Wrap(errors.ErrMalformedEntity, err)
				}
			}

			return errors.Wrap(errors.ErrUpdateEntity, err)
		}

		cnt, err := row.RowsAffected()
		if err != nil {
			return errors.Wrap(errors.ErrUpdateEntity, err)
		}

		if cnt != 1 {
			return errors.Wrap(errors.ErrNotFound, err)
		}
	}

	return nil
}

type dbGroupPolicy struct {
	MemberID string `db:"member_id"`
	GroupID  string `db:"group_id"`
	Policy   string `db:"policy"`
}

func toDBGroupPolicy(gp things.GroupPolicy) dbGroupPolicy {
	return dbGroupPolicy{
		MemberID: gp.MemberID,
		GroupID:  gp.GroupID,
		Policy:   gp.Policy,
	}
}

func toGroupPolicy(dbgp dbGroupPolicy) things.GroupPolicy {
	return things.GroupPolicy{
		GroupID:  dbgp.GroupID,
		MemberID: dbgp.MemberID,
		Policy:   dbgp.Policy,
	}
}
