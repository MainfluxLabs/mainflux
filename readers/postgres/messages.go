// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"encoding/json"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/jmoiron/sqlx" // required for DB access
	"github.com/lib/pq"
)

const (
	// Table for SenML messages
	defTable = "messages"
	// noLimit is used to indicate that no limit is set
	noLimit = -1

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

func (tr postgresRepository) readAll(chanID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
	order := "time"
	format := defTable

	if rpm.Format != "" && rpm.Format != defTable {
		order = "created"
		format = rpm.Format
	}

	q := fmt.Sprintf(`SELECT * FROM %s %s ORDER BY %s DESC LIMIT :limit OFFSET :offset;`, format, fmtCondition(chanID, rpm), order)

	qNoLimit := fmt.Sprintf(`SELECT * FROM %s %s ORDER BY %s DESC;`, format, fmtCondition(chanID, rpm), order)

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

	var query string
	switch rpm.Limit {
	case noLimit:
		query = qNoLimit
	default:
		query = q
	}

	rows, err := tr.db.NamedQuery(query, params)
	if err != nil {
		if e, ok := err.(*pq.Error); ok {
			if e.Code == undefinedTableCode {
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
		return "ERRRROR"
	}
	json.Unmarshal(meta, &query)

	condition := ""
	op := "WHERE"
	if chanID != "" {
		condition = fmt.Sprintf(`%s channel = :channel`, op)
		op = "AND"
	}

	if _, ok := query["subtopic"]; ok {
		condition = fmt.Sprintf(`%s %s subtopic = :subtopic`, condition, op)
		op = "AND"
	}

	if _, ok := query["publisher"]; ok {
		condition = fmt.Sprintf(`%s %s publisher = :publisher`, condition, op)
		op = "AND"
	}

	if _, ok := query["name"]; ok {
		condition = fmt.Sprintf(`%s %s name = :name`, condition, op)
		op = "AND"
	}

	if _, ok := query["protocol"]; ok {
		condition = fmt.Sprintf(`%s %s protocol = :protocol`, condition, op)
		op = "AND"
	}

	if _, ok := query["v"]; ok {
		comparator := readers.ParseValueComparator(query)
		condition = fmt.Sprintf(`%s %s value %s :value`, condition, op, comparator)
		op = "AND"
	}

	if _, ok := query["vb"]; ok {
		condition = fmt.Sprintf(`%s %s bool_value = :bool_value`, condition, op)
		op = "AND"
	}

	if _, ok := query["vs"]; ok {
		condition = fmt.Sprintf(`%s %s string_value = :string_value`, condition, op)
		op = "AND"
	}

	if _, ok := query["vd"]; ok {
		condition = fmt.Sprintf(`%s %s data_value = :data_value`, condition, op)
		op = "AND"
	}

	if _, ok := query["from"]; ok {
		condition = fmt.Sprintf(`%s %s time >= :from`, condition, op)
		op = "AND"
	}

	if _, ok := query["to"]; ok {
		condition = fmt.Sprintf(`%s %s time < :to`, condition, op)
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
