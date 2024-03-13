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
	ID     string `db:"id"`
	Name   string `db:"name"`
	Format string `db:"format"`
	Url    string `db:"url"`
}

func toDBWebhook(wh webhooks.Webhook) (dbWebhook, error) {
	return dbWebhook{
		ID:     wh.ID,
		Name:   wh.Name,
		Format: wh.Format,
		Url:    wh.Url,
	}, nil
}

func toWebhook(dbW dbWebhook) (webhooks.Webhook, error) {
	return webhooks.Webhook{
		ID:     dbW.ID,
		Name:   dbW.Name,
		Format: dbW.Format,
		Url:    dbW.Url,
	}, nil
}
