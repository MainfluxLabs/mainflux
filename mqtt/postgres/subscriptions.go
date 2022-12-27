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

func (mr *mqttRepository) RetrieveAll(ctx context.Context, pm mqtt.PageMetadata) (mqtt.Page, error) {
	q := fmt.Sprintf(`SELECT * FROM %s ORDER BY %s DESC LIMIT %d OFFSET %d`, format, pm.Order, pm.Limit, pm.Offset)
	qnoLimit := fmt.Sprintf(`SELECT * FROM %s ORDER BY %s DESC`, format, pm.Order)

	if pm.Limit == noLimit {
		q = qnoLimit
	}

	rows, err := mr.db.QueryxContext(ctx, q)
	if err != nil {
		return mqtt.Page{}, err
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
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Total:  total,
			Order:  pm.Order,
		},
		Subscriptions: items,
	}

	return page, nil
}
