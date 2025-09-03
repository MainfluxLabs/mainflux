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

func TestListAllMessagesSenML(t *testing.T) {
	reader := preader.New(db)
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
		pageMeta readers.SenMLMetadata
		page     readers.MessagesPage
	}{
		"read all messages": {
			pageMeta: readers.SenMLMetadata{
				Limit: noLimit,
			},
			page: readers.MessagesPage{
				Total:    msgsNum,
				Messages: fromSenml(messages),
			},
		},
		"read messages with non-existent subtopic": {
			pageMeta: readers.SenMLMetadata{
				Limit:    noLimit,
				Subtopic: "not-present",
			},
			page: readers.MessagesPage{
				Messages: []readers.Message{},
			},
		},
		"read messages with subtopic": {
			pageMeta: readers.SenMLMetadata{
				Limit:    noLimit,
				Subtopic: subtopic,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read messages with publisher": {
			pageMeta: readers.SenMLMetadata{
				Limit:     noLimit,
				Publisher: pubID2,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read messages with protocol": {
			pageMeta: readers.SenMLMetadata{
				Limit:    noLimit,
				Protocol: httpProt,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read messages with name": {
			pageMeta: readers.SenMLMetadata{
				Limit: noLimit,
				Name:  msgName,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read messages with value": {
			pageMeta: readers.SenMLMetadata{
				Limit: noLimit,
				Value: v,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs),
			},
		},
		"read messages with value and equal comparator": {
			pageMeta: readers.SenMLMetadata{
				Limit:      noLimit,
				Value:      v,
				Comparator: readers.EqualKey,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs),
			},
		},
		"read messages with value and lower-than comparator": {
			pageMeta: readers.SenMLMetadata{
				Limit:      noLimit,
				Value:      v + 1,
				Comparator: readers.LowerThanKey,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs),
			},
		},
		"read messages with value and lower-than-or-equal comparator": {
			pageMeta: readers.SenMLMetadata{
				Limit:      noLimit,
				Value:      v + 1,
				Comparator: readers.LowerThanEqualKey,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs),
			},
		},
		"read messages with value and greater-than comparator": {
			pageMeta: readers.SenMLMetadata{
				Limit:      noLimit,
				Value:      v - 1,
				Comparator: readers.GreaterThanKey,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs),
			},
		},
		"read messages with value and greater-than-or-equal comparator": {
			pageMeta: readers.SenMLMetadata{
				Limit:      noLimit,
				Value:      v - 1,
				Comparator: readers.GreaterThanEqualKey,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs),
			},
		},
		"read messages with boolean value": {
			pageMeta: readers.SenMLMetadata{
				Limit:     noLimit,
				BoolValue: vb,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(boolMsgs)),
				Messages: fromSenml(boolMsgs),
			},
		},
		"read messages with string value": {
			pageMeta: readers.SenMLMetadata{
				Limit:       noLimit,
				StringValue: vs,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(stringMsgs)),
				Messages: fromSenml(stringMsgs),
			},
		},
		"read messages with data value": {
			pageMeta: readers.SenMLMetadata{
				Limit:     noLimit,
				DataValue: vd,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(dataMsgs)),
				Messages: fromSenml(dataMsgs),
			},
		},
		"read messages with from": {
			pageMeta: readers.SenMLMetadata{
				Limit: noLimit,
				From:  messages[20].Time,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(messages[0:21])),
				Messages: fromSenml(messages[0:21]),
			},
		},
		"read messages with to": {
			pageMeta: readers.SenMLMetadata{
				Limit: noLimit,
				To:    messages[20].Time,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(messages[20:])),
				Messages: fromSenml(messages[20:]),
			},
		},
		"read messages with from/to": {
			pageMeta: readers.SenMLMetadata{
				Limit: noLimit,
				From:  messages[5].Time,
				To:    messages[0].Time,
			},
			page: readers.MessagesPage{
				Total:    6,
				Messages: fromSenml(messages[0:6]),
			},
		},
		"count aggregation": {
			pageMeta: readers.SenMLMetadata{
				Limit:   noLimit,
				AggType: countAgg,
			},
			page: readers.MessagesPage{
				Total:    msgsNum,
				Messages: fromSenml(messages),
			},
		},
		"min aggregation with name filter": {
			pageMeta: readers.SenMLMetadata{
				Limit:   noLimit,
				Name:    msgName,
				AggType: minAgg,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"max aggregation with name filter": {
			pageMeta: readers.SenMLMetadata{
				Limit:   noLimit,
				Name:    msgName,
				AggType: maxAgg,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"avg aggregation on sum field": {
			pageMeta: readers.SenMLMetadata{
				Limit:    noLimit,
				Name:     msgName,
				AggType:  avgAgg,
				AggField: "sum",
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
	}

	for desc, tc := range cases {
		result, err := reader.ListSenMLMessages(tc.pageMeta)
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))
		assert.ElementsMatch(t, tc.page.Messages, result.Messages, fmt.Sprintf("%s: expected %v got %v", desc, tc.page.Messages, result.Messages))
		assert.Equal(t, tc.page.Total, result.Total, fmt.Sprintf("%s: expected %v got %v", desc, tc.page.Total, result.Total))
	}
}

func TestListAllMessagesJSON(t *testing.T) {
	reader := preader.New(db)
	writer := pwriter.New(db)

	id1, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pyd := map[string]interface{}{
		"field_1": 123.0,
		"field_2": "value",
		"field_3": false,
		"field_4": 12.344,
		"field_5": map[string]interface{}{
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
	pyd2 := map[string]interface{}{
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

	var msgs, httpMsgs []map[string]interface{}
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
		pageMeta readers.JSONMetadata
		page     readers.MessagesPage
	}{
		"read all messages": {
			pageMeta: readers.JSONMetadata{
				Limit: noLimit,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(msgs)),
				Messages: fromJSON(msgs),
			},
		},
		"read messages with protocol": {
			pageMeta: readers.JSONMetadata{
				Limit:    noLimit,
				Protocol: httpProt,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(httpMsgs)),
				Messages: fromJSON(httpMsgs),
			},
		},
	}

	for desc, tc := range cases {
		result, err := reader.ListJSONMessages(tc.pageMeta)
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))
		assert.ElementsMatch(t, tc.page.Messages, result.Messages, fmt.Sprintf("%s: expected %v got %v", desc, tc.page.Messages, result.Messages))
		assert.Equal(t, tc.page.Total, result.Total, fmt.Sprintf("%s: expected %v got %v", desc, tc.page.Total, result.Total))
	}
}

