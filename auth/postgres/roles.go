package postgres

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ auth.RolesRepository = (*rolesRepository)(nil)

type rolesRepository struct {
	db dbutil.Database
}

// NewRolesRepo instantiates a PostgreSQL implementation of roles repository.
func NewRolesRepo(db dbutil.Database) auth.RolesRepository {
	return &rolesRepository{
		db: db,
	}
}

func (rr rolesRepository) SaveRole(ctx context.Context, userID, role string) error {
	q := `INSERT INTO users_roles (user_id, role) VALUES (:user_id, :role);`

	dbur := toDBUsersRole(userID, role)

	if _, err := rr.db.NamedExecContext(ctx, q, dbur); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.UniqueViolation {
			return errors.Wrap(dbutil.ErrConflict, err)
		}

		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return nil
}

func (rr rolesRepository) RetrieveRole(ctx context.Context, userID string) (string, error) {
	q := `SELECT role FROM users_roles WHERE user_id = :user_id;`

	params := map[string]any{"user_id": userID}

	rows, err := rr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return "", errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var role string
	for rows.Next() {
		if err := rows.Scan(&role); err != nil {
			return "", errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
	}

	return role, nil
}

func (rr rolesRepository) RetrieveRoles(ctx context.Context, userIDs []string) (map[string]string, error) {
	q := `SELECT user_id, role FROM users_roles WHERE user_id = ANY($1::uuid[]);`

	var dburs []dbUserRole
	if err := rr.db.SelectContext(ctx, &dburs, q, userIDs); err != nil {
		return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	roles := make(map[string]string, len(dburs))
	for _, dbur := range dburs {
		roles[dbur.UserID] = dbur.Role
	}

	return roles, nil
}

func (rr rolesRepository) UpdateRole(ctx context.Context, userID, role string) error {
	q := `UPDATE users_roles SET role = :role WHERE user_id = :user_id;`

	dbur := toDBUsersRole(userID, role)

	if _, err := rr.db.NamedExecContext(ctx, q, dbur); err != nil {
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	return nil
}

func (rr rolesRepository) RemoveRole(ctx context.Context, userID string) error {
	q := `DELETE FROM users_roles WHERE user_id = :user_id;`

	dbur := dbUserRole{UserID: userID}

	res, err := rr.db.NamedExecContext(ctx, q, dbur)
	if err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	if cnt != 1 {
		return dbutil.ErrNotFound
	}

	return nil
}

func (rr rolesRepository) BackupAll(ctx context.Context) ([]auth.UserRole, error) {
	q := `SELECT user_id, role FROM users_roles;`

	var dburs []dbUserRole
	if err := rr.db.SelectContext(ctx, &dburs, q); err != nil {
		return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	var urs []auth.UserRole
	for _, dbur := range dburs {
		urs = append(urs, toUserRole(dbur))
	}

	return urs, nil
}

func (rr rolesRepository) RestoreAll(ctx context.Context, urs ...auth.UserRole) error {
	tx, err := rr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	q := `INSERT INTO users_roles (user_id, role) VALUES (:user_id, :role)
	      ON CONFLICT (user_id) DO NOTHING;`

	for _, ur := range urs {
		dbur := toDBUsersRole(ur.UserID, ur.Role)

		if _, err := tx.NamedExecContext(ctx, q, dbur); err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation, pgerrcode.CheckViolation:
					return errors.Wrap(dbutil.ErrMalformedEntity, err)
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

type dbUserRole struct {
	UserID string `db:"user_id"`
	Role   string `db:"role"`
}

func toDBUsersRole(userID, role string) dbUserRole {
	return dbUserRole{
		UserID: userID,
		Role:   role,
	}
}

func toUserRole(dbur dbUserRole) auth.UserRole {
	return auth.UserRole{
		UserID: dbur.UserID,
		Role:   dbur.Role,
	}
}
