// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"fmt"
	"testing"
	"time"

	pwriter "github.com/MainfluxLabs/mainflux/consumers/writers/postgres"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/json"
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
	limit       = 10
	noLimit     = 0
	valueFields = 5
	zeroOffset  = 0
	mqttProt    = "mqtt"
	httpProt    = "http"
	msgName     = "temperature"
	msgFormat   = "messages"
	format1     = "format1"
	format2     = "format2"
	wrongID     = "0"
	wrongFormat = "wrong"
)

var (
	v   float64 = 5
	vs          = "value"
	vb          = true
	vd          = "dataValue"
	sum float64 = 42

	idProvider = uuid.New()
)

func TestListAllMessagesJSON(t *testing.T) {
	writer := pwriter.New(db)

	id1, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	m := json.Message{
		Publisher: id1,
		Created:   time.Now().Unix(),
		Subtopic:  "subtopic/format/some_json",
		Protocol:  "coap",
		Payload: map[string]interface{}{
			"field_1": 123.0,
			"field_2": "value",
			"field_3": false,
			"field_4": 12.344,
			"field_5": map[string]interface{}{
				"field_1": "value",
				"field_2": 42.0,
			},
		},
	}
	messages1 := json.Messages{
		Format: format1,
	}
	msgs1 := []map[string]interface{}{}
	for i := 0; i < msgsNum; i++ {
		msg := m
		messages1.Data = append(messages1.Data, msg)
		m := toMap(msg)
		msgs1 = append(msgs1, m)
	}
	err = writer.Consume(messages1)
	assert.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))

	id2, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	m = json.Message{
		Publisher: id2,
		Created:   time.Now().Unix(),
		Subtopic:  "subtopic/other_format/some_other_json",
		Protocol:  "udp",
		Payload: map[string]interface{}{
			"field_1":     "other_value",
			"false_value": false,
			"field_pi":    3.14159265,
		},
	}
	messages2 := json.Messages{
		Format: format2,
	}
	msgs2 := []map[string]interface{}{}
	httpMsgs := []map[string]interface{}{}
	for i := 0; i < msgsNum; i++ {
		msg := m
		if i%2 == 0 {
			msg.Protocol = httpProt
			httpMsgs = append(httpMsgs, toMap(msg))
		}

		messages2.Data = append(messages2.Data, msg)
		msgs2 = append(msgs2, toMap(msg))
	}

	err = writer.Consume(messages2)
	assert.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))

	reader := preader.New(db)

	cases := map[string]struct {
		pageMeta readers.PageMetadata
		page     readers.MessagesPage
	}{
		"read all messages": {
			pageMeta: readers.PageMetadata{
				Format: messages1.Format,
				Limit:  noLimit,
			},
			page: readers.MessagesPage{
				Total:    msgsNum,
				Messages: fromJSON(msgs1),
			},
		},
		"read messages with protocol": {
			pageMeta: readers.PageMetadata{
				Format:   messages2.Format,
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
		for i := 0; i < len(result.Messages); i++ {
			m := result.Messages[i]
			// Remove id as it is not sent by the client.
			delete(m.(map[string]interface{}), "id")
			result.Messages[i] = m
		}
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

func toMap(msg json.Message) map[string]interface{} {
	return map[string]interface{}{
		"created":   msg.Created,
		"subtopic":  msg.Subtopic,
		"publisher": msg.Publisher,
		"protocol":  msg.Protocol,
		"payload":   map[string]interface{}(msg.Payload),
	}
}
