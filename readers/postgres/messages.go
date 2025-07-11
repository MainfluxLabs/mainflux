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

type AggregationType string

const (
	defTable                         = "messages"
	jsonTable                        = "json"
	AggregationNone  AggregationType = ""
	AggregationMin   AggregationType = "min"
	AggregationMax   AggregationType = "max"
	AggregationAvg   AggregationType = "avg"
	AggregationCount AggregationType = "count"
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

func (tr postgresRepository) Backup(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return tr.readAll(rpm)
}

func (tr postgresRepository) DeleteMessages(ctx context.Context, rpm readers.PageMetadata) error {
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

	tables := []string{defTable, jsonTable}
	for _, table := range tables {

		condition := fmtCondition(rpm, table)
		q := fmt.Sprintf("DELETE FROM %s %s", table, condition)
		params := tr.buildDeleteQueryParams(rpm)

		_, err := tx.NamedExecContext(ctx, q, params)
		if err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.UndefinedTable:
					return errors.Wrap(errors.ErrDeleteMessages, err)
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errors.ErrDeleteMessages, errInvalidMessage)
				default:
					return errors.Wrap(errors.ErrDeleteMessages, err)
				}
			}
			return errors.Wrap(errors.ErrDeleteMessages, err)
		}
	}

	return nil
}

func (tr postgresRepository) Restore(ctx context.Context, messages ...senml.Message) error {
	q := `INSERT INTO messages (subtopic, publisher, protocol,
          name, unit, value, string_value, bool_value, data_value, sum,
          time, update_time)
          VALUES (:subtopic, :publisher, :protocol, :name, :unit,
          :value, :string_value, :bool_value, :data_value, :sum,
          :time, :update_time);`

	tx, err := tr.db.BeginTxx(context.Background(), nil)
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

	for _, msg := range messages {
		m := msg

		if _, err := tx.NamedExec(q, m); err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok && pgErr.Code == pgerrcode.InvalidTextRepresentation {
				return errors.Wrap(errors.ErrSaveMessages, errInvalidMessage)
			}
			return errors.Wrap(errors.ErrSaveMessages, err)
		}
	}

	return err
}

func (tr postgresRepository) readAll(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	format, order := tr.getFormatAndOrder(rpm)
	params := tr.buildQueryParams(rpm)

	page := readers.MessagesPage{
		PageMetadata: rpm,
		Messages:     []readers.Message{},
	}

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

	if rpm.Aggregation != "" {
		aggregation, err := tr.readAggregation(rpm, format, order, params)
		if err != nil {
			return page, err
		}
		page.Aggregation = aggregation
	}

	return page, nil
}

func (tr postgresRepository) readMessages(rpm readers.PageMetadata, format, order string, params map[string]interface{}) ([]readers.Message, error) {
	query := tr.buildRegularQuery(rpm, format, order)
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
	query := tr.buildCountQuery(rpm, format, order)
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

func (tr postgresRepository) readAggregation(rpm readers.PageMetadata, format, order string, params map[string]interface{}) (readers.Aggregation, error) {
	query := tr.buildAggregationQuery(rpm, format, order)
	rows, err := tr.executeQuery(query, params)
	if err != nil {
		return readers.Aggregation{}, err
	}
	if rows == nil {
		return readers.Aggregation{}, nil
	}
	defer rows.Close()

	aggregationType := AggregationType(rpm.Aggregation)
	var result interface{}
	var count uint64

	if rows.Next() {
		if aggregationType == AggregationCount {
			if err := rows.Scan(&count, &count); err != nil {
				return readers.Aggregation{}, errors.Wrap(readers.ErrReadMessages, err)
			}
			result = count
		} else {
			if err := rows.Scan(&result, &count); err != nil {
				return readers.Aggregation{}, errors.Wrap(readers.ErrReadMessages, err)
			}
		}
	}

	aggregateField := tr.getAggregateField(rpm, format)
	return readers.Aggregation{
		Field: aggregateField,
		Value: result,
		Count: count,
	}, nil
}

func (tr postgresRepository) executeQuery(query string, params map[string]interface{}) (*sqlx.Rows, error) {
	rows, err := tr.db.NamedQuery(query, params)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == pgerrcode.UndefinedTable {
				return nil, nil
			}
		}
		return nil, errors.Wrap(readers.ErrReadMessages, err)
	}
	return rows, nil
}

func (tr postgresRepository) buildRegularQuery(rpm readers.PageMetadata, format, order string) string {
	olq := dbutil.GetOffsetLimitQuery(rpm.Limit)
	interval := rpm.Interval
	condition := fmtCondition(rpm, format)

	if interval != "" {
		switch format {
		case defTable:
			return fmt.Sprintf(`
				SELECT * FROM (
					SELECT DISTINCT ON (date_trunc('%[1]s', to_timestamp(%[2]s))) *
					FROM %[3]s %[4]s
					ORDER BY date_trunc('%[1]s', to_timestamp(%[2]s)), %[2]s DESC
				) sub
				%[5]s;`, interval, order, format, condition, olq)

		case jsonTable:
			return fmt.Sprintf(`
				SELECT * FROM (
					SELECT DISTINCT ON (date_trunc('%[1]s', to_timestamp(created / 1000000000))) *
					FROM %[2]s %[3]s
					ORDER BY date_trunc('%[1]s', to_timestamp(created / 1000000000)), created DESC
				) sub
				%[4]s;`, interval, format, condition, olq)
		}
	}

	return fmt.Sprintf(`SELECT * FROM %s %s ORDER BY %s DESC %s;`, format, condition, order, olq)
}

