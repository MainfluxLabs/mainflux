package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ notifiers.NotifierRepository = (*notifierRepository)(nil)

type notifierRepository struct {
	db dbutil.Database
}

// NewNotifierRepository instantiates a PostgreSQL implementation of notifier repository.
func NewNotifierRepository(db dbutil.Database) notifiers.NotifierRepository {
	return &notifierRepository{
		db: db,
	}
}

func (nr notifierRepository) Save(ctx context.Context, nfs ...things.Notifier) ([]things.Notifier, error) {
	tx, err := nr.db.BeginTxx(ctx, nil)
	if err != nil {
		return []things.Notifier{}, errors.Wrap(errors.ErrCreateEntity, err)
	}

	q := `INSERT INTO notifiers (id, group_id, name, contacts, metadata) VALUES (:id, :group_id, :name, :contacts, :metadata);`

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

func (nr notifierRepository) RetrieveByGroupID(ctx context.Context, groupID string, pm apiutil.PageMetadata) (things.NotifiersPage, error) {
	if _, err := uuid.FromString(groupID); err != nil {
		return things.NotifiersPage{}, errors.Wrap(errors.ErrNotFound, err)
	}

	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	q := fmt.Sprintf(`SELECT id, group_id, name, contacts, metadata FROM notifiers WHERE group_id = :group_id ORDER BY %s %s %s;`, oq, dq, olq)
	qc := `SELECT COUNT(*) FROM notifiers WHERE group_id = $1;`

	params := map[string]interface{}{
		"group_id": groupID,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	rows, err := nr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.NotifiersPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []things.Notifier
	for rows.Next() {
		dbNf := dbNotifier{}
		if err := rows.StructScan(&dbNf); err != nil {
			return things.NotifiersPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}
		notifier, err := toNotifier(dbNf)
		if err != nil {
			return things.NotifiersPage{}, err
		}

		items = append(items, notifier)
	}

	var total uint64
	if err := nr.db.GetContext(ctx, &total, qc, groupID); err != nil {
		return things.NotifiersPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	page := things.NotifiersPage{
		Notifiers: items,
		PageMetadata: apiutil.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}
	return page, nil
}

func (nr notifierRepository) RetrieveByID(ctx context.Context, id string) (things.Notifier, error) {
	q := `SELECT group_id, name, contacts, metadata FROM notifiers WHERE id = $1;`

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
	q := `UPDATE notifiers SET name = :name, contacts = :contacts, metadata = :metadata WHERE id = :id;`

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

func (nr notifierRepository) Remove(ctx context.Context, ids ...string) error {
	for _, id := range ids {
		dbNf := dbNotifier{ID: id}
		q := `DELETE FROM notifiers WHERE id = :id;`

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
	Name     string `db:"name"`
	Contacts string `json:"contacts"`
	Metadata []byte `db:"metadata"`
}

func toDBNotifier(nf things.Notifier) (dbNotifier, error) {
	contacts := strings.Join(nf.Contacts, ",")
	metadata := []byte("{}")
	if len(nf.Metadata) > 0 {
		b, err := json.Marshal(nf.Metadata)
		if err != nil {
			return dbNotifier{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
		metadata = b
	}

	return dbNotifier{
		ID:       nf.ID,
		GroupID:  nf.GroupID,
		Name:     nf.Name,
		Contacts: contacts,
		Metadata: metadata,
	}, nil
}

func toNotifier(dbN dbNotifier) (things.Notifier, error) {
	var metadata map[string]interface{}
	contacts := strings.Split(dbN.Contacts, ",")

	if err := json.Unmarshal([]byte(dbN.Metadata), &metadata); err != nil {
		return things.Notifier{}, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return things.Notifier{
		ID:       dbN.ID,
		GroupID:  dbN.GroupID,
		Name:     dbN.Name,
		Contacts: contacts,
		Metadata: metadata,
	}, nil
}
