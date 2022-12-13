package influxdb_test

import (
	"fmt"
	"testing"
	"time"

	iwriter "github.com/MainfluxLabs/mainflux/consumers/writers/influxdb"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/readers"
	ireader "github.com/MainfluxLabs/mainflux/readers/influxdb"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDB      = "test"
	subtopic    = "topic"
	msgsNum     = 1001
	limit       = 10
	oneMsgLimit = 1
	noLimit     = 0
	valueFields = 5
	mqttProt    = "mqtt"
	httpProt    = "http"
	msgName     = "temperature"
	offset      = 21
	zeroOffset  = 0

	format1 = "format1"
	format2 = "format2"
	wrongID = "wrong_id"
)

var (
	v   float64 = 5
	vs          = "a"
	vb          = true
	vd          = "dataValue"
	sum float64 = 42

	client  influxdb2.Client
	repoCfg = struct {
		Bucket string
		Org    string
	}{
		Bucket: dbBucket,
		Org:    dbOrg,
	}
	idProvider = uuid.New()
)

func TestListChannelMessagesSenML(t *testing.T) {
	writer := iwriter.New(client, repoCfg)

	chanID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pubID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pubID2, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	wrongID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	m := senml.Message{
		Channel:    chanID,
		Publisher:  pubID,
		Protocol:   mqttProt,
		Name:       "name",
		Unit:       "U",
		UpdateTime: 1234,
	}

	messages := []senml.Message{}
	valueMsgs := []senml.Message{}
	boolMsgs := []senml.Message{}
	stringMsgs := []senml.Message{}
	dataMsgs := []senml.Message{}
	queryMsgs := []senml.Message{}
	now := time.Now().UnixNano()

	for i := 0; i < msgsNum; i++ {
		// Mix possible values as well as value sum.
		msg := m
		msg.Time = float64(now)/float64(1e9) - 10*float64(i)

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

	err = writer.Consume(messages)
	require.Nil(t, err, fmt.Sprintf("failed to store message to InfluxDB: %s", err))

	reader := ireader.New(client, repoCfg)

	cases := map[string]struct {
		chanID   string
		pageMeta readers.PageMetadata
		page     readers.MessagesPage
	}{
		"read messages page for existing channel": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: zeroOffset,
				Limit:  msgsNum,
			},
			page: readers.MessagesPage{
				Total:    msgsNum,
				Messages: fromSenml(messages),
			},
		},
		"read messages page for non-existent channel": {
			chanID: wrongID,
			pageMeta: readers.PageMetadata{
				Offset: zeroOffset,
				Limit:  msgsNum,
			},
			page: readers.MessagesPage{
				Messages: []readers.Message{},
			},
		},
		"read messages last page": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: msgsNum - 20,
				Limit:  msgsNum,
			},
			page: readers.MessagesPage{
				Total:    msgsNum,
				Messages: fromSenml(messages[msgsNum-20 : msgsNum]),
			},
		},
		"read messages with non-existent subtopic": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:   zeroOffset,
				Limit:    msgsNum,
				Subtopic: "not-present",
			},
			page: readers.MessagesPage{
				Messages: []readers.Message{},
			},
		},
		"read messages with subtopic": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:   zeroOffset,
				Limit:    uint64(len(queryMsgs)),
				Subtopic: subtopic,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read messages with publisher": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:    zeroOffset,
				Limit:     uint64(len(queryMsgs)),
				Publisher: pubID2,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read messages with wrong format": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Format:    "messagess",
				Offset:    zeroOffset,
				Limit:     uint64(len(queryMsgs)),
				Publisher: pubID2,
			},
			page: readers.MessagesPage{
				Total:    0,
				Messages: []readers.Message{},
			},
		},
		"read messages with protocol": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:   zeroOffset,
				Limit:    uint64(len(queryMsgs)),
				Protocol: httpProt,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read messages with name": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: zeroOffset,
				Limit:  limit,
				Name:   msgName,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs[0:limit]),
			},
		},
		"read messages with value": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: zeroOffset,
				Limit:  limit,
				Value:  v,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		"read messages with value and equal comparator": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:     zeroOffset,
				Limit:      limit,
				Value:      v,
				Comparator: readers.EqualKey,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		"read messages with value and lower-than comparator": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:     zeroOffset,
				Limit:      limit,
				Value:      v + 1,
				Comparator: readers.LowerThanKey,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		"read messages with value and lower-than-or-equal comparator": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:     zeroOffset,
				Limit:      limit,
				Value:      v + 1,
				Comparator: readers.LowerThanEqualKey,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		"read messages with value and greater-than comparator": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:     zeroOffset,
				Limit:      limit,
				Value:      v - 1,
				Comparator: readers.GreaterThanKey,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		"read messages with value and greater-than-or-equal comparator": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:     zeroOffset,
				Limit:      limit,
				Value:      v - 1,
				Comparator: readers.GreaterThanEqualKey,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		"read messages with boolean value": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:    zeroOffset,
				Limit:     limit,
				BoolValue: vb,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(boolMsgs)),
				Messages: fromSenml(boolMsgs[0:limit]),
			},
		},
		"read messages with string value": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:      zeroOffset,
				Limit:       limit,
				StringValue: vs,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(stringMsgs)),
				Messages: fromSenml(stringMsgs[0:limit]),
			},
		},
		"read messages with data value": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:    zeroOffset,
				Limit:     limit,
				DataValue: vd,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(dataMsgs)),
				Messages: fromSenml(dataMsgs[0:limit]),
			},
		},
		"read messages with from": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: zeroOffset,
				Limit:  uint64(len(messages[0 : offset+1])),
				From:   messages[offset].Time,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(messages[0 : offset+1])),
				Messages: fromSenml(messages[0 : offset+1]),
			},
		},
		"read messages with to": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: zeroOffset,
				Limit:  uint64(len(messages[offset-1:])),
				To:     messages[offset-1].Time,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(messages[offset:])),
				Messages: fromSenml(messages[offset:]),
			},
		},
		"read messages with from/to": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: zeroOffset,
				Limit:  limit,
				From:   messages[5].Time,
				To:     messages[0].Time,
			},
			page: readers.MessagesPage{
				Total:    5,
				Messages: fromSenml(messages[1:6]),
			},
		},
	}

	for desc, tc := range cases {
		result, err := reader.ListChannelMessages(tc.chanID, tc.pageMeta)
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))
		assert.ElementsMatch(t, tc.page.Messages, result.Messages, fmt.Sprintf("%s: expected: %v, got: %v", desc, tc.page.Messages, result.Messages))
		assert.Equal(t, tc.page.Total, result.Total, fmt.Sprintf("%s: expected %d got %d", desc, tc.page.Total, result.Total))
	}
}

