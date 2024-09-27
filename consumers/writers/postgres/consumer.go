// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"

	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx" // required for DB access
)

var (
	errInvalidMessage = errors.New("invalid message representation")
	errTransRollback  = errors.New("failed to rollback transaction")
)

var _ consumers.Consumer = (*postgresRepo)(nil)

type postgresRepo struct {
	db *sqlx.DB
}

// New returns new PostgreSQL writer.
func New(db *sqlx.DB) consumers.Consumer {
	return &postgresRepo{db: db}
}

func (pr postgresRepo) Consume(message interface{}) (err error) {
	switch m := message.(type) {
	case mfjson.Messages:
		return pr.saveJSON(m)
	default:
		return pr.saveSenml(m)
	}
}

func (pr postgresRepo) saveSenml(messages interface{}) (err error) {
	msgs, ok := messages.([]senml.Message)
	if !ok {
		return errors.ErrSaveMessage
	}
	q := `INSERT INTO messages (subtopic, publisher, protocol,
          name, unit, value, string_value, bool_value, data_value, sum,
          time, update_time)
          VALUES (:subtopic, :publisher, :protocol, :name, :unit,
          :value, :string_value, :bool_value, :data_value, :sum,
          :time, :update_time);`

	tx, err := pr.db.BeginTxx(context.Background(), nil)
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

	for _, msg := range msgs {
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

func (pr postgresRepo) saveJSON(messages interface{}) error {
	msgs, ok := messages.(mfjson.Messages)
	if !ok {
		return errors.ErrSaveMessage
	}
	q := `INSERT INTO json (created, subtopic, publisher, protocol, payload)
          VALUES (:created, :subtopic, :publisher, :protocol, :payload);`

	tx, err := pr.db.BeginTxx(context.Background(), nil)
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

	for _, m := range msgs.Data {
		var dbmsg jsonMessage
		dbmsg, err = toJSONMessage(m)
		if err != nil {
			return errors.Wrap(errors.ErrSaveMessage, err)
		}

		if _, err = tx.NamedExec(q, dbmsg); err != nil {
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

type senmlMessage struct {
	senml.Message
}

type jsonMessage struct {
	Created   int64  `db:"created"`
	Subtopic  string `db:"subtopic"`
	Publisher string `db:"publisher"`
	Protocol  string `db:"protocol"`
	Payload   []byte `db:"payload"`
}

func toJSONMessage(msg mfjson.Message) (jsonMessage, error) {
	data := []byte("{}")
	if msg.Payload != nil {
		b, err := json.Marshal(msg.Payload)
		if err != nil {
			return jsonMessage{}, errors.Wrap(errors.ErrSaveMessage, err)
		}
		data = b
	}

	m := jsonMessage{
		Created:   msg.Created,
		Subtopic:  msg.Subtopic,
		Publisher: msg.Publisher,
		Protocol:  msg.Protocol,
		Payload:   data,
	}

	return m, nil
}
