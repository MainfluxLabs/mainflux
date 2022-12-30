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
	order  = "time"
	sort   = "desc"
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

func (mr *mqttRepository) Save(ctx context.Context, sub mqtt.Subscription) error {
	q := fmt.Sprintf(`INSERT INTO %s (owner_id, subtopic, thing_id, chan_id) VALUES (:owner_id, :subtopic, :thing_id, :chan_id)`, format)
	if _, err := mr.db.NamedExecContext(ctx, q, sub); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (mr *mqttRepository) Remove(ctx context.Context, sub mqtt.Subscription) error {
	q := fmt.Sprintf(`DELETE FROM %s WHERE subtopic =$1 AND thing_id=$2 AND chan_id=$3`, format)
	if _, err := mr.db.ExecContext(ctx, q, sub.Subtopic, sub.ThingID, sub.ChanID); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	return nil
}

func (mr *mqttRepository) RetrieveByOwnerID(ctx context.Context, pm mqtt.PageMetadata, ownerID string) (mqtt.Page, error) {
	olq := "LIMIT :limit OFFSET :offset"
	if pm.Limit == 0 {
		olq = ""
	}

	q := fmt.Sprintf(`SELECT subtopic, chan_id, thing_id, time FROM %s WHERE owner_id= :ownerID %s ORDER BY %s %s`, format, olq, order, sort)
	params := map[string]interface{}{
		"ownerID": ownerID,
		"limit":   pm.Limit,
		"offset":  pm.Offset,
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

	q = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE owner_id=$1", format)

	total, err := mr.total(ctx, q, ownerID)
	if err != nil {
		return mqtt.Page{}, errors.Wrap(errors.ErrViewEntity, err)
	}

	return mqtt.Page{
		PageMetadata: mqtt.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
		Subscriptions: items,
	}, nil

}

func (mr *mqttRepository) total(ctx context.Context, query string, params interface{}) (uint64, error) {
	rows, err := mr.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	total := uint64(0)
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, err
		}
	}
	return total, nil
}