func TestListChannelMessagesJSON(t *testing.T) {
	writer := iwriter.New(client, repoCfg)

	id1, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	m := json.Message{
		Channel:   id1,
		Publisher: id1,
		Created:   time.Now().UnixNano(),
		Subtopic:  "subtopic/format/some_json",
		Protocol:  "coap",
		Payload: map[string]interface{}{
			"field_1": 123.0,
			"field_2": "value",
			"field_3": false,
		},
	}
	messages1 := json.Messages{
		Format: format1,
	}
	msgs1 := []map[string]interface{}{}
	for i := 0; i < msgsNum; i++ {
		messages1.Data = append(messages1.Data, m)
		m := toMap(m)
		msgs1 = append(msgs1, m)
	}
	err = writer.Consume(messages1)
	assert.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))

	id2, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	m = json.Message{
		Channel:   id2,
		Publisher: id2,
		Created:   time.Now().UnixNano() + msgsNum,
		Subtopic:  "subtopic/other_format/some_other_json",
		Protocol:  "udp",
		Payload: map[string]interface{}{
			"field_pi": 3.14159265,
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

	reader := ireader.New(client, repoCfg)

	cases := map[string]struct {
		chanID   string
		pageMeta readers.PageMetadata
		page     readers.MessagesPage
	}{
		"read messages page for existing channel": {
			chanID: id1,
			pageMeta: readers.PageMetadata{
				Format: messages1.Format,
				Offset: zeroOffset,
				Limit:  oneMsgLimit,
			},
			page: readers.MessagesPage{
				Total:    msgsNum,
				Messages: fromJSON(msgs1[:oneMsgLimit]),
			},
		},
		"read all messages for existing channel": {
			chanID: id1,
			pageMeta: readers.PageMetadata{
				Format: messages1.Format,
				Offset: zeroOffset,
				Limit:  noLimit,
			},
			page: readers.MessagesPage{
				Total:    msgsNum,
				Messages: fromJSON(msgs1),
			},
		},
		"read messages page for non-existent channel": {
			chanID: wrongID,
			pageMeta: readers.PageMetadata{
				Format: messages1.Format,
				Offset: zeroOffset,
				Limit:  limit,
			},
			page: readers.MessagesPage{
				Messages: []readers.Message{},
			},
		},
		"read messages last page": {
			chanID: id2,
			pageMeta: readers.PageMetadata{
				Format: messages2.Format,
				Offset: msgsNum - 20,
				Limit:  msgsNum,
			},
			page: readers.MessagesPage{
				Total:    msgsNum,
				Messages: fromJSON(msgs2[msgsNum-20 : msgsNum]),
			},
		},
		"read messages with protocol": {
			chanID: id2,
			pageMeta: readers.PageMetadata{
				Format:   messages2.Format,
				Offset:   zeroOffset,
				Limit:    uint64(len(httpMsgs)),
				Protocol: httpProt,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(httpMsgs)),
				Messages: fromJSON(httpMsgs),
			},
		},
	}

	for desc, tc := range cases {
		result, err := reader.ListChannelMessages(tc.chanID, tc.pageMeta)
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))

		for i := 0; i < len(result.Messages); i++ {
			m := result.Messages[i]
			// Remove time as it is not sent by the client.
			delete(m.(map[string]interface{}), "time")

			result.Messages[i] = m
		}
		assert.ElementsMatch(t, tc.page.Messages, result.Messages, fmt.Sprintf("%s: expected \n%v got \n%v", desc, tc.page.Messages, result.Messages))
		assert.Equal(t, tc.page.Total, result.Total, fmt.Sprintf("%s: expected %v got %v", desc, tc.page.Total, result.Total))
	}
}

