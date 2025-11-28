// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mongodb_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	mwriter "github.com/MainfluxLabs/mainflux/consumers/writers/mongodb"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/readers"
	mreader "github.com/MainfluxLabs/mainflux/readers/mongodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestListJSONMessages(t *testing.T) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	reader := mreader.NewJSONRepository(db)
	writer := mwriter.New(db)

	id1, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	pyd := map[string]any{
		"field_1": 123.0,
		"field_2": "value",
		"field_3": false,
		"field_4": 12.344,
		"field_5": map[string]any{
			"field_1": "value",
			"field_2": 42.0,
		},
	}
	payload, err := json.Marshal(pyd)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	m := protomfx.Message{
		Publisher:   id1,
		Subtopic:    subtopic,
		Protocol:    coapProt,
		Payload:     payload,
		ContentType: jsonCT,
	}

	var messages []protomfx.Message
	created := time.Now().Unix()
	for i := 0; i < msgsNum; i++ {
		msg := m
		msg.Created = created + int64(i)
		messages = append(messages, msg)
	}

	id2, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pyd2 := map[string]any{
		"field_1":     "other_value",
		"false_value": false,
		"field_pi":    3.14159265,
	}
	payload2, err := json.Marshal(pyd2)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	m2 := protomfx.Message{
		Publisher:   id2,
		Subtopic:    subtopic,
		Protocol:    udpProt,
		Payload:     payload2,
		ContentType: jsonCT,
	}

	for i := 0; i < 3; i++ {
		msg := m2
		msg.Created = created + int64(i)
		if i%2 == 0 {
			msg.Protocol = httpProt
		}
		messages = append(messages, msg)
	}

	var msgs, httpMsgs []map[string]any
	for _, m := range messages {
		err := writer.Consume(m)
		assert.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))

		mapped := toMap(m)
		msgs = append(msgs, mapped)
		if m.Protocol == httpProt {
			httpMsgs = append(httpMsgs, mapped)
		}
	}

	cases := map[string]struct {
		pageMeta readers.JSONPageMetadata
		page     readers.JSONMessagesPage
	}{
		"read all messages": {
			pageMeta: readers.JSONPageMetadata{
				Limit: noLimit,
			},
			page: readers.JSONMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(msgs)),
					Messages: fromJSON(msgs),
				},
			},
		},
		"read messages with protocol": {
			pageMeta: readers.JSONPageMetadata{
				Limit:    noLimit,
				Protocol: httpProt,
			},
			page: readers.JSONMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(httpMsgs)),
					Messages: fromJSON(httpMsgs),
				},
			},
		},
	}

	for desc, tc := range cases {
		result, err := reader.Retrieve(context.Background(), tc.pageMeta)
		require.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))

		for i := 0; i < len(result.Messages); i++ {
			if msgMap, ok := result.Messages[i].(map[string]any); ok {
				result.Messages[i] = cleanMap(msgMap)
			}
		}
		assert.ElementsMatch(t, tc.page.Messages, result.Messages, fmt.Sprintf("%s: expected %v got %v", desc, tc.page.Messages, result.Messages))
		assert.Equal(t, tc.page.Total, result.Total, fmt.Sprintf("%s: expected %v got %v", desc, tc.page.Total, result.Total))
	}
}

func fromJSON(msg []map[string]any) []readers.Message {
	var ret []readers.Message
	for _, m := range msg {
		ret = append(ret, m)
	}
	return ret
}

func toMap(msg protomfx.Message) map[string]any {
	return map[string]any{
		"created":   msg.Created,
		"subtopic":  msg.Subtopic,
		"publisher": msg.Publisher,
		"protocol":  msg.Protocol,
		"payload":   msg.Payload,
	}
}

func cleanMap(msg map[string]any) map[string]any {
	delete(msg, "_id")

	if bin, ok := msg["payload"].(primitive.Binary); ok {
		msg["payload"] = bin.Data
	}

	return msg
}
