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
	senmlFormat = "messages"
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
	reader := mreader.NewSenMLRepository(db)
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
					Total:    uint64(len(messages[21:])),
					Messages: fromSenml(messages[21:]),
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
					Total:    5,
					Messages: fromSenml(messages[1:6]),
				},
			},
		},
	}

	for desc, tc := range cases {
		result, err := reader.ListMessages(context.Background(), tc.pageMeta)
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))
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
