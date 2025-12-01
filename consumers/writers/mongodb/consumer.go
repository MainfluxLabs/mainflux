// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mongodb

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const (
	senmlCollection = "messages"
	jsonCollection  = "json"
)

var _ consumers.Consumer = (*mongoRepo)(nil)

type mongoRepo struct {
	db *mongo.Database
}

// New returns new MongoDB writer.
func New(db *mongo.Database) consumers.Consumer {
	return &mongoRepo{db}
}

func (repo *mongoRepo) Consume(message any) error {
	if msg, ok := message.(protomfx.Message); ok {
		msgs, err := messaging.SplitMessage(msg)
		if err != nil {
			return err
		}

		switch msg.ContentType {
		case messaging.JSONContentType:
			return repo.saveJSON(msgs)
		default:
			return repo.saveSenML(msgs)
		}
	}

	return errors.ErrMessage
}

func (repo *mongoRepo) saveSenML(msgs []protomfx.Message) error {
	coll := repo.db.Collection(senmlCollection)
	var dbMsgs []any
	for _, msg := range msgs {
		mapped, err := messaging.ToSenMLMessage(msg)
		if err != nil {
			return errors.Wrap(errors.ErrSaveMessages, err)
		}

		dbMsgs = append(dbMsgs, mapped)
	}

	_, err := coll.InsertMany(context.Background(), dbMsgs)
	if err != nil {
		return errors.Wrap(errors.ErrSaveMessages, err)
	}

	return nil
}

func (repo *mongoRepo) saveJSON(msgs []protomfx.Message) error {
	m := []any{}
	for _, msg := range msgs {
		mapped := messaging.ToJSONMessage(msg)
		m = append(m, mapped)
	}

	coll := repo.db.Collection(jsonCollection)

	_, err := coll.InsertMany(context.Background(), m)
	if err != nil {
		return errors.Wrap(errors.ErrSaveMessages, err)
	}

	return nil
}
