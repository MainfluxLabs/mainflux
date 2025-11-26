// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/consumers/writers/postgres"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gofrs/uuid"
)

const (
	msgsNum     = 42
	valueFields = 5
	mqttProt    = "mqtt"
	subtopic    = "topic"
)

var (
	v       float64 = 5
	stringV         = "value"
	boolV           = true
	dataV           = "base64"
	sum     float64 = 42
)

func TestSaveSenML(t *testing.T) {
	repo := postgres.New(db)

	pubid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	now := time.Now().Unix()

	for i := 0; i < msgsNum; i++ {
		msg := senml.Message{
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
}

func TestSaveJSON(t *testing.T) {
	repo := postgres.New(db)

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
	assert.Nil(t, err, fmt.Sprintf("expected no error on Consume, got %s", err))
}
