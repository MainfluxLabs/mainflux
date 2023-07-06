// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx" // required for DB access
)

const (
	// Table for SenML messages
	defTable = "messages"
	// noLimit is used to indicate that there is no limit for the number of results.
	noLimit = 0

	// Error code for Undefined table error.
	undefinedTableCode = "42P01"
)

var _ readers.MessageRepository = (*postgresRepository)(nil)

type postgresRepository struct {
	db *sqlx.DB
}

// New returns new PostgreSQL writer.
func New(db *sqlx.DB) readers.MessageRepository {
	return &postgresRepository{
		db: db,
	}
}

func (tr postgresRepository) ListAllMessages(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return tr.readAll("", rpm)
}

func (tr postgresRepository) ListChannelMessages(chanID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return tr.readAll(chanID, rpm)
}

func (tr postgresRepository) Save(ctx context.Context, messages ...readers.BackupMessage) error {
	tx, err := tr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	q := fmt.Sprintf(`INSERT INTO %s (id, channel, subtopic, publisher, protocol, name,unit, value, bool_value, string_value, data_value, sum, time, update_time)
	VALUES (:id, :channel, :subtopic, :publisher, :protocol, :name, :unit, :value, :bool_value, :string_value, :data_value, :sum, :time, :update_time);`, defTable)

	for _, msg := range messages {

		dbms := toDBMessage(msg)

		if _, err := tx.NamedExecContext(ctx, q, dbms); err != nil {
			tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return errors.Wrap(errors.ErrConflict, err)
				case pgerrcode.StringDataRightTruncationDataException:
					return errors.Wrap(errors.ErrMalformedEntity, err)

				}
			}
			return errors.Wrap(errors.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (tr postgresRepository) readAll(chanID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
	order := "time"
	format := defTable

	if rpm.Format != "" && rpm.Format != defTable {
		order = "created"
		format = rpm.Format
	}

	olq := "LIMIT :limit OFFSET :offset"
	if rpm.Limit == 0 {
		olq = ""
	}

	q := fmt.Sprintf(`SELECT * FROM %s %s ORDER BY %s DESC %s;`, format, fmtCondition(chanID, rpm), order, olq)

	params := map[string]interface{}{
		"channel":      chanID,
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

	rows, err := tr.db.NamedQuery(q, params)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == pgerrcode.UndefinedTable {
				return readers.MessagesPage{}, nil
			}
		}
		return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	defer rows.Close()

	page := readers.MessagesPage{
		PageMetadata: rpm,
		Messages:     []readers.Message{},
	}
	switch format {
	case defTable:
		for rows.Next() {
			msg := senmlMessage{Message: senml.Message{}}
			if err := rows.StructScan(&msg); err != nil {
				return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
			}

			page.Messages = append(page.Messages, msg.Message)
		}
	default:
		for rows.Next() {
			msg := jsonMessage{}
			if err := rows.StructScan(&msg); err != nil {
				return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
			}
			m, err := msg.toMap()
			if err != nil {
				return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
			}
			page.Messages = append(page.Messages, m)
		}

	}

	q = fmt.Sprintf(`SELECT COUNT(*) FROM %s %s;`, format, fmtCondition(chanID, rpm))
	rows, err = tr.db.NamedQuery(q, params)
	if err != nil {
		return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	defer rows.Close()

	total := uint64(0)
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return page, err
		}
	}
	page.Total = total

	return page, nil
}

func fmtCondition(chanID string, rpm readers.PageMetadata) string {
	var query map[string]interface{}
	meta, err := json.Marshal(rpm)
	if err != nil {
		return ""
	}
	json.Unmarshal(meta, &query)

	condition := ""
	op := "WHERE"
	if chanID != "" {
		condition = fmt.Sprintf(`%s channel = :channel`, op)
		op = "AND"
	}

	for name := range query {
		switch name {
		case
			"subtopic",
			"publisher",
			"name",
			"protocol":
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
			condition = fmt.Sprintf(`%s %s time >= :from`, condition, op)
			op = "AND"
		case "to":
			condition = fmt.Sprintf(`%s %s time < :to`, condition, op)
			op = "AND"
		}
	}
	return condition
}

type senmlMessage struct {
	ID string `db:"id"`
	senml.Message
}

type jsonMessage struct {
	ID        string `db:"id"`
	Channel   string `db:"channel"`
	Created   int64  `db:"created"`
	Subtopic  string `db:"subtopic"`
	Publisher string `db:"publisher"`
	Protocol  string `db:"protocol"`
	Payload   []byte `db:"payload"`
}

func (msg jsonMessage) toMap() (map[string]interface{}, error) {
	ret := map[string]interface{}{
		"id":        msg.ID,
		"channel":   msg.Channel,
		"created":   msg.Created,
		"subtopic":  msg.Subtopic,
		"publisher": msg.Publisher,
		"protocol":  msg.Protocol,
		"payload":   map[string]interface{}{},
	}
	pld := make(map[string]interface{})
	if err := json.Unmarshal(msg.Payload, &pld); err != nil {
		return nil, err
	}
	ret["payload"] = pld
	return ret, nil
}

type dbMessage struct {
	ID           string  `json:"id,omitempty"`
	Channel      string  `json:"channel"`
	Subtopic     string  `json:"subtopic"`
	Publisher    string  `json:"publisher"`
	Protocol     string  `json:"protocol"`
	Name         string  `json:"name"`
	Unit         string  `json:"unit,omitempty"`
	Value        float64 `json:"value"`
	String_value string  `json:"string_value,omitempty"`
	Bool_value   bool    `json:"bool_value,omitempty"`
	Data_value   []byte  `json:"data_value,omitempty"`
	Sum          float64 `json:"sum,omitempty"`
	Time         float64 `json:"time"`
	Update_time  float64 `json:"update_time,omitempty"`
}

func toDBMessage(ms readers.BackupMessage) dbMessage {
	return dbMessage{
		ID:           ms.ID,
		Channel:      ms.Channel,
		Subtopic:     ms.Subtopic,
		Publisher:    ms.Publisher,
		Protocol:     ms.Protocol,
		Name:         ms.Name,
		Unit:         ms.Unit,
		Value:        ms.Value,
		String_value: ms.String_value,
		Bool_value:   ms.Bool_value,
		Data_value:   ms.Data_value,
		Sum:          ms.Sum,
		Time:         ms.Time,
		Update_time:  ms.Update_time,
	}
}
