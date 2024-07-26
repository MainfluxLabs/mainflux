package postgres

import (
	"context"
	"database/sql"
	"strings"

	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ notifiers.NotifierRepository = (*notifierRepository)(nil)

type notifierRepository struct {
	db Database
}

// NewNotifierRepository instantiates a PostgreSQL implementation of notifier repository.
func NewNotifierRepository(db Database) notifiers.NotifierRepository {
	return &notifierRepository{
		db: db,
	}
}

func (nr notifierRepository) Save(ctx context.Context, nfs ...things.Notifier) ([]things.Notifier, error) {
	tx, err := nr.db.BeginTxx(ctx, nil)
	if err != nil {
		return []things.Notifier{}, errors.Wrap(errors.ErrCreateEntity, err)
	}

	q := `INSERT INTO notifiers (id, group_id, contacts) VALUES (:id, :group_id, :contacts);`

	for _, notifier := range nfs {
		dbNf, err := toDBNotifier(notifier)
		if err != nil {
			return []things.Notifier{}, errors.Wrap(errors.ErrCreateEntity, err)
		}

		if _, err := tx.NamedExecContext(ctx, q, dbNf); err != nil {
			tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return []things.Notifier{}, errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return []things.Notifier{}, errors.Wrap(errors.ErrConflict, err)
				case pgerrcode.StringDataRightTruncationWarning:
					return []things.Notifier{}, errors.Wrap(errors.ErrMalformedEntity, err)
				}
			}

			return []things.Notifier{}, errors.Wrap(errors.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return []things.Notifier{}, errors.Wrap(errors.ErrCreateEntity, err)
	}
	return nfs, nil
}

func (nr notifierRepository) RetrieveByGroupID(ctx context.Context, groupID string) ([]things.Notifier, error) {
	if _, err := uuid.FromString(groupID); err != nil {
		return []things.Notifier{}, errors.Wrap(errors.ErrNotFound, err)
	}
	q := `SELECT id, group_id, contacts FROM notifiers WHERE group_id = :group_id;`

	params := map[string]interface{}{
		"group_id": groupID,
	}

	rows, err := nr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return nil, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []things.Notifier
	for rows.Next() {
		dbNf := dbNotifier{GroupID: groupID}
		if err := rows.StructScan(&dbNf); err != nil {
			return nil, errors.Wrap(errors.ErrRetrieveEntity, err)
		}
		notifier, err := toNotifier(dbNf)
		if err != nil {
			return nil, err
		}

		items = append(items, notifier)
	}
	return items, nil
}

func (nr notifierRepository) RetrieveByID(ctx context.Context, id string) (things.Notifier, error) {
	q := `SELECT group_id, contacts FROM notifiers WHERE id = $1;`

	dbNf := dbNotifier{ID: id}
	if err := nr.db.QueryRowxContext(ctx, q, id).StructScan(&dbNf); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return things.Notifier{}, errors.Wrap(errors.ErrNotFound, err)
		}
		return things.Notifier{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	return toNotifier(dbNf)
}

func (nr notifierRepository) Update(ctx context.Context, n things.Notifier) error {
	q := `UPDATE notifiers SET contacts = :contacts WHERE id = :id;`

	dbNf, err := toDBNotifier(n)
	if err != nil {
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	res, errdb := nr.db.NamedExecContext(ctx, q, dbNf)
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

func (nr notifierRepository) Remove(ctx context.Context, groupID string, ids ...string) error {
	for _, id := range ids {
		dbNf := dbNotifier{
			ID:      id,
			GroupID: groupID,
		}
		q := `DELETE FROM notifiers WHERE id = :id AND group_id = :group_id;`
		_, err := nr.db.NamedExecContext(ctx, q, dbNf)
		if err != nil {
			return errors.Wrap(errors.ErrRemoveEntity, err)
		}
	}

	return nil
}

type dbNotifier struct {
	ID       string `db:"id"`
	GroupID  string `db:"group_id"`
	Contacts string `json:"contacts"`
}

func toDBNotifier(nf things.Notifier) (dbNotifier, error) {
	contacts := strings.Join(nf.Contacts, ",")

	return dbNotifier{
		ID:       nf.ID,
		GroupID:  nf.GroupID,
		Contacts: contacts,
	}, nil
}

func toNotifier(dbN dbNotifier) (things.Notifier, error) {
	contacts := strings.Split(dbN.Contacts, ",")

	return things.Notifier{
		ID:       dbN.ID,
		GroupID:  dbN.GroupID,
		Contacts: contacts,
	}, nil
}
