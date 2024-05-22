package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/webhooks"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jackc/pgerrcode"
)

var _ webhooks.WebhookRepository = (*webhookRepository)(nil)

type webhookRepository struct {
	db Database
}

// NewWebhookRepository instantiates a PostgreSQL implementation of webhook repository.
func NewWebhookRepository(db Database) webhooks.WebhookRepository {
	return &webhookRepository{
		db: db,
	}
}

func (wr webhookRepository) Save(ctx context.Context, whs ...webhooks.Webhook) ([]webhooks.Webhook, error) {
	tx, err := wr.db.BeginTxx(ctx, nil)
	if err != nil {
		return []webhooks.Webhook{}, errors.Wrap(errors.ErrCreateEntity, err)
	}

	q := `INSERT INTO webhooks (id, group_id, name, url, headers) VALUES (:id, :group_id, :name, :url, :headers);`

	for _, webhook := range whs {
		dbWh, err := toDBWebhook(webhook)
		if err != nil {
			return []webhooks.Webhook{}, errors.Wrap(errors.ErrCreateEntity, err)
		}

		if _, err := tx.NamedExecContext(ctx, q, dbWh); err != nil {
			tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return []webhooks.Webhook{}, errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return []webhooks.Webhook{}, errors.Wrap(errors.ErrConflict, err)
				case pgerrcode.StringDataRightTruncationWarning:
					return []webhooks.Webhook{}, errors.Wrap(errors.ErrMalformedEntity, err)
				}
			}

			return []webhooks.Webhook{}, errors.Wrap(errors.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return []webhooks.Webhook{}, errors.Wrap(errors.ErrCreateEntity, err)
	}
	return whs, nil
}

func (wr webhookRepository) RetrieveByGroupID(ctx context.Context, groupID string) ([]webhooks.Webhook, error) {
	if _, err := uuid.FromString(groupID); err != nil {
		return []webhooks.Webhook{}, errors.Wrap(errors.ErrNotFound, err)
	}
	q := `SELECT id, name, url, headers FROM webhooks WHERE group_id = :group_id;`

	params := map[string]interface{}{
		"group_id": groupID,
	}

	rows, err := wr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return nil, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []webhooks.Webhook
	for rows.Next() {
		dbWh := dbWebhook{GroupID: groupID}
		if err := rows.StructScan(&dbWh); err != nil {
			return nil, errors.Wrap(errors.ErrRetrieveEntity, err)
		}
		webhook, err := toWebhook(dbWh)
		if err != nil {
			return nil, err
		}

		items = append(items, webhook)
	}
	return items, nil
}

func (wr webhookRepository) RetrieveByID(ctx context.Context, id string) (webhooks.Webhook, error) {
	q := `SELECT group_id, name, url, headers FROM webhooks WHERE id = $1;`

	dbwh := dbWebhook{ID: id}
	if err := wr.db.QueryRowxContext(ctx, q, id).StructScan(&dbwh); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		//  If there is no result or ID is in an invalid format, return ErrNotFound.
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return webhooks.Webhook{}, errors.Wrap(errors.ErrNotFound, err)
		}
		return webhooks.Webhook{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	return toWebhook(dbwh)
}

func (wr webhookRepository) Update(ctx context.Context, w webhooks.Webhook) error {
	q := `UPDATE webhooks SET name = :name, url = :url, headers = :headers WHERE id = :id;`

	dbwh, err := toDBWebhook(w)
	if err != nil {
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	res, errdb := wr.db.NamedExecContext(ctx, q, dbwh)
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

func (wr webhookRepository) Remove(ctx context.Context, groupID string, ids ...string) error {
	for _, id := range ids {
		dbwh := dbWebhook{
			ID:      id,
			GroupID: groupID,
		}
		q := `DELETE FROM webhooks WHERE id = :id AND group_id = :group_id;`
		_, err := wr.db.NamedExecContext(ctx, q, dbwh)
		if err != nil {
			return errors.Wrap(errors.ErrRemoveEntity, err)
		}
	}

	return nil
}

type dbWebhook struct {
	ID      string `db:"id"`
	GroupID string `db:"group_id"`
	Name    string `db:"name"`
	Url     string `db:"url"`
	Headers []byte `db:"headers"`
}

func toDBWebhook(wh webhooks.Webhook) (dbWebhook, error) {
	data := []byte("{}")
	if len(wh.Headers) > 0 {
		b, err := json.Marshal(wh.Headers)
		if err != nil {
			return dbWebhook{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
		data = b
	}

	return dbWebhook{
		ID:      wh.ID,
		GroupID: wh.GroupID,
		Name:    wh.Name,
		Url:     wh.Url,
		Headers: data,
	}, nil
}

func toWebhook(dbW dbWebhook) (webhooks.Webhook, error) {
	var headers map[string]string
	if err := json.Unmarshal([]byte(dbW.Headers), &headers); err != nil {
		return webhooks.Webhook{}, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return webhooks.Webhook{
		ID:      dbW.ID,
		GroupID: dbW.GroupID,
		Name:    dbW.Name,
		Url:     dbW.Url,
		Headers: headers,
	}, nil
}
