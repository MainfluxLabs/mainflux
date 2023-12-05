// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package timescale

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

var _ readers.MessageRepository = (*timescaleRepository)(nil)

var (
	errInvalidMessage = errors.New("invalid message representation")
	errTransRollback  = errors.New("failed to rollback transaction")
)

type timescaleRepository struct {
	db *sqlx.DB
}

// New returns new TimescaleSQL writer.
func New(db *sqlx.DB) readers.MessageRepository {
	return &timescaleRepository{
		db: db,
	}
}
func (tr timescaleRepository) ListAllMessages(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return tr.readAll("", rpm)
}

func (tr timescaleRepository) ListChannelMessages(chanID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return tr.readAll(chanID, rpm)
}

func (tr timescaleRepository) Backup(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return tr.readAll("", rpm)
}

func (tr timescaleRepository) Restore(ctx context.Context, messages ...senml.Message) error {
	q := `INSERT INTO messages (channel, subtopic, publisher, protocol,
		name, unit, value, string_value, bool_value, data_value, sum,
		time, update_time)
		VALUES (:channel, :subtopic, :publisher, :protocol, :name, :unit,
		:value, :string_value, :bool_value, :data_value, :sum,
		:time, :update_time);`

	tx, err := tr.db.BeginTxx(context.Background(), nil)
	if err != nil {
		return errors.Wrap(errors.ErrSaveMessage, err)
	}

	defer func() {
		if err != nil {
			if txErr := tx.Rollback(); txErr != nil {
				err = errors.Wrap(err, errors.Wrap(errTransRollback, txErr))
			}
			return
		}

		if err = tx.Commit(); err != nil {
			err = errors.Wrap(errors.ErrSaveMessage, err)
		}
	}()

	for _, msg := range messages {
		m := senmlMessage{Message: msg}
		if _, err := tx.NamedExec(q, m); err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errors.ErrSaveMessage, errInvalidMessage)
				}
			}

			return errors.Wrap(errors.ErrSaveMessage, err)
		}
	}

	return err
}

func (tr timescaleRepository) readAll(chanID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
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
		}
	}
	return condition
}

type senmlMessage struct {
	ID string `db:"id"`
	senml.Message
}

type jsonMessage struct {
	Channel   string `db:"channel"`
	Created   int64  `db:"created"`
	Subtopic  string `db:"subtopic"`
	Publisher string `db:"publisher"`
	Protocol  string `db:"protocol"`
	Payload   []byte `db:"payload"`
}

func (msg jsonMessage) toMap() (map[string]interface{}, error) {
	ret := map[string]interface{}{
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
