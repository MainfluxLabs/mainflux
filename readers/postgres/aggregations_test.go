// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"testing"

	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/stretchr/testify/assert"
)

func TestBuildJSONPath(t *testing.T) {
	cases := []struct {
		desc   string
		field  string
		result string
	}{
		{
			desc:   "single field",
			field:  "temperature",
			result: "payload->>'temperature'",
		},
		{
			desc:   "nested field",
			field:  "sensor.temperature",
			result: "payload->'sensor'->>'temperature'",
		},
		{
			desc:   "deeply nested field",
			field:  "data.sensor.readings.temperature",
			result: "payload->'data'->'sensor'->'readings'->>'temperature'",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := buildJSONPath(tc.field)
			assert.Equal(t, tc.result, got)
		})
	}
}

func TestBuildJSONSelect(t *testing.T) {
	cases := []struct {
		desc       string
		aggFields  []string
		resultPart string
	}{
		{
			desc:       "empty fields",
			aggFields:  []string{},
			resultPart: "CAST('{}' AS jsonb) as payload",
		},
		{
			desc:       "single field",
			aggFields:  []string{"temperature"},
			resultPart: "jsonb_build_object('temperature', ia.agg_value_0)",
		},
		{
			desc:       "multiple fields",
			aggFields:  []string{"temperature", "humidity"},
			resultPart: "jsonb_build_object('temperature', ia.agg_value_0, 'humidity', ia.agg_value_1)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := buildJSONSelect(tc.aggFields)
			assert.Contains(t, got, tc.resultPart)
			assert.Contains(t, got, "ia.max_time as created")
		})
	}
}

func TestBuildConditionForCount(t *testing.T) {
	cases := []struct {
		desc   string
		qp     queryParams
		result string
	}{
		{
			desc:   "no agg fields",
			qp:     queryParams{},
			result: "1=1",
		},
		{
			desc: "senml table",
			qp: queryParams{
				table:     senmlTable,
				aggFields: []string{"value"},
			},
			result: "MAX(m.value) IS NOT NULL",
		},
		{
			desc: "json table single field",
			qp: queryParams{
				table:     jsonTable,
				aggFields: []string{"temperature"},
			},
			result: "MAX(CAST(m.payload->>'temperature' AS FLOAT)) IS NOT NULL",
		},
		{
			desc: "json table multiple fields",
			qp: queryParams{
				table:     jsonTable,
				aggFields: []string{"temperature", "humidity"},
			},
			result: "MAX(CAST(m.payload->>'temperature' AS FLOAT)) IS NOT NULL OR MAX(CAST(m.payload->>'humidity' AS FLOAT)) IS NOT NULL",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := buildConditionForCount(tc.qp)
			assert.Equal(t, tc.result, got)
		})
	}
}

func TestBuildTruncTimeExpression(t *testing.T) {
	cases := []struct {
		desc         string
		intervalVal  uint64
		intervalUnit string
		timeColumn   string
		resultPart   string
	}{
		{
			desc:         "single hour",
			intervalVal:  1,
			intervalUnit: "hour",
			timeColumn:   "time",
			resultPart:   "date_trunc('hour'",
		},
		{
			desc:         "multiple hours",
			intervalVal:  5,
			intervalUnit: "hour",
			timeColumn:   "time",
			resultPart:   "interval '5 hours'",
		},
		{
			desc:         "single day",
			intervalVal:  1,
			intervalUnit: "day",
			timeColumn:   "created",
			resultPart:   "date_trunc('day'",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := buildTruncTimeExpression(tc.intervalVal, tc.intervalUnit, tc.timeColumn)
			assert.Contains(t, got, tc.resultPart)
			assert.Contains(t, got, "to_timestamp")
		})
	}
}

func TestBuildTimeJoinCondition(t *testing.T) {
	qp := queryParams{
		aggValue:    1,
		aggInterval: "hour",
		timeColumn:  "time",
	}

	result := buildTimeJoinCondition(qp)

	assert.Contains(t, result, "ti.interval_time")
	assert.Contains(t, result, "m.time")
}