func TestListAllMessagesSenML(t *testing.T) {
	writer := iwriter.New(client, repoCfg)

	chanID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pubID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pubID2, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	m := senml.Message{
		Channel:    chanID,
		Publisher:  pubID,
		Protocol:   mqttProt,
		Name:       "name",
		Unit:       "U",
		UpdateTime: 1234,
	}

	messages := []senml.Message{}
	valueMsgs := []senml.Message{}
	boolMsgs := []senml.Message{}
	stringMsgs := []senml.Message{}
	dataMsgs := []senml.Message{}
	queryMsgs := []senml.Message{}
	now := time.Now().UnixNano()

	for i := 0; i < msgsNum; i++ {
		// Mix possible values as well as value sum.
		msg := m
		msg.Time = float64(now)/float64(1e9) - 10*float64(i)

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

	err = writer.Consume(messages)
	require.Nil(t, err, fmt.Sprintf("failed to store message to InfluxDB: %s", err))

	reader := ireader.New(client, repoCfg)

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
		"read messages last page": {
			pageMeta: readers.PageMetadata{
				Offset: msgsNum - 20,
				Limit:  msgsNum,
			},
			page: readers.MessagesPage{
				Total:    msgsNum,
				Messages: fromSenml(messages[msgsNum-20 : msgsNum]),
			},
		},
		"read messages with non-existent subtopic": {
			pageMeta: readers.PageMetadata{
				Offset:   zeroOffset,
				Limit:    msgsNum,
				Subtopic: "not-present",
			},
			page: readers.MessagesPage{
				Messages: []readers.Message{},
			},
		},
		"read messages with subtopic": {
			pageMeta: readers.PageMetadata{
				Offset:   zeroOffset,
				Limit:    uint64(len(queryMsgs)),
				Subtopic: subtopic,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read messages with publisher": {
			pageMeta: readers.PageMetadata{
				Offset:    zeroOffset,
				Limit:     uint64(len(queryMsgs)),
				Publisher: pubID2,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read messages with wrong format": {
			pageMeta: readers.PageMetadata{
				Format:    "messagess",
				Offset:    zeroOffset,
				Limit:     uint64(len(queryMsgs)),
				Publisher: pubID2,
			},
			page: readers.MessagesPage{
				Total:    0,
				Messages: []readers.Message{},
			},
		},
		"read messages with protocol": {
			pageMeta: readers.PageMetadata{
				Offset:   zeroOffset,
				Limit:    uint64(len(queryMsgs)),
				Protocol: httpProt,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read messages with name": {
			pageMeta: readers.PageMetadata{
				Offset: zeroOffset,
				Limit:  limit,
				Name:   msgName,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs[0:limit]),
			},
		},
		"read messages with value": {
			pageMeta: readers.PageMetadata{
				Offset: zeroOffset,
				Limit:  limit,
				Value:  v,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		"read messages with value and equal comparator": {
			pageMeta: readers.PageMetadata{
				Offset:     zeroOffset,
				Limit:      limit,
				Value:      v,
				Comparator: readers.EqualKey,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		"read messages with value and lower-than comparator": {
			pageMeta: readers.PageMetadata{
				Offset:     zeroOffset,
				Limit:      limit,
				Value:      v + 1,
				Comparator: readers.LowerThanKey,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		"read messages with value and lower-than-or-equal comparator": {
			pageMeta: readers.PageMetadata{
				Offset:     zeroOffset,
				Limit:      limit,
				Value:      v + 1,
				Comparator: readers.LowerThanEqualKey,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		"read messages with value and greater-than comparator": {
			pageMeta: readers.PageMetadata{
				Offset:     zeroOffset,
				Limit:      limit,
				Value:      v - 1,
				Comparator: readers.GreaterThanKey,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		"read messages with value and greater-than-or-equal comparator": {
			pageMeta: readers.PageMetadata{
				Offset:     zeroOffset,
				Limit:      limit,
				Value:      v - 1,
				Comparator: readers.GreaterThanEqualKey,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		"read messages with boolean value": {
			pageMeta: readers.PageMetadata{
				Offset:    zeroOffset,
				Limit:     limit,
				BoolValue: vb,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(boolMsgs)),
				Messages: fromSenml(boolMsgs[0:limit]),
			},
		},
		"read messages with string value": {
			pageMeta: readers.PageMetadata{
				Offset:      zeroOffset,
				Limit:       limit,
				StringValue: vs,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(stringMsgs)),
				Messages: fromSenml(stringMsgs[0:limit]),
			},
		},
		"read messages with data value": {
			pageMeta: readers.PageMetadata{
				Offset:    zeroOffset,
				Limit:     limit,
				DataValue: vd,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(dataMsgs)),
				Messages: fromSenml(dataMsgs[0:limit]),
			},
		},
		"read messages with from": {
			pageMeta: readers.PageMetadata{
				Offset: zeroOffset,
				Limit:  uint64(len(messages[0 : offset+1])),
				From:   messages[offset].Time,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(messages[0 : offset+1])),
				Messages: fromSenml(messages[0 : offset+1]),
			},
		},
		"read messages with to": {
			pageMeta: readers.PageMetadata{
				Offset: zeroOffset,
				Limit:  uint64(len(messages[offset-1:])),
				To:     messages[offset-1].Time,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(messages[offset:])),
				Messages: fromSenml(messages[offset:]),
			},
		},
		"read messages with from/to": {
			pageMeta: readers.PageMetadata{
				Offset: zeroOffset,
				Limit:  limit,
				From:   messages[5].Time,
				To:     messages[0].Time,
			},
			page: readers.MessagesPage{
				Total:    5,
				Messages: fromSenml(messages[0+1 : 5+1]),
			},
		},
	}

	for desc, tc := range cases {
		result, err := reader.ListAllMessages(tc.pageMeta)
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))
		assert.ElementsMatch(t, tc.page.Messages, result.Messages, fmt.Sprintf("%s: expected: %v, got: %v", desc, tc.page.Messages, result.Messages))
		assert.Equal(t, tc.page.Total, result.Total, fmt.Sprintf("%s: expected %d got %d", desc, tc.page.Total, result.Total))
	}
}

func TestListAllMessagesJSON(t *testing.T) {
	writer := iwriter.New(client, repoCfg)

	id1, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	m := json.Message{
		Channel:   id1,
		Publisher: id1,
		Created:   time.Now().UnixNano(),
		Subtopic:  "subtopic/format/some_json",
		Protocol:  "coap",
		Payload: map[string]interface{}{
			"field_1": 123.0,
			"field_2": "value",
			"field_3": false,
		},
	}
	messages1 := json.Messages{
		Format: format1,
	}
	msgs1 := []map[string]interface{}{}
	for i := 0; i < msgsNum; i++ {
		messages1.Data = append(messages1.Data, m)
		m := toMap(m)
		msgs1 = append(msgs1, m)
	}
	err = writer.Consume(messages1)
	assert.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))

	id2, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	m = json.Message{
		Channel:   id2,
		Publisher: id2,
		Created:   time.Now().UnixNano() + msgsNum,
		Subtopic:  "subtopic/other_format/some_other_json",
		Protocol:  "udp",
		Payload: map[string]interface{}{
			"field_pi": 3.14159265,
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

	reader := ireader.New(client, repoCfg)

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
		"read messages last page": {
			pageMeta: readers.PageMetadata{
				Format: messages2.Format,
				Offset: msgsNum - 20,
				Limit:  msgsNum,
			},
			page: readers.MessagesPage{
				Total:    msgsNum,
				Messages: fromJSON(msgs2[msgsNum-20 : msgsNum]),
			},
		},
		"read messages with protocol": {
			pageMeta: readers.PageMetadata{
				Format:   messages2.Format,
				Offset:   zeroOffset,
				Limit:    uint64(len(httpMsgs)),
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

		for i := 0; i < len(result.Messages); i++ {
			m := result.Messages[i]
			// Remove time as it is not sent by the client.
			delete(m.(map[string]interface{}), "time")

			result.Messages[i] = m
		}
		assert.ElementsMatch(t, tc.page.Messages, result.Messages, fmt.Sprintf("%s: expected \n%v got \n%v", desc, tc.page.Messages, result.Messages))
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

func toMap(msg json.Message) map[string]interface{} {
	return map[string]interface{}{
		"channel":   msg.Channel,
		"subtopic":  msg.Subtopic,
		"publisher": msg.Publisher,
		"protocol":  msg.Protocol,
		"payload":   map[string]interface{}(msg.Payload),
	}
}
