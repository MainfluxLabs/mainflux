package postgres

import (
	"context"
	"fmt"

	"github.com/MainfluxLabs/mainflux/mqtt"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	format = "subscriptions"
	order  = "time"
)

var _ mqtt.Repository = (*mqttRepository)(nil)

type mqttRepository struct {
	db Database
}

// NewRepository instantiates a PostgreSQL implementation of mqt repository.
func NewRepository(db Database) mqtt.Repository {
	return &mqttRepository{db: db}
}

func (mr *mqttRepository) Save(ctx context.Context, sub mqtt.Subscription) error {
	q := fmt.Sprintf(`INSERT INTO %s (subtopic, thing_id, channel_id, time) VALUES (:subtopic, :thing_id, :channel_id, :time)`, format)
	dbSub := dbSubscription{
		Subtopic: sub.Subtopic,
		ThingID:  sub.ThingID,
		ChanID:   sub.ChanID,
		Time:     sub.Time,
	}

	row, err := mr.db.NamedQueryContext(ctx, q, dbSub)
	if err != nil {
		if pqErr, ok := err.(*pgconn.PgError); ok && pqErr.Code == pgerrcode.UniqueViolation {
			return errors.Wrap(errors.ErrConflict, err)
		}
		return errors.Wrap(errors.ErrCreateEntity, err)
	}
	defer row.Close()

	return nil
}

func (mr *mqttRepository) Remove(ctx context.Context, sub mqtt.Subscription) error {
	q := fmt.Sprintf(`DELETE FROM %s WHERE subtopic =$1 AND thing_id=$2 AND channel_id=$3`, format)

	if r := mr.db.QueryRowxContext(ctx, q, sub.Subtopic, sub.ThingID, sub.ChanID); r.Err() != nil {
		return errors.Wrap(errors.ErrRemoveEntity, r.Err())
	}

	return nil
}

func (mr *mqttRepository) RetrieveByChannelID(ctx context.Context, pm mqtt.PageMetadata, chanID string) (mqtt.Page, error) {
	olq := "LIMIT :limit OFFSET :offset"
	if pm.Limit == 0 {
		olq = ""
	}
	q := fmt.Sprintf(`SELECT subtopic, channel_id, thing_id, time FROM %s WHERE channel_id= :chanID ORDER BY %s %s;`, format, order, olq)
	params := map[string]interface{}{
		"chanID": chanID,
		"limit":  pm.Limit,
		"offset": pm.Offset,
	}

	rows, err := mr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return mqtt.Page{}, errors.Wrap(errors.ErrViewEntity, err)
	}
	defer rows.Close()

	var items []mqtt.Subscription
	for rows.Next() {
		item := dbSubscription{}
		if err := rows.StructScan(&item); err != nil {
			return mqtt.Page{}, errors.Wrap(errors.ErrViewEntity, err)
		}
		items = append(items, fromDBSub(item))
	}

	if len(items) == 0 {
		return mqtt.Page{}, errors.ErrNotFound
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE channel_id= :chanID;`, format)
	total, err := mr.total(ctx, mr.db, cq, params)
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

func (mr *mqttRepository) total(ctx context.Context, db Database, query string, params interface{}) (uint64, error) {
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

type dbSubscription struct {
	Subtopic string  `db:"subtopic"`
	ThingID  string  `db:"thing_id"`
	ChanID   string  `db:"channel_id"`
	Time     float64 `db:"time"`
}

func fromDBSub(sub dbSubscription) mqtt.Subscription {
	return mqtt.Subscription{
		Subtopic: sub.Subtopic,
		ThingID:  sub.ThingID,
		ChanID:   sub.ChanID,
		Time:     sub.Time,
	}
}
