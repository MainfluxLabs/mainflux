// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package timescale

import (
	"testing"

	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/stretchr/testify/assert"
)

func TestBuildJSONPath(t *testing.T) {
	cases := []struct {
		desc  string
		field string
		res   string
	}{
		{
			desc:  "single field",
			field: "temperature",
			res:   "payload->>'temperature'",
		},
		{
			desc:  "nested field",
			field: "sensor.temperature",
			res:   "payload->'sensor'->>'temperature'",
		},
		{
			desc:  "deeply nested field",
			field: "data.sensor.readings.temperature",
			res:   "payload->'data'->'sensor'->'readings'->>'temperature'",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result := buildJSONPath(tc.field)
			assert.Equal(t, tc.res, result)
		})
	}
}

func TestTimeBucketExpr(t *testing.T) {
	cases := []struct {
		desc         string
		intervalVal  uint64
		intervalUnit string
		timeColumn   string
		resPart      string
	}{
		{
			desc:         "1 hour on time column",
			intervalVal:  1,
			intervalUnit: "hour",
			timeColumn:   "time",
			resPart:      "time_bucket('1 hour', to_timestamp(time / 1000000000))",
		},
		{
			desc:         "5 minutes on created column",
			intervalVal:  5,
			intervalUnit: "minute",
			timeColumn:   "created",
			resPart:      "time_bucket('5 minute', to_timestamp(created / 1000000000))",
		},
		{
			desc:         "1 day",
			intervalVal:  1,
			intervalUnit: "day",
			timeColumn:   "time",
			resPart:      "time_bucket('1 day', to_timestamp(time / 1000000000))",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result := timeBucketExpr(tc.intervalVal, tc.intervalUnit, tc.timeColumn)
			assert.Equal(t, tc.resPart, result)
		})
	}
}

func TestSqlAggFunc(t *testing.T) {
	cases := []struct {
		desc    string
		aggType string
		res     string
	}{
		{
			desc:    "max",
			aggType: readers.AggregationMax,
			res:     "MAX",
		},
		{
			desc:    "min",
			aggType: readers.AggregationMin,
			res:     "MIN",
		},
		{
			desc:    "avg",
			aggType: readers.AggregationAvg,
			res:     "AVG",
		},
		{
			desc:    "count",
			aggType: readers.AggregationCount,
			res:     "COUNT",
		},
		{
			desc:    "invalid",
			aggType: "invalid",
			res:     "",
		},
		{
			desc:    "empty",
			aggType: "",
			res:     "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result := sqlAggFunc(tc.aggType)
			assert.Equal(t, tc.res, result)
		})
	}
}

func TestJsonAggExpr(t *testing.T) {
	cases := []struct {
		desc      string
		aggType   string
		aggFields []string
		res       string
		isEmpty   bool
	}{
		{
			desc:      "empty fields",
			aggType:   readers.AggregationMax,
			aggFields: []string{},
			isEmpty:   true,
		},
		{
			desc:      "invalid agg type",
			aggType:   "invalid",
			aggFields: []string{"temperature"},
			isEmpty:   true,
		},
		{
			desc:      "max single field",
			aggType:   readers.AggregationMax,
			aggFields: []string{"temperature"},
			res:       "MAX(CAST(payload->>'temperature' AS FLOAT)) AS agg_value_0",
		},
		{
			desc:      "count single field",
			aggType:   readers.AggregationCount,
			aggFields: []string{"temperature"},
			res:       "COUNT(payload->>'temperature') AS agg_value_0",
		},
		{
			desc:      "max multiple fields",
			aggType:   readers.AggregationMax,
			aggFields: []string{"temperature", "humidity"},
			res:       "MAX(CAST(payload->>'temperature' AS FLOAT)) AS agg_value_0",
		},
		{
			desc:      "avg nested field",
			aggType:   readers.AggregationAvg,
			aggFields: []string{"sensor.temp"},
			res:       "AVG(CAST(payload->'sensor'->>'temp' AS FLOAT)) AS agg_value_0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result := jsonAggExpr(tc.aggType, tc.aggFields)
			if tc.isEmpty {
				assert.Empty(t, result)
			} else {
				assert.Contains(t, result, tc.res)
			}
		})
	}
}