func (tr postgresRepository) buildCountQuery(rpm readers.PageMetadata, format, order string) string {
	interval := rpm.Interval
	condition := fmtCondition(rpm, format)

	if interval != "" {
		switch format {
		case defTable:
			return fmt.Sprintf(`
				SELECT COUNT(*) FROM (
					SELECT DISTINCT ON (date_trunc('%[1]s', to_timestamp(%[2]s))) *
					FROM %[3]s %[4]s
					ORDER BY date_trunc('%[1]s', to_timestamp(%[2]s)), %[2]s DESC
				) sub;`, interval, order, format, condition)
		case jsonTable:
			return fmt.Sprintf(`
				SELECT COUNT(*) FROM (
					SELECT DISTINCT ON (date_trunc('%[1]s', to_timestamp(created / 1000000000))) *
					FROM %[2]s %[3]s
					ORDER BY date_trunc('%[1]s', to_timestamp(created / 1000000000)), created DESC
				) sub;`, interval, format, condition)
		}
	}

	return fmt.Sprintf(`SELECT COUNT(*) FROM %s %s;`, format, condition)
}

func (tr postgresRepository) buildAggregationQuery(rpm readers.PageMetadata, format, order string) string {
	aggregateField := tr.getAggregateField(rpm, format)
	aggFunc := tr.buildAggregationFunction(rpm.Aggregation, aggregateField)
	subQuery := tr.buildSubQuery(rpm, format, order)

	return fmt.Sprintf(`SELECT %s as result, COUNT(*) as count FROM (%s) as paginated_results;`, aggFunc, subQuery)
}

func (tr postgresRepository) buildSubQuery(rpm readers.PageMetadata, format, order string) string {
	olq := dbutil.GetOffsetLimitQuery(rpm.Limit)
	interval := rpm.Interval
	condition := fmtCondition(rpm, format)

	if interval != "" {
		return tr.buildIntervalSubQuery(interval, format, order, condition, olq)
	}
	return fmt.Sprintf(`SELECT * FROM %s %s ORDER BY %s DESC %s`, format, condition, order, olq)
}

func (tr postgresRepository) buildIntervalSubQuery(interval, format, order, condition, olq string) string {
	switch format {
	case defTable:
		return fmt.Sprintf(`
				SELECT * FROM (
					SELECT DISTINCT ON (date_trunc('%[1]s', to_timestamp(%[2]s))) *
					FROM %[3]s %[4]s
					ORDER BY date_trunc('%[1]s', to_timestamp(%[2]s)), %[2]s DESC
				) sub
				%[5]s`, interval, order, format, condition, olq)

	case jsonTable:
		return fmt.Sprintf(`
				SELECT * FROM (
					SELECT DISTINCT ON (date_trunc('%[1]s', to_timestamp(created / 1000000000))) *
					FROM %[2]s %[3]s
					ORDER BY date_trunc('%[1]s', to_timestamp(created / 1000000000)), created DESC
				) sub
				%[4]s`, interval, format, condition, olq)
	default:
		return ""
	}
}

func (tr postgresRepository) buildAggregationFunction(aggregationType, aggregateField string) string {
	var aggFunc string

	switch AggregationType(aggregationType) {
	case AggregationMin:
		aggFunc = fmt.Sprintf("MIN(%s)", aggregateField)
	case AggregationMax:
		aggFunc = fmt.Sprintf("MAX(%s)", aggregateField)
	case AggregationAvg:
		aggFunc = fmt.Sprintf("AVG(%s)", aggregateField)
	case AggregationCount:
		aggFunc = "COUNT(*)"
	}

	return aggFunc
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

func fmtCondition(rpm readers.PageMetadata, table string) string {
	var query map[string]interface{}
	meta, err := json.Marshal(rpm)
	if err != nil {
		return ""
	}
	json.Unmarshal(meta, &query)

	condition := ""
	op := "WHERE"
	timeColumn := "time"

	if table != "" && table == jsonTable {
		timeColumn = "created"
	}

	for name := range query {
		switch name {
		case "subtopic", "publisher", "name", "protocol":
			condition = fmt.Sprintf(`%s %s %s = :%s`, condition, op, name, name)
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

func (tr postgresRepository) getAggregateField(rpm readers.PageMetadata, format string) string {
	switch rpm.AggregationField {
	case "":
		if format == jsonTable {
			return "created"
		} else {
			return "value"
		}
	default:
		return rpm.AggregationField
	}
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

func (tr postgresRepository) buildDeleteQueryParams(rpm readers.PageMetadata) map[string]interface{} {
	return map[string]interface{}{
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
