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
	pointsDataField       = "Points.Value"
	pointsTimeField       = "Time"
	tsCreated             = int64(1638310819000000000)
	tsListPayload         = `[{"custom_ts_key": "1638310819", "key1": "val1", "key5": "val5"}]`
	singlePointPayload    = `{"Points": [{"Value": {"key1": "val1", "Time": "1638310819"}}]}`
	invalidPointsPayload  = `{"Points": [{"Value": {"key1": "val1"}}, "val2"]}`
	singlePointResPayload = `{"Time":"1638310819","key1":"val1"}`

	partialPayload = `{"Points": [{"Value": {"key1": "val1", "Time": "1638310819"}}, ` +
		`{"PointId": 2}]}`

	pointsPayload = `{"Points": [` +
		`{"PointId": 1, "Value": [{"Value": "63.6", "Time": "1638310819"}, ` +
		`{"Value": "63.8", "Time": "1638310820"}]}, ` +
		`{"PointId": 2, "Value": [{"Value": "12.1", "Time": "1638310821"}]}]}`

	missingKeyPayload = `{"Points": [{"Value": {"key1": "val1"}}, {"PointId": 2}, ` +
		`{"Value": {"key1": "val2"}}]}`
)

var (
	dataFilters = []string{"key1", "key2", "key3"}
)

var transformer = domain.Transformer{DataFilters: dataFilters, TimeField: "nanos_key", TimeFormat: timeFieldFormat, TimeLocation: timeFieldLocation}

func TestTransformJSON(t *testing.T) {
	now := time.Now().UnixNano()

	tsMsgTr := domain.Transformer{DataFilters: dataFilters, TimeField: timeFieldName, TimeFormat: timeFieldFormat, TimeLocation: timeFieldLocation}
	microsMsgTr := domain.Transformer{DataFilters: dataFilters, TimeField: "custom_ts_micro_key", TimeFormat: "unix_us", TimeLocation: timeFieldLocation}
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

	cases := []struct {
		desc        string
		transformer domain.Transformer
		payload     string
		records     []*protomfx.Record
		err         error
	}{
		{
			desc:        "test transform JSON",
			payload:     validPayload,
			transformer: transformer,
			records: []*protomfx.Record{
				{
					Payload: payload,
					Time:    now,
				},
			},
			err: nil,
		},
		{
			desc:        "test transform JSON array",
			payload:     listPayload,
			transformer: transformer,
			records: []*protomfx.Record{
				{
					Payload: payload,
					Time:    now,
				},
				{
					Payload: payload,
					Time:    now,
				},
			},
			err: nil,
		},
		{
			desc:    "test transform JSON with invalid payload",
			payload: invalidPayload,
			records: nil,
			err:     mfjson.ErrTransform,
		},
		{
			desc:        "test transform JSON with timestamp transformation",
			payload:     tsPayload,
			transformer: tsMsgTr,
			records: []*protomfx.Record{
				{
					Payload: payload,
					Time:    tsCreated,
				},
			},
			err: nil,
		},
		{
			desc:        "test transform JSON with timestamp transformation in micros",
			transformer: microsMsgTr,
			payload:     microsPayload,
			records: []*protomfx.Record{
				{
					Payload: payload,
					Time:    tsCreated,
				},
			},
			err: nil,
		},
		{
			desc:        "test transform JSON with invalid timestamp transformation in micros",
			transformer: invalidTimeFieldTr,
			payload:     invalidTsPayload,
			records:     nil,
			err:         mfjson.ErrInvalidTimeField,
		},
	}

	for _, tc := range cases {
		records, err := mfjson.TransformPayload(tc.transformer, []byte(tc.payload), now)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))
		resMsg := fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.records, records)
		assert.Equal(t, tc.records, records, resMsg)
	}
}

func TestTransformJSONArray(t *testing.T) {
	now := time.Now().UnixNano()

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
		return &protomfx.Record{
			Payload: []byte(payload),
			Time:    created,
		}
	}

	cases := []struct {
		desc        string
		transformer domain.Transformer
		payload     string
		records     []*protomfx.Record
		err         error
	}{
		{
			desc:        "transform JSON array with time field",
			transformer: tsTr,
			payload:     tsListPayload,
			records: []*protomfx.Record{
				record(`{"custom_ts_key":"1638310819","key1":"val1","key5":"val5"}`, tsCreated),
			},
			err: nil,
		},
		{
			desc:        "transform JSON array with time field and data filters",
			transformer: tsFilterTr,
			payload:     tsListPayload,
			records:     []*protomfx.Record{record(`{"key1":"val1"}`, tsCreated)},
			err:         nil,
		},
		{
			desc:        "transform JSON with data field in all array items",
			transformer: pointsFilterTr,
			payload:     pointsPayload,
			records: []*protomfx.Record{
				record(`{"Value":"63.6"}`, tsCreated),
				record(`{"Value":"63.8"}`, tsCreated+1e9),
				record(`{"Value":"12.1"}`, tsCreated+2e9),
			},
			err: nil,
		},
		{
			desc:        "transform JSON with data field missing in some array items",
			transformer: dataFieldTr,
			payload:     missingKeyPayload,
			records: []*protomfx.Record{
				record(`{"key1":"val1"}`, now),
				record(`{"key1":"val2"}`, now),
			},
			err: nil,
		},
		{
			desc:        "transform JSON with data field in single array item",
			transformer: pointsTr,
			payload:     singlePointPayload,
			records:     []*protomfx.Record{record(singlePointResPayload, tsCreated)},
			err:         nil,
		},
		{
			desc:        "transform JSON with data field in one of several array items",
			transformer: pointsTr,
			payload:     partialPayload,
			records: []*protomfx.Record{
				record(`{"Time":"1638310819","key1":"val1"}`, tsCreated),
			},
			err: nil,
		},
		{
			desc:        "transform JSON with data field in non-object array items",
			transformer: dataFieldTr,
			payload:     invalidPointsPayload,
			records:     nil,
			err:         mfjson.ErrTransform,
		},
	}

	for _, tc := range cases {
		records, err := mfjson.TransformPayload(tc.transformer, []byte(tc.payload), now)
		errMsg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), errMsg)
		resMsg := fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.records, records)
		assert.Equal(t, tc.records, records, resMsg)
	}
}
