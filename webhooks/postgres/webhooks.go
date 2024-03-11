package postgres

import (
	"context"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/webhooks"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ webhooks.WebhookRepository = (*webhookRepository)(nil)

type webhookRepository struct {
	db Database
}

// NewWebhookRepository instantiates a PostgreSQL implementation of webhook
// repository.
func NewWebhookRepository(db Database) webhooks.WebhookRepository {
	return &webhookRepository{
		db: db,
	}
}

func (wr webhookRepository) Save(ctx context.Context, w webhooks.Webhook) (webhooks.Webhook, error) {
	tx, err := wr.db.BeginTxx(ctx, nil)
	if err != nil {
		return webhooks.Webhook{}, errors.Wrap(errors.ErrCreateEntity, err)
	}

	q := `INSERT INTO webhooks (id, name, format, url) VALUES (:id, :name, :format, :url);`

	dbWh, err := toDBWebhook(w)
	if err != nil {
		return webhooks.Webhook{}, errors.Wrap(errors.ErrCreateEntity, err)
	}

	if _, err := tx.NamedExecContext(ctx, q, dbWh); err != nil {
		tx.Rollback()
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return webhooks.Webhook{}, errors.Wrap(errors.ErrMalformedEntity, err)
			case pgerrcode.UniqueViolation:
				return webhooks.Webhook{}, errors.Wrap(errors.ErrConflict, err)
			case pgerrcode.StringDataRightTruncationWarning:
				return webhooks.Webhook{}, errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}

		return webhooks.Webhook{}, errors.Wrap(errors.ErrCreateEntity, err)
	}
	if err = tx.Commit(); err != nil {
		return webhooks.Webhook{}, errors.Wrap(errors.ErrCreateEntity, err)
	}
	return w, nil
}

type dbWebhook struct {
	id     string `db:"id"`
	name   string `db:"name"`
	format string `db:"format"`
	url    string `db:"url"`
}

func toDBWebhook(wh webhooks.Webhook) (dbWebhook, error) {
	return dbWebhook{
		id:     wh.ID,
		name:   wh.Name,
		format: wh.Format,
		url:    wh.Url,
	}, nil
}

func toWebhook(dbW dbWebhook) (webhooks.Webhook, error) {
	return webhooks.Webhook{
		ID:     dbW.id,
		Name:   dbW.name,
		Format: dbW.format,
		Url:    dbW.url,
	}, nil
}
