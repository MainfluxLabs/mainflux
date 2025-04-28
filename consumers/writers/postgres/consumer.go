// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"

	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
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
	msg, ok := message.(protomfx.Message)
	if !ok {
		return errors.ErrMessage
	}

	switch msg.ContentType {
	case messaging.JSONContentType:
		return pr.saveJSON(msg)
	default:
		return pr.saveSenML(msg)
	}
}

func (pr postgresRepo) saveSenML(msg protomfx.Message) (err error) {
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

	dbmsg, err := toSenMLMessage(msg)
	if err != nil {
		return errors.Wrap(errors.ErrSaveMessage, err)
	}

	if _, err := tx.NamedExec(q, dbmsg); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(errors.ErrSaveMessage, errInvalidMessage)
			}
		}

		return errors.Wrap(errors.ErrSaveMessage, err)
	}

	return err
}

func (pr postgresRepo) saveJSON(msg protomfx.Message) error {
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

	dbmsg := toJSONMessage(msg)
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

	return err
}

type jsonMessage struct {
	Created   int64  `db:"created"`
	Subtopic  string `db:"subtopic"`
	Publisher string `db:"publisher"`
	Protocol  string `db:"protocol"`
	Payload   []byte `db:"payload"`
}

func toJSONMessage(message protomfx.Message) jsonMessage {
	return jsonMessage{
		Created:   message.Created,
		Subtopic:  message.Subtopic,
		Publisher: message.Publisher,
		Protocol:  message.Protocol,
		Payload:   message.Payload,
	}
}

func toSenMLMessage(message protomfx.Message) (senml.Message, error) {
	var msg senml.Message
	if err := json.Unmarshal(message.Payload, &msg); err != nil {
		return senml.Message{}, err
	}

	msg.Publisher = message.Publisher
	msg.Subtopic = message.Subtopic
	msg.Protocol = message.Protocol

	return msg, nil
}
