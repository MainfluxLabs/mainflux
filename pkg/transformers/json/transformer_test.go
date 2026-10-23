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
	tsListFilterResPayload = `[{"Created":1638310819000000000,"key1":"val1"}]`
	missingKeyResPayload   = `[{"key1":"val1"},{"key1":"val2"}]`
	singlePointResPayload  = `{"Time":"1638310819","key1":"val1"}`

	pointsPayload = `{"Points": [` +
		`{"PointId": 1, "Value": [{"Value": "63.6", "Time": "1638310819"}, ` +
		`{"Value": "63.8", "Time": "1638310820"}]}, ` +
		`{"PointId": 2, "Value": [{"Value": "12.1", "Time": "1638310821"}]}]}`

	missingKeyPayload = `{"Points": [{"Value": {"key1": "val1"}}, {"PointId": 2}, ` +
		`{"Value": {"key1": "val2"}}]}`

	tsListResPayload = `[{"Created":1638310819000000000,"custom_ts_key":"1638310819",` +
		`"key1":"val1","key5":"val5"}]`

	pointsResPayload = `[{"Created":1638310819000000000,"Value":"63.6"},` +
		`{"Created":1638310820000000000,"Value":"63.8"},` +
		`{"Created":1638310821000000000,"Value":"12.1"}]`
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

	tsListRes := tsListMsg
	tsListRes.Payload = []byte(tsListResPayload)

	tsListFilterRes := tsListMsg
	tsListFilterRes.Payload = []byte(tsListFilterResPayload)

	pointsRes := pointsMsg
	pointsRes.Payload = []byte(pointsResPayload)

	missingKeyRes := missingKeyMsg
	missingKeyRes.Payload = []byte(missingKeyResPayload)

	singlePointRes := singlePointMsg
	singlePointRes.Payload = []byte(singlePointResPayload)
	singlePointRes.Created = tsCreated

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
			desc:        "transform JSON with data field in single array item",
			transformer: pointsTr,
			msg:         singlePointMsg,
			res:         singlePointRes,
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
