// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package timescale_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/consumers/writers/timescale"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gofrs/uuid"
)

const (
	msgsNum     = 42
	valueFields = 5
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
	repo := timescale.New(db)

	msg := senml.Message{}

	pubid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	msg.Publisher = pubid.String()

	now := time.Now().Unix()
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
	assert.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))
}

func TestSaveJSON(t *testing.T) {
	repo := timescale.New(db)

	pubid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	msg := json.Message{
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
