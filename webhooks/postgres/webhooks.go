package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/webhooks"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jackc/pgerrcode"
)

var _ webhooks.WebhookRepository = (*webhookRepository)(nil)

type webhookRepository struct {
	db dbutil.Database
}

// NewWebhookRepository instantiates a PostgreSQL implementation of webhook repository.
func NewWebhookRepository(db dbutil.Database) webhooks.WebhookRepository {
	return &webhookRepository{
		db: db,
	}
}

func (wr webhookRepository) Save(ctx context.Context, whs ...webhooks.Webhook) ([]webhooks.Webhook, error) {
	tx, err := wr.db.BeginTxx(ctx, nil)
	if err != nil {
		return []webhooks.Webhook{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	q := `INSERT INTO webhooks (id, thing_id, group_id, name, url, headers, metadata) VALUES (:id, :thing_id, :group_id, :name, :url, :headers, :metadata);`

	for _, webhook := range whs {
		dbWh, err := toDBWebhook(webhook)
		if err != nil {
			return []webhooks.Webhook{}, errors.Wrap(dbutil.ErrCreateEntity, err)
		}

		if _, err := tx.NamedExecContext(ctx, q, dbWh); err != nil {
			tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return []webhooks.Webhook{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return []webhooks.Webhook{}, errors.Wrap(dbutil.ErrConflict, err)
				case pgerrcode.StringDataRightTruncationWarning:
					return []webhooks.Webhook{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
				}
			}

			return []webhooks.Webhook{}, errors.Wrap(dbutil.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return []webhooks.Webhook{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	return whs, nil
}

func (wr webhookRepository) RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (webhooks.WebhooksPage, error) {
	if _, err := uuid.FromString(groupID); err != nil {
		return webhooks.WebhooksPage{}, errors.Wrap(dbutil.ErrNotFound, err)
	}

	gq := "group_id = :group_id"
	oq := dbutil.GetOrderQuery(pm.Order, webhooks.AllowedOrders)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	nq, name := dbutil.GetNameQuery(pm.Name)
	m, mq, err := dbutil.GetMetadataQuery(pm.Metadata)
	if err != nil {
		return webhooks.WebhooksPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	whereClause := dbutil.BuildWhereClause(gq, nq, mq)

	q := fmt.Sprintf(`SELECT id, thing_id, group_id, name, url, headers, metadata FROM webhooks %s ORDER BY %s %s %s;`, whereClause, oq, dq, olq)
	qc := fmt.Sprintf(`SELECT COUNT(*) FROM webhooks %s;`, whereClause)

	params := map[string]any{
		"group_id": groupID,
		"name":     name,
		"metadata": m,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	return wr.retrieve(ctx, q, qc, params)
}

func (wr webhookRepository) RetrieveByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (webhooks.WebhooksPage, error) {
	if _, err := uuid.FromString(thingID); err != nil {
		return webhooks.WebhooksPage{}, errors.Wrap(dbutil.ErrNotFound, err)
	}

	tq := "thing_id = :thing_id"
	oq := dbutil.GetOrderQuery(pm.Order, webhooks.AllowedOrders)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	nq, name := dbutil.GetNameQuery(pm.Name)
	m, mq, err := dbutil.GetMetadataQuery(pm.Metadata)
	if err != nil {
		return webhooks.WebhooksPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	whereClause := dbutil.BuildWhereClause(tq, nq, mq)

	q := fmt.Sprintf(`SELECT id, thing_id, group_id, name, url, headers, metadata FROM webhooks %s ORDER BY %s %s %s;`, whereClause, oq, dq, olq)
	qc := fmt.Sprintf(`SELECT COUNT(*) FROM webhooks %s;`, whereClause)

	params := map[string]any{
		"thing_id": thingID,
		"name":     name,
		"metadata": m,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	return wr.retrieve(ctx, q, qc, params)
}

func (wr webhookRepository) RetrieveByID(ctx context.Context, id string) (webhooks.Webhook, error) {
	q := `SELECT id, thing_id, group_id, name, url, headers, metadata FROM webhooks WHERE id = $1;`

	dbwh := dbWebhook{ID: id}
	if err := wr.db.QueryRowxContext(ctx, q, id).StructScan(&dbwh); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		//  If there is no result or ID is in an invalid format, return ErrNotFound.
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return webhooks.Webhook{}, errors.Wrap(dbutil.ErrNotFound, err)
		}
		return webhooks.Webhook{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toWebhook(dbwh)
}

func (wr webhookRepository) Update(ctx context.Context, w webhooks.Webhook) error {
	q := `UPDATE webhooks SET name = :name, url = :url, headers = :headers, metadata = :metadata WHERE id = :id;`

	dbwh, err := toDBWebhook(w)
	if err != nil {
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	res, errdb := wr.db.NamedExecContext(ctx, q, dbwh)
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

func (wr webhookRepository) Remove(ctx context.Context, ids ...string) error {
	for _, id := range ids {
		dbwh := dbWebhook{ID: id}
		q := `DELETE FROM webhooks WHERE id = :id;`

		if _, err := wr.db.NamedExecContext(ctx, q, dbwh); err != nil {
			return errors.Wrap(dbutil.ErrRemoveEntity, err)
		}
	}

	return nil
}

func (wr webhookRepository) RemoveByThing(ctx context.Context, thingID string) error {
	dbwh := dbWebhook{ThingID: thingID}
	q := `DELETE FROM webhooks WHERE thing_id = :thing_id;`

	if _, err := wr.db.NamedExecContext(ctx, q, dbwh); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (wr webhookRepository) RemoveByGroup(ctx context.Context, groupID string) error {
	dbwh := dbWebhook{GroupID: groupID}
	q := `DELETE FROM webhooks WHERE group_id = :group_id;`

	if _, err := wr.db.NamedExecContext(ctx, q, dbwh); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (wr webhookRepository) retrieve(ctx context.Context, query, cquery string, params map[string]any) (webhooks.WebhooksPage, error) {
	rows, err := wr.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return webhooks.WebhooksPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []webhooks.Webhook
	for rows.Next() {
		dbWh := dbWebhook{}
		if err := rows.StructScan(&dbWh); err != nil {
			return webhooks.WebhooksPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		wh, err := toWebhook(dbWh)
		if err != nil {
			return webhooks.WebhooksPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		items = append(items, wh)
	}

	total, err := dbutil.Total(ctx, wr.db, cquery, params)
	if err != nil {
		return webhooks.WebhooksPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	page := webhooks.WebhooksPage{
		Webhooks: items,
		Total:    total,
	}

	return page, nil
}

type dbWebhook struct {
	ID       string `db:"id"`
	ThingID  string `db:"thing_id"`
	GroupID  string `db:"group_id"`
	Name     string `db:"name"`
	Url      string `db:"url"`
	Headers  []byte `db:"headers"`
	Metadata []byte `db:"metadata"`
}

func toDBWebhook(wh webhooks.Webhook) (dbWebhook, error) {
	headers := []byte("{}")
	if len(wh.Headers) > 0 {
		b, err := json.Marshal(wh.Headers)
		if err != nil {
			return dbWebhook{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
		}
		headers = b
	}

	metadata := []byte("{}")
	if len(wh.Metadata) > 0 {
		b, err := json.Marshal(wh.Metadata)
		if err != nil {
			return dbWebhook{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
		}
		metadata = b
	}

	return dbWebhook{
		ID:       wh.ID,
		ThingID:  wh.ThingID,
		GroupID:  wh.GroupID,
		Name:     wh.Name,
		Url:      wh.Url,
		Headers:  headers,
		Metadata: metadata,
	}, nil
}

func toWebhook(dbW dbWebhook) (webhooks.Webhook, error) {
	var headers map[string]string
	if err := json.Unmarshal(dbW.Headers, &headers); err != nil {
		return webhooks.Webhook{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	var metadata map[string]any
	if err := json.Unmarshal(dbW.Metadata, &metadata); err != nil {
		return webhooks.Webhook{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	return webhooks.Webhook{
		ID:       dbW.ID,
		ThingID:  dbW.ThingID,
		GroupID:  dbW.GroupID,
		Name:     dbW.Name,
		Url:      dbW.Url,
		Headers:  headers,
		Metadata: metadata,
	}, nil
}
