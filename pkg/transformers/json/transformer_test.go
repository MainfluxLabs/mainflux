// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package json_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	validPayload      = `{"key1": "val1", "key2": 123, "key3": {"key4": "val4"}, "key5": "val5"}`
	tsPayload         = `{"custom_ts_key": "1638310819", "key1": "val1", "key2": 123, "key3": {"key4": "val4"}, "key5": "val5"}`
	microsPayload     = `{"custom_ts_micro_key": "1638310819000000", "key1": "val1", "key2": 123, "key3": {"key4": "val4"}, "key5": "val5"}`
	invalidTsPayload  = `{"custom_ts_key": "abc", "key1": "val1", "key2": 123, "key3": {"key4": "val4"}, "key5": "val5"}`
	listPayload       = `[{"key1": "val1", "key2": 123, "key3": {"key4": "val4"}, "key5": "val5"}, {"key1": "val1", "key2": 123, "key3": {"key4": "val4"}, "key5": "val5"}]`
	invalidPayload    = `{"key1": }`
	subtopic          = "subtopic"
	format            = "format"
	timeFieldLocation = "UTC"
	timeFieldFormat   = "unix"
	timeFieldName     = "custom_ts_key"
)

const (
	pointsDataField        = "Points.Value"
	pointsTimeField        = "Time"
	tsCreated              = int64(1638310819000000000)
	tsListPayload          = `[{"custom_ts_key": "1638310819", "key1": "val1", "key5": "val5"}]`
	singlePointPayload     = `{"Points": [{"Value": {"key1": "val1", "Time": "1638310819"}}]}`
	invalidPointsPayload   = `{"Points": [{"Value": {"key1": "val1"}}, "val2"]}`
	tsListFilterResPayload = `[{"key1":"val1"}]`
	missingKeyResPayload   = `[{"key1":"val1"},{"key1":"val2"}]`
	singlePointResPayload  = `{"Time":"1638310819","key1":"val1"}`
	partialResPayload      = `[{"Time":"1638310819","key1":"val1"}]`

	partialPayload = `{"Points": [{"Value": {"key1": "val1", "Time": "1638310819"}}, ` +
		`{"PointId": 2}]}`

	pointsPayload = `{"Points": [` +
		`{"PointId": 1, "Value": [{"Value": "63.6", "Time": "1638310819"}, ` +
		`{"Value": "63.8", "Time": "1638310820"}]}, ` +
		`{"PointId": 2, "Value": [{"Value": "12.1", "Time": "1638310821"}]}]}`

	missingKeyPayload = `{"Points": [{"Value": {"key1": "val1"}}, {"PointId": 2}, ` +
		`{"Value": {"key1": "val2"}}]}`

	tsListResPayload = `[{"custom_ts_key":"1638310819","key1":"val1","key5":"val5"}]`
	pointsResPayload = `[{"Value":"63.6"},{"Value":"63.8"},{"Value":"12.1"}]`
)

var (
	dataFilters = []string{"key1", "key2", "key3"}
)

var transformer = domain.Transformer{DataFilters: dataFilters, TimeField: "nanos_key", TimeFormat: timeFieldFormat, TimeLocation: timeFieldLocation}

