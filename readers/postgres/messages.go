// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

const (
	defTable  = "messages"
	jsonTable = "json"
)

var _ readers.MessageRepository = (*postgresRepository)(nil)

var (
	errInvalidMessage = errors.New("invalid message representation")
	errTransRollback  = errors.New("failed to rollback transaction")
)

type postgresRepository struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) readers.MessageRepository {
	return &postgresRepository{
		db: db,
	}
}

func (tr postgresRepository) ListAllMessages(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return tr.readAll(rpm)
}

func (tr postgresRepository) Backup(rpm readers.PageMetadata, table string) (readers.MessagesPage, error) {
	if table == jsonTable {
		rpm.Format = jsonTable
	}
	return tr.readAll(rpm)
}

func (tr postgresRepository) DeleteMessages(ctx context.Context, rpm readers.PageMetadata, table string) error {
	tx, err := tr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(errors.ErrSaveMessages, err)
	}

	defer func() {
		if err != nil {
			if txErr := tx.Rollback(); txErr != nil {
				err = errors.Wrap(err, errors.Wrap(errTransRollback, txErr))
			}
			return
		}

		if err = tx.Commit(); err != nil {
			err = errors.Wrap(errors.ErrDeleteMessages, err)
		}
	}()

	condition := tr.fmtCondition(rpm, table)
	q := fmt.Sprintf("DELETE FROM %s %s", table, condition)
	params := map[string]interface{}{
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

	_, err = tx.NamedExecContext(ctx, q, params)
	if err != nil {
		return tr.handlePgError(err, errors.ErrDeleteMessages)
	}

	return nil
}

func (tr postgresRepository) Restore(ctx context.Context, format string, messages ...readers.Message) error {
	tx, err := tr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(errors.ErrSaveMessages, err)
	}

	defer func() {
		if err != nil {
			if txErr := tx.Rollback(); txErr != nil {
				err = errors.Wrap(err, errors.Wrap(errTransRollback, txErr))
			}
			return
		}

		if err = tx.Commit(); err != nil {
			err = errors.Wrap(errors.ErrSaveMessages, err)
		}
	}()

	switch format {
	case defTable:
		return tr.restoreSenMLMessages(ctx, tx, messages...)
	default:
		return tr.restoreJSONMessages(ctx, tx, messages...)
	}
}

func (tr postgresRepository) restoreJSONMessages(ctx context.Context, tx *sqlx.Tx, messages ...readers.Message) error {
	q := `INSERT INTO json (subtopic, publisher, protocol, payload, created)
          VALUES (:subtopic, :publisher, :protocol, :payload, :created);`

	for _, msg := range messages {
		jsonMsg, ok := msg.(mfjson.Message)
		if !ok {
			return errors.Wrap(errors.ErrSaveMessages, errInvalidMessage)
		}

		if _, err := tx.NamedExecContext(ctx, q, jsonMsg); err != nil {
			return tr.handlePgError(err, errors.ErrSaveMessages)
		}
	}

	return nil
}

func (tr postgresRepository) restoreSenMLMessages(ctx context.Context, tx *sqlx.Tx, messages ...readers.Message) error {
	q := `INSERT INTO messages (subtopic, publisher, protocol,
          name, unit, value, string_value, bool_value, data_value, sum,
          time, update_time)
          VALUES (:subtopic, :publisher, :protocol, :name, :unit,
          :value, :string_value, :bool_value, :data_value, :sum,
          :time, :update_time);`

	for _, msg := range messages {
		senmlMesage, ok := msg.(senml.Message)
		if !ok {
			return errors.Wrap(errors.ErrSaveMessages, errInvalidMessage)
		}

		if _, err := tx.NamedExecContext(ctx, q, senmlMesage); err != nil {
			return tr.handlePgError(err, errors.ErrSaveMessages)
		}
	}

	return nil
}

func (tr postgresRepository) readAll(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	page := readers.MessagesPage{
		PageMetadata: rpm,
		Messages:     []readers.Message{},
	}

	format, order := tr.getFormatAndOrder(rpm)
	params := tr.buildQueryParams(rpm)

	messages, err := tr.readMessages(rpm, format, order, params)
	if err != nil {
		return page, err
	}
	page.Messages = messages

	total, err := tr.readCount(rpm, format, order, params)
	if err != nil {
		return page, err
	}
	page.Total = total

	return page, nil
}

