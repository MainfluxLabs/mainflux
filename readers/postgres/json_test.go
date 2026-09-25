// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	pwriter "github.com/MainfluxLabs/mainflux/consumers/writers/postgres"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	mfreaders "github.com/MainfluxLabs/mainflux/pkg/readers"
	"github.com/MainfluxLabs/mainflux/readers"
	preader "github.com/MainfluxLabs/mainflux/readers/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fromIndex = 20
	toIndex   = 50
)

func TestListJSONMessages(t *testing.T) {
	reader := preader.NewJSONRepository(db)
	writer := pwriter.New(db)

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
		ThingId:     id1,
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
		ThingId:     id2,
		Subtopic:    subtopic,
		Protocol:    udpProt,
		Payload:     payload2,
		ContentType: jsonCT,
	}

	for i := 0; i < msgsNum; i++ {
		msg := m2
		msg.Created = created + int64(i)
		if i%2 == 0 {
			msg.Protocol = httpProt
		}

		messages = append(messages, msg)
	}

	var msgs, httpMsgs []map[string]any
	for _, m := range messages {
		err := writer.ConsumeMessage(subject, m)
		assert.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))

		mapped, err := toMap(m)
		require.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))

		if m.Protocol == httpProt {
			httpMsgs = append(httpMsgs, mapped)
		}
		msgs = append(msgs, mapped)
	}

	cases := map[string]struct {
		pageMeta readers.JSONPageMetadata
		page     readers.JSONMessagesPage
	}{
		"read all messages": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: readers.MessagesPageMetadata{
					Limit: noLimit,
				},
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
				MessagesPageMetadata: readers.MessagesPageMetadata{
					Limit:    noLimit,
					Protocol: httpProt,
				},
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
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))
		assert.ElementsMatch(t, tc.page.Messages, result.Messages, fmt.Sprintf("%s: expected %v got %v", desc, tc.page.Messages, result.Messages))
		assert.Equal(t, tc.page.Total, result.Total, fmt.Sprintf("%s: expected %v got %v", desc, tc.page.Total, result.Total))
	}
}

func TestJSONAggregation(t *testing.T) {
	reader := preader.NewJSONRepository(db)
	writer := pwriter.New(db)

	pubID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	pyd := map[string]any{
		"temperature": 25.5,
		"humidity":    60.0,
		"nested": map[string]any{
			"value": 42.0,
		},
	}
	payload, err := json.Marshal(pyd)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	now := time.Now().Unix()
	for i := 0; i < 10; i++ {
		msg := protomfx.Message{
			ThingId:     pubID,
			Subtopic:    subtopic,
			Protocol:    mqttProt,
			Payload:     payload,
			ContentType: jsonCT,
			Created:     now + int64(i),
		}
		err := writer.ConsumeMessage(subject, msg)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	cases := map[string]struct {
		pageMeta readers.JSONPageMetadata
	}{
		"max aggregation": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: readers.MessagesPageMetadata{
					Limit:       noLimit,
					Publisher:   pubID,
					AggType:     maxAgg,
					AggInterval: "hour",
					AggValue:    1,
					AggFields:   []string{"temperature"},
				},
			},
		},
		"avg aggregation": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: readers.MessagesPageMetadata{
					Limit:       noLimit,
					Publisher:   pubID,
					AggType:     avgAgg,
					AggInterval: "hour",
					AggValue:    1,
					AggFields:   []string{"humidity"},
				},
			},
		},
		"count aggregation": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: readers.MessagesPageMetadata{
					Limit:       noLimit,
					Publisher:   pubID,
					AggType:     countAgg,
					AggInterval: "hour",
					AggValue:    1,
					AggFields:   []string{"temperature"},
				},
			},
		},
		"nested field aggregation": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: readers.MessagesPageMetadata{
					Limit:       noLimit,
					Publisher:   pubID,
					AggType:     maxAgg,
					AggInterval: "hour",
					AggValue:    1,
					AggFields:   []string{"nested.value"},
				},
			},
		},
	}

	for desc, tc := range cases {
		result, err := reader.Retrieve(context.Background(), tc.pageMeta)
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))
		assert.NotEmpty(t, result.Messages, fmt.Sprintf("%s: expected non-empty messages", desc))
		assert.GreaterOrEqual(t, result.Total, uint64(1), fmt.Sprintf("%s: expected total >= 1", desc))
	}
}

