package postgres

import (
	"context"

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

	q := `INSERT INTO webhooks (thing_id, name, format, url) VALUES (:thing_id, :name, :format, :url);`

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

func (wr webhookRepository) RetrieveByThingID(ctx context.Context, thingID string) ([]webhooks.Webhook, error) {
	if _, err := uuid.FromString(thingID); err != nil {
		return []webhooks.Webhook{}, errors.Wrap(errors.ErrNotFound, err)
	}
	q := `SELECT thing_id, name, format, url FROM webhooks WHERE thing_id = :thing_id;`

	params := map[string]interface{}{
		"thing_id": thingID,
	}

	rows, err := wr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return nil, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []webhooks.Webhook
	for rows.Next() {
		dbWh := dbWebhook{}
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

type dbWebhook struct {
	ThingID string `db:"thing_id"`
	Name    string `db:"name"`
	Format  string `db:"format"`
	Url     string `db:"url"`
}

func toDBWebhook(wh webhooks.Webhook) (dbWebhook, error) {
	return dbWebhook{
		ThingID: wh.ThingID,
		Name:    wh.Name,
		Format:  wh.Format,
		Url:     wh.Url,
	}, nil
}

func toWebhook(dbW dbWebhook) (webhooks.Webhook, error) {
	return webhooks.Webhook{
		ThingID: dbW.ThingID,
		Name:    dbW.Name,
		Format:  dbW.Format,
		Url:     dbW.Url,
	}, nil
}
