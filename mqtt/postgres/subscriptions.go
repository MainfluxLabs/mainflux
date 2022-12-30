package postgres

import (
	"context"
	"fmt"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/mqtt"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/jmoiron/sqlx"
)

const (
	format = "subscriptions"
	// noLimit is used to indicate that no limit is set.
	noLimit = 0
)

var _ mqtt.Repository = (*mqttRepository)(nil)

type mqttRepository struct {
	db  *sqlx.DB
	log logger.Logger
}

// NewRepository instantiates a PostgreSQL implementation of mqtt
// repository.
func NewRepository(db *sqlx.DB, log logger.Logger) mqtt.Repository {
	return &mqttRepository{db: db, log: log}
}

func (mr *mqttRepository) Save(ctx context.Context, sub mqtt.Subscription) (string, error) {
	q := fmt.Sprintf(`INSERT INTO %s (id, owner_id, subtopic, thing_id, chan_id) VALUES (:id, :owner_id, :subtopic, :thing_id, :chan_id)`, format)
	if _, err := mr.db.NamedExecContext(ctx, q, sub); err != nil {
		return "", errors.Wrap(errors.ErrCreateEntity, err)
	}

	return sub.ID, nil
}

func (mr *mqttRepository) Remove(ctx context.Context, id string) error {
	q := fmt.Sprintf(`DELETE FROM %s WHERE id = :id`, format)
	if _, err := mr.db.NamedExecContext(ctx, q, id); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	return nil
}

func (mr *mqttRepository) RetrieveAll(ctx context.Context, pm mqtt.PageMetadata) (mqtt.Page, error) {
	olq := "LIMIT :limit OFFSET :offset"
	if pm.Limit == 0 {
		olq = ""
	}

	q := fmt.Sprintf(`SELECT id, owner_id, subtopic, channel_id, thing_id, time FROM %s ORDER BY :order :direction %s`, format, olq)
	params := map[string]interface{}{
		"limit":     pm.Limit,
		"offset":    pm.Offset,
		"order":     pm.Order,
		"direction": pm.Direction,
	}

	rows, err := mr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return mqtt.Page{}, errors.Wrap(errors.ErrViewEntity, err)
	}
	defer rows.Close()

	var items []mqtt.Subscription
	for rows.Next() {
		var item mqtt.Subscription
		if err := rows.StructScan(&item); err != nil {
			return mqtt.Page{}, errors.Wrap(errors.ErrViewEntity, err)
		}
		items = append(items, item)
	}

	q = "SELECT COUNT(*) FROM subscriptions"

	var total uint64
	if err := mr.db.QueryRowxContext(ctx, q).Scan(&total); err != nil {
		return mqtt.Page{}, errors.Wrap(errors.ErrViewEntity, err)
	}

	page := mqtt.Page{
		PageMetadata: mqtt.PageMetadata{
			Offset:    pm.Offset,
			Limit:     pm.Limit,
			Total:     total,
			Order:     pm.Order,
			Direction: pm.Direction,
		},
		Subscriptions: items,
	}

	return page, nil
}