func TestBuildTimeIntervals(t *testing.T) {
	cases := []struct {
		desc       string
		qp         queryParams
		resultPart string
		noLimit    bool
	}{
		{
			desc: "with limit",
			qp: queryParams{
				table:       senmlTable,
				timeColumn:  senmlOrder,
				aggValue:    1,
				aggInterval: "hour",
				limit:       100,
				dir:         "desc",
			},
			resultPart: "LIMIT 100",
		},
		{
			desc: "without limit",
			qp: queryParams{
				table:       senmlTable,
				timeColumn:  senmlOrder,
				aggValue:    1,
				aggInterval: "hour",
				limit:       0,
				dir:         "asc",
			},
			noLimit: true,
		},
		{
			desc: "with condition",
			qp: queryParams{
				table:       senmlTable,
				timeColumn:  senmlOrder,
				condition:   "WHERE publisher = :publisher",
				aggValue:    1,
				aggInterval: "hour",
			},
			resultPart: "WHERE publisher = :publisher",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := buildTimeIntervals(tc.qp)
			assert.Contains(t, got, "SELECT DISTINCT")
			assert.Contains(t, got, tc.qp.table)
			if tc.noLimit {
				assert.NotContains(t, got, "LIMIT")
			} else if tc.resultPart != "" {
				assert.Contains(t, got, tc.resultPart)
			}
		})
	}
}

func TestNewAggStrategy(t *testing.T) {
	cases := []struct {
		desc    string
		aggType string
		result  sqlAggFunc
		isNil   bool
	}{
		{
			desc:    "max",
			aggType: readers.AggregationMax,
			result:  sqlAggFunc("MAX"),
		},
		{
			desc:    "min",
			aggType: readers.AggregationMin,
			result:  sqlAggFunc("MIN"),
		},
		{
			desc:    "avg",
			aggType: readers.AggregationAvg,
			result:  sqlAggFunc("AVG"),
		},
		{
			desc:    "count",
			aggType: readers.AggregationCount,
			result:  sqlAggFunc("COUNT"),
		},
		{
			desc:    "invalid",
			aggType: "invalid",
			isNil:   true,
		},
		{
			desc:    "empty",
			aggType: "",
			isNil:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := newAggStrategy(tc.aggType)
			if tc.isNil {
				assert.Nil(t, got)
			} else {
				assert.Equal(t, tc.result, got)
			}
		})
	}
}

func TestSqlAggFuncSelectedFields(t *testing.T) {
	cases := []struct {
		desc       string
		fn         sqlAggFunc
		qp         queryParams
		resultPart string
	}{
		{
			desc: "senml table",
			fn:   sqlAggFunc("MAX"),
			qp: queryParams{
				table: senmlTable,
			},
			resultPart: "ia.agg_value as value",
		},
		{
			desc: "json table",
			fn:   sqlAggFunc("MAX"),
			qp: queryParams{
				table:     jsonTable,
				aggFields: []string{"temperature"},
			},
			resultPart: "jsonb_build_object",
		},
		{
			desc: "json table empty fields",
			fn:   sqlAggFunc("MAX"),
			qp: queryParams{
				table:     jsonTable,
				aggFields: []string{},
			},
			resultPart: "CAST('{}' AS jsonb)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := tc.fn.selectedFields(tc.qp)
			assert.Contains(t, got, tc.resultPart)
		})
	}
}

