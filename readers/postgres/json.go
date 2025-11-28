// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

type jsonRepository struct {
	db         dbutil.Database
	aggregator *aggregationService
}

func NewJSONRepository(db dbutil.Database) readers.JSONMessageRepository {
	return &jsonRepository{
		db:         db,
		aggregator: newAggregationService(db),
	}
}

func (jr *jsonRepository) Retrieve(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	return jr.readAll(ctx, rpm)
}

func (jr *jsonRepository) readAll(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	page := readers.JSONMessagesPage{
		JSONPageMetadata: rpm,
		MessagesPage: readers.MessagesPage{
			Messages: []readers.Message{},
			Total:    0,
		},
	}

	params := jr.buildQueryParams(rpm)

	if rpm.AggType != "" && rpm.AggInterval != "" {
		messages, total, err := jr.aggregator.readAggregatedJSONMessages(ctx, rpm)
		if err != nil {
			return page, err
		}
		page.Messages = messages
		page.Total = total

		return page, nil
	}

	messages, err := jr.readMessages(ctx, rpm, params)
	if err != nil {
		return page, err
	}
	page.Messages = messages

	condition := jr.fmtCondition(rpm)
	query := fmt.Sprintf(`SELECT COUNT(*) FROM json %s;`, condition)
	total, err := dbutil.Total(ctx, jr.db, query, params)
	if err != nil {
		return page, err
	}
	page.Total = total

	return page, nil
}

func (jr *jsonRepository) readMessages(ctx context.Context, rpm readers.JSONPageMetadata, params map[string]any) ([]readers.Message, error) {
	olq := dbutil.GetOffsetLimitQuery(rpm.Limit)
	dq := dbutil.GetDirQuery(rpm.Dir)
	condition := jr.fmtCondition(rpm)

	query := fmt.Sprintf(`SELECT * FROM json %s ORDER BY created %s %s;`, condition, dq, olq)
	rows, err := jr.db.NamedQueryContext(ctx, query, params)
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

	return jr.scanMessages(rows)
}

func (jr *jsonRepository) scanMessages(rows *sqlx.Rows) ([]readers.Message, error) {
	var messages []readers.Message

	for rows.Next() {
		msg := mfjson.Message{}
		if err := rows.StructScan(&msg); err != nil {
			return nil, errors.Wrap(readers.ErrReadMessages, err)
		}

		m, err := msg.ToMap()
		if err != nil {
			return nil, errors.Wrap(readers.ErrReadMessages, err)
		}
		messages = append(messages, m)
	}

	return messages, nil
}

func (jr *jsonRepository) fmtCondition(rpm readers.JSONPageMetadata) string {
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
		case "from":
			condition = fmt.Sprintf(`%s %s created >= :from`, condition, op)
			op = "AND"
		case "to":
			condition = fmt.Sprintf(`%s %s created <= :to`, condition, op)
			op = "AND"
		case "filter":
			filterPath := buildPayloadFilterPath(rpm.Filter)
			condition = fmt.Sprintf(`%s %s %s IS NOT NULL`, condition, op, filterPath)
			op = "AND"
		}
	}
	return condition
}

func buildPayloadFilterPath(field string) string {
	parts := strings.Split(field, ".")
	if len(parts) == 1 {
		return fmt.Sprintf("payload->>'%s'", parts[0])
	}

	var path strings.Builder
	path.WriteString("payload")

	for i, part := range parts {
		if i == len(parts)-1 {
			path.WriteString(fmt.Sprintf("->>'%s'", part))
		} else {
			path.WriteString(fmt.Sprintf("->'%s'", part))
		}
	}

	return path.String()
}

func (jr *jsonRepository) buildQueryParams(rpm readers.JSONPageMetadata) map[string]any {
	return map[string]any{
		"limit":     rpm.Limit,
		"offset":    rpm.Offset,
		"subtopic":  rpm.Subtopic,
		"publisher": rpm.Publisher,
		"protocol":  rpm.Protocol,
		"filter":    rpm.Filter,
		"from":      rpm.From,
		"to":        rpm.To,
	}
}

func (jr *jsonRepository) Backup(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	return jr.readAll(ctx, rpm)
}

func (jr *jsonRepository) Remove(ctx context.Context, rpm readers.JSONPageMetadata) error {
	condition := jr.fmtCondition(rpm)
	q := fmt.Sprintf("DELETE FROM json %s", condition)
	params := map[string]any{
		"subtopic":  rpm.Subtopic,
		"publisher": rpm.Publisher,
		"protocol":  rpm.Protocol,
		"from":      rpm.From,
		"to":        rpm.To,
	}

	if _, err := jr.db.NamedExecContext(ctx, q, params); err != nil {
		return jr.handlePgError(err, errors.ErrDeleteMessages)
	}

	return nil
}

func (jr *jsonRepository) handlePgError(err error, wrapErr error) error {
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

func (jr *jsonRepository) Restore(ctx context.Context, messages ...readers.Message) error {
	tx, err := jr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(errors.ErrSaveMessages, err)
	}

	q := `INSERT INTO json (subtopic, publisher, protocol, payload, created)
          VALUES (:subtopic, :publisher, :protocol, :payload, :created);`

	for _, msg := range messages {
		jsonMsg, ok := msg.(mfjson.Message)
		if !ok {
			return errors.Wrap(errors.ErrSaveMessages, errors.ErrInvalidMessage)
		}

		if _, err := tx.NamedExecContext(ctx, q, jsonMsg); err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				return errors.Wrap(errors.ErrSaveMessages, err)
			}
			return jr.handlePgError(err, errors.ErrSaveMessages)
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(errors.ErrSaveMessages, err)
	}

	return nil
}
