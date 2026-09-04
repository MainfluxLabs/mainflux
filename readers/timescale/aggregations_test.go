// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package timescale

import (
	"testing"

	mfreaders "github.com/MainfluxLabs/mainflux/pkg/readers"
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
		{
			desc:  "field with spaces, brackets and percent",
			field: "Ambient T [%]",
			res:   "payload->>'Ambient T [%]'",
		},
		{
			desc:  "field with parentheses",
			field: "Sensor MC A (38mm) [%]",
			res:   "payload->>'Sensor MC A (38mm) [%]'",
		},
		{
			desc:  "field with single quote is escaped",
			field: "bob's field",
			res:   "payload->>'bob''s field'",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result, err := buildJSONPath(tc.field)
			assert.NoError(t, err)
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
		{
			desc:      "first orders ascending",
			aggType:   readers.AggregationFirst,
			aggFields: []string{"temperature"},
			res:       "(array_agg(COALESCE(payload, CAST('{}' AS jsonb)) ORDER BY created ASC))[1] AS agg_payload",
		},
		{
			desc:      "last orders descending",
			aggType:   readers.AggregationLast,
			aggFields: []string{"temperature"},
			res:       "(array_agg(COALESCE(payload, CAST('{}' AS jsonb)) ORDER BY created DESC))[1] AS agg_payload",
		},
		{
			desc:      "first selects the picked row time",
			aggType:   readers.AggregationFirst,
			aggFields: []string{"temperature"},
			res:       "(array_agg(created ORDER BY created ASC))[1] AS agg_time",
		},
		{
			desc:      "identity columns come from the picked row",
			aggType:   readers.AggregationFirst,
			aggFields: []string{"temperature"},
			res:       "(array_agg(publisher ORDER BY created ASC))[1] AS agg_publisher",
		},
		{
			desc:      "first without agg fields is still valid",
			aggType:   readers.AggregationFirst,
			aggFields: []string{},
			res:       "AS agg_payload",
		},
		{
			desc:      "last without agg fields is still valid",
			aggType:   readers.AggregationLast,
			aggFields: []string{},
			res:       "AS agg_time",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result, err := jsonAggExpr(tc.aggType, tc.aggFields)
			assert.NoError(t, err)
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
		aggType   string
		aggFields []string
		resPart   string
		timeCol   string
	}{
		{
			desc:      "single field",
			aggType:   readers.AggregationMax,
			aggFields: []string{"temperature"},
			resPart:   "jsonb_build_object('temperature', agg.agg_value_0)",
			timeCol:   "agg.max_time AS created",
		},
		{
			desc:      "multiple fields",
			aggType:   readers.AggregationMax,
			aggFields: []string{"temperature", "humidity"},
			resPart:   "jsonb_build_object('temperature', agg.agg_value_0, 'humidity', agg.agg_value_1)",
			timeCol:   "agg.max_time AS created",
		},
		{
			desc:      "first reads the payload of the picked row",
			aggType:   readers.AggregationFirst,
			aggFields: []string{"temperature"},
			resPart:   "jsonb_build_object('temperature', agg.agg_payload->'temperature')",
			timeCol:   "agg.agg_time AS created",
		},
		{
			desc:      "last with multiple fields",
			aggType:   readers.AggregationLast,
			aggFields: []string{"temperature", "humidity"},
			resPart:   "jsonb_build_object('temperature', agg.agg_payload->'temperature', 'humidity', agg.agg_payload->'humidity')",
			timeCol:   "agg.agg_time AS created",
		},
		{
			desc:      "identity columns come from the picked row",
			aggType:   readers.AggregationLast,
			aggFields: []string{"temperature"},
			resPart:   "agg.agg_publisher AS publisher",
			timeCol:   "agg.agg_time AS created",
		},
		{
			desc:      "first without agg fields returns the whole payload",
			aggType:   readers.AggregationFirst,
			aggFields: []string{},
			resPart:   "agg.agg_payload AS payload",
			timeCol:   "agg.agg_time AS created",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result, err := jsonSelectFields(tc.aggFields, tc.aggType)
			assert.NoError(t, err)
			assert.Contains(t, result, tc.resPart)
			assert.Contains(t, result, tc.timeCol)
		})
	}
}