func TestDeleteJSONMessages(t *testing.T) {
	reader := preader.NewJSONRepository(db)
	writer := pwriter.New(db)

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
		ThingId:     id1,
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
		ThingId:     id2,
		Subtopic:    subtopic,
		Protocol:    udpProt,
		Payload:     payload2,
		ContentType: jsonCT,
	}

	var httpMsgCount int
	for i := 0; i < msgsNum; i++ {
		msg := m2
		msg.Created = created + int64(i)
		if i%2 == 0 {
			msg.Protocol = httpProt
			httpMsgCount++
		}
		messages = append(messages, msg)
	}

	for _, m := range messages {
		err := writer.ConsumeMessage(subject, m)
		assert.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))
	}

	cases := map[string]struct {
		pageMeta      readers.JSONPageMetadata
		expectedCount uint64
		description   string
	}{
		"delete JSON messages with publisher id1": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: readers.MessagesPageMetadata{
					Publisher: id1,
					From:      0,
					To:        int64(created + int64(msgsNum)),
				},
			},
			expectedCount: uint64(msgsNum),
			description:   "should delete JSON messages from specific publisher id1",
		},
		"delete JSON messages with publisher id2": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: readers.MessagesPageMetadata{
					Publisher: id2,
					From:      0,
					To:        int64(created + int64(msgsNum)),
				},
			},
			expectedCount: uint64(msgsNum),
			description:   "should delete JSON messages from specific publisher id2",
		},
		"delete JSON messages with protocol": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: readers.MessagesPageMetadata{
					Publisher: id2,
					Protocol:  httpProt,
					From:      0,
					To:        int64(created + int64(msgsNum)),
				},
			},
			expectedCount: uint64(httpMsgCount),
			description:   "should delete JSON messages with HTTP protocol",
		},
		"delete JSON messages with subtopic": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: readers.MessagesPageMetadata{
					Publisher: id1,
					Subtopic:  subtopic,
					From:      0,
					To:        int64(created + int64(msgsNum)),
				},
			},
			expectedCount: uint64(msgsNum),
			description:   "should delete JSON messages with specific subtopic",
		},
		"delete JSON messages with time range": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: readers.MessagesPageMetadata{
					Publisher: id1,
					From:      int64(created + fromIndex),
					To:        int64(created + toIndex + 1),
				},
			},
			expectedCount: uint64(toIndex - fromIndex + 1),
			description:   "should delete JSON messages within time range",
		},
	}

	for desc, tc := range cases {
		_ = reader.Remove(context.Background(), readers.JSONPageMetadata{
			MessagesPageMetadata: readers.MessagesPageMetadata{
				Publisher: id1,
				From:      0,
				To:        int64(created + int64(msgsNum)),
			},
		})

		_ = reader.Remove(context.Background(), readers.JSONPageMetadata{
			MessagesPageMetadata: readers.MessagesPageMetadata{
				Publisher: id2,
				From:      0,
				To:        int64(created + int64(msgsNum)),
			},
		})

		for _, m := range messages {
			err := writer.ConsumeMessage(subject, m)
			require.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))
		}

		beforePage, err := reader.Retrieve(context.Background(), readers.JSONPageMetadata{
			MessagesPageMetadata: readers.MessagesPageMetadata{
				Publisher: tc.pageMeta.Publisher,
				Subtopic:  tc.pageMeta.Subtopic,
				Protocol:  tc.pageMeta.Protocol,
				From:      tc.pageMeta.From,
				To:        tc.pageMeta.To,
				Limit:     noLimit,
			},
		})
		require.Nil(t, err)
		beforeCount := beforePage.Total

		err = reader.Remove(context.Background(), tc.pageMeta)
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))

		afterPage, err := reader.Retrieve(context.Background(), readers.JSONPageMetadata{
			MessagesPageMetadata: readers.MessagesPageMetadata{
				Publisher: tc.pageMeta.Publisher,
				Subtopic:  tc.pageMeta.Subtopic,
				Protocol:  tc.pageMeta.Protocol,
				From:      tc.pageMeta.From,
				To:        tc.pageMeta.To,
				Limit:     noLimit,
			},
		})
		require.Nil(t, err)
		afterCount := afterPage.Total

		actualDeleted := beforeCount - afterCount
		assert.Equal(t, tc.expectedCount, actualDeleted, fmt.Sprintf("%s: %s - expected %d deleted, got %d", desc, tc.description, tc.expectedCount, actualDeleted))
	}
}

