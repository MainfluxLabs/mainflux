// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package timescale

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
	"github.com/jmoiron/sqlx" // required for DB access
)

const (
	// Table for SenML messages
	defTable = "senml"
	// Table for JSON messages
	jsonTable = "json"
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

type PageMetadata struct {
	Offset      uint64  `json:"offset"`
	Limit       uint64  `json:"limit"`
	Subtopic    string  `json:"subtopic,omitempty"`
	Publisher   string  `json:"publisher,omitempty"`
	Protocol    string  `json:"protocol,omitempty"`
	Name        string  `json:"name,omitempty"`
	Value       float64 `json:"v,omitempty"`
	Comparator  string  `json:"comparator,omitempty"`
	BoolValue   bool    `json:"vb,omitempty"`
	StringValue string  `json:"vs,omitempty"`
	DataValue   string  `json:"vd,omitempty"`
	From        int64   `json:"from,omitempty"`
	To          int64   `json:"to,omitempty"`
	Format      string  `json:"format,omitempty"`
	AggInterval string  `json:"agg_interval,omitempty"`
	AggType     string  `json:"agg_type,omitempty"`
	AggField    string  `json:"agg_field,omitempty"`
}

func jsonPageMetaToPageMeta(jm readers.JSONPageMetadata) PageMetadata {
	return PageMetadata{
		Offset:    jm.Offset,
		Limit:     jm.Limit,
		Subtopic:  jm.Subtopic,
		Publisher: jm.Publisher,
		Protocol:  jm.Protocol,
		From:      jm.From,
		To:        jm.To,
	}
}

func senmlPageMetaToPageMeta(sm readers.SenMLPageMetadata) PageMetadata {
	return PageMetadata{
		Offset:      sm.Offset,
		Limit:       sm.Limit,
		Subtopic:    sm.Subtopic,
		Publisher:   sm.Publisher,
		Protocol:    sm.Protocol,
		Name:        sm.Name,
		Value:       sm.Value,
		Comparator:  sm.Comparator,
		BoolValue:   sm.BoolValue,
		StringValue: sm.StringValue,
		DataValue:   sm.DataValue,
		From:        sm.From,
		To:          sm.To,
	}
}

func fmtCondition(rpm PageMetadata) string {
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
			condition = fmt.Sprintf(`%s %s time <= :to`, condition, op)
			op = "AND"
		}
	}
	return condition
}

func (tr timescaleRepository) DeleteMessages(ctx context.Context, rpm PageMetadata) error {
	return nil
}

func (tr timescaleRepository) Restore(ctx context.Context, format string, messages ...readers.Message) error {
	q := `INSERT INTO senml (subtopic, publisher, protocol,
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
		m := senmlMessage{Message: msg.(senml.Message)}
		if _, err := tx.NamedExec(q, m); err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errors.ErrSaveMessages, errInvalidMessage)
				}
			}

			return errors.Wrap(errors.ErrSaveMessages, err)
		}
	}

	return err
}

func (tr timescaleRepository) readAllSenML(rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	pageMetadata := senmlPageMetaToPageMeta(rpm)

	olq := dbutil.GetOffsetLimitQuery(rpm.Limit)
	q := fmt.Sprintf(`SELECT * FROM %s %s ORDER BY time DESC %s;`, defTable, fmtCondition(pageMetadata), olq)

	params := map[string]interface{}{
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
				return readers.SenMLMessagesPage{}, nil
			}
		}
		return readers.SenMLMessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	defer rows.Close()

	messages := []readers.Message{}
	for rows.Next() {
		msg := senmlMessage{Message: senml.Message{}}
		if err := rows.StructScan(&msg); err != nil {
			return readers.SenMLMessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
		}
		messages = append(messages, msg.Message)
	}

	q = fmt.Sprintf(`SELECT COUNT(*) FROM %s %s;`, defTable, fmtCondition(pageMetadata))
	rows, err = tr.db.NamedQuery(q, params)
	if err != nil {
		return readers.SenMLMessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	defer rows.Close()

	total := uint64(0)
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return readers.SenMLMessagesPage{}, err
		}
	}

	return readers.SenMLMessagesPage{
		SenMLPageMetadata: rpm,
		MessagesPage: readers.MessagesPage{
			Total:    total,
			Messages: messages,
		},
	}, nil
}

func (tr timescaleRepository) readAllJSON(rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	pageMetadata := jsonPageMetaToPageMeta(rpm)

	olq := dbutil.GetOffsetLimitQuery(rpm.Limit)
	q := fmt.Sprintf(`SELECT * FROM %s %s ORDER BY created DESC %s;`, jsonTable, fmtCondition(pageMetadata), olq)

	params := map[string]interface{}{
		"limit":     rpm.Limit,
		"offset":    rpm.Offset,
		"subtopic":  rpm.Subtopic,
		"publisher": rpm.Publisher,
		"protocol":  rpm.Protocol,
		"from":      rpm.From,
		"to":        rpm.To,
	}

	rows, err := tr.db.NamedQuery(q, params)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == pgerrcode.UndefinedTable {
				return readers.JSONMessagesPage{}, nil
			}
		}
		return readers.JSONMessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	defer rows.Close()

	messages := []readers.Message{}
	for rows.Next() {
		msg := jsonMessage{}
		if err := rows.StructScan(&msg); err != nil {
			return readers.JSONMessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
		}
		m, err := msg.toMap()
		if err != nil {
			return readers.JSONMessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
		}
		messages = append(messages, m)
	}

	q = fmt.Sprintf(`SELECT COUNT(*) FROM %s %s;`, jsonTable, fmtCondition(pageMetadata))
	rows, err = tr.db.NamedQuery(q, params)
	if err != nil {
		return readers.JSONMessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	defer rows.Close()

	total := uint64(0)
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return readers.JSONMessagesPage{}, err
		}
	}

	return readers.JSONMessagesPage{
		JSONPageMetadata: rpm,
		MessagesPage: readers.MessagesPage{
			Total:    total,
			Messages: messages,
		},
	}, nil
}

type senmlMessage struct {
	ID string `db:"id"`
	senml.Message
}

type jsonMessage struct {
	Created   int64  `db:"created"`
	Subtopic  string `db:"subtopic"`
	Publisher string `db:"publisher"`
	Protocol  string `db:"protocol"`
	Payload   []byte `db:"payload"`
}

func (msg jsonMessage) toMap() (map[string]interface{}, error) {
	ret := map[string]interface{}{
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

func (tr timescaleRepository) ListJSONMessages(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	return tr.readAllJSON(rpm)
}
func (tr timescaleRepository) ListSenMLMessages(ctx context.Context, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	return tr.readAllSenML(rpm)
}

func (tr timescaleRepository) BackupJSONMessages(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	backup := rpm
	backup.Limit = 0
	backup.Offset = 0
	return tr.readAllJSON(backup)
}

func (tr timescaleRepository) BackupSenMLMessages(ctx context.Context, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	backup := rpm
	backup.Limit = 0
	backup.Offset = 0
	return tr.readAllSenML(backup)
}

func (tr timescaleRepository) RestoreJSONMessages(ctx context.Context, messages ...readers.Message) error {
	return nil
}

func (tr timescaleRepository) RestoreSenMLMessages(ctx context.Context, messages ...readers.Message) error {
	return nil
}

func (tr timescaleRepository) DeleteJSONMessages(ctx context.Context, rpm readers.JSONPageMetadata) error {
	return nil
}

func (tr timescaleRepository) DeleteSenMLMessages(ctx context.Context, rpm readers.SenMLPageMetadata) error {
	return nil
}
