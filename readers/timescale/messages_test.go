// SPDX-License-Identifier: Apache-2.0

package timescale_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	twriter "github.com/MainfluxLabs/mainflux/consumers/writers/timescale"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/readers"
	treader "github.com/MainfluxLabs/mainflux/readers/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	subtopic    = "subtopic"
	msgsNum     = 101
	valueFields = 5
	mqttProt    = "mqtt"
	httpProt    = "http"
	udpProt     = "udp"
	coapProt    = "coap"
	msgName     = "temperature"
	jsonFormat  = "json"
	senmlFormat = "senml"
	jsonCT      = "application/json"
	noLimit     = 0
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
	reader := treader.New(db)
	writer := twriter.New(db)

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
		page     readers.SenMLMessagesPage
	}{
		"read all senml messages": {
			pageMeta: readers.SenMLMetadata{
				Limit: noLimit,
			},
			page: readers.SenMLMessagesPage{
				Total:    msgsNum,
				Messages: fromSenml(messages),
			},
		},
		"read senml messages with non-existent subtopic": {
			pageMeta: readers.SenMLMetadata{
				Limit:    noLimit,
				Subtopic: "not-present",
			},
			page: readers.SenMLMessagesPage{
				Messages: []readers.Message{},
			},
		},
		"read senml messages with subtopic": {
			pageMeta: readers.SenMLMetadata{
				Limit:    noLimit,
				Subtopic: subtopic,
			},
			page: readers.SenMLMessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read senml messages with publisher": {
			pageMeta: readers.SenMLMetadata{
				Limit:     noLimit,
				Publisher: pubID2,
			},
			page: readers.SenMLMessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read senml messages with protocol": {
			pageMeta: readers.SenMLMetadata{
				Limit:    noLimit,
				Protocol: httpProt,
			},
			page: readers.SenMLMessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read senml messages with name": {
			pageMeta: readers.SenMLMetadata{
				Limit: noLimit,
				Name:  msgName,
			},
			page: readers.SenMLMessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read senml messages with value": {
			pageMeta: readers.SenMLMetadata{
				Limit: noLimit,
				Value: v,
			},
			page: readers.SenMLMessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs),
			},
		},
		"read senml messages with value and equal comparator": {
			pageMeta: readers.SenMLMetadata{
				Limit:      noLimit,
				Value:      v,
				Comparator: readers.EqualKey,
			},
			page: readers.SenMLMessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs),
			},
		},
		"read senml messages with value and lower-than comparator": {
			pageMeta: readers.SenMLMetadata{
				Limit:      noLimit,
				Value:      v + 1,
				Comparator: readers.LowerThanKey,
			},
			page: readers.SenMLMessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs),
			},
		},
		"read senml messages with value and lower-than-or-equal comparator": {
			pageMeta: readers.SenMLMetadata{
				Limit:      noLimit,
				Value:      v + 1,
				Comparator: readers.LowerThanEqualKey,
			},
			page: readers.SenMLMessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs),
			},
		},
		"read senml messages with value and greater-than comparator": {
			pageMeta: readers.SenMLMetadata{
				Limit:      noLimit,
				Value:      v - 1,
				Comparator: readers.GreaterThanKey,
			},
			page: readers.SenMLMessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs),
			},
		},
		"read senml messages with value and greater-than-or-equal comparator": {
			pageMeta: readers.SenMLMetadata{
				Limit:      noLimit,
				Value:      v - 1,
				Comparator: readers.GreaterThanEqualKey,
			},
			page: readers.SenMLMessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs),
			},
		},
		"read senml messages with boolean value": {
			pageMeta: readers.SenMLMetadata{
				Limit:     noLimit,
				BoolValue: vb,
			},
			page: readers.SenMLMessagesPage{
				Total:    uint64(len(boolMsgs)),
				Messages: fromSenml(boolMsgs),
			},
		},
		"read senml messages with string value": {
			pageMeta: readers.SenMLMetadata{
				Limit:       noLimit,
				StringValue: vs,
			},
			page: readers.SenMLMessagesPage{
				Total:    uint64(len(stringMsgs)),
				Messages: fromSenml(stringMsgs),
			},
		},
		"read senml messages with data value": {
			pageMeta: readers.SenMLMetadata{
				Limit:     noLimit,
				DataValue: vd,
			},
			page: readers.SenMLMessagesPage{
				Total:    uint64(len(dataMsgs)),
				Messages: fromSenml(dataMsgs),
			},
		},
		"read senml messages with from": {
			pageMeta: readers.SenMLMetadata{
				Limit: noLimit,
				From:  messages[20].Time,
			},
			page: readers.SenMLMessagesPage{
				Total:    uint64(len(messages[0:21])),
				Messages: fromSenml(messages[0:21]),
			},
		},
		"read senml messages with to": {
			pageMeta: readers.SenMLMetadata{
				Limit: noLimit,
				To:    messages[20].Time,
			},
			page: readers.SenMLMessagesPage{
				Total:    uint64(len(messages[20:])),
				Messages: fromSenml(messages[20:]),
			},
		},
		"read senml messages with from/to": {
			pageMeta: readers.SenMLMetadata{
				Limit: noLimit,
				From:  messages[5].Time,
				To:    messages[0].Time,
			},
			page: readers.SenMLMessagesPage{
				Total:    6,
				Messages: fromSenml(messages[0:6]),
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
	reader := treader.New(db)
	writer := twriter.New(db)

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
		page     readers.JSONMessagesPage
	}{
		"read all json messages": {
			pageMeta: readers.JSONMetadata{
				Limit: noLimit,
			},
			page: readers.JSONMessagesPage{
				Total:    uint64(len(msgs)),
				Messages: fromJSON(msgs),
			},
		},
		"read json messages with protocol": {
			pageMeta: readers.JSONMetadata{
				Limit:    noLimit,
				Protocol: httpProt,
			},
			page: readers.JSONMessagesPage{
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
