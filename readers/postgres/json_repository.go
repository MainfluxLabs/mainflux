package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

type jsonRepository struct {
	db         *sqlx.DB
	aggregator *aggregationService
}

func newJSONRepository(db *sqlx.DB) *jsonRepository {
	return &jsonRepository{
		db:         db,
		aggregator: newAggregationService(db),
	}
}

func (jr *jsonRepository) ListMessages(rpm readers.JSONMetadata) (readers.JSONMessagesPage, error) {
	return jr.readAll(rpm)
}

func (jr *jsonRepository) readAll(rpm readers.JSONMetadata) (readers.JSONMessagesPage, error) {
	page := readers.JSONMessagesPage{
		JSONMetadata: rpm,
		MessagesPage: readers.MessagesPage{
			Messages: []readers.Message{},
			Total:    0,
		},
	}

	params := jr.buildQueryParams(rpm)

	if rpm.AggType != "" && rpm.AggInterval != "" {
		messages, err := jr.aggregator.readAggregatedJSONMessages(rpm)
		if err != nil {
			return page, err
		}
		page.Messages = messages

		total, err := jr.aggregator.readAggregatedJSONCount(rpm)
		if err != nil {
			return page, err
		}
		page.Total = total

		return page, nil
	}

	messages, err := jr.readMessages(rpm, params)
	if err != nil {
		return page, err
	}
	page.Messages = messages

	total, err := jr.readCount(rpm, params)
	if err != nil {
		return page, err
	}
	page.Total = total

	return page, nil
}

func (jr *jsonRepository) readMessages(rpm readers.JSONMetadata, params map[string]interface{}) ([]readers.Message, error) {
	olq := dbutil.GetOffsetLimitQuery(rpm.Limit)
	condition := jr.fmtCondition(rpm)

	query := fmt.Sprintf(`SELECT * FROM json %s ORDER BY created DESC %s;`, condition, olq)
	rows, err := jr.executeQuery(query, params)
	if err != nil {
		return nil, err
	}

	if rows == nil {
		return []readers.Message{}, nil
	}
	defer rows.Close()

	return jr.scanMessages(rows)
}

func (jr *jsonRepository) readCount(rpm readers.JSONMetadata, params map[string]interface{}) (uint64, error) {
	condition := jr.fmtCondition(rpm)
	query := fmt.Sprintf(`SELECT COUNT(*) FROM json %s;`, condition)

	rows, err := jr.executeQuery(query, params)
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

func (jr *jsonRepository) executeQuery(query string, params map[string]interface{}) (*sqlx.Rows, error) {
	rows, err := jr.db.NamedQuery(query, params)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.UndefinedTable {
			return nil, nil
		}
		return nil, errors.Wrap(readers.ErrReadMessages, err)
	}
	return rows, nil
}

func (jr *jsonRepository) fmtCondition(rpm readers.JSONMetadata) string {
	var query map[string]interface{}
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
		}
	}
	return condition
}

func (jr *jsonRepository) buildQueryParams(rpm readers.JSONMetadata) map[string]interface{} {
	return map[string]interface{}{
		"limit":     rpm.Limit,
		"offset":    rpm.Offset,
		"subtopic":  rpm.Subtopic,
		"publisher": rpm.Publisher,
		"protocol":  rpm.Protocol,
		"from":      rpm.From,
		"to":        rpm.To,
	}
}

func (jr *jsonRepository) Backup(rpm readers.JSONMetadata) (readers.JSONMessagesPage, error) {
	return jr.readAll(rpm)
}

func (jr *jsonRepository) DeleteMessages(ctx context.Context, rpm readers.JSONMetadata) error {
	tx, err := jr.db.BeginTxx(ctx, nil)
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

	condition := jr.fmtCondition(rpm)
	q := fmt.Sprintf("DELETE FROM json %s", condition)
	params := map[string]interface{}{
		"subtopic":  rpm.Subtopic,
		"publisher": rpm.Publisher,
		"protocol":  rpm.Protocol,
		"from":      rpm.From,
		"to":        rpm.To,
	}

	_, err = tx.NamedExecContext(ctx, q, params)
	if err != nil {
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
			return errors.Wrap(wrapErr, errInvalidMessage)
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

	q := `INSERT INTO json (subtopic, publisher, protocol, payload, created)
          VALUES (:subtopic, :publisher, :protocol, :payload, :created);`

	for _, msg := range messages {
		jsonMsg, ok := msg.(mfjson.Message)
		if !ok {
			return errors.Wrap(errors.ErrSaveMessages, errInvalidMessage)
		}

		if _, err := tx.NamedExecContext(ctx, q, jsonMsg); err != nil {
			return jr.handlePgError(err, errors.ErrSaveMessages)
		}
	}
	return nil
}