func TestTransformJSON(t *testing.T) {
	now := time.Now().UnixNano()

	msg := protomfx.Message{
		Subtopic: subtopic,
		ThingId:  "thing-1",
		Protocol: "protocol",
		Payload:  []byte(validPayload),
		Created:  now,
	}
	invalid := msg
	invalid.Payload = []byte(invalidPayload)

	listMsg := msg
	listMsg.Payload = []byte(listPayload)

	tsMsg := msg
	tsMsg.Payload = []byte(tsPayload)
	tsMsgTr := domain.Transformer{DataFilters: dataFilters, TimeField: timeFieldName, TimeFormat: timeFieldFormat, TimeLocation: timeFieldLocation}

	microsMsg := msg
	microsMsg.Payload = []byte(microsPayload)
	microsMsgTr := domain.Transformer{DataFilters: dataFilters, TimeField: "custom_ts_micro_key", TimeFormat: "unix_us", TimeLocation: timeFieldLocation}

	invalidTimeField := msg
	invalidTimeField.Payload = []byte(invalidTsPayload)
	invalidTimeFieldTr := domain.Transformer{DataFilters: dataFilters, TimeField: timeFieldName, TimeFormat: timeFieldName, TimeLocation: timeFieldLocation}

	pyd := map[string]any{
		"key1": "val1",
		"key2": float64(123),
		"key3": map[string]any{
			"key4": "val4",
		},
	}

	payload, err := json.Marshal(pyd)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	msgs := []protomfx.Message{
		{
			Subtopic: subtopic,
			ThingId:  msg.ThingId,
			Protocol: msg.Protocol,
			Created:  msg.Created,
			Payload:  payload,
		},
	}

	tsMsgs := []protomfx.Message{
		{
			Subtopic: subtopic,
			ThingId:  msg.ThingId,
			Protocol: msg.Protocol,
			Created:  int64(1638310819000000000),
			Payload:  payload,
		},
	}

	microsMsgs := []protomfx.Message{
		{
			Subtopic: subtopic,
			ThingId:  msg.ThingId,
			Protocol: msg.Protocol,
			Created:  int64(1638310819000000000),
			Payload:  payload,
		},
	}

	listMsgs := []protomfx.Message{
		{
			Subtopic: subtopic,
			ThingId:  msg.ThingId,
			Protocol: msg.Protocol,
			Created:  msg.Created,
			Payload:  payload,
		},
		{
			Subtopic: subtopic,
			ThingId:  msg.ThingId,
			Protocol: msg.Protocol,
			Created:  msg.Created,
			Payload:  payload,
		},
	}

	cases := []struct {
		desc        string
		transformer domain.Transformer
		msg         protomfx.Message
		msgs        []protomfx.Message
		err         error
	}{
		{
			desc:        "test transform JSON",
			msg:         msg,
			transformer: transformer,
			msgs:        msgs,
			err:         nil,
		},
		{
			desc:        "test transform JSON array",
			msg:         listMsg,
			transformer: transformer,
			msgs:        listMsgs,
			err:         nil,
		},
		{
			desc: "test transform JSON with invalid payload",
			msg:  invalid,
			msgs: nil,
			err:  mfjson.ErrTransform,
		},
		{
			desc:        "test transform JSON with timestamp transformation",
			msg:         tsMsg,
			transformer: tsMsgTr,
			msgs:        tsMsgs,
			err:         nil,
		},
		{
			desc:        "test transform JSON with timestamp transformation in micros",
			transformer: microsMsgTr,
			msg:         microsMsg,
			msgs:        microsMsgs,
			err:         nil,
		},
		{
			desc:        "test transform JSON with invalid timestamp transformation in micros",
			transformer: invalidTimeFieldTr,
			msg:         invalidTimeField,
			msgs:        nil,
			err:         mfjson.ErrInvalidTimeField,
		},
	}

	for _, tc := range cases {
		err := mfjson.TransformPayload(tc.transformer, &tc.msg)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))
	}
}

