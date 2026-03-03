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
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/readers"
	preader "github.com/MainfluxLabs/mainflux/readers/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	subtopic    = "subtopic"
	msgsNum     = 101
	noLimit     = 0
	valueFields = 5
	mqttProt    = "mqtt"
	httpProt    = "http"
	coapProt    = "coap"
	udpProt     = "udp"
	msgName     = "temperature"
	jsonFormat  = "json"
	jsonCT      = "application/json"
	jsonTable   = "json"
	senmlFormat = "messages"
	minAgg      = "min"
	maxAgg      = "max"
	countAgg    = "count"
	avgAgg      = "avg"
)

var (
	v   float64 = 5
	vs          = "value"
	vb          = true
	vd          = "dataValue"
	sum float64 = 42

	idProvider = uuid.New()
)

func TestListSenMLMessages(t *testing.T) {
	reader := preader.NewSenMLRepository(db)
	writer := pwriter.New(db)

	pubID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pubID2, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	m := senml.Message{
		Publisher: pubID,
		Protocol:  mqttProt,
	}

	messages := []senml.Message{}
	valueMsgs := []senml.Message{}
	boolMsgs := []senml.Message{}
	stringMsgs := []senml.Message{}
	dataMsgs := []senml.Message{}
	queryMsgs := []senml.Message{}

	now := int64(time.Now().Unix())
	for i := 0; i < msgsNum; i++ {
		// Mix possible values as well as value sum.
		msg := m
		msg.Time = now - int64(i)

		count := i % valueFields
		switch count {
		case 0:
			msg.Value = &v
			valueMsgs = append(valueMsgs, msg)
		case 1:
			msg.BoolValue = &vb
			boolMsgs = append(boolMsgs, msg)
		case 2:
			msg.StringValue = &vs
			stringMsgs = append(stringMsgs, msg)
		case 3:
			msg.DataValue = &vd
			dataMsgs = append(dataMsgs, msg)
		case 4:
			msg.Sum = &sum
			msg.Subtopic = subtopic
			msg.Protocol = httpProt
			msg.Publisher = pubID2
			msg.Name = msgName
			queryMsgs = append(queryMsgs, msg)
		}

		messages = append(messages, msg)
	}

	for _, m := range messages {
		pyd := senml.Message{
			Name:        m.Name,
			Unit:        m.Unit,
			Time:        m.Time,
			Value:       m.Value,
			BoolValue:   m.BoolValue,
			StringValue: m.StringValue,
			DataValue:   m.DataValue,
			Sum:         m.Sum,
		}

		payload, err := json.Marshal(pyd)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		pm := protomfx.Message{
			Publisher:   m.Publisher,
			Subtopic:    m.Subtopic,
			Protocol:    m.Protocol,
			ContentType: senml.JSON,
			Payload:     payload,
		}

		err = writer.Consume(pm)
		assert.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))
	}

	// Since messages are not saved in natural order,
	// cases that return subset of messages are only
	// checking data result set size, but not content.
	cases := map[string]struct {
		pageMeta readers.SenMLPageMetadata
		page     readers.SenMLMessagesPage
	}{
		"read all messages": {
			pageMeta: readers.SenMLPageMetadata{
				Limit: noLimit,
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    msgsNum,
					Messages: fromSenml(messages),
				},
			},
		},
		"read messages with non-existent subtopic": {
			pageMeta: readers.SenMLPageMetadata{
				Limit:    noLimit,
				Subtopic: "not-present",
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Messages: []readers.Message{},
				},
			},
		},
		"read messages with subtopic": {
			pageMeta: readers.SenMLPageMetadata{
				Limit:    noLimit,
				Subtopic: subtopic,
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(queryMsgs)),
					Messages: fromSenml(queryMsgs),
				},
			},
		},
		"read messages with publisher": {
			pageMeta: readers.SenMLPageMetadata{
				Limit:     noLimit,
				Publisher: pubID2,
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(queryMsgs)),
					Messages: fromSenml(queryMsgs),
				},
			},
		},
		"read messages with protocol": {
			pageMeta: readers.SenMLPageMetadata{
				Limit:    noLimit,
				Protocol: httpProt,
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(queryMsgs)),
					Messages: fromSenml(queryMsgs),
				},
			},
		},
		"read messages with name": {
			pageMeta: readers.SenMLPageMetadata{
				Limit: noLimit,
				Name:  msgName,
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(queryMsgs)),
					Messages: fromSenml(queryMsgs),
				},
			},
		},
		"read messages with value": {
			pageMeta: readers.SenMLPageMetadata{
				Limit: noLimit,
				Value: v,
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(valueMsgs)),
					Messages: fromSenml(valueMsgs),
				},
			},
		},
		"read messages with value and equal comparator": {
			pageMeta: readers.SenMLPageMetadata{
				Limit:      noLimit,
				Value:      v,
				Comparator: readers.EqualKey,
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(valueMsgs)),
					Messages: fromSenml(valueMsgs),
				},
			},
		},
		"read messages with value and lower-than comparator": {
			pageMeta: readers.SenMLPageMetadata{
				Limit:      noLimit,
				Value:      v + 1,
				Comparator: readers.LowerThanKey,
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(valueMsgs)),
					Messages: fromSenml(valueMsgs),
				},
			},
		},
		"read messages with value and lower-than-or-equal comparator": {
			pageMeta: readers.SenMLPageMetadata{
				Limit:      noLimit,
				Value:      v + 1,
				Comparator: readers.LowerThanEqualKey,
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(valueMsgs)),
					Messages: fromSenml(valueMsgs),
				},
			},
		},
		"read messages with value and greater-than comparator": {
			pageMeta: readers.SenMLPageMetadata{
				Limit:      noLimit,
				Value:      v - 1,
				Comparator: readers.GreaterThanKey,
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(valueMsgs)),
					Messages: fromSenml(valueMsgs),
				},
			},
		},
		"read messages with value and greater-than-or-equal comparator": {
			pageMeta: readers.SenMLPageMetadata{
				Limit:      noLimit,
				Value:      v - 1,
				Comparator: readers.GreaterThanEqualKey,
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(valueMsgs)),
					Messages: fromSenml(valueMsgs),
				},
			},
		},
		"read messages with boolean value": {
			pageMeta: readers.SenMLPageMetadata{
				Limit:     noLimit,
				BoolValue: vb,
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(boolMsgs)),
					Messages: fromSenml(boolMsgs),
				},
			},
		},
		"read messages with string value": {
			pageMeta: readers.SenMLPageMetadata{
				Limit:       noLimit,
				StringValue: vs,
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(stringMsgs)),
					Messages: fromSenml(stringMsgs),
				},
			},
		},
		"read messages with data value": {
			pageMeta: readers.SenMLPageMetadata{
				Limit:     noLimit,
				DataValue: vd,
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(dataMsgs)),
					Messages: fromSenml(dataMsgs),
				},
			},
		},
		"read messages with from": {
			pageMeta: readers.SenMLPageMetadata{
				Limit: noLimit,
				From:  messages[20].Time,
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(messages[0:21])),
					Messages: fromSenml(messages[0:21]),
				},
			},
		},
		"read messages with to": {
			pageMeta: readers.SenMLPageMetadata{
				Limit: noLimit,
				To:    messages[20].Time,
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(messages[20:])),
					Messages: fromSenml(messages[20:]),
				},
			},
		},
		"read messages with from/to": {
			pageMeta: readers.SenMLPageMetadata{
				Limit: noLimit,
				From:  messages[5].Time,
				To:    messages[0].Time,
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    6,
					Messages: fromSenml(messages[0:6]),
				},
			},
		},
		"count aggregation": {
			pageMeta: readers.SenMLPageMetadata{
				Limit:   noLimit,
				AggType: countAgg,
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    msgsNum,
					Messages: fromSenml(messages),
				},
			},
		},
		"min aggregation with name filter": {
			pageMeta: readers.SenMLPageMetadata{
				Limit:   noLimit,
				Name:    msgName,
				AggType: minAgg,
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(queryMsgs)),
					Messages: fromSenml(queryMsgs),
				},
			},
		},
		"max aggregation with name filter": {
			pageMeta: readers.SenMLPageMetadata{
				Limit:   noLimit,
				Name:    msgName,
				AggType: maxAgg,
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(queryMsgs)),
					Messages: fromSenml(queryMsgs),
				},
			},
		},
		"avg aggregation on sum field": {
			pageMeta: readers.SenMLPageMetadata{
				Limit:     noLimit,
				Name:      msgName,
				AggType:   avgAgg,
				AggFields: []string{"sum"},
			},
			page: readers.SenMLMessagesPage{
				MessagesPage: readers.MessagesPage{
					Total:    uint64(len(queryMsgs)),
					Messages: fromSenml(queryMsgs),
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

func TestSenMLAggregation(t *testing.T) {
	reader := preader.NewSenMLRepository(db)
	writer := pwriter.New(db)

	pubID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	now := time.Now().Unix()
	val := 10.0
	for i := 0; i < 10; i++ {
		msg := senml.Message{
			Name:  "sensor",
			Value: &val,
			Time:  now + int64(i),
		}
		payload, err := json.Marshal(msg)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		err = writer.Consume(protomfx.Message{
			Publisher:   pubID,
			Protocol:    mqttProt,
			ContentType: senml.JSON,
			Payload:     payload,
		})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	cases := map[string]readers.SenMLPageMetadata{
		"max aggregation": {
			Limit:       noLimit,
			Publisher:   pubID,
			AggType:     maxAgg,
			AggInterval: "hour",
			AggValue:    1,
			AggFields:   []string{"value"},
		},
		"min aggregation": {
			Limit:       noLimit,
			Publisher:   pubID,
			AggType:     minAgg,
			AggInterval: "hour",
			AggValue:    1,
			AggFields:   []string{"value"},
		},
		"avg aggregation": {
			Limit:       noLimit,
			Publisher:   pubID,
			AggType:     avgAgg,
			AggInterval: "hour",
			AggValue:    1,
			AggFields:   []string{"value"},
		},
		"count aggregation": {
			Limit:       noLimit,
			Publisher:   pubID,
			AggType:     countAgg,
			AggInterval: "hour",
			AggValue:    1,
			AggFields:   []string{"value"},
		},
	}

	for desc, pm := range cases {
		result, err := reader.Retrieve(context.Background(), pm)
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))
		assert.NotEmpty(t, result.Messages, fmt.Sprintf("%s: expected non-empty messages", desc))
		assert.GreaterOrEqual(t, result.Total, uint64(1), fmt.Sprintf("%s: expected total >= 1", desc))
	}
}

func TestDeleteSenMLMessages(t *testing.T) {
	reader := preader.NewSenMLRepository(db)
	writer := pwriter.New(db)

	pubID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pubID2, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	m := senml.Message{
		Publisher: pubID,
		Protocol:  mqttProt,
	}

	messages := []senml.Message{}
	queryMsgs := []senml.Message{}

	now := int64(time.Now().Unix())
	for i := 0; i < msgsNum; i++ {
		msg := m
		msg.Time = now - int64(i)

		count := i % valueFields
		switch count {
		case 0:
			msg.Value = &v
		case 1:
			msg.BoolValue = &vb
		case 2:
			msg.StringValue = &vs
		case 3:
			msg.DataValue = &vd
		case 4:
			msg.Sum = &sum
			msg.Subtopic = subtopic
			msg.Protocol = httpProt
			msg.Publisher = pubID2
			msg.Name = msgName
			queryMsgs = append(queryMsgs, msg)
		}

		messages = append(messages, msg)
	}

	for _, m := range messages {
		pyd := senml.Message{
			Name:        m.Name,
			Unit:        m.Unit,
			Time:        m.Time,
			Value:       m.Value,
			BoolValue:   m.BoolValue,
			StringValue: m.StringValue,
			DataValue:   m.DataValue,
			Sum:         m.Sum,
		}

		payload, err := json.Marshal(pyd)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		pm := protomfx.Message{
			Publisher:   m.Publisher,
			Subtopic:    m.Subtopic,
			Protocol:    m.Protocol,
			ContentType: senml.JSON,
			Payload:     payload,
		}

		err = writer.Consume(pm)
		assert.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))
	}

	cases := map[string]struct {
		pageMeta      readers.SenMLPageMetadata
		expectedCount uint64
		description   string
	}{
		"delete senml messages with subtopic": {
			pageMeta: readers.SenMLPageMetadata{
				Publisher: pubID2,
				Subtopic:  subtopic,
				From:      0,
				To:        now + 1,
			},
			expectedCount: uint64(len(queryMsgs)),
			description:   "should delete messages with specific subtopic",
		},
		"delete senml messages with protocol": {
			pageMeta: readers.SenMLPageMetadata{
				Publisher: pubID2,
				Protocol:  httpProt,
				From:      0,
				To:        now + 1,
			},
			expectedCount: uint64(len(queryMsgs)),
			description:   "should delete messages with specific protocol",
		},
		"delete senml messages with time range from": {
			pageMeta: readers.SenMLPageMetadata{
				Publisher: pubID,
				From:      messages[20].Time,
				To:        now + 1,
			},
			expectedCount: 17,
			description:   "should delete messages from specific time",
		},
		"delete senml messages with time range to": {
			pageMeta: readers.SenMLPageMetadata{
				Publisher: pubID,
				From:      0,
				To:        messages[20].Time,
			},
			expectedCount: 65,
			description:   "should delete messages to specific time",
		},
		"delete senml messages with time range from/to": {
			pageMeta: readers.SenMLPageMetadata{
				Publisher: pubID,
				From:      messages[50].Time,
				To:        messages[20].Time,
			},
			expectedCount: 25,
			description:   "should delete messages within time range",
		},
		"delete all senml messages for publisher": {
			pageMeta: readers.SenMLPageMetadata{
				Publisher: pubID,
				From:      0,
				To:        now + 1,
			},
			expectedCount: uint64(msgsNum - len(queryMsgs)),
			description:   "should delete all messages for specific publisher",
		},
	}

	for desc, tc := range cases {
		_ = reader.Remove(context.Background(), readers.SenMLPageMetadata{
			Publisher: pubID,
			From:      0,
			To:        now,
		})
		_ = reader.Remove(context.Background(), readers.SenMLPageMetadata{
			Publisher: pubID2,
			From:      0,
			To:        now,
		})

		for _, m := range messages {
			pyd := senml.Message{
				Name:        m.Name,
				Unit:        m.Unit,
				Time:        m.Time,
				Value:       m.Value,
				BoolValue:   m.BoolValue,
				StringValue: m.StringValue,
				DataValue:   m.DataValue,
				Sum:         m.Sum,
			}

			payload, err := json.Marshal(pyd)
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

			pm := protomfx.Message{
				Publisher:   m.Publisher,
				Subtopic:    m.Subtopic,
				Protocol:    m.Protocol,
				ContentType: senml.JSON,
				Payload:     payload,
			}

			err = writer.Consume(pm)
			require.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))
		}

		beforePage, err := reader.Retrieve(context.Background(), readers.SenMLPageMetadata{
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

		afterPage, err := reader.Retrieve(context.Background(), readers.SenMLPageMetadata{
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

func fromSenml(msg []senml.Message) []readers.Message {
	var ret []readers.Message
	for _, m := range msg {
		ret = append(ret, m)
	}
	return ret
}
