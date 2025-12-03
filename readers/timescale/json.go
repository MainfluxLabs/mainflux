// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package timescale

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

var _ readers.JSONMessageRepository = (*jsonRepository)(nil)

type jsonRepository struct {
	db *sqlx.DB
}

func NewJSONRepository(db *sqlx.DB) readers.JSONMessageRepository {
	return &jsonRepository{
		db: db,
	}
}

func (jr *jsonRepository) Retrieve(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	return jr.readAll(ctx, rpm)
}

func (jr *jsonRepository) Backup(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	backup := rpm
	backup.Limit = 0
	backup.Offset = 0
	return jr.readAll(ctx, backup)
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
			msgMap, mapOk := msg.(map[string]any)
			if !mapOk {
				return errors.Wrap(errors.ErrSaveMessages, errors.ErrInvalidMessage)
			}

			jsonMsg = mfjson.Message{
				Subtopic:  msgMap["subtopic"].(string),
				Publisher: msgMap["publisher"].(string),
				Protocol:  msgMap["protocol"].(string),
				Created:   int64(msgMap["created"].(float64)),
			}

			if payload, ok := msgMap["payload"]; ok {
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					return errors.Wrap(errors.ErrSaveMessages, err)
				}
				jsonMsg.Payload = payloadBytes
			}
		}

		_, err := tx.NamedExecContext(ctx, q, jsonMsg)
		if err != nil {
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

func (jr *jsonRepository) readAll(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	page := readers.JSONMessagesPage{
		JSONPageMetadata: rpm,
		MessagesPage: readers.MessagesPage{
			Messages: []readers.Message{},
			Total:    0,
		},
	}

	params := jr.buildQueryParams(rpm)

	messages, err := jr.readMessages(ctx, rpm, params)
	if err != nil {
		return page, err
	}
	page.Messages = messages

	condition := jr.fmtCondition(rpm)
	q := fmt.Sprintf(`SELECT COUNT(*) FROM json %s;`, condition)
	total, err := dbutil.Total(ctx, jr.db, q, params)
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

	q := fmt.Sprintf(`SELECT * FROM json %s ORDER BY created %s %s;`, condition, dq, olq)
	rows, err := jr.db.NamedQueryContext(ctx, q, params)
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
		}
	}
	return condition
}

func (jr *jsonRepository) buildQueryParams(rpm readers.JSONPageMetadata) map[string]any {
	return map[string]any{
		"limit":     rpm.Limit,
		"offset":    rpm.Offset,
		"subtopic":  rpm.Subtopic,
		"publisher": rpm.Publisher,
		"protocol":  rpm.Protocol,
		"from":      rpm.From,
		"to":        rpm.To,
	}
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
