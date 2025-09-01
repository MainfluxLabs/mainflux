package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ users.EmailVerificationRepository = (*emailVerificationRepository)(nil)

type emailVerificationRepository struct {
	db dbutil.Database
}

func NewEmailVerificationRepo(db dbutil.Database) users.EmailVerificationRepository {
	return &emailVerificationRepository{
		db: db,
	}
}

func (evr emailVerificationRepository) Save(ctx context.Context, ev users.EmailVerification) (string, error) {
	q := `
		INSERT INTO verifications (token, email, password, created_at, expires_at)
		VALUES (:token, :email, :password, :created_at, :expires_at)
		RETURNING token
	`
	dbV := toDBVerification(ev)

	rows, err := evr.db.NamedQueryContext(ctx, q, dbV)
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return "", errors.Wrap(dbutil.ErrMalformedEntity, err)
			case pgerrcode.UniqueViolation:
				return "", errors.Wrap(dbutil.ErrConflict, err)
			}
		}

		return "", errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	defer rows.Close()
	rows.Next()

	var token string
	if err := rows.Scan(&token); err != nil {
		return "", err
	}

	return token, nil
}

func (evr emailVerificationRepository) RetrieveByToken(ctx context.Context, confirmToken string) (users.EmailVerification, error) {
	q := `
		SELECT token, email, password, created_at, expires_at
		FROM verifications	
		WHERE token = $1
	`

	dbv := dbEmailVerification{
		Token: confirmToken,
	}

	if err := evr.db.QueryRowxContext(ctx, q, confirmToken).StructScan(&dbv); err != nil {
		if err == sql.ErrNoRows {
			return users.EmailVerification{}, errors.Wrap(dbutil.ErrNotFound, err)
		}

		return users.EmailVerification{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toVerification(dbv), nil
}

func (evr emailVerificationRepository) Remove(ctx context.Context, confirmToken string) error {
	q := `
		DELETE FROM verifications
		WHERE token = :token	
	`

	res, err := evr.db.NamedExecContext(ctx, q, map[string]any{"token": confirmToken})
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}

		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	rowsDeleted, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsDeleted != 1 {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func toDBVerification(ev users.EmailVerification) dbEmailVerification {
	return dbEmailVerification{
		Email:     ev.User.Email,
		Password:  ev.User.Password,
		Token:     ev.Token,
		CreatedAt: ev.CreatedAt,
		ExpiresAt: ev.ExpiresAt,
	}
}

func toVerification(dbv dbEmailVerification) users.EmailVerification {
	return users.EmailVerification{
		User: users.User{
			Email:    dbv.Email,
			Password: dbv.Password,
		},
		Token:     dbv.Token,
		CreatedAt: dbv.CreatedAt,
		ExpiresAt: dbv.ExpiresAt,
	}
}

type dbEmailVerification struct {
	Email     string    `db:"email"`
	Password  string    `db:"password"`
	Token     string    `db:"token"`
	CreatedAt time.Time `db:"created_at"`
	ExpiresAt time.Time `db:"expires_at"`
}
