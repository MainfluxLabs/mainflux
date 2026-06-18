// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/audit"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ audit.EventRepository = (*eventRepository)(nil)

type eventRepository struct {
	db dbutil.Database
}

func NewEventRepository(db dbutil.Database) audit.EventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) SaveEvent(ctx context.Context, e audit.Event) error {
	query := `INSERT INTO events (id, occurred_at, operation, actor_id, actor_email, org_id, group_id, data)
	      VALUES (:id, :occurred_at, :operation, :actor_id, :actor_email, :org_id, :group_id, :data)`

	dbe, err := toDBEvent(e)
	if err != nil {
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	if _, err := r.db.NamedExecContext(ctx, query, dbe); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation, pgerrcode.StringDataRightTruncationDataException:
				return errors.Wrap(dbutil.ErrMalformedEntity, err)
			case pgerrcode.UniqueViolation:
				return errors.Wrap(dbutil.ErrConflict, err)
			}
		}
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	return nil
}

func (r *eventRepository) RetrieveEvents(ctx context.Context, pm audit.PageMetadata) (audit.EventsPage, error) {
	emailQ, emailVal := emailQuery(pm.Email)
	opQ, opVal := operationQuery(pm.Operation)
	fromQ, fromVal := occurredFromQuery(pm.From)
	toQ, toVal := occurredToQuery(pm.To)

	dataQ, dataVal, err := dataQuery(pm.Data)
	if err != nil {
		return audit.EventsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	whereClause := dbutil.BuildWhereClause(emailQ, opQ, fromQ, toQ, dataQ)
	query := fmt.Sprintf(
		`SELECT id, occurred_at, operation, actor_id, actor_email, org_id, group_id, data FROM events %s ORDER BY %s %s %s`,
		whereClause, dbutil.GetOrderQuery(pm.Order), dbutil.GetDirQuery(pm.Dir), dbutil.GetOffsetLimitQuery(pm.Limit),
	)
	cquery := fmt.Sprintf(`SELECT COUNT(*) FROM events %s`, whereClause)

	params := map[string]any{
		"email":     emailVal,
		"operation": opVal,
		"from":      fromVal,
		"to":        toVal,
		"data":      dataVal,
		"limit":     pm.Limit,
		"offset":    pm.Offset,
	}

	return r.retrieve(ctx, query, cquery, params)
}

func (r *eventRepository) RetrieveEventsByOrg(ctx context.Context, orgID string, pm audit.PageMetadata) (audit.EventsPage, error) {
	emailQ, emailVal := emailQuery(pm.Email)
	opQ, opVal := operationQuery(pm.Operation)
	orgQ, orgVal := orgQuery(orgID)
	fromQ, fromVal := occurredFromQuery(pm.From)
	toQ, toVal := occurredToQuery(pm.To)

	dataQ, dataVal, err := dataQuery(pm.Data)
	if err != nil {
		return audit.EventsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	whereClause := dbutil.BuildWhereClause(emailQ, opQ, orgQ, fromQ, toQ, dataQ)
	query := fmt.Sprintf(
		`SELECT id, occurred_at, operation, actor_id, actor_email, org_id, group_id, data FROM events %s ORDER BY %s %s %s`,
		whereClause, dbutil.GetOrderQuery(pm.Order), dbutil.GetDirQuery(pm.Dir), dbutil.GetOffsetLimitQuery(pm.Limit),
	)
	cquery := fmt.Sprintf(`SELECT COUNT(*) FROM events %s`, whereClause)

	params := map[string]any{
		"email":     emailVal,
		"operation": opVal,
		"org_id":    orgVal,
		"from":      fromVal,
		"to":        toVal,
		"data":      dataVal,
		"limit":     pm.Limit,
		"offset":    pm.Offset,
	}

	return r.retrieve(ctx, query, cquery, params)
}

func (r *eventRepository) RetrieveEventsByGroup(ctx context.Context, groupID string, pm audit.PageMetadata) (audit.EventsPage, error) {
	emailQ, emailVal := emailQuery(pm.Email)
	opQ, opVal := operationQuery(pm.Operation)
	groupQ, groupVal := groupQuery(groupID)
	fromQ, fromVal := occurredFromQuery(pm.From)
	toQ, toVal := occurredToQuery(pm.To)

	dataQ, dataVal, err := dataQuery(pm.Data)
	if err != nil {
		return audit.EventsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	whereClause := dbutil.BuildWhereClause(emailQ, opQ, groupQ, fromQ, toQ, dataQ)
	query := fmt.Sprintf(
		`SELECT id, occurred_at, operation, actor_id, actor_email, org_id, group_id, data FROM events %s ORDER BY %s %s %s`,
		whereClause, dbutil.GetOrderQuery(pm.Order), dbutil.GetDirQuery(pm.Dir), dbutil.GetOffsetLimitQuery(pm.Limit),
	)
	cquery := fmt.Sprintf(`SELECT COUNT(*) FROM events %s`, whereClause)

	params := map[string]any{
		"email":     emailVal,
		"operation": opVal,
		"group_id":  groupVal,
		"from":      fromVal,
		"to":        toVal,
		"data":      dataVal,
		"limit":     pm.Limit,
		"offset":    pm.Offset,
	}

	return r.retrieve(ctx, query, cquery, params)
}

func (r *eventRepository) retrieve(ctx context.Context, query, cquery string, params map[string]any) (audit.EventsPage, error) {
	rows, err := r.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return audit.EventsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []audit.Event
	for rows.Next() {
		dbe := dbEvent{}
		if err := rows.StructScan(&dbe); err != nil {
			return audit.EventsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		ev, err := toEvent(dbe)
		if err != nil {
			return audit.EventsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		items = append(items, ev)
	}

	total, err := dbutil.Total(ctx, r.db, cquery, params)
	if err != nil {
		return audit.EventsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return audit.EventsPage{
		Total:  total,
		Events: items,
	}, nil
}

func emailQuery(email string) (string, string) {
	if email == "" {
		return "", ""
	}
	return "actor_email = :email", email
}

func operationQuery(op string) (string, string) {
	if op == "" {
		return "", ""
	}
	return "operation = :operation", op
}

func orgQuery(orgID string) (string, string) {
	if orgID == "" {
		return "", ""
	}
	return "org_id = :org_id", orgID
}

func groupQuery(groupID string) (string, string) {
	if groupID == "" {
		return "", ""
	}
	return "group_id = :group_id", groupID
}

func occurredFromQuery(from time.Time) (string, time.Time) {
	if from.IsZero() {
		return "", time.Time{}
	}
	return "occurred_at >= :from", from
}

func occurredToQuery(to time.Time) (string, time.Time) {
	if to.IsZero() {
		return "", time.Time{}
	}
	return "occurred_at <= :to", to
}

func dataQuery(m map[string]any) (string, []byte, error) {
	if len(m) == 0 {
		return "", nil, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", nil, err
	}
	return "data @> :data", b, nil
}

type dbEvent struct {
	ID         string         `db:"id"`
	OccurredAt time.Time      `db:"occurred_at"`
	Operation  string         `db:"operation"`
	ActorID    sql.NullString `db:"actor_id"`
	ActorEmail sql.NullString `db:"actor_email"`
	OrgID      sql.NullString `db:"org_id"`
	GroupID    sql.NullString `db:"group_id"`
	Data       []byte         `db:"data"`
}

func toDBEvent(e audit.Event) (dbEvent, error) {
	var data []byte
	if e.Data != nil {
		b, err := json.Marshal(e.Data)
		if err != nil {
			return dbEvent{}, err
		}
		data = b
	} else {
		data = []byte("{}")
	}

	return dbEvent{
		ID:         e.ID,
		OccurredAt: e.OccurredAt,
		Operation:  e.Operation,
		ActorID:    nullableString(e.Actor.ID),
		ActorEmail: nullableString(e.Actor.Email),
		OrgID:      nullableString(e.OrgID),
		GroupID:    nullableString(e.GroupID),
		Data:       data,
	}, nil
}

func toEvent(d dbEvent) (audit.Event, error) {
	var data map[string]any
	if len(d.Data) > 0 {
		if err := json.Unmarshal(d.Data, &data); err != nil {
			return audit.Event{}, err
		}
	}

	return audit.Event{
		ID:         d.ID,
		OccurredAt: d.OccurredAt,
		Operation:  d.Operation,
		Actor: domain.Identity{
			ID:    d.ActorID.String,
			Email: d.ActorEmail.String,
		},
		OrgID:   d.OrgID.String,
		GroupID: d.GroupID.String,
		Data:    data,
	}, nil
}

func nullableString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