func TestJsonSelectFields(t *testing.T) {
	cases := []struct {
		desc      string
		aggFields []string
		resPart   string
	}{
		{
			desc:      "empty fields",
			aggFields: []string{},
			resPart:   "CAST('{}' AS jsonb) AS payload",
		},
		{
			desc:      "single field",
			aggFields: []string{"temperature"},
			resPart:   "jsonb_build_object('temperature', agg.agg_value_0)",
		},
		{
			desc:      "multiple fields",
			aggFields: []string{"temperature", "humidity"},
			resPart:   "jsonb_build_object('temperature', agg.agg_value_0, 'humidity', agg.agg_value_1)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result := jsonSelectFields(tc.aggFields)
			assert.Contains(t, result, tc.resPart)
			assert.Contains(t, result, "agg.max_time AS created")
		})
	}
}

func TestJsonHaving(t *testing.T) {
	cases := []struct {
		desc      string
		aggFields []string
		res       string
	}{
		{
			desc:      "no fields",
			aggFields: []string{},
			res:       "1=1",
		},
		{
			desc:      "single field",
			aggFields: []string{"temperature"},
			res:       "MAX(CAST(payload->>'temperature' AS FLOAT)) IS NOT NULL",
		},
		{
			desc:      "multiple fields",
			aggFields: []string{"temperature", "humidity"},
			res:       "MAX(CAST(payload->>'temperature' AS FLOAT)) IS NOT NULL OR MAX(CAST(payload->>'humidity' AS FLOAT)) IS NOT NULL",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result := jsonHaving(tc.aggFields)
			assert.Equal(t, tc.res, result)
		})
	}
}

func TestBaseConditions(t *testing.T) {
	cases := []struct {
		desc       string
		subtopic   string
		publisher  string
		protocol   string
		from       int64
		to         int64
		timeColumn string
		res        []string
	}{
		{
			desc:       "empty filter",
			timeColumn: "time",
			res:        nil,
		},
		{
			desc:       "all fields",
			subtopic:   "sub",
			publisher:  "pub",
			protocol:   "mqtt",
			from:       1000,
			to:         2000,
			timeColumn: "time",
			res: []string{
				"subtopic = :subtopic",
				"publisher = :publisher",
				"protocol = :protocol",
				"time >= :from",
				"time <= :to",
			},
		},
		{
			desc:       "partial fields",
			publisher:  "pub",
			from:       1000,
			timeColumn: "created",
			res: []string{
				"publisher = :publisher",
				"created >= :from",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result := baseConditions(tc.subtopic, tc.publisher, tc.protocol, tc.from, tc.to, tc.timeColumn)
			assert.Equal(t, tc.res, result)
		})
	}
}

func TestJsonConditions(t *testing.T) {
	pm := readers.JSONPageMetadata{
		Subtopic:  "test",
		Publisher: "pub1",
		From:      1000,
	}

	result := jsonConditions(pm)

	assert.Contains(t, result, "subtopic = :subtopic")
	assert.Contains(t, result, "publisher = :publisher")
	assert.Contains(t, result, "created >= :from")
}

func TestSenmlConditions(t *testing.T) {
	cases := []struct {
		desc    string
		pm      readers.SenMLPageMetadata
		resPart string
	}{
		{
			desc: "with name",
			pm: readers.SenMLPageMetadata{
				Name: "temperature",
			},
			resPart: "name = :name",
		},
		{
			desc: "with value",
			pm: readers.SenMLPageMetadata{
				Value:      5.0,
				Comparator: "gt",
			},
			resPart: "value > :value",
		},
		{
			desc: "with bool value",
			pm: readers.SenMLPageMetadata{
				BoolValue: true,
			},
			resPart: "bool_value = :bool_value",
		},
		{
			desc: "with string value",
			pm: readers.SenMLPageMetadata{
				StringValue: "test",
			},
			resPart: "string_value = :string_value",
		},
		{
			desc: "with data value",
			pm: readers.SenMLPageMetadata{
				DataValue: "data",
			},
			resPart: "data_value = :data_value",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result := senmlConditions(tc.pm)
			assert.Contains(t, result, tc.resPart)
		})
	}
}
