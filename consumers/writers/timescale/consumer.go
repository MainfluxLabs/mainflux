// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package timescale

import (
	"context"
	"hash/fnv"

	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx" // required for DB access
)

var (
	errInvalidMessage = errors.New("invalid message representation")
	errTransRollback  = errors.New("failed to rollback transaction")
)

var _ consumers.MessageConsumer = (*timescaleRepo)(nil)

type timescaleRepo struct {
	db *sqlx.DB
}

// New returns new TimescaleSQL writer.
func New(db *sqlx.DB) consumers.MessageConsumer {
	return &timescaleRepo{db: db}
}

func (tr timescaleRepo) ConsumeMessage(_ string, msg protomfx.Message) error {
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

type jsonRow struct {
	mfjson.Message
	PayloadHash int32 `db:"payload_hash"`
}

// jsonPayloadHash hashes the payload, letting the unique index dedupe exact
// repeats while still storing distinct content at the same timestamp.
func jsonPayloadHash(msg mfjson.Message) int32 {
	h := fnv.New32a()
	h.Write(msg.Payload)
	return int32(h.Sum32())
}

func (tr timescaleRepo) saveJSON(msgs []protomfx.Message) error {
	q := `INSERT INTO json (created, subtopic, publisher, protocol, payload, payload_hash)
          VALUES (:created, :subtopic, :publisher, :protocol, :payload, :payload_hash)
          ON CONFLICT (created, publisher, subtopic, payload_hash) DO NOTHING;`

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

		row := jsonRow{Message: dbmsg, PayloadHash: jsonPayloadHash(dbmsg)}

		if _, err := tx.NamedExec(q, row); err != nil {
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