func TestDeleteMessagesSenML(t *testing.T) {
	reader := preader.New(db)
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

	cases := map[string]struct {
		pageMeta      readers.SenMLMetadata
		expectedCount uint64
		description   string
	}{
		"delete messages with subtopic": {
			pageMeta: readers.SenMLMetadata{
				Publisher: pubID2,
				Subtopic:  subtopic,
				From:      0,
				To:        now + 1,
			},
			expectedCount: uint64(len(queryMsgs)),
			description:   "should delete messages with specific subtopic",
		},
		"delete messages with protocol": {
			pageMeta: readers.SenMLMetadata{
				Publisher: pubID2,
				Protocol:  httpProt,
				From:      0,
				To:        now + 1,
			},
			expectedCount: uint64(len(queryMsgs)),
			description:   "should delete messages with specific protocol",
		},
		"delete messages with time range from": {
			pageMeta: readers.SenMLMetadata{
				Publisher: pubID,
				From:      messages[20].Time,
				To:        now + 1,
			},
			expectedCount: 17,
			description:   "should delete messages from specific time",
		},
		"delete messages with time range to": {
			pageMeta: readers.SenMLMetadata{
				Publisher: pubID,
				From:      0,
				To:        messages[20].Time,
			},
			expectedCount: 65,
			description:   "should delete messages to specific time",
		},
		"delete messages with time range from/to": {
			pageMeta: readers.SenMLMetadata{
				Publisher: pubID,
				From:      messages[50].Time,
				To:        messages[20].Time,
			},
			expectedCount: 25,
			description:   "should delete messages within time range",
		},
		"delete all messages for publisher": {
			pageMeta: readers.SenMLMetadata{
				Publisher: pubID,
				From:      0,
				To:        now + 1,
			},
			expectedCount: uint64(msgsNum - len(queryMsgs)),
			description:   "should delete all messages for specific publisher",
		},
	}

	for desc, tc := range cases {
		_ = reader.DeleteSenMLMessages(context.Background(), readers.SenMLMetadata{
			Publisher: pubID,
			From:      0,
			To:        now,
		})
		_ = reader.DeleteSenMLMessages(context.Background(), readers.SenMLMetadata{
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

		beforePage, err := reader.ListSenMLMessages(readers.SenMLMetadata{
			Publisher: tc.pageMeta.Publisher,
			Subtopic:  tc.pageMeta.Subtopic,
			Protocol:  tc.pageMeta.Protocol,
			From:      tc.pageMeta.From,
			To:        tc.pageMeta.To,
			Limit:     noLimit,
			Format:    senmlFormat,
		})

		require.Nil(t, err)
		beforeCount := beforePage.Total

		err = reader.DeleteSenMLMessages(context.Background(), tc.pageMeta)
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))

		afterPage, err := reader.ListSenMLMessages(readers.SenMLMetadata{
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

func TestDeleteMessagesJSON(t *testing.T) {
	reader := preader.New(db)
	writer := pwriter.New(db)

	id1, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pyd := map[string]interface{}{
		"field_1": 123.0,
		"field_2": "value",
		"field_3": false,
		"field_4": 12.344,
		"field_5": map[string]interface{}{
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
	pyd2 := map[string]interface{}{
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
		pageMeta      readers.JSONMetadata
		expectedCount uint64
		description   string
	}{
		"delete JSON messages with publisher id1": {
			pageMeta: readers.JSONMetadata{
				Publisher: id1,
				From:      0,
				To:        int64(created + int64(msgsNum)),
			},
			expectedCount: uint64(msgsNum),
			description:   "should delete JSON messages from specific publisher id1",
		},
		"delete JSON messages with publisher id2": {
			pageMeta: readers.JSONMetadata{
				Publisher: id2,
				From:      0,
				To:        int64(created + int64(msgsNum)),
			},
			expectedCount: uint64(msgsNum),
			description:   "should delete JSON messages from specific publisher id2",
		},
		"delete JSON messages with protocol": {
			pageMeta: readers.JSONMetadata{
				Publisher: id2,
				Protocol:  httpProt,
				From:      0,
				To:        int64(created + int64(msgsNum)),
			},
			expectedCount: uint64(httpMsgCount),
			description:   "should delete JSON messages with HTTP protocol",
		},
		"delete JSON messages with subtopic": {
			pageMeta: readers.JSONMetadata{
				Publisher: id1,
				Subtopic:  subtopic,
				From:      0,
				To:        int64(created + int64(msgsNum)),
			},
			expectedCount: uint64(msgsNum),
			description:   "should delete JSON messages with specific subtopic",
		},
		"delete JSON messages with time range": {
			pageMeta: readers.JSONMetadata{
				Publisher: id1,
				From:      int64(created + 20),
				To:        int64(created + 50),
			},
			expectedCount: 31,
			description:   "should delete JSON messages within time range",
		},
	}

	for desc, tc := range cases {
		_ = reader.DeleteJSONMessages(context.Background(), readers.JSONMetadata{
			Publisher: id1,
			From:      0,
			To:        int64(created + int64(msgsNum)),
		})

		_ = reader.DeleteJSONMessages(context.Background(), readers.JSONMetadata{
			Publisher: id2,
			From:      0,
			To:        int64(created + int64(msgsNum)),
		})

		for _, m := range messages {
			err := writer.Consume(m)
			require.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))
		}

		beforePage, err := reader.ListJSONMessages(readers.JSONMetadata{
			Publisher: tc.pageMeta.Publisher,
			Subtopic:  tc.pageMeta.Subtopic,
			Protocol:  tc.pageMeta.Protocol,
			From:      tc.pageMeta.From,
			To:        tc.pageMeta.To,
			Limit:     noLimit,
		})
		require.Nil(t, err)
		beforeCount := beforePage.Total

		err = reader.DeleteJSONMessages(context.Background(), tc.pageMeta)
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))

		afterPage, err := reader.ListJSONMessages(readers.JSONMetadata{
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

func fromJSON(msg []map[string]interface{}) []readers.Message {
	var ret []readers.Message
	for _, m := range msg {
		ret = append(ret, m)
	}
	return ret
}

func toMap(msg protomfx.Message) (map[string]interface{}, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"created":   msg.Created,
		"subtopic":  msg.Subtopic,
		"publisher": msg.Publisher,
		"protocol":  msg.Protocol,
		"payload":   payload,
	}, nil
}
