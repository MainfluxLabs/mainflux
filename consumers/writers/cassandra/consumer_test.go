// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cassandra_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/MainfluxLabs/mainflux/consumers/writers/cassandra"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	keyspace    = "mainflux"
	msgsNum     = 42
	valueFields = 5
	subtopic    = "topic"
)

var addr = "localhost"

var (
	v       float64 = 5
	stringV         = "value"
	boolV           = true
	dataV           = "base64"
	sum     float64 = 42
)

func TestSaveSenML(t *testing.T) {
	session, err := cassandra.Connect(cassandra.DBConfig{
		Hosts:    []string{addr},
		Keyspace: keyspace,
	})
	require.Nil(t, err, fmt.Sprintf("failed to connect to Cassandra: %s", err))
	repo := cassandra.New(session)
	now := time.Now().Unix()
	msg := senml.Message{
		Channel:   "1",
		Publisher: "1",
		Protocol:  "mqtt",
	}
	var msgs []senml.Message

	for i := 0; i < msgsNum; i++ {
		// Mix possible values as well as value sum.
		count := i % valueFields
		switch count {
		case 0:
			msg.Subtopic = subtopic
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

		msg.Time = float64(now + int64(i))
		msgs = append(msgs, msg)
	}

	err = repo.Consume(msgs)
	assert.Nil(t, err, fmt.Sprintf("expected no error, got %s", err))
}

func TestSaveJSON(t *testing.T) {
	session, err := cassandra.Connect(cassandra.DBConfig{
		Hosts:    []string{addr},
		Keyspace: keyspace,
	})
	require.Nil(t, err, fmt.Sprintf("failed to connect to Cassandra: %s", err))
	repo := cassandra.New(session)
	chid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pubid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	msg := json.Message{
		Channel:   chid.String(),
		Publisher: pubid.String(),
		Created:   time.Now().Unix(),
		Subtopic:  "subtopic/format/some_json",
		Protocol:  "mqtt",
		Payload: map[string]interface{}{
			"field_1": 123,
			"field_2": "value",
			"field_3": false,
			"field_4": 12.344,
			"field_5": map[string]interface{}{
				"field_1": "value",
				"field_2": 42,
			},
		},
	}

	now := time.Now().Unix()
	msgs := json.Messages{
		Format: "some_json",
	}

	for i := 0; i < msgsNum; i++ {
		msg.Created = now + int64(i)
		msgs.Data = append(msgs.Data, msg)
	}

	err = repo.Consume(msgs)
	assert.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))
}
