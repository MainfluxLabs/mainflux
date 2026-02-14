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
	"github.com/MainfluxLabs/mainflux/readers"
	preader "github.com/MainfluxLabs/mainflux/readers/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		err := writer.Consume(m)
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
			Publisher:   pubID,
			Subtopic:    subtopic,
			Protocol:    mqttProt,
			Payload:     payload,
			ContentType: jsonCT,
			Created:     now + int64(i),
		}
		err := writer.Consume(msg)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	cases := map[string]struct {
		pageMeta readers.JSONPageMetadata
	}{
		"max aggregation": {
			pageMeta: readers.JSONPageMetadata{
				Limit:       noLimit,
				Publisher:   pubID,
				AggType:     maxAgg,
				AggInterval: "hour",
				AggValue:    1,
				AggFields:   []string{"temperature"},
			},
		},
		"avg aggregation": {
			pageMeta: readers.JSONPageMetadata{
				Limit:       noLimit,
				Publisher:   pubID,
				AggType:     avgAgg,
				AggInterval: "hour",
				AggValue:    1,
				AggFields:   []string{"humidity"},
			},
		},
		"count aggregation": {
			pageMeta: readers.JSONPageMetadata{
				Limit:       noLimit,
				Publisher:   pubID,
				AggType:     countAgg,
				AggInterval: "hour",
				AggValue:    1,
				AggFields:   []string{"temperature"},
			},
		},
		"nested field aggregation": {
			pageMeta: readers.JSONPageMetadata{
				Limit:       noLimit,
				Publisher:   pubID,
				AggType:     maxAgg,
				AggInterval: "hour",
				AggValue:    1,
				AggFields:   []string{"nested.value"},
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
		err := writer.Consume(m)
		assert.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))
	}

	cases := map[string]struct {
		pageMeta      readers.JSONPageMetadata
		expectedCount uint64
		description   string
	}{
		"delete JSON messages with publisher id1": {
			pageMeta: readers.JSONPageMetadata{
				Publisher: id1,
				From:      0,
				To:        int64(created + int64(msgsNum)),
			},
			expectedCount: uint64(msgsNum),
			description:   "should delete JSON messages from specific publisher id1",
		},
		"delete JSON messages with publisher id2": {
			pageMeta: readers.JSONPageMetadata{
				Publisher: id2,
				From:      0,
				To:        int64(created + int64(msgsNum)),
			},
			expectedCount: uint64(msgsNum),
			description:   "should delete JSON messages from specific publisher id2",
		},
		"delete JSON messages with protocol": {
			pageMeta: readers.JSONPageMetadata{
				Publisher: id2,
				Protocol:  httpProt,
				From:      0,
				To:        int64(created + int64(msgsNum)),
			},
			expectedCount: uint64(httpMsgCount),
			description:   "should delete JSON messages with HTTP protocol",
		},
		"delete JSON messages with subtopic": {
			pageMeta: readers.JSONPageMetadata{
				Publisher: id1,
				Subtopic:  subtopic,
				From:      0,
				To:        int64(created + int64(msgsNum)),
			},
			expectedCount: uint64(msgsNum),
			description:   "should delete JSON messages with specific subtopic",
		},
		"delete JSON messages with time range": {
			pageMeta: readers.JSONPageMetadata{
				Publisher: id1,
				From:      int64(created + 20),
				To:        int64(created + 50),
			},
			expectedCount: 31,
			description:   "should delete JSON messages within time range",
		},
	}

	for desc, tc := range cases {
		_ = reader.Remove(context.Background(), readers.JSONPageMetadata{
			Publisher: id1,
			From:      0,
			To:        int64(created + int64(msgsNum)),
		})

		_ = reader.Remove(context.Background(), readers.JSONPageMetadata{
			Publisher: id2,
			From:      0,
			To:        int64(created + int64(msgsNum)),
		})

		for _, m := range messages {
			err := writer.Consume(m)
			require.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))
		}

		beforePage, err := reader.Retrieve(context.Background(), readers.JSONPageMetadata{
			Publisher: tc.pageMeta.Publisher,
			Subtopic:  tc.pageMeta.Subtopic,
			Protocol:  tc.pageMeta.Protocol,
			From:      tc.pageMeta.From,
			To:        tc.pageMeta.To,
			Limit:     noLimit,
		})
		require.Nil(t, err)
		beforeCount := beforePage.Total

		err = reader.Remove(context.Background(), tc.pageMeta)
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))

		afterPage, err := reader.Retrieve(context.Background(), readers.JSONPageMetadata{
			Publisher: tc.pageMeta.Publisher,
			Subtopic:  tc.pageMeta.Subtopic,
			Protocol:  tc.pageMeta.Protocol,
			From:      tc.pageMeta.From,
			To:        tc.pageMeta.To,
			Limit:     noLimit,
		})
		require.Nil(t, err)
		afterCount := afterPage.Total

		actualDeleted := beforeCount - afterCount
		assert.Equal(t, tc.expectedCount, actualDeleted, fmt.Sprintf("%s: %s - expected %d deleted, got %d", desc, tc.description, tc.expectedCount, actualDeleted))
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
		"publisher": msg.Publisher,
		"protocol":  msg.Protocol,
		"payload":   payload,
	}, nil
}
