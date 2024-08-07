// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx" // required for DB access
)

var (
	errInvalidMessage = errors.New("invalid message representation")
	errTransRollback  = errors.New("failed to rollback transaction")
	errNoTable        = errors.New("relation does not exist")
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
	q := `INSERT INTO messages (id, subtopic, publisher, protocol,
          name, unit, value, string_value, bool_value, data_value, sum,
          time, update_time)
          VALUES (:id, :subtopic, :publisher, :protocol, :name, :unit,
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
		id, err := uuid.NewV4()
		if err != nil {
			return err
		}
		m := senmlMessage{Message: msg, ID: id.String()}
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

func (pr postgresRepo) saveJSON(msgs mfjson.Messages) error {
	if err := pr.insertJSON(msgs); err != nil {
		if err == errNoTable {
			if err := pr.createTable(msgs.Format); err != nil {
				return err
			}
			return pr.insertJSON(msgs)
		}
		return err
	}
	return nil
}

func (pr postgresRepo) insertJSON(msgs mfjson.Messages) error {
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

	q := `INSERT INTO %s (id, created, subtopic, publisher, protocol, payload)
          VALUES (:id, :created, :subtopic, :publisher, :protocol, :payload);`
	q = fmt.Sprintf(q, msgs.Format)

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
				case pgerrcode.UndefinedTable:
					return errNoTable
				}
			}
			return err
		}
	}
	return nil
}

func (pr postgresRepo) createTable(name string) error {
	q := `CREATE TABLE IF NOT EXISTS %s (
            id            UUID,
            created       BIGINT,
            subtopic      VARCHAR(254),
            publisher     VARCHAR(254),
            protocol      TEXT,
            payload       JSONB,
            PRIMARY KEY (id)
        )`
	q = fmt.Sprintf(q, name)

	_, err := pr.db.Exec(q)
	return err
}

type senmlMessage struct {
	senml.Message
	ID string `db:"id"`
}

type jsonMessage struct {
	ID        string `db:"id"`
	Created   int64  `db:"created"`
	Subtopic  string `db:"subtopic"`
	Publisher string `db:"publisher"`
	Protocol  string `db:"protocol"`
	Payload   []byte `db:"payload"`
}

func toJSONMessage(msg mfjson.Message) (jsonMessage, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return jsonMessage{}, err
	}

	data := []byte("{}")
	if msg.Payload != nil {
		b, err := json.Marshal(msg.Payload)
		if err != nil {
			return jsonMessage{}, errors.Wrap(errors.ErrSaveMessage, err)
		}
		data = b
	}

	m := jsonMessage{
		ID:        id.String(),
		Created:   msg.Created,
		Subtopic:  msg.Subtopic,
		Publisher: msg.Publisher,
		Protocol:  msg.Protocol,
		Payload:   data,
	}

	return m, nil
}
