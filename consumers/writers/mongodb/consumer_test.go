// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mongodb_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MainfluxLabs/mainflux/consumers/writers/mongodb"

	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"

	log "github.com/MainfluxLabs/mainflux/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	port        string
	addr        string
	testLog, _  = log.New(os.Stdout, log.Info.String())
	testDB      = "test"
	collection  = "messages"
	msgsNum     = 100
	valueFields = 5
	subtopic    = "topic"
	mqttProt    = "mqtt"
)

var (
	v       float64 = 5
	stringV         = "value"
	boolV           = true
	dataV           = "base64"
	sum     float64 = 42
)

func TestSaveSenML(t *testing.T) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	repo := mongodb.New(db)

	now := time.Now().Unix()
	pubid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	for i := 0; i < msgsNum; i++ {
		msg := senml.Message{
			Name: "test name",
			Unit: "km",
			Time: int64(now + int64(i)),
		}
		switch i % valueFields {
		case 0:
			msg.Value = &v
		case 1:
			msg.BoolValue = &boolV
		case 2:
			msg.StringValue = &stringV
		case 3:
			msg.DataValue = &dataV
		case 4:
			msg.Sum = &sum
		}

		payload, err := json.Marshal(msg)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		pm := protomfx.Message{
			Publisher:   pubid.String(),
			Subtopic:    subtopic,
			Protocol:    mqttProt,
			Payload:     payload,
			ContentType: senml.JSON,
			Created:     now + int64(i),
		}

		err = repo.Consume(pm)
		assert.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))
	}

	count, err := db.Collection(collection).CountDocuments(context.Background(), bson.D{})
	assert.Nil(t, err, fmt.Sprintf("Querying database expected to succeed: %s.\n", err))
	assert.Equal(t, int64(msgsNum), count, fmt.Sprintf("Expected to have %d value, found %d instead.\n", msgsNum, count))
}

func TestSaveJSON(t *testing.T) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	repo := mongodb.New(db)

	pubid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	payload := map[string]any{
		"field_1": 123,
		"field_2": "value",
		"field_3": false,
		"field_4": 12.344,
		"field_5": map[string]any{
			"field_1": "value",
			"field_2": 42,
		},
	}
	payloadBytes, err := json.Marshal(payload)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	pm := protomfx.Message{
		Publisher:   pubid.String(),
		Created:     time.Now().Unix(),
		Subtopic:    subtopic,
		Protocol:    mqttProt,
		Payload:     payloadBytes,
		ContentType: messaging.JSONContentType,
	}

	err = repo.Consume(pm)
	assert.Nil(t, err, fmt.Sprintf("expected no error got %s", err))
}
