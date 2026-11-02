// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messaging_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/stretchr/testify/assert"
)

const (
	deviceTime         = int64(1638310819000000000)
	timeFieldName      = "custom_ts_key"
	timeFieldFormat    = "unix"
	timeFieldLocation  = "UTC"
	pointsDataField    = "Points.Value"
	pointsTimeField    = "Time"
	objectPayload      = `{"custom_ts_key": "1638310819", "key1": "val1"}`
	tsListPayload      = `[{"custom_ts_key": "1638310819", "key1": "val1", "key5": "val5"}]`
	singlePointPayload = `{"Points": [{"Value": {"key1": "val1", "Time": "1638310819"}}]}`
	invalidPayload     = `{"Points": [{"Value": {"key1": "val1"}}, "val2"]}`

	partialPayload = `{"Points": [{"Value": {"key1": "val1", "Time": "1638310819"}}, ` +
		`{"PointId": 2}]}`

	pointsPayload = `{"Points": [` +
		`{"PointId": 1, "Value": [{"Value": "63.6", "Time": "1638310819"}, ` +
		`{"Value": "63.8", "Time": "1638310820"}]}, ` +
		`{"PointId": 2, "Value": [{"Value": "12.1", "Time": "1638310821"}]}]}`

	missingKeyPayload = `{"Points": [{"Value": {"key1": "val1"}}, {"PointId": 2}, ` +
		`{"Value": {"key1": "val2"}}]}`

	objectResPayload       = `{"custom_ts_key":"1638310819","key1":"val1"}`
	tsListResPayload       = `[{"custom_ts_key":"1638310819","key1":"val1","key5":"val5"}]`
	tsListFilterResPayload = `[{"key1":"val1"}]`
	pointsResPayload       = `[{"Value":"63.6"},{"Value":"63.8"},{"Value":"12.1"}]`
	missingKeyResPayload   = `[{"key1":"val1"},{"key1":"val2"}]`
	singlePointResPayload  = `{"Time":"1638310819","key1":"val1"}`
	partialResPayload      = `[{"Time":"1638310819","key1":"val1"}]`
)

func TestFormatMessage(t *testing.T) {
	tsTr := domain.Transformer{
		TimeField:    timeFieldName,
		TimeFormat:   timeFieldFormat,
		TimeLocation: timeFieldLocation,
	}

	tsFilterTr := tsTr
	tsFilterTr.DataFilters = []string{"key1"}

	pointsTr := domain.Transformer{
		DataField:    pointsDataField,
		TimeField:    pointsTimeField,
		TimeFormat:   timeFieldFormat,
		TimeLocation: timeFieldLocation,
	}

	pointsFilterTr := pointsTr
	pointsFilterTr.DataFilters = []string{"Value"}

	dataFieldTr := domain.Transformer{
		DataField: pointsDataField,
	}

	cases := []struct {
		desc        string
		transformer domain.Transformer
		payload     string
		res         string
		deviceTime  bool
		err         error
	}{
		{
			desc:        "format JSON object with time field",
			transformer: tsTr,
			payload:     objectPayload,
			res:         objectResPayload,
			deviceTime:  true,
			err:         nil,
		},
		{
			desc:        "format JSON array with time field",
			transformer: tsTr,
			payload:     tsListPayload,
			res:         tsListResPayload,
			deviceTime:  false,
			err:         nil,
		},
		{
			desc:        "format JSON array with time field and data filters",
			transformer: tsFilterTr,
			payload:     tsListPayload,
			res:         tsListFilterResPayload,
			deviceTime:  false,
			err:         nil,
		},
		{
			desc:        "format JSON with data field in all array items",
			transformer: pointsFilterTr,
			payload:     pointsPayload,
			res:         pointsResPayload,
			deviceTime:  false,
			err:         nil,
		},
		{
			desc:        "format JSON with data field missing in some array items",
			transformer: dataFieldTr,
			payload:     missingKeyPayload,
			res:         missingKeyResPayload,
			deviceTime:  false,
			err:         nil,
		},
		{
			desc:        "format JSON with data field in single array item",
			transformer: pointsTr,
			payload:     singlePointPayload,
			res:         singlePointResPayload,
			deviceTime:  true,
			err:         nil,
		},
		{
			desc:        "format JSON with data field in one of several array items",
			transformer: pointsTr,
			payload:     partialPayload,
			res:         partialResPayload,
			deviceTime:  false,
			err:         nil,
		},
		{
			desc:        "format JSON with data field in non-object array items",
			transformer: dataFieldTr,
			payload:     invalidPayload,
			res:         "",
			deviceTime:  false,
			err:         mfjson.ErrTransform,
		},
	}

	for _, tc := range cases {
		pc := domain.PubConfigInfo{
			ThingID: "thing-1",
			ProfileConfig: &domain.ProfileConfig{
				ContentType: messaging.JSONContentType,
				Transformer: tc.transformer,
			},
		}
		msg := protomfx.Message{
			Records: []*protomfx.Record{{Payload: []byte(tc.payload)}},
		}

		start := time.Now().UnixNano()
		err := messaging.FormatMessage(pc, &msg)
		errMsg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), errMsg)
		if tc.err != nil {
			continue
		}

		payload, err := messaging.ToPayload(msg)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
		resMsg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.res, payload)
		assert.Equal(t, tc.res, string(payload), resMsg)

		createdMsg := fmt.Sprintf("%s: unexpected created %d\n", tc.desc, msg.Created)
		if tc.deviceTime {
			assert.Equal(t, deviceTime, msg.Created, createdMsg)
			continue
		}
		assert.GreaterOrEqual(t, msg.Created, start, createdMsg)
	}
}