func TestSqlAggFuncAggregateExpr(t *testing.T) {
	cases := []struct {
		desc       string
		fn         sqlAggFunc
		qp         queryParams
		resultPart string
		isEmpty    bool
	}{
		{
			desc: "empty agg fields",
			fn:   sqlAggFunc("MAX"),
			qp: queryParams{
				aggFields: []string{},
			},
			isEmpty: true,
		},
		{
			desc: "senml max",
			fn:   sqlAggFunc("MAX"),
			qp: queryParams{
				table:     senmlTable,
				aggFields: []string{"value"},
			},
			resultPart: "MAX(m.value) as agg_value",
		},
		{
			desc: "senml count",
			fn:   sqlAggFunc("COUNT"),
			qp: queryParams{
				table:     senmlTable,
				aggFields: []string{"value"},
			},
			resultPart: "COUNT(m.value) as agg_value",
		},
		{
			desc: "json max",
			fn:   sqlAggFunc("MAX"),
			qp: queryParams{
				table:     jsonTable,
				aggFields: []string{"temperature"},
			},
			resultPart: "MAX(CAST(m.payload->>'temperature' as FLOAT)) as agg_value_0",
		},
		{
			desc: "json count",
			fn:   sqlAggFunc("COUNT"),
			qp: queryParams{
				table:     jsonTable,
				aggFields: []string{"temperature"},
			},
			resultPart: "COUNT(m.payload->>'temperature') as agg_value_0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := tc.fn.aggregateExpr(tc.qp)
			if tc.isEmpty {
				assert.Empty(t, got)
			} else {
				assert.Contains(t, got, tc.resultPart)
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
		result     []string
	}{
		{
			desc:       "empty filter",
			timeColumn: "time",
			result:     nil,
		},
		{
			desc:       "all fields",
			subtopic:   "sub",
			publisher:  "pub",
			protocol:   "mqtt",
			from:       1000,
			to:         2000,
			timeColumn: "time",
			result: []string{
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
			result: []string{
				"publisher = :publisher",
				"created >= :from",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := baseConditions(tc.subtopic, tc.publisher, tc.protocol, tc.from, tc.to, tc.timeColumn)
			assert.Equal(t, tc.result, got)
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
		desc       string
		pm         readers.SenMLPageMetadata
		resultPart string
	}{
		{
			desc: "with name",
			pm: readers.SenMLPageMetadata{
				Name: "temperature",
			},
			resultPart: "name = :name",
		},
		{
			desc: "with value",
			pm: readers.SenMLPageMetadata{
				Value:      5.0,
				Comparator: "gt",
			},
			resultPart: "value > :value",
		},
		{
			desc: "with bool value",
			pm: readers.SenMLPageMetadata{
				BoolValue: true,
			},
			resultPart: "bool_value = :bool_value",
		},
		{
			desc: "with string value",
			pm: readers.SenMLPageMetadata{
				StringValue: "test",
			},
			resultPart: "string_value = :string_value",
		},
		{
			desc: "with data value",
			pm: readers.SenMLPageMetadata{
				DataValue: "data",
			},
			resultPart: "data_value = :data_value",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := senmlConditions(tc.pm)
			assert.Contains(t, got, tc.resultPart)
		})
	}
}

func TestBuildAggQuery(t *testing.T) {
	qp := queryParams{
		table:            senmlTable,
		timeColumn:       senmlOrder,
		aggValue:         1,
		aggInterval:      "hour",
		aggFields:        []string{"value"},
		aggType:          readers.AggregationMax,
		conditionForJoin: "AND publisher = :publisher",
		dir:              "desc",
	}
	strategy := sqlAggFunc("MAX")

	result := buildAggQuery(qp, strategy)

	assert.Contains(t, result, "WITH time_intervals AS")
	assert.Contains(t, result, "interval_aggs AS")
	assert.Contains(t, result, "MAX(m.value) as agg_value")
	assert.Contains(t, result, "LEFT JOIN senml m ON")
	assert.Contains(t, result, "ORDER BY ia.interval_time")
}

func TestBuildAggCountQuery(t *testing.T) {
	qp := queryParams{
		table:            senmlTable,
		timeColumn:       senmlOrder,
		aggValue:         1,
		aggInterval:      "hour",
		aggFields:        []string{"value"},
		conditionForJoin: "AND publisher = :publisher",
	}

	result := buildAggCountQuery(qp)

	assert.Contains(t, result, "WITH time_intervals AS")
	assert.Contains(t, result, "SELECT COUNT(*) FROM")
	assert.Contains(t, result, "LEFT JOIN senml m ON")
	assert.Contains(t, result, "HAVING")
}
