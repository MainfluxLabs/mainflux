// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package timescale_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	twriter "github.com/MainfluxLabs/mainflux/consumers/writers/timescale"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/readers"
	treader "github.com/MainfluxLabs/mainflux/readers/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAllMessagesJSON(t *testing.T) {
	reader := treader.NewJSONRepository(db)
	writer := twriter.New(db)

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
		"read all json messages": {
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
		"read json messages with protocol": {
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
