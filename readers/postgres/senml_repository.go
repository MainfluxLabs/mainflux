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

const (
	senmlTable = "messages"
	senmlOrder = "time"
)

type senmlRepository struct {
	db         *sqlx.DB
	aggregator *aggregationService
}

func newSenMLRepository(db *sqlx.DB) *senmlRepository {
	return &senmlRepository{
		db: db,
	}
}

func (sr *senmlRepository) ListMessages(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return sr.readAll(rpm)
}

func (sr *senmlRepository) Backup(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return sr.readAll(rpm)
}

func (sr *senmlRepository) DeleteMessages(ctx context.Context, rpm readers.PageMetadata) error {
	tx, err := sr.db.BeginTxx(ctx, nil)
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

	condition := sr.fmtCondition(rpm)
	q := fmt.Sprintf("DELETE FROM messages %s", condition)
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
		return sr.handlePgError(err, errors.ErrDeleteMessages)
	}

	return nil
}

func (sr *senmlRepository) readAll(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	page := readers.MessagesPage{
		PageMetadata: rpm,
		Messages:     []readers.Message{},
	}

	params := sr.buildQueryParams(rpm)

	if rpm.AggType != "" && rpm.AggInterval != "" {
		messages, err := sr.aggregator.readAggregatedMessages(rpm)
		if err != nil {
			return page, err
		}
		page.Messages = messages

		total, err := sr.aggregator.readAggregatedCount(rpm)
		if err != nil {
			return page, err
		}
		page.Total = total

		return page, nil
	}

	messages, err := sr.readMessages(rpm, params)
	if err != nil {
		return page, err
	}
	page.Messages = messages

	total, err := sr.readCount(rpm, params)
	if err != nil {
		return page, err
	}
	page.Total = total

	return page, nil
}

func (sr *senmlRepository) readMessages(rpm readers.PageMetadata, params map[string]interface{}) ([]readers.Message, error) {
	olq := dbutil.GetOffsetLimitQuery(rpm.Limit)
	condition := sr.fmtCondition(rpm)
	query := fmt.Sprintf(`SELECT * FROM messages %s ORDER BY time DESC %s;`, condition, olq)

	rows, err := sr.executeQuery(query, params)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []readers.Message{}, nil
	}
	defer rows.Close()

	return sr.scanMessages(rows)
}

func (sr *senmlRepository) readCount(rpm readers.PageMetadata, params map[string]interface{}) (uint64, error) {
	condition := sr.fmtCondition(rpm)
	query := fmt.Sprintf(`SELECT COUNT(*) FROM messages %s;`, condition)

	rows, err := sr.executeQuery(query, params)
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

func (sr *senmlRepository) executeQuery(query string, params map[string]interface{}) (*sqlx.Rows, error) {
	rows, err := sr.db.NamedQuery(query, params)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.UndefinedTable {
			return nil, nil
		}
		return nil, errors.Wrap(readers.ErrReadMessages, err)
	}
	return rows, nil
}

func (sr *senmlRepository) fmtCondition(rpm readers.PageMetadata) string {
	var query map[string]interface{}
	meta, err := json.Marshal(rpm)
	if err != nil {
		return ""
	}
	json.Unmarshal(meta, &query)

	condition := ""
	op := "WHERE"
	timeColumn := senmlOrder

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
			condition = fmt.Sprintf(`%s %s %s >= :from`, condition, op, timeColumn)
			op = "AND"
		case "to":
			condition = fmt.Sprintf(`%s %s %s <= :to`, condition, op, timeColumn)
			op = "AND"
		}
	}
	return condition
}

func (sr *senmlRepository) buildQueryParams(rpm readers.PageMetadata) map[string]interface{} {
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

func (sr *senmlRepository) Restore(ctx context.Context, messages ...readers.Message) error {
	tx, err := sr.db.BeginTxx(ctx, nil)
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
			return sr.handlePgError(err, errors.ErrSaveMessages)
		}
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
			return errors.Wrap(wrapErr, errInvalidMessage)
		default:
			return errors.Wrap(wrapErr, err)
		}
	}
	return errors.Wrap(wrapErr, err)
}