func TestJsonHaving(t *testing.T) {
	cases := []struct {
		desc      string
		aggType   string
		aggFields []string
		res       string
	}{
		{
			desc:      "no fields",
			aggType:   readers.AggregationMax,
			aggFields: []string{},
			res:       "1=1",
		},
		{
			desc:      "single field",
			aggType:   readers.AggregationMax,
			aggFields: []string{"temperature"},
			res:       "MAX(CAST(payload->>'temperature' AS FLOAT)) IS NOT NULL",
		},
		{
			desc:      "multiple fields",
			aggType:   readers.AggregationMax,
			aggFields: []string{"temperature", "humidity"},
			res:       "MAX(CAST(payload->>'temperature' AS FLOAT)) IS NOT NULL OR MAX(CAST(payload->>'humidity' AS FLOAT)) IS NOT NULL",
		},
		{
			desc:      "first tests presence, not a numeric value",
			aggType:   readers.AggregationFirst,
			aggFields: []string{"temperature"},
			res:       "MAX(payload->>'temperature') IS NOT NULL",
		},
		{
			desc:      "last with multiple fields",
			aggType:   readers.AggregationLast,
			aggFields: []string{"temperature", "humidity"},
			res:       "MAX(payload->>'temperature') IS NOT NULL OR MAX(payload->>'humidity') IS NOT NULL",
		},
		{
			desc:      "first with no fields",
			aggType:   readers.AggregationFirst,
			aggFields: []string{},
			res:       "1=1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result, err := jsonFilterNullFields(tc.aggFields, tc.aggType)
			assert.NoError(t, err)
			assert.Equal(t, tc.res, result)
			if isFirstLast(tc.aggType) {
				assert.NotContains(t, result, "AS FLOAT")
			}
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
				"time < :to",
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
			result := mfreaders.BaseConditions(readers.MessagesPageMetadata{
				Subtopic:  tc.subtopic,
				Publisher: tc.publisher,
				Protocol:  tc.protocol,
				From:      tc.from,
				To:        tc.to,
			}, tc.timeColumn)
			assert.Equal(t, tc.res, result)
		})
	}
}

func TestJsonConditions(t *testing.T) {
	pm := readers.JSONPageMetadata{
		MessagesPageMetadata: readers.MessagesPageMetadata{
			Subtopic:  "test",
			Publisher: "pub1",
			From:      1000,
		},
	}

	result := mfreaders.BaseConditions(pm.MessagesPageMetadata, mfreaders.JSONOrder)

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
			result := mfreaders.SenMLConditions(tc.pm)
			assert.Contains(t, result, tc.resPart)
		})
	}
}

func TestIsFirstLast(t *testing.T) {
	cases := []struct {
		desc    string
		aggType string
		res     bool
	}{
		{desc: "first", aggType: readers.AggregationFirst, res: true},
		{desc: "last", aggType: readers.AggregationLast, res: true},
		{desc: "max", aggType: readers.AggregationMax, res: false},
		{desc: "count", aggType: readers.AggregationCount, res: false},
		{desc: "invalid", aggType: "invalid", res: false},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.res, isFirstLast(tc.aggType))
			// first and last are not scalar SQL functions.
			if tc.res {
				assert.Empty(t, sqlAggFunc(tc.aggType))
			}
		})
	}
}

func TestAggOrderDir(t *testing.T) {
	cases := []struct {
		desc    string
		aggType string
		res     string
	}{
		{desc: "first is ascending", aggType: readers.AggregationFirst, res: "ASC"},
		{desc: "last is descending", aggType: readers.AggregationLast, res: "DESC"},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.res, aggOrderDir(tc.aggType))
		})
	}
}

func TestSenmlFirstLastSubquery(t *testing.T) {
	bucket := timeBucketExpr(1, "hour", mfreaders.SenMLOrder)

	cases := []struct {
		desc      string
		dir       string
		condition string
		resParts  []string
	}{
		{
			desc: "first picks the earliest row of the bucket",
			dir:  "ASC",
			resParts: []string{
				"(array_agg(time ORDER BY time ASC))[1] AS time",
				"(array_agg(value ORDER BY time ASC))[1] AS value",
				"(array_agg(string_value ORDER BY time ASC))[1] AS string_value",
			},
		},
		{
			desc: "last picks the latest row of the bucket",
			dir:  "DESC",
			resParts: []string{
				"(array_agg(time ORDER BY time DESC))[1] AS time",
				"(array_agg(data_value ORDER BY time DESC))[1] AS data_value",
			},
		},
		{
			desc: "identity columns come from the picked row",
			dir:  "ASC",
			resParts: []string{
				"(array_agg(subtopic ORDER BY time ASC))[1] AS subtopic",
				"(array_agg(publisher ORDER BY time ASC))[1] AS publisher",
				"(array_agg(protocol ORDER BY time ASC))[1] AS protocol",
			},
		},
		{
			desc: "nullable non-pointer columns are coalesced",
			dir:  "ASC",
			resParts: []string{
				"COALESCE((array_agg(unit ORDER BY time ASC))[1], '') AS unit",
				"COALESCE((array_agg(update_time ORDER BY time ASC))[1], 0) AS update_time",
			},
		},
		{
			desc:      "carries the where clause and groups by the bucket",
			dir:       "ASC",
			condition: "WHERE publisher = :publisher",
			resParts: []string{
				"WHERE publisher = :publisher",
				"GROUP BY " + bucket,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result := senmlFirstLastSubquery(tc.dir, tc.condition, bucket)
			for _, part := range tc.resParts {
				assert.Contains(t, result, part)
			}
			assert.NotContains(t, result, "HAVING")
		})
	}
}
