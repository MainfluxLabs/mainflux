// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mongodb

import (
	"context"
	"encoding/json"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	senmlCollection = "messages"
	senmlOrder      = "time"
)

var _ readers.SenMLMessageRepository = (*senmlRepository)(nil)

type senmlRepository struct {
	db *mongo.Database
}

func NewSenMLRepository(db *mongo.Database) readers.SenMLMessageRepository {
	return &senmlRepository{
		db: db,
	}
}

func (sr *senmlRepository) Retrieve(ctx context.Context, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	return sr.readAll(ctx, rpm)
}

func (sr *senmlRepository) Backup(ctx context.Context, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	backupMetadata := rpm
	backupMetadata.Limit = noLimit
	backupMetadata.Offset = 0
	return sr.readAll(ctx, backupMetadata)
}

func (sr *senmlRepository) Remove(ctx context.Context, rpm readers.SenMLPageMetadata) error {
	coll := sr.db.Collection(senmlCollection)
	filter := sr.fmtCondition(rpm)

	if len(filter) == 0 {
		return errors.Wrap(errors.ErrDeleteMessages, errors.New("no delete criteria specified"))
	}

	_, err := coll.DeleteMany(ctx, filter)
	if err != nil {
		return errors.Wrap(errors.ErrDeleteMessages, err)
	}

	return nil
}

func (sr *senmlRepository) Restore(ctx context.Context, messages ...readers.Message) error {
	if len(messages) == 0 {
		return nil
	}

	coll := sr.db.Collection(senmlCollection)
	var docs []any
	for _, msg := range messages {
		senmlMessage, ok := msg.(senml.Message)
		if !ok {
			return errors.Wrap(errors.ErrSaveMessages, errors.ErrInvalidMessage)
		}
		docs = append(docs, senmlMessage)
	}

	_, err := coll.InsertMany(ctx, docs)
	if err != nil {
		return errors.Wrap(errors.ErrSaveMessages, err)
	}

	return nil
}

func (sr *senmlRepository) readAll(ctx context.Context, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	coll := sr.db.Collection(senmlCollection)
	filter := sr.fmtCondition(rpm)
	dir := 1
	if rpm.Dir == apiutil.DescDir {
		dir = -1
	}

	sortMap := bson.D{{Key: senmlOrder, Value: dir}}

	findOpts := options.Find().SetSort(sortMap)
	if rpm.Limit != noLimit {
		findOpts.SetLimit(int64(rpm.Limit)).SetSkip(int64(rpm.Offset))
	}

	cursor, err := coll.Find(ctx, filter, findOpts)
	if err != nil {
		return readers.SenMLMessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	defer cursor.Close(ctx)

	var messages []readers.Message
	for cursor.Next(ctx) {
		var m senml.Message
		if err := cursor.Decode(&m); err != nil {
			return readers.SenMLMessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
		}
		messages = append(messages, m)
	}

	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return readers.SenMLMessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}

	return readers.SenMLMessagesPage{
		SenMLPageMetadata: rpm,
		MessagesPage: readers.MessagesPage{
			Total:    uint64(total),
			Messages: messages,
		},
	}, nil
}

func (sr *senmlRepository) fmtCondition(rpm readers.SenMLPageMetadata) bson.D {
	filter := bson.D{}

	var query map[string]any
	meta, err := json.Marshal(rpm)
	if err != nil {
		return filter
	}
	json.Unmarshal(meta, &query)

	for name, value := range query {
		switch name {
		case "subtopic", "publisher", "name", "protocol":
			filter = append(filter, bson.E{Key: name, Value: value})
		case "v":
			var bsonFilter any = value
			val, ok := query["comparator"]
			if ok {
				switch val.(string) {
				case readers.EqualKey:
					bsonFilter = value
				case readers.LowerThanKey:
					bsonFilter = bson.M{"$lt": value}
				case readers.LowerThanEqualKey:
					bsonFilter = bson.M{"$lte": value}
				case readers.GreaterThanKey:
					bsonFilter = bson.M{"$gt": value}
				case readers.GreaterThanEqualKey:
					bsonFilter = bson.M{"$gte": value}
				}
			}
			filter = append(filter, bson.E{Key: "value", Value: bsonFilter})
		case "vb":
			filter = append(filter, bson.E{Key: "bool_value", Value: value})
		case "vs":
			filter = append(filter, bson.E{Key: "string_value", Value: value})
		case "vd":
			filter = append(filter, bson.E{Key: "data_value", Value: value})
		case "from":
			filter = append(filter, bson.E{Key: senmlOrder, Value: bson.M{"$gte": value}})
		case "to":
			filter = append(filter, bson.E{Key: senmlOrder, Value: bson.M{"$lt": value}})
		}
	}

	return filter
}
