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

	now := float64(time.Now().Unix())
	for i := 0; i < msgsNum; i++ {
		// Mix possible values as well as value sum.
		msg := m
		msg.Time = now - float64(i)

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
		pageMeta readers.PageMetadata
		page     readers.MessagesPage
	}{
		"read all messages": {
			pageMeta: readers.PageMetadata{
				Limit: noLimit,
			},
			page: readers.MessagesPage{
				Total:    msgsNum,
				Messages: fromSenml(messages),
			},
		},
		"read messages with non-existent subtopic": {
			pageMeta: readers.PageMetadata{
				Limit:    noLimit,
				Subtopic: "not-present",
			},
			page: readers.MessagesPage{
				Messages: []readers.Message{},
			},
		},
		"read messages with subtopic": {
			pageMeta: readers.PageMetadata{
				Limit:    noLimit,
				Subtopic: subtopic,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read messages with publisher": {
			pageMeta: readers.PageMetadata{
				Limit:     noLimit,
				Publisher: pubID2,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read messages with protocol": {
			pageMeta: readers.PageMetadata{
				Limit:    noLimit,
				Protocol: httpProt,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read messages with name": {
			pageMeta: readers.PageMetadata{
				Limit: noLimit,
				Name:  msgName,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read messages with value": {
			pageMeta: readers.PageMetadata{
				Limit: noLimit,
				Value: v,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs),
			},
		},
		"read messages with value and equal comparator": {
			pageMeta: readers.PageMetadata{
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
			pageMeta: readers.PageMetadata{
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
			pageMeta: readers.PageMetadata{
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
			pageMeta: readers.PageMetadata{
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
			pageMeta: readers.PageMetadata{
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
			pageMeta: readers.PageMetadata{
				Limit:     noLimit,
				BoolValue: vb,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(boolMsgs)),
				Messages: fromSenml(boolMsgs),
			},
		},
		"read messages with string value": {
			pageMeta: readers.PageMetadata{
				Limit:       noLimit,
				StringValue: vs,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(stringMsgs)),
				Messages: fromSenml(stringMsgs),
			},
		},
		"read messages with data value": {
			pageMeta: readers.PageMetadata{
				Limit:     noLimit,
				DataValue: vd,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(dataMsgs)),
				Messages: fromSenml(dataMsgs),
			},
		},
		"read messages with from": {
			pageMeta: readers.PageMetadata{
				Limit: noLimit,
				From:  messages[20].Time,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(messages[0:21])),
				Messages: fromSenml(messages[0:21]),
			},
		},
		"read messages with to": {
			pageMeta: readers.PageMetadata{
				Limit: noLimit,
				To:    messages[20].Time,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(messages[21:])),
				Messages: fromSenml(messages[21:]),
			},
		},
		"read messages with from/to": {
			pageMeta: readers.PageMetadata{
				Limit: noLimit,
				From:  messages[5].Time,
				To:    messages[0].Time,
			},
			page: readers.MessagesPage{
				Total:    5,
				Messages: fromSenml(messages[1:6]),
			},
		},
	}

	for desc, tc := range cases {
		result, err := reader.ListAllMessages(tc.pageMeta)
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
		pageMeta readers.PageMetadata
		page     readers.MessagesPage
	}{
		"read all messages": {
			pageMeta: readers.PageMetadata{
				Format: jsonFormat,
				Limit:  noLimit,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(msgs)),
				Messages: fromJSON(msgs),
			},
		},
		"read messages with protocol": {
			pageMeta: readers.PageMetadata{
				Format:   jsonFormat,
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
		result, err := reader.ListAllMessages(tc.pageMeta)
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

	now := float64(time.Now().Unix())
	for i := 0; i < msgsNum; i++ {
		// Mix possible values as well as value sum.
		msg := m
		msg.Time = now - float64(i)

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
		pageMeta      readers.PageMetadata
		expectedCount uint64
		description   string
	}{
		"delete messages with publisher": {
			pageMeta: readers.PageMetadata{
				Limit:     noLimit,
				Publisher: pubID,
			},
			expectedCount: uint64(msgsNum - len(queryMsgs)),
			description:   "should delete messages from specific publisher",
		},
		"delete messages with time range (from)": {
			pageMeta: readers.PageMetadata{
				Publisher: pubID,
				Limit:     noLimit,
				From:      messages[20].Time,
				To:        now + 1,
			},
			expectedCount: uint64(len(messages[0:21])),
			description:   "should delete messages from specific time",
		},
		"delete messages with time range (to)": {
			pageMeta: readers.PageMetadata{
				Publisher: pubID,
				Limit:     noLimit,
				From:      0,
				To:        messages[80].Time,
			},
			expectedCount: uint64(len(messages[81:])),
			description:   "should delete messages to specific time",
		},
		"delete messages with time range (from/to)": {
			pageMeta: readers.PageMetadata{
				Publisher: pubID,
				Limit:     noLimit,
				From:      messages[15].Time,
				To:        messages[10].Time,
			},
			expectedCount: 5,
			description:   "should delete messages within time range",
		},
		"delete all messages": {
			pageMeta: readers.PageMetadata{
				Publisher: pubID,
				Limit:     noLimit,
			},
			expectedCount: uint64(msgsNum - len(queryMsgs)), // only messages with pubID
			description:   "should delete all messages for specific publisher",
		},
	}

	for desc, tc := range cases {
		t.Run(desc, func(t *testing.T) {
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

			deletedCount, err := reader.DeleteMessages(context.Background(), tc.pageMeta)
			assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))
			assert.Equal(t, tc.expectedCount, deletedCount, fmt.Sprint("%s: %s - expected %s deleted, got %d", desc, tc.description, tc.expectedCount, deletedCount))
			

			// cleanup
			_, err = reader.DeleteMessages(context.Background(), readers.PageMetadata{Limit: noLimit})
			require.Nil(t, err, fmt.Sprintf("cleanup failed: %s", err))
		})
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
		Protocol:    payload,
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

	cases := map[string]struct {
		pageMeta      readers.PageMetadata
		expectedCount uint64
		description   string
	}{
		"delete JSON messages with specific publisher": {
			pageMeta: readers.PageMetadata{
				Format:    jsonFormat,
				Limit:     noLimit,
				Publisher: id1,
			},
			expectedCount: msgsNum,
			description:   "should delete JSON messages from specific publisher",
		},
		"delete JSON messages with non-existent publisher": {
			pageMeta: readers.PageMetadata{
				Format:    jsonFormat,
				Limit:     noLimit,
				Publisher: "non-existent-id",
			},
			expectedCount: 0,
			description:   "should delete 0 messages with non-existent publisher",
		},
	}

	for desc, tc := range cases {
		t.Run(desc, func(t *testing.T) {
			for _, m := range messages {
				err := writer.Consume(m)
				require.Nil(t, err, fmt.Sprintf("expected no error got %s", err))
			}

			deletedCount, err := reader.DeleteMessages(context.Background(), tc.pageMeta)
			assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))
			assert.Equal(t, tc.expectedCount, deletedCount, fmt.Sprintf("%s: %s - expected %d deleted, got %d", desc, tc.description, tc.expectedCount, deletedCount))

			// cleanup
			_, err = reader.DeleteMessages(context.Background(), readers.PageMetadata{Format: jsonFormat, Limit: noLimit})
			require.Nil(t, err, fmt.Sprintf("cleanup failed: %s, err"))
		})
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
