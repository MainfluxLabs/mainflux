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
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/readers"
	mreader "github.com/MainfluxLabs/mainflux/readers/mongodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	testDB      = "test"
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
	port string
	addr string

	v   float64 = 5
	vs          = "value"
	vb          = true
	vd          = "dataValue"
	sum float64 = 42

	idProvider = uuid.New()
)

func TestListAllMessagesSenML(t *testing.T) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	reader := mreader.New(db)
	writer := mwriter.New(db)

	err = db.Drop(context.Background())
	require.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))

	pubID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pubID2, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	messages := []senml.Message{}
	valueMsgs := []senml.Message{}
	boolMsgs := []senml.Message{}
	stringMsgs := []senml.Message{}
	dataMsgs := []senml.Message{}
	queryMsgs := []senml.Message{}
	now := time.Now().Unix()

	for i := 0; i < msgsNum; i++ {
		// Mix possible values as well as value sum.
		msg := senml.Message{
			Publisher: pubID,
			Protocol:  mqttProt,
			Time:      int64(now - int64(i)),
		}

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
			Subject:     "",
		}

		err = writer.Consume(pm)
		assert.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))
	}

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
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	reader := mreader.New(db)
	writer := mwriter.New(db)

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

	for i := 0; i < 3; i++ {
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

		mapped := toMap(m)
		msgs = append(msgs, mapped)
		if m.Protocol == httpProt {
			httpMsgs = append(httpMsgs, mapped)
		}
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
		require.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))

		for i := 0; i < len(result.Messages); i++ {
			if msgMap, ok := result.Messages[i].(map[string]interface{}); ok {
				result.Messages[i] = cleanMap(msgMap)
			}
		}
		assert.ElementsMatch(t, tc.page.Messages, result.Messages, fmt.Sprintf("%s: expected %v got %v", desc, tc.page.Messages, result.Messages))
		assert.Equal(t, tc.page.Total, result.Total, fmt.Sprintf("%s: expected %v got %v", desc, tc.page.Total, result.Total))
	}
}

func fromSenml(in []senml.Message) []readers.Message {
	var ret []readers.Message
	for _, m := range in {
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

func toMap(msg protomfx.Message) map[string]interface{} {
	return map[string]interface{}{
		"created":   msg.Created,
		"subtopic":  msg.Subtopic,
		"publisher": msg.Publisher,
		"protocol":  msg.Protocol,
		"payload":   msg.Payload,
	}
}

func cleanMap(msg map[string]interface{}) map[string]interface{} {
	delete(msg, "_id")

	if bin, ok := msg["payload"].(primitive.Binary); ok {
		msg["payload"] = bin.Data
	}

	return msg
}
