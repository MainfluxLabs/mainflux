package postgres

import (
	"context"
	"fmt"

	"github.com/MainfluxLabs/mainflux/mqtt"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
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
	q := `INSERT INTO subscriptions (subtopic, thing_id, channel_id, client_id, status, created_at)
		VALUES (:subtopic, :thing_id, :channel_id, :client_id, :status, :created_at)`
	dbSub := dbSubscription{
		Subtopic:  sub.Subtopic,
		ThingID:   sub.ThingID,
		ChanID:    sub.ChanID,
		ClientID:  sub.ClientID,
		Status:    sub.Status,
		CreatedAt: sub.CreatedAt,
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

func (mr *mqttRepository) UpdateStatus(ctx context.Context, sub mqtt.Subscription) error {
	q := `UPDATE subscriptions SET status = :status, created_at = :created_at WHERE client_id = :client_id;`

	dbSub := dbSubscription{
		ClientID:  sub.ClientID,
		Status:    sub.Status,
		CreatedAt: sub.CreatedAt,
	}

	row, err := mr.db.NamedQueryContext(ctx, q, dbSub)
	if err != nil {
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}
	defer row.Close()

	return nil
}

func (mr *mqttRepository) Remove(ctx context.Context, sub mqtt.Subscription) error {
	q := `DELETE FROM subscriptions WHERE client_id =$1 AND subtopic =$2 AND thing_id=$3 AND channel_id=$4;`

	if r := mr.db.QueryRowxContext(ctx, q, sub.ClientID, sub.Subtopic, sub.ThingID, sub.ChanID); r.Err() != nil {
		return errors.Wrap(errors.ErrRemoveEntity, r.Err())
	}

	return nil
}

func (mr *mqttRepository) HasClientID(ctx context.Context, clientID string) error {
	q := `SELECT EXISTS (SELECT 1 FROM subscriptions WHERE client_id = $1);`
	exists := false
	if err := mr.db.QueryRowxContext(ctx, q, clientID).Scan(&exists); err != nil {
		return errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	if !exists {
		return errors.ErrNotFound
	}

	return nil
}

func (mr *mqttRepository) RetrieveByChannelID(ctx context.Context, pm mqtt.PageMetadata, chanID string) (mqtt.Page, error) {
	olq := "LIMIT :limit OFFSET :offset"
	if pm.Limit == 0 {
		olq = ""
	}

	q := fmt.Sprintf(`SELECT subtopic, channel_id, client_id, thing_id, status, created_at FROM subscriptions WHERE channel_id= :channel_id ORDER BY created_at %s;`, olq)
	params := map[string]interface{}{
		"channel_id": chanID,
		"limit":      pm.Limit,
		"offset":     pm.Offset,
	}

	rows, err := mr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return mqtt.Page{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []mqtt.Subscription
	for rows.Next() {
		item := dbSubscription{}
		if err := rows.StructScan(&item); err != nil {
			return mqtt.Page{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}
		items = append(items, fromDBSub(item))
	}

	cq := `SELECT COUNT(*) FROM subscriptions WHERE channel_id= :channel_id;`
	total, err := mr.total(ctx, mr.db, cq, params)
	if err != nil {
		return mqtt.Page{}, errors.Wrap(errors.ErrRetrieveEntity, err)
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
	Subtopic  string  `db:"subtopic"`
	ThingID   string  `db:"thing_id"`
	ChanID    string  `db:"channel_id"`
	ClientID  string  `db:"client_id"`
	Status    string  `db:"status"`
	CreatedAt float64 `db:"created_at"`
}

func fromDBSub(sub dbSubscription) mqtt.Subscription {
	return mqtt.Subscription{
		Subtopic:  sub.Subtopic,
		ThingID:   sub.ThingID,
		ChanID:    sub.ChanID,
		ClientID:  sub.ClientID,
		Status:    sub.Status,
		CreatedAt: sub.CreatedAt,
	}
}