func TestListJSONMessagesByPayloadKeyValue(t *testing.T) {
	reader := preader.NewJSONRepository(db)
	writer := pwriter.New(db)

	pubID, err := idProvider.ID()
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
	pyd2 := map[string]any{
		"field_1":     "other_value",
		"false_value": false,
		"field_pi":    3.14159265,
		"field_array": []any{
			map[string]any{
				"field_1": "value_1",
			},
			map[string]any{
				"field_1": "value_2",
			},
		},
	}

	var msgs []map[string]any
	created := time.Now().Unix()
	for i, p := range []map[string]any{pyd, pyd2} {
		payload, err := json.Marshal(p)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		msg := protomfx.Message{
			ThingId:     pubID,
			Subtopic:    subtopic,
			Protocol:    mqttProt,
			Payload:     payload,
			ContentType: jsonCT,
			Created:     created + int64(i),
		}
		err = writer.ConsumeMessage(subject, msg)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		mapped, err := toMap(msg)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		msgs = append(msgs, mapped)
	}

	pageMeta := readers.MessagesPageMetadata{
		Limit:     noLimit,
		Publisher: pubID,
	}

	cases := map[string]struct {
		pageMeta readers.JSONPageMetadata
		page     readers.JSONMessagesPage
	}{
		"read messages with nested payload key": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: pageMeta,
				PayloadKey:           "field_5.field_2",
			},
			page: readers.JSONMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(msgs[0:1])),
					Messages: fromJSON(msgs[0:1]),
				},
			},
		},
		"read messages with payload key containing a quote": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: pageMeta,
				PayloadKey:           "it's",
			},
			page: readers.JSONMessagesPage{},
		},
		"read messages with array index payload key": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: pageMeta,
				PayloadKey:           "field_array[1].field_1",
				PayloadValue:         "value_2",
			},
			page: readers.JSONMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(msgs[1:2])),
					Messages: fromJSON(msgs[1:2]),
				},
			},
		},
		"read messages with payload key and exact value": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: pageMeta,
				PayloadKey:           "field_5.field_1",
				PayloadValue:         "value",
			},
			page: readers.JSONMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(msgs[0:1])),
					Messages: fromJSON(msgs[0:1]),
				},
			},
		},
		"read messages with payload key and value in different case": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: pageMeta,
				PayloadKey:           "field_5.field_1",
				PayloadValue:         "VALUE",
			},
			page: readers.JSONMessagesPage{},
		},
		"read messages with payload key and value prefix": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: pageMeta,
				PayloadKey:           "field_1",
				PayloadValue:         "OTHER",
				Comparator:           mfreaders.StartsWithKey,
			},
			page: readers.JSONMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(msgs[1:2])),
					Messages: fromJSON(msgs[1:2]),
				},
			},
		},
		"read messages with payload key and value substring": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: pageMeta,
				PayloadKey:           "field_1",
				PayloadValue:         "her_va",
				Comparator:           mfreaders.ContainsKey,
			},
			page: readers.JSONMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(msgs[1:2])),
					Messages: fromJSON(msgs[1:2]),
				},
			},
		},
		"read messages with payload key and underscore not used as wildcard": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: pageMeta,
				PayloadKey:           "field_1",
				PayloadValue:         "othe_",
				Comparator:           mfreaders.StartsWithKey,
			},
			page: readers.JSONMessagesPage{},
		},
		"read messages with payload key and number value": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: pageMeta,
				PayloadKey:           "field_1",
				PayloadValue:         "123",
			},
			page: readers.JSONMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(msgs[0:1])),
					Messages: fromJSON(msgs[0:1]),
				},
			},
		},
		"read messages with payload key and value at object": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: pageMeta,
				PayloadKey:           "field_5",
				PayloadValue:         "value",
				Comparator:           mfreaders.ContainsKey,
			},
			page: readers.JSONMessagesPage{},
		},
		"read messages with payload value in nested fields": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: pageMeta,
				PayloadValue:         "42",
			},
			page: readers.JSONMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(msgs[0:1])),
					Messages: fromJSON(msgs[0:1]),
				},
			},
		},
		"read messages with payload value in array element": {
			pageMeta: readers.JSONPageMetadata{
				MessagesPageMetadata: pageMeta,
				PayloadValue:         "value_2",
			},
			page: readers.JSONMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(msgs[1:2])),
					Messages: fromJSON(msgs[1:2]),
				},
			},
		},
	}

	for desc, tc := range cases {
		result, err := reader.Retrieve(context.Background(), tc.pageMeta)
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))
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

func toMap(msg protomfx.Message) (map[string]any, error) {
	var payload map[string]any
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, err
	}

	return map[string]any{
		"created":   msg.Created,
		"subtopic":  msg.Subtopic,
		"publisher": msg.ThingId,
		"protocol":  msg.Protocol,
		"payload":   payload,
	}, nil
}
