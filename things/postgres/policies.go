package postgres

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ things.RolesRepository = (*rolesRepository)(nil)

type rolesRepository struct {
	db Database
}

// NewRolesRepository instantiates a PostgreSQL implementation of policies repository.
func NewRolesRepository(db Database) things.RolesRepository {
	return &rolesRepository{
		db: db,
	}
}

func (pr rolesRepository) SaveRolesByGroup(ctx context.Context, groupID string, gps ...things.GroupRoles) error {
	tx, err := pr.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	q := `INSERT INTO group_roles (member_id, group_id, role) VALUES (:member_id, :group_id, :role);`

	for _, g := range gps {
		gp := things.GroupMembers{
			MemberID: g.MemberID,
			GroupID:  groupID,
			Role:     g.Role,
		}
		dbgp := toDBGroupMembers(gp)

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

func (pr rolesRepository) RetrieveRole(ctc context.Context, gp things.GroupMembers) (string, error) {
	q := `SELECT role FROM group_roles WHERE member_id = :member_id AND group_id = :group_id;`

	params := map[string]interface{}{
		"member_id": gp.MemberID,
		"group_id":  gp.GroupID,
	}

	rows, err := pr.db.NamedQueryContext(ctc, q, params)
	if err != nil {
		return "", errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var role string
	for rows.Next() {
		if err := rows.Scan(&role); err != nil {
			return "", errors.Wrap(errors.ErrRetrieveEntity, err)
		}
	}

	return role, nil
}

func (pr rolesRepository) RetrieveRolesByGroup(ctx context.Context, groupID string, pm things.PageMetadata) (things.GroupRolesPage, error) {
	q := `SELECT member_id, role FROM group_roles WHERE group_id = :group_id LIMIT :limit OFFSET :offset;`

	params := map[string]interface{}{
		"group_id": groupID,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	rows, err := pr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.GroupRolesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []things.GroupMembers
	for rows.Next() {
		dbgp := dbGroupMembers{}
		if err := rows.StructScan(&dbgp); err != nil {
			return things.GroupRolesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		gp := toGroupMembers(dbgp)
		items = append(items, gp)
	}

	cq := `SELECT COUNT(*) FROM group_roles WHERE group_id = :group_id;`

	total, err := total(ctx, pr.db, cq, params)
	if err != nil {
		return things.GroupRolesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	page := things.GroupRolesPage{
		GroupRoles: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (pr rolesRepository) RetrieveAllRolesByGroup(ctx context.Context) ([]things.GroupMembers, error) {
	q := `SELECT member_id, group_id, role FROM group_roles;`

	rows, err := pr.db.NamedQueryContext(ctx, q, map[string]interface{}{})
	if err != nil {
		return nil, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []things.GroupMembers
	for rows.Next() {
		dbgp := dbGroupMembers{}
		if err := rows.StructScan(&dbgp); err != nil {
			return nil, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		gp := toGroupMembers(dbgp)
		items = append(items, gp)
	}

	return items, nil
}

func (pr rolesRepository) RemoveRolesByGroup(ctx context.Context, groupID string, memberIDs ...string) error {
	q := `DELETE FROM group_roles WHERE member_id = :member_id AND group_id = :group_id;`

	for _, memberID := range memberIDs {
		dbgp := dbGroupMembers{
			MemberID: memberID,
			GroupID:  groupID,
		}

		if _, err := pr.db.NamedExecContext(ctx, q, dbgp); err != nil {
			return errors.Wrap(errors.ErrRemoveEntity, err)
		}
	}
	return nil
}

func (pr rolesRepository) UpdateRolesByGroup(ctx context.Context, groupID string, gps ...things.GroupRoles) error {
	q := `UPDATE group_roles SET role = :role WHERE member_id = :member_id AND group_id = :group_id;`

	for _, g := range gps {
		gp := things.GroupMembers{
			MemberID: g.MemberID,
			GroupID:  groupID,
			Role:     g.Role,
		}
		dbgp := toDBGroupMembers(gp)

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

type dbGroupMembers struct {
	MemberID string `db:"member_id"`
	GroupID  string `db:"group_id"`
	Role     string `db:"role"`
}

func toDBGroupMembers(gp things.GroupMembers) dbGroupMembers {
	return dbGroupMembers{
		MemberID: gp.MemberID,
		GroupID:  gp.GroupID,
		Role:     gp.Role,
	}
}

func toGroupMembers(dbgp dbGroupMembers) things.GroupMembers {
	return things.GroupMembers{
		GroupID:  dbgp.GroupID,
		MemberID: dbgp.MemberID,
		Role:     dbgp.Role,
	}
}