func (tr postgresRepository) readMessages(rpm readers.PageMetadata, format, order string, params map[string]interface{}) ([]readers.Message, error) {
	olq := dbutil.GetOffsetLimitQuery(rpm.Limit)
	condition := tr.fmtCondition(rpm, format)
	query := fmt.Sprintf(`SELECT * FROM %s %s ORDER BY %s DESC %s;`, format, condition, order, olq)

	rows, err := tr.executeQuery(query, params)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []readers.Message{}, nil
	}
	defer rows.Close()

	return tr.scanMessages(rows, format)
}

func (tr postgresRepository) readCount(rpm readers.PageMetadata, format, order string, params map[string]interface{}) (uint64, error) {
	condition := tr.fmtCondition(rpm, format)
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s %s;`, format, condition)

	rows, err := tr.executeQuery(query, params)
	if err != nil {
		return 0, err
	}
	if rows == nil {
		return 0, nil
	}
	defer rows.Close()

	var total uint64
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, err
		}
	}
	return total, nil
}

func (tr postgresRepository) executeQuery(query string, params map[string]interface{}) (*sqlx.Rows, error) {
	rows, err := tr.db.NamedQuery(query, params)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.UndefinedTable {
			return nil, nil
		}
		return nil, errors.Wrap(readers.ErrReadMessages, err)
	}
	return rows, nil
}

func (tr postgresRepository) scanMessages(rows *sqlx.Rows, format string) ([]readers.Message, error) {
	var messages []readers.Message

	switch format {
	case defTable:
		for rows.Next() {
			msg := senml.Message{}
			if err := rows.StructScan(&msg); err != nil {
				return nil, errors.Wrap(readers.ErrReadMessages, err)
			}
			messages = append(messages, msg)
		}
	default:
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
	}

	return messages, nil
}

func (tr postgresRepository) fmtCondition(rpm readers.PageMetadata, table string) string {
	var query map[string]interface{}
	meta, err := json.Marshal(rpm)
	if err != nil {
		return ""
	}
	json.Unmarshal(meta, &query)

	condition := ""
	op := "WHERE"
	timeColumn := tr.getTimeColumn(table)

	for name := range query {
		switch name {
		case "subtopic", "publisher", "protocol":
			condition = fmt.Sprintf(`%s %s %s = :%s`, condition, op, name, name)
			op = "AND"
		case "name":
			switch table {
			case jsonTable:
				condition = fmt.Sprintf(`%s %s payload->>'n' = :name`, condition, op)
			default:
				condition = fmt.Sprintf(`%s %s name = :name`, condition, op)
			}

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
			condition = fmt.Sprintf(`%s %s %s >= :from`, condition, op, timeColumn)
			op = "AND"
		case "to":
			condition = fmt.Sprintf(`%s %s %s <= :to`, condition, op, timeColumn)
			op = "AND"
		}
	}
	return condition
}

func (tr postgresRepository) getFormatAndOrder(rpm readers.PageMetadata) (format, order string) {
	format = defTable
	order = "time"

	if rpm.Format == jsonTable {
		format = jsonTable
		order = "created"
	}

	return format, order
}

func (tr postgresRepository) getTimeColumn(table string) string {
	if table == jsonTable {
		return "created"
	}
	return "time"
}

func (tr postgresRepository) buildQueryParams(rpm readers.PageMetadata) map[string]interface{} {
	return map[string]interface{}{
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

func (tr postgresRepository) handlePgError(err error, wrapErr error) error {
	pgErr, ok := err.(*pgconn.PgError)
	if ok {
		switch pgErr.Code {
		case pgerrcode.UndefinedTable:
			return errors.Wrap(wrapErr, err)
		case pgerrcode.InvalidTextRepresentation:
			return errors.Wrap(wrapErr, errInvalidMessage)
		default:
			return errors.Wrap(wrapErr, err)
		}
	}
	return errors.Wrap(wrapErr, err)
}
