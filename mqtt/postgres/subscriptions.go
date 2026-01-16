package postgres

import (
	"context"
	"fmt"

	"github.com/MainfluxLabs/mainflux/mqtt"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ mqtt.Repository = (*mqttRepository)(nil)

type mqttRepository struct {
	db dbutil.Database
}

// NewRepository instantiates a PostgreSQL implementation of mqt repository.
func NewRepository(db dbutil.Database) mqtt.Repository {
	return &mqttRepository{db: db}
}

func (mr *mqttRepository) Save(ctx context.Context, sub mqtt.Subscription) error {
	q := `INSERT INTO subscriptions (subtopic, thing_id, group_id, created_at)
		  VALUES (:subtopic, :thing_id, :group_id, :created_at)`
	dbSub := dbSubscription{
		Subtopic:  sub.Subtopic,
		ThingID:   sub.ThingID,
		GroupID:   sub.GroupID,
		CreatedAt: sub.CreatedAt,
	}

	row, err := mr.db.NamedQueryContext(ctx, q, dbSub)
	if err != nil {
		if pqErr, ok := err.(*pgconn.PgError); ok && pqErr.Code == pgerrcode.UniqueViolation {
			return errors.Wrap(dbutil.ErrConflict, err)
		}
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer row.Close()

	return nil
}

func (mr *mqttRepository) Remove(ctx context.Context, sub mqtt.Subscription) error {
	q := `DELETE FROM subscriptions 
          WHERE subtopic = :subtopic AND thing_id = :thing_id AND group_id = :group_id;`

	dbSub := dbSubscription{
		Subtopic: sub.Subtopic,
		ThingID:  sub.ThingID,
		GroupID:  sub.GroupID,
	}

	if _, err := mr.db.NamedExecContext(ctx, q, dbSub); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (mr *mqttRepository) RetrieveByGroup(ctx context.Context, pm mqtt.PageMetadata, groupID string) (mqtt.Page, error) {
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	q := fmt.Sprintf(`SELECT subtopic, group_id, thing_id, created_at 
					  FROM subscriptions 
					  WHERE group_id = :group_id 
					  ORDER BY created_at 
					  %s;`, olq)
	params := map[string]any{
		"group_id": groupID,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	rows, err := mr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return mqtt.Page{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []mqtt.Subscription
	for rows.Next() {
		item := dbSubscription{}
		if err := rows.StructScan(&item); err != nil {
			return mqtt.Page{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		items = append(items, fromDBSub(item))
	}

	cq := `SELECT COUNT(*) FROM subscriptions WHERE group_id = :group_id;`
	total, err := dbutil.Total(ctx, mr.db, cq, params)
	if err != nil {
		return mqtt.Page{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return mqtt.Page{
		Total:         total,
		Subscriptions: items,
	}, nil

}

type dbSubscription struct {
	Subtopic  string  `db:"subtopic"`
	ThingID   string  `db:"thing_id"`
	GroupID   string  `db:"group_id"`
	CreatedAt float64 `db:"created_at"`
}

func fromDBSub(sub dbSubscription) mqtt.Subscription {
	return mqtt.Subscription{
		Subtopic:  sub.Subtopic,
		ThingID:   sub.ThingID,
		GroupID:   sub.GroupID,
		CreatedAt: sub.CreatedAt,
	}
}
