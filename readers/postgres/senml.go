// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

type senmlRepository struct {
	db         dbutil.Database
	aggregator *aggregationService
}

func NewSenMLRepository(db dbutil.Database) readers.SenMLMessageRepository {
	return &senmlRepository{
		db:         db,
		aggregator: newAggregationService(db),
	}
}

func (sr *senmlRepository) Retrieve(ctx context.Context, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	return sr.readAll(ctx, rpm)
}

func (sr *senmlRepository) Backup(ctx context.Context, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	return sr.readAll(ctx, rpm)
}

func (sr *senmlRepository) Remove(ctx context.Context, rpm readers.SenMLPageMetadata) error {
	condition := sr.fmtCondition(rpm)
	q := fmt.Sprintf("DELETE FROM senml %s", condition)
	params := map[string]any{
		"subtopic":     rpm.Subtopic,
		"publisher":    rpm.Publisher,
		"name":         rpm.Name,
		"protocol":     rpm.Protocol,
		"value":        rpm.Value,
		"bool_value":   rpm.BoolValue,
		"string_value": rpm.StringValue,
		"data_value":   rpm.DataValue,
		"from":         rpm.From,
		"to":           rpm.To,
	}

	if _, err := sr.db.NamedExecContext(ctx, q, params); err != nil {
		return sr.handlePgError(err, errors.ErrDeleteMessages)
	}

	return nil
}

func (sr *senmlRepository) readAll(ctx context.Context, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	page := readers.SenMLMessagesPage{
		SenMLPageMetadata: rpm,
		MessagesPage: readers.MessagesPage{
			Messages: []readers.Message{},
			Total:    0,
		},
	}

	params := sr.buildQueryParams(rpm)

	if rpm.AggType != "" && rpm.AggInterval != "" {
		messages, total, err := sr.aggregator.readAggregatedSenMLMessages(ctx, rpm)
		if err != nil {
			return page, err
		}
		page.Messages = messages
		page.Total = total

		return page, nil
	}

	messages, err := sr.readMessages(ctx, rpm, params)
	if err != nil {
		return page, err
	}
	page.Messages = messages

	condition := sr.fmtCondition(rpm)
	query := fmt.Sprintf(`SELECT COUNT(*) FROM senml %s;`, condition)
	total, err := dbutil.Total(ctx, sr.db, query, params)
	if err != nil {
		return page, err
	}
	page.Total = total

	return page, nil
}

func (sr *senmlRepository) readMessages(ctx context.Context, rpm readers.SenMLPageMetadata, params map[string]any) ([]readers.Message, error) {
	olq := dbutil.GetOffsetLimitQuery(rpm.Limit)
	dq := dbutil.GetDirQuery(rpm.Dir)
	condition := sr.fmtCondition(rpm)

	query := fmt.Sprintf(`SELECT * FROM senml %s ORDER BY time %s %s;`, condition, dq, olq)
	rows, err := sr.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.UndefinedTable {
			return []readers.Message{}, nil
		}
		return nil, errors.Wrap(readers.ErrReadMessages, err)
	}

	if rows == nil {
		return []readers.Message{}, nil
	}
	defer rows.Close()

	return sr.scanMessages(rows)
}

func (sr *senmlRepository) scanMessages(rows *sqlx.Rows) ([]readers.Message, error) {
	var messages []readers.Message

	for rows.Next() {
		msg := senml.Message{}
		if err := rows.StructScan(&msg); err != nil {
			return nil, errors.Wrap(readers.ErrReadMessages, err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (sr *senmlRepository) fmtCondition(rpm readers.SenMLPageMetadata) string {
	var query map[string]any
	meta, err := json.Marshal(rpm)
	if err != nil {
		return ""
	}
	json.Unmarshal(meta, &query)

	condition := ""
	op := "WHERE"

	for name := range query {
		switch name {
		case "subtopic", "publisher", "protocol":
			condition = fmt.Sprintf(`%s %s %s = :%s`, condition, op, name, name)
			op = "AND"
		case "name":
			condition = fmt.Sprintf(`%s %s name = :name`, condition, op)
			op = "AND"
		case "v":
			comparator := readers.ParseValueComparator(query)
			condition = fmt.Sprintf(`%s %s value %s :value`, condition, op, comparator)
			op = "AND"
		case "vb":
			condition = fmt.Sprintf(`%s %s bool_value = :bool_value`, condition, op)
			op = "AND"
		case "vs":
			condition = fmt.Sprintf(`%s %s string_value = :string_value`, condition, op)
			op = "AND"
		case "vd":
			condition = fmt.Sprintf(`%s %s data_value = :data_value`, condition, op)
			op = "AND"
		case "from":
			condition = fmt.Sprintf(`%s %s time >= :from`, condition, op)
			op = "AND"
		case "to":
			condition = fmt.Sprintf(`%s %s time <= :to`, condition, op)
			op = "AND"
		}
	}
	return condition
}

func (sr *senmlRepository) buildQueryParams(rpm readers.SenMLPageMetadata) map[string]any {
	return map[string]any{
		"limit":        rpm.Limit,
		"offset":       rpm.Offset,
		"subtopic":     rpm.Subtopic,
		"publisher":    rpm.Publisher,
		"name":         rpm.Name,
		"protocol":     rpm.Protocol,
		"value":        rpm.Value,
		"bool_value":   rpm.BoolValue,
		"string_value": rpm.StringValue,
		"data_value":   rpm.DataValue,
		"from":         rpm.From,
		"to":           rpm.To,
	}
}

func (sr *senmlRepository) Restore(ctx context.Context, messages ...readers.Message) error {
	tx, err := sr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(errors.ErrSaveMessages, err)
	}

	q := `INSERT INTO senml (subtopic, publisher, protocol,
          name, unit, value, string_value, bool_value, data_value, sum,
          time, update_time)
          VALUES (:subtopic, :publisher, :protocol, :name, :unit,
          :value, :string_value, :bool_value, :data_value, :sum,
          :time, :update_time);`

	for _, msg := range messages {
		senmlMesage, ok := msg.(senml.Message)
		if !ok {
			return errors.Wrap(errors.ErrSaveMessages, errors.ErrInvalidMessage)
		}

		if _, err := tx.NamedExecContext(ctx, q, senmlMesage); err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				return errors.Wrap(errors.ErrSaveMessages, err)
			}
			return sr.handlePgError(err, errors.ErrSaveMessages)
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(errors.ErrSaveMessages, err)
	}

	return nil

}

func (sr *senmlRepository) handlePgError(err error, wrapErr error) error {
	pgErr, ok := err.(*pgconn.PgError)
	if ok {
		switch pgErr.Code {
		case pgerrcode.UndefinedTable:
			return errors.Wrap(wrapErr, err)
		case pgerrcode.InvalidTextRepresentation:
			return errors.Wrap(wrapErr, errors.ErrInvalidMessage)
		default:
			return errors.Wrap(wrapErr, err)
		}
	}
	return errors.Wrap(wrapErr, err)
}
