package postgres

import (
	"context"
	"database/sql"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var _ auth.RolesRepository = (*rolesRepository)(nil)

type rolesRepository struct {
	db Database
}

// NewRolesRepo instantiates a PostgreSQL implementation of roles
// repository.
func NewRolesRepo(db Database) auth.RolesRepository {
	return &rolesRepository{
		db: db,
	}
}

func (rr rolesRepository) SaveRole(ctx context.Context, userID, role string) error {
	q := `INSERT INTO users_roles (user_id, role) VALUES (:user_id, :role);`

	dbur := toDBUsersRole(userID, role)

	if _, err := rr.db.NamedExecContext(ctx, q, dbur); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (rr rolesRepository) RetrieveRole(ctx context.Context, userID string) (string, error) {
	q := `SELECT role FROM users_roles WHERE user_id = $1;`

	dbur := dbUserRole{UserID: userID}

	if err := rr.db.QueryRowxContext(ctx, q, userID).StructScan(&dbur); err != nil {
		if err == sql.ErrNoRows {
			return "", errors.Wrap(errors.ErrNotFound, err)

		}
		return "", errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	return dbur.Role, nil
}

func (rr rolesRepository) UpdateRole(ctx context.Context, userID, role string) error {
	q := `UPDATE users_roles SET role = :role WHERE user_id = :user_id;`

	dbur := toDBUsersRole(userID, role)

	if _, err := rr.db.NamedExecContext(ctx, q, dbur); err != nil {
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	return nil
}

func (rr rolesRepository) RemoveRole(ctx context.Context, userID string) error {
	q := `DELETE FROM users_roles WHERE user_id = :user_id;`

	dbur := dbUserRole{UserID: userID}

	if _, err := rr.db.NamedExecContext(ctx, q, dbur); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
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
