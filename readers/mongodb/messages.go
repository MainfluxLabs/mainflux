// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mongodb

import (
	"context"
	"encoding/json"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Collection for SenML messages
const (
	defCollection  = "messages"
	jsonCollection = "json"
	jsonOrder      = "created"
	senmlOrder     = "time"
	// noLimit is used to indicate that there is no limit for the number of results.
	noLimit = 0
)

var _ readers.MessageRepository = (*mongoRepository)(nil)

type mongoRepository struct {
	db *mongo.Database
}

// New returns new MongoDB reader.
func New(db *mongo.Database) readers.MessageRepository {
	return mongoRepository{
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
		Format:    jsonCollection,
	}
}

func senMLPageMetaToPageMeta(sm readers.SenMLPageMetadata) PageMetadata {
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
		Format:      defCollection,
	}
}

func fmtCondition(rpm PageMetadata) bson.D {
	filter := bson.D{}

	var query map[string]interface{}
	meta, err := json.Marshal(rpm)
	if err != nil {
		return filter
	}
	json.Unmarshal(meta, &query)

	timeField := senmlOrder
	if rpm.Format == jsonCollection {
		timeField = jsonOrder
	}

	for name, value := range query {
		switch name {
		case "profile", "subtopic", "publisher", "name", "protocol":
			filter = append(filter, bson.E{Key: name, Value: value})
		case "v":
			var bsonFilter interface{} = value
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
			filter = append(filter, bson.E{Key: timeField, Value: bson.M{"$gte": value}})
		case "to":
			filter = append(filter, bson.E{Key: timeField, Value: bson.M{"$lt": value}})
		}
	}

	return filter
}

func (repo mongoRepository) readAllJSON(rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	col := repo.db.Collection(jsonCollection)

	pageMetadata := jsonPageMetaToPageMeta(rpm)
	filter := fmtCondition(pageMetadata)

	sortMap := bson.D{{Key: "created", Value: -1}}

	findOpts := options.Find().SetSort(sortMap)
	if rpm.Limit != noLimit {
		findOpts.SetLimit(int64(rpm.Limit)).SetSkip(int64(rpm.Offset))
	}

	cursor, err := col.Find(context.Background(), filter, findOpts)
	if err != nil {
		return readers.JSONMessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	defer cursor.Close(context.Background())

	var messages []readers.Message
	for cursor.Next(context.Background()) {
		var m map[string]interface{}
		if err := cursor.Decode(&m); err != nil {
			return readers.JSONMessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
		}
		messages = append(messages, m)
	}

	total, err := col.CountDocuments(context.Background(), filter)
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

func (repo mongoRepository) readAllSenML(rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	col := repo.db.Collection(defCollection)

	pageMetadata := senMLPageMetaToPageMeta(rpm)
	filter := fmtCondition(pageMetadata)

	sortMap := bson.D{{Key: "time", Value: -1}}

	findOpts := options.Find().SetSort(sortMap)
	if rpm.Limit != noLimit {
		findOpts.SetLimit(int64(rpm.Limit)).SetSkip(int64(rpm.Offset))
	}

	cursor, err := col.Find(context.Background(), filter, findOpts)
	if err != nil {
		return readers.SenMLMessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	defer cursor.Close(context.Background())

	var messages []readers.Message
	for cursor.Next(context.Background()) {
		var m senml.Message
		if err := cursor.Decode(&m); err != nil {
			return readers.SenMLMessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
		}
		messages = append(messages, m)
	}

	total, err := col.CountDocuments(context.Background(), filter)
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

func (repo mongoRepository) ListJSONMessages(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	return repo.readAllJSON(rpm)
}

func (repo mongoRepository) ListSenMLMessages(ctx context.Context, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	return repo.readAllSenML(rpm)
}

func (repo mongoRepository) BackupJSONMessages(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	backupMetadata := rpm
	backupMetadata.Limit = noLimit
	backupMetadata.Offset = 0
	return repo.readAllJSON(backupMetadata)
}

func (repo mongoRepository) BackupSenMLMessages(ctx context.Context, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	backupMetadata := rpm
	backupMetadata.Limit = noLimit
	backupMetadata.Offset = 0
	return repo.readAllSenML(backupMetadata)
}

func (repo mongoRepository) RestoreJSONMessages(ctx context.Context, messages ...readers.Message) error {
	if len(messages) == 0 {
		return nil
	}

	coll := repo.db.Collection(jsonCollection)
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

func (repo mongoRepository) RestoreSenMLMessages(ctx context.Context, messages ...readers.Message) error {
	if len(messages) == 0 {
		return nil
	}

	coll := repo.db.Collection(defCollection)
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

func (repo mongoRepository) DeleteJSONMessages(ctx context.Context, rpm readers.JSONPageMetadata) error {
	coll := repo.db.Collection(jsonCollection)

	pageMetadata := jsonPageMetaToPageMeta(rpm)
	filter := fmtCondition(pageMetadata)

	if len(filter) == 0 {
		return errors.Wrap(errors.ErrDeleteMessages, errors.New("no delete criteria specified"))
	}

	_, err := coll.DeleteMany(ctx, filter)
	if err != nil {
		return errors.Wrap(errors.ErrDeleteMessages, err)
	}

	return nil
}

func (repo mongoRepository) DeleteSenMLMessages(ctx context.Context, rpm readers.SenMLPageMetadata) error {
	coll := repo.db.Collection(defCollection)

	pageMetadata := senMLPageMetaToPageMeta(rpm)
	filter := fmtCondition(pageMetadata)

	if len(filter) == 0 {
		return errors.Wrap(errors.ErrDeleteMessages, errors.New("no delete criteria specified"))
	}

	_, err := coll.DeleteMany(ctx, filter)
	if err != nil {
		return errors.Wrap(errors.ErrDeleteMessages, err)
	}

	return nil
}
