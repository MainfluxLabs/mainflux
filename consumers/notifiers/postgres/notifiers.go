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

func (nr notifierRepository) Save(ctx context.Context, nfs ...notifiers.Notifier) ([]notifiers.Notifier, error) {
	tx, err := nr.db.BeginTxx(ctx, nil)
	if err != nil {
		return []notifiers.Notifier{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	q := `INSERT INTO notifiers (id, group_id, name, contacts, metadata) VALUES (:id, :group_id, :name, :contacts, :metadata);`

	for _, notifier := range nfs {
		dbNf, err := toDBNotifier(notifier)
		if err != nil {
			return []notifiers.Notifier{}, errors.Wrap(dbutil.ErrCreateEntity, err)
		}

		if _, err := tx.NamedExecContext(ctx, q, dbNf); err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return []notifiers.Notifier{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return []notifiers.Notifier{}, errors.Wrap(dbutil.ErrConflict, err)
				case pgerrcode.StringDataRightTruncationWarning:
					return []notifiers.Notifier{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
				}
			}

			return []notifiers.Notifier{}, errors.Wrap(dbutil.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return []notifiers.Notifier{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	return nfs, nil
}

func (nr notifierRepository) RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (notifiers.NotifiersPage, error) {
	if _, err := uuid.FromString(groupID); err != nil {
		return notifiers.NotifiersPage{}, errors.Wrap(dbutil.ErrNotFound, err)
	}

	oq := dbutil.GetOrderQuery(pm.Order, notifiers.AllowedOrders)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	nq, name := dbutil.GetNameQuery(pm.Name)
	m, mq, err := dbutil.GetMetadataQuery(pm.Metadata)
	if err != nil {
		return notifiers.NotifiersPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	groupFilter := "group_id = :group_id"
	whereClause := dbutil.BuildWhereClause(groupFilter, nq, mq)

	q := fmt.Sprintf(`SELECT id, group_id, name, contacts, metadata FROM notifiers %s ORDER BY %s %s %s;`, whereClause, oq, dq, olq)
	qc := fmt.Sprintf(`SELECT COUNT(*) FROM notifiers %s;`, whereClause)

	params := map[string]any{
		"group_id": groupID,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
		"name":     name,
		"metadata": m,
	}

	rows, err := nr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return notifiers.NotifiersPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []notifiers.Notifier
	for rows.Next() {
		dbNf := dbNotifier{}
		if err := rows.StructScan(&dbNf); err != nil {
			return notifiers.NotifiersPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		notifier, err := toNotifier(dbNf)
		if err != nil {
			return notifiers.NotifiersPage{}, err
		}

		items = append(items, notifier)
	}

	total, err := dbutil.Total(ctx, nr.db, qc, params)
	if err != nil {
		return notifiers.NotifiersPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	page := notifiers.NotifiersPage{
		Notifiers: items,
		Total:     total,
	}
	return page, nil
}

func (nr notifierRepository) RetrieveByID(ctx context.Context, id string) (notifiers.Notifier, error) {
	q := `SELECT group_id, name, contacts, metadata FROM notifiers WHERE id = $1;`

	dbNf := dbNotifier{ID: id}
	if err := nr.db.QueryRowxContext(ctx, q, id).StructScan(&dbNf); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return notifiers.Notifier{}, errors.Wrap(dbutil.ErrNotFound, err)
		}
		return notifiers.Notifier{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toNotifier(dbNf)
}

func (nr notifierRepository) Update(ctx context.Context, n notifiers.Notifier) error {
	q := `UPDATE notifiers SET name = :name, contacts = :contacts, metadata = :metadata WHERE id = :id;`

	dbNf, err := toDBNotifier(n)
	if err != nil {
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	res, errdb := nr.db.NamedExecContext(ctx, q, dbNf)
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

func (nr notifierRepository) Remove(ctx context.Context, ids ...string) error {
	q := `DELETE FROM notifiers WHERE id = :id;`

	for _, id := range ids {
		dbNf := dbNotifier{ID: id}
		if _, err := nr.db.NamedExecContext(ctx, q, dbNf); err != nil {
			return errors.Wrap(dbutil.ErrRemoveEntity, err)
		}
	}

	return nil
}

func (nr notifierRepository) RemoveByGroup(ctx context.Context, groupID string) error {
	q := `DELETE FROM notifiers WHERE group_id = :group_id;`

	dbNf := dbNotifier{GroupID: groupID}
	if _, err := nr.db.NamedExecContext(ctx, q, dbNf); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
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

func toDBNotifier(nf notifiers.Notifier) (dbNotifier, error) {
	contacts := strings.Join(nf.Contacts, ",")
	metadata := []byte("{}")
	if len(nf.Metadata) > 0 {
		b, err := json.Marshal(nf.Metadata)
		if err != nil {
			return dbNotifier{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
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

func toNotifier(dbN dbNotifier) (notifiers.Notifier, error) {
	var metadata map[string]any
	contacts := strings.Split(dbN.Contacts, ",")

	if err := json.Unmarshal([]byte(dbN.Metadata), &metadata); err != nil {
		return notifiers.Notifier{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	return notifiers.Notifier{
		ID:       dbN.ID,
		GroupID:  dbN.GroupID,
		Name:     dbN.Name,
		Contacts: contacts,
		Metadata: metadata,
	}, nil
}
