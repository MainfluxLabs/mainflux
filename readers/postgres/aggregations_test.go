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

func TestBuildJSONSelect(t *testing.T) {
	cases := []struct {
		desc      string
		aggFields []string
		resPart   string
	}{
		{
			desc:      "empty fields",
			aggFields: []string{},
			resPart:   "CAST('{}' AS jsonb) as payload",
		},
		{
			desc:      "single field",
			aggFields: []string{"temperature"},
			resPart:   "jsonb_build_object('temperature', ia.agg_value_0)",
		},
		{
			desc:      "multiple fields",
			aggFields: []string{"temperature", "humidity"},
			resPart:   "jsonb_build_object('temperature', ia.agg_value_0, 'humidity', ia.agg_value_1)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result := buildJSONSelect(tc.aggFields)
			assert.Contains(t, result, tc.resPart)
			assert.Contains(t, result, "ia.max_time as created")
		})
	}
}

func TestBuildConditionForCount(t *testing.T) {
	cases := []struct {
		desc string
		qp   queryParams
		res  string
	}{
		{
			desc: "no agg fields",
			qp:   queryParams{},
			res:  "1=1",
		},
		{
			desc: "senml table",
			qp: queryParams{
				table:     senmlTable,
				aggFields: []string{"value"},
			},
			res: "MAX(m.value) IS NOT NULL",
		},
		{
			desc: "json table single field",
			qp: queryParams{
				table:     jsonTable,
				aggFields: []string{"temperature"},
			},
			res: "MAX(CAST(m.payload->>'temperature' AS FLOAT)) IS NOT NULL",
		},
		{
			desc: "json table multiple fields",
			qp: queryParams{
				table:     jsonTable,
				aggFields: []string{"temperature", "humidity"},
			},
			res: "MAX(CAST(m.payload->>'temperature' AS FLOAT)) IS NOT NULL OR MAX(CAST(m.payload->>'humidity' AS FLOAT)) IS NOT NULL",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result := buildConditionForCount(tc.qp)
			assert.Equal(t, tc.res, result)
		})
	}
}

func TestBuildTruncTimeExpression(t *testing.T) {
	cases := []struct {
		desc         string
		intervalVal  uint64
		intervalUnit string
		timeColumn   string
		resPart      string
	}{
		{
			desc:         "single hour",
			intervalVal:  1,
			intervalUnit: "hour",
			timeColumn:   "time",
			resPart:      "date_trunc('hour'",
		},
		{
			desc:         "multiple hours",
			intervalVal:  5,
			intervalUnit: "hour",
			timeColumn:   "time",
			resPart:      "interval '5 hours'",
		},
		{
			desc:         "single day",
			intervalVal:  1,
			intervalUnit: "day",
			timeColumn:   "created",
			resPart:      "date_trunc('day'",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result := buildTruncTimeExpression(tc.intervalVal, tc.intervalUnit, tc.timeColumn)
			assert.Contains(t, result, tc.resPart)
			assert.Contains(t, result, "to_timestamp")
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
		desc        string
		qp          queryParams
		resPart     string
		notContains string
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
			resPart: "LIMIT 100",
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
			notContains: "LIMIT",
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
			resPart: "WHERE publisher = :publisher",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result := buildTimeIntervals(tc.qp)
			assert.Contains(t, result, "SELECT DISTINCT")
			assert.Contains(t, result, tc.qp.table)
			switch {
			case tc.notContains != "":
				assert.NotContains(t, result, tc.notContains)
			case tc.resPart != "":
				assert.Contains(t, result, tc.resPart)
			}
		})
	}
}

func TestNewAggStrategy(t *testing.T) {
	max := sqlAggFunc("MAX")
	min := sqlAggFunc("MIN")
	avg := sqlAggFunc("AVG")
	count := sqlAggFunc("COUNT")

	cases := []struct {
		desc    string
		aggType string
		res     *sqlAggFunc
	}{
		{
			desc:    "max",
			aggType: readers.AggregationMax,
			res:     &max,
		},
		{
			desc:    "min",
			aggType: readers.AggregationMin,
			res:     &min,
		},
		{
			desc:    "avg",
			aggType: readers.AggregationAvg,
			res:     &avg,
		},
		{
			desc:    "count",
			aggType: readers.AggregationCount,
			res:     &count,
		},
		{
			desc:    "invalid",
			aggType: "invalid",
			res:     nil,
		},
		{
			desc:    "empty",
			aggType: "",
			res:     nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result := newAggStrategy(tc.aggType)
			switch {
			case tc.res == nil:
				assert.Nil(t, result)
			default:
				assert.Equal(t, *tc.res, result)
			}
		})
	}
}

func TestSqlAggFuncSelectedFields(t *testing.T) {
	cases := []struct {
		desc    string
		fn      sqlAggFunc
		qp      queryParams
		resPart string
	}{
		{
			desc: "senml table",
			fn:   sqlAggFunc("MAX"),
			qp: queryParams{
				table: senmlTable,
			},
			resPart: "ia.agg_value as value",
		},
		{
			desc: "json table",
			fn:   sqlAggFunc("MAX"),
			qp: queryParams{
				table:     jsonTable,
				aggFields: []string{"temperature"},
			},
			resPart: "jsonb_build_object",
		},
		{
			desc: "json table empty fields",
			fn:   sqlAggFunc("MAX"),
			qp: queryParams{
				table:     jsonTable,
				aggFields: []string{},
			},
			resPart: "CAST('{}' AS jsonb)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result := tc.fn.selectedFields(tc.qp)
			assert.Contains(t, result, tc.resPart)
		})
	}
}

func TestSqlAggFuncAggregateExpr(t *testing.T) {
	cases := []struct {
		desc    string
		fn      sqlAggFunc
		qp      queryParams
		resPart string
		isEmpty bool
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
			resPart: "MAX(m.value) as agg_value",
		},
		{
			desc: "senml count",
			fn:   sqlAggFunc("COUNT"),
			qp: queryParams{
				table:     senmlTable,
				aggFields: []string{"value"},
			},
			resPart: "COUNT(m.value) as agg_value",
		},
		{
			desc: "json max",
			fn:   sqlAggFunc("MAX"),
			qp: queryParams{
				table:     jsonTable,
				aggFields: []string{"temperature"},
			},
			resPart: "MAX(CAST(m.payload->>'temperature' as FLOAT)) as agg_value_0",
		},
		{
			desc: "json count",
			fn:   sqlAggFunc("COUNT"),
			qp: queryParams{
				table:     jsonTable,
				aggFields: []string{"temperature"},
			},
			resPart: "COUNT(m.payload->>'temperature') as agg_value_0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result := tc.fn.aggregateExpr(tc.qp)
			if tc.isEmpty {
				assert.Empty(t, result)
			} else {
				assert.Contains(t, result, tc.resPart)
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