func TestToPayload(t *testing.T) {
	first := []byte(`{"key":"val1"}`)
	second := []byte(`{"key":"val2"}`)

	cases := []struct {
		desc string
		msg  protomfx.Message
		res  []byte
	}{
		{
			desc: "get payload of message with single record",
			msg: protomfx.Message{
				ContentType: messaging.JSONContentType,
				Records:     []*protomfx.Record{{Payload: first}},
			},
			res: first,
		},
		{
			desc: "get payload of message with single record encoded as array",
			msg: protomfx.Message{
				ContentType: messaging.JSONContentType,
				Records:     []*protomfx.Record{{Payload: first}},
				Array:       true,
			},
			res: []byte(`[{"key":"val1"}]`),
		},
		{
			desc: "get payload of message with multiple records",
			msg: protomfx.Message{
				ContentType: messaging.JSONContentType,
				Records:     []*protomfx.Record{{Payload: first}, {Payload: second}},
			},
			res: []byte(`[{"key":"val1"},{"key":"val2"}]`),
		},
		{
			desc: "get payload of SenML message with single record",
			msg: protomfx.Message{
				ContentType: messaging.SenMLContentType,
				Records:     []*protomfx.Record{{Payload: first}},
			},
			res: []byte(`[{"key":"val1"}]`),
		},
	}

	for _, tc := range cases {
		res, err := messaging.ToPayload(tc.msg)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
		assert.Equal(t, tc.res, res, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.res, res))
	}
}

func TestToJSONMessages(t *testing.T) {
	msg := protomfx.Message{
		ThingId:  "thing-1",
		Subtopic: "subtopic",
		Protocol: "http",
		Created:  1,
		Records: []*protomfx.Record{
			{
				Payload: []byte(`{"key":"val1"}`),
				Time:    2,
			},
			{
				Payload: []byte(`{"key":"val2"}`),
				Time:    3,
			},
		},
	}

	jsonMsg := mfjson.Message{
		ThingID:  msg.ThingId,
		Subtopic: msg.Subtopic,
		Protocol: msg.Protocol,
	}

	firstRes := jsonMsg
	firstRes.Payload = msg.Records[0].Payload
	firstRes.Created = msg.Records[0].Time

	secondRes := jsonMsg
	secondRes.Payload = msg.Records[1].Payload
	secondRes.Created = msg.Records[1].Time

	res := messaging.ToJSONMessages(msg)
	expected := []mfjson.Message{firstRes, secondRes}
	assert.Equal(t, expected, res, fmt.Sprintf("expected %v got %v\n", expected, res))
}

func TestToSenMLMessages(t *testing.T) {
	msg := protomfx.Message{
		ThingId:  "thing-1",
		Subtopic: "subtopic",
		Protocol: "http",
		Records: []*protomfx.Record{
			{
				Payload: []byte(`{"name":"temp","time":2}`),
				Time:    2,
			},
			{
				Payload: []byte(`{"name":"hum","time":3}`),
				Time:    3,
			},
		},
	}

	invalidMsg := msg
	invalidMsg.Records = []*protomfx.Record{{Payload: []byte(`invalid`)}}

	senmlMsg := senml.Message{
		ThingID:  msg.ThingId,
		Subtopic: msg.Subtopic,
		Protocol: msg.Protocol,
	}

	tempRes := senmlMsg
	tempRes.Name = "temp"
	tempRes.Time = 2

	humRes := senmlMsg
	humRes.Name = "hum"
	humRes.Time = 3

	cases := []struct {
		desc string
		msg  protomfx.Message
		res  []senml.Message
		err  bool
	}{
		{
			desc: "convert message with records",
			msg:  msg,
			res:  []senml.Message{tempRes, humRes},
			err:  false,
		},
		{
			desc: "convert message with invalid record payload",
			msg:  invalidMsg,
			res:  nil,
			err:  true,
		},
	}

	for _, tc := range cases {
		res, err := messaging.ToSenMLMessages(tc.msg)
		assert.Equal(t, tc.err, err != nil, fmt.Sprintf("%s: unexpected error %v\n", tc.desc, err))
		assert.Equal(t, tc.res, res, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.res, res))
	}
}
