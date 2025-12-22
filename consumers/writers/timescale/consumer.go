// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package timescale

import (
	"context"

	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx" // required for DB access
)

var (
	errInvalidMessage = errors.New("invalid message representation")
	errTransRollback  = errors.New("failed to rollback transaction")
)

var _ consumers.Consumer = (*timescaleRepo)(nil)

type timescaleRepo struct {
	db *sqlx.DB
}

// New returns new TimescaleSQL writer.
func New(db *sqlx.DB) consumers.Consumer {
	return &timescaleRepo{db: db}
}

func (tr timescaleRepo) Consume(message any) error {
	if msg, ok := message.(protomfx.Message); ok {
		msgs, err := messaging.SplitMessage(msg)
		if err != nil {
			return err
		}

		switch msg.ContentType {
		case messaging.JSONContentType:
			return tr.saveJSON(msgs)
		default:
			return tr.saveSenML(msgs)
		}
	}

	return errors.ErrMessage
}

func (tr timescaleRepo) saveSenML(msgs []protomfx.Message) (err error) {
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

	for _, msg := range msgs {
		dbmsg, err := messaging.ToSenMLMessage(msg)
		if err != nil {
			return errors.Wrap(errors.ErrSaveMessages, err)
		}

		if _, err := tx.NamedExec(q, dbmsg); err != nil {
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

func (tr timescaleRepo) saveJSON(msgs []protomfx.Message) error {
	q := `INSERT INTO json (created, subtopic, publisher, protocol, payload)
          VALUES (:created, :subtopic, :publisher, :protocol, :payload);`

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

	for _, msg := range msgs {
		dbmsg := messaging.ToJSONMessage(msg)

		if _, err := tx.NamedExec(q, dbmsg); err != nil {
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
