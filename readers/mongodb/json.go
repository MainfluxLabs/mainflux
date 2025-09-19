// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mongodb

import (
	"context"
	"encoding/json"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/readers"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	jsonCollection = "json"
	jsonOrder      = "created"
	noLimit        = 0
)

var _ readers.JSONMessaageRepository = (*jsonRepository)(nil)

type jsonRepository struct {
	db *mongo.Database
}

func NewJSONRepository(db *mongo.Database) readers.JSONMessaageRepository {
	return &jsonRepository{
		db: db,
	}
}

func (jr *jsonRepository) ListMessages(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	return jr.readAll(ctx, rpm)
}

func (jr *jsonRepository) Backup(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	backupMetadata := rpm
	backupMetadata.Limit = noLimit
	backupMetadata.Offset = 0
	return jr.readAll(ctx, backupMetadata)
}

func (jr *jsonRepository) DeleteMessages(ctx context.Context, rpm readers.JSONPageMetadata) error {
	col := jr.db.Collection(jsonCollection)

	filter := jr.fmtCondition(rpm)

	if len(filter) == 0 {
		return errors.Wrap(errors.ErrDeleteMessages, errors.New("no delete criteria specified"))
	}

	_, err := col.DeleteMany(ctx, filter)
	if err != nil {
		return errors.Wrap(errors.ErrDeleteMessages, err)
	}

	return nil
}

func (jr *jsonRepository) Restore(ctx context.Context, messages ...readers.Message) error {
	if len(messages) == 0 {
		return nil
	}

	coll := jr.db.Collection(jsonCollection)
	var docs []interface{}
	for _, msg := range messages {
		docs = append(docs, msg)
	}

	_, err := coll.InsertMany(ctx, docs)
	if err != nil {
		return errors.Wrap(errors.ErrSaveMessages, err)
	}

	return nil
}

func (jr *jsonRepository) readAll(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	col := jr.db.Collection(jsonCollection)

	filter := jr.fmtCondition(rpm)

	sortMap := bson.D{{Key: jsonOrder, Value: -1}}

	findOpts := options.Find().SetSort(sortMap)
	if rpm.Limit != noLimit {
		findOpts.SetLimit(int64(rpm.Limit)).SetSkip(int64(rpm.Offset))
	}

	cursor, err := col.Find(ctx, filter, findOpts)
	if err != nil {
		return readers.JSONMessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	defer cursor.Close(ctx)

	var messages []readers.Message
	for cursor.Next(ctx) {
		var m map[string]interface{}
		if err := cursor.Decode(&m); err != nil {
			return readers.JSONMessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
		}
		messages = append(messages, m)
	}

	total, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return readers.JSONMessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}

	return readers.JSONMessagesPage{
		JSONPageMetadata: rpm,
		MessagesPage: readers.MessagesPage{
			Total:    uint64(total),
			Messages: messages,
		},
	}, nil
}

func (jr *jsonRepository) fmtCondition(rpm readers.JSONPageMetadata) bson.D {
	filter := bson.D{}

	var query map[string]interface{}
	meta, err := json.Marshal(rpm)
	if err != nil {
		return filter
	}
	json.Unmarshal(meta, &query)

	for name, value := range query {
		switch name {
		case "subtopic", "publisher", "protocol":
			filter = append(filter, bson.E{Key: name, Value: value})
		case "from":
			filter = append(filter, bson.E{Key: jsonOrder, Value: bson.M{"$gte": value}})
		case "to":
			filter = append(filter, bson.E{Key: jsonOrder, Value: bson.M{"$lt": value}})
		}
	}
	return filter
}
