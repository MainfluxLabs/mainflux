package postgres

import (
	"context"
	"database/sql"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var _ auth.RoleRepository = (*roleRepository)(nil)

type roleRepository struct {
	db Database
}

// NewOrgRepo instantiates a PostgreSQL implementation of org
// repository.
func NewRoleRepo(db Database) auth.RoleRepository {
	return &roleRepository{
		db: db,
	}
}

func (rr roleRepository) SaveRole(ctx context.Context, id, role string) error {
	q := `INSERT INTO users_roles (user_id, role) VALUES (:user_id, :role);`

	dbur := toDBUsersRole(id, role)

	if _, err := rr.db.NamedExecContext(ctx, q, dbur); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (rr roleRepository) RetrieveRole(ctx context.Context, id string) (string, error) {
	q := `SELECT role FROM users_roles WHERE user_id = $1;`

	dbur := dbUserRole{ID: id}

	if err := rr.db.QueryRowxContext(ctx, q, id).StructScan(&dbur); err != nil {
		if err == sql.ErrNoRows {
			return "", errors.Wrap(errors.ErrNotFound, err)

		}
		return "", errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	return dbur.Role, nil
}

func (rr roleRepository) UpdateRole(ctx context.Context, id, role string) error {
	q := `UPDATE users_roles SET role = :role WHERE user_id = :user_id;`

	dbur := toDBUsersRole(id, role)

	if _, err := rr.db.NamedExecContext(ctx, q, dbur); err != nil {
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	return nil
}

func (rr roleRepository) RemoveRole(ctx context.Context, id string) error {
	q := `DELETE FROM users_roles WHERE user_id = :user_id;`

	dbur := dbUserRole{ID: id}

	if _, err := rr.db.NamedExecContext(ctx, q, dbur); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	return nil
}

type dbUserRole struct {
	ID   string `db:"user_id"`
	Role string `db:"role"`
}

func toDBUsersRole(id, role string) dbUserRole {
	return dbUserRole{
		ID:   id,
		Role: role,
	}
}