func TestTransformJSONArray(t *testing.T) {
	now := time.Now().UnixNano()

	msg := protomfx.Message{
		Subtopic: subtopic,
		ThingId:  "thing-1",
		Protocol: "protocol",
		Created:  now,
	}

	tsListMsg := msg
	tsListMsg.Payload = []byte(tsListPayload)

	pointsMsg := msg
	pointsMsg.Payload = []byte(pointsPayload)

	missingKeyMsg := msg
	missingKeyMsg.Payload = []byte(missingKeyPayload)

	singlePointMsg := msg
	singlePointMsg.Payload = []byte(singlePointPayload)

	partialMsg := msg
	partialMsg.Payload = []byte(partialPayload)

	invalidPointsMsg := msg
	invalidPointsMsg.Payload = []byte(invalidPointsPayload)

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

	record := func(payload string, created int64) *protomfx.Record {
		return &protomfx.Record{Payload: []byte(payload), Created: created}
	}

	tsListRes := tsListMsg
	tsListRes.Payload = []byte(tsListResPayload)
	tsListRes.Records = []*protomfx.Record{
		record(`{"custom_ts_key":"1638310819","key1":"val1","key5":"val5"}`, tsCreated),
	}

	tsListFilterRes := tsListMsg
	tsListFilterRes.Payload = []byte(tsListFilterResPayload)
	tsListFilterRes.Records = []*protomfx.Record{record(`{"key1":"val1"}`, tsCreated)}

	pointsRes := pointsMsg
	pointsRes.Payload = []byte(pointsResPayload)
	pointsRes.Records = []*protomfx.Record{
		record(`{"Value":"63.6"}`, tsCreated),
		record(`{"Value":"63.8"}`, tsCreated+1e9),
		record(`{"Value":"12.1"}`, tsCreated+2e9),
	}

	missingKeyRes := missingKeyMsg
	missingKeyRes.Payload = []byte(missingKeyResPayload)
	missingKeyRes.Records = []*protomfx.Record{
		record(`{"key1":"val1"}`, now),
		record(`{"key1":"val2"}`, now),
	}

	recordsMsg := missingKeyMsg
	recordsMsg.Records = []*protomfx.Record{{Created: tsCreated}, {Created: tsCreated + 1e9}}

	recordsRes := missingKeyRes
	recordsRes.Records = []*protomfx.Record{
		record(`{"key1":"val1"}`, tsCreated),
		record(`{"key1":"val2"}`, tsCreated+1e9),
	}

	singlePointRes := singlePointMsg
	singlePointRes.Payload = []byte(singlePointResPayload)
	singlePointRes.Created = tsCreated
	singlePointRes.Records = []*protomfx.Record{record(singlePointResPayload, tsCreated)}

	partialRes := partialMsg
	partialRes.Payload = []byte(partialResPayload)
	partialRes.Records = []*protomfx.Record{record(`{"Time":"1638310819","key1":"val1"}`, tsCreated)}

	cases := []struct {
		desc        string
		transformer domain.Transformer
		msg         protomfx.Message
		res         protomfx.Message
		err         error
	}{
		{
			desc:        "transform JSON array with time field",
			transformer: tsTr,
			msg:         tsListMsg,
			res:         tsListRes,
			err:         nil,
		},
		{
			desc:        "transform JSON array with time field and data filters",
			transformer: tsFilterTr,
			msg:         tsListMsg,
			res:         tsListFilterRes,
			err:         nil,
		},
		{
			desc:        "transform JSON with data field in all array items",
			transformer: pointsFilterTr,
			msg:         pointsMsg,
			res:         pointsRes,
			err:         nil,
		},
		{
			desc:        "transform JSON with data field missing in some array items",
			transformer: dataFieldTr,
			msg:         missingKeyMsg,
			res:         missingKeyRes,
			err:         nil,
		},
		{
			desc:        "transform JSON array with record timestamps assigned by publisher",
			transformer: dataFieldTr,
			msg:         recordsMsg,
			res:         recordsRes,
			err:         nil,
		},
		{
			desc:        "transform JSON with data field in single array item",
			transformer: pointsTr,
			msg:         singlePointMsg,
			res:         singlePointRes,
			err:         nil,
		},
		{
			desc:        "transform JSON with data field in one of several array items",
			transformer: pointsTr,
			msg:         partialMsg,
			res:         partialRes,
			err:         nil,
		},
		{
			desc:        "transform JSON with data field in non-object array items",
			transformer: dataFieldTr,
			msg:         invalidPointsMsg,
			res:         invalidPointsMsg,
			err:         mfjson.ErrTransform,
		},
	}

	for _, tc := range cases {
		err := mfjson.TransformPayload(tc.transformer, &tc.msg)
		errMsg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), errMsg)
		resMsg := fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.res, tc.msg)
		assert.Equal(t, tc.res, tc.msg, resMsg)
	}
}
