// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

const (
	timeIntervals        = "ti"
	intervalAggregations = "ia"
	jsonTable            = "json"
	jsonOrder            = "created"
	senmlTable           = "senml"
	senmlOrder           = "time"
	minuteInterval       = "minute"
	hourInterval         = "hour"
	dayInterval          = "day"
	weekInterval         = "week"
	monthInterval        = "month"
	yearInterval         = "year"
)

type QueryParams struct {
	Table            string
	TimeColumn       string
	Condition        string
	ConditionForJoin string
	Limit            uint64
	AggInterval      string
	AggValue         int64
	AggField         string
	AggType          string
}

type AggStrategy interface {
	// Function that builds the query for aggregation.
	BuildQuery(qp QueryParams) string

	// Function that returns selected strings.
	GetSelectedFields(qp QueryParams) string

	//Function containing aggregation expression.
	GetAggregateExpression(qp QueryParams) string
}

type aggregationService struct {
	db dbutil.Database
}

func newAggregationService(db dbutil.Database) *aggregationService {
	return &aggregationService{db: db}
}

func (as *aggregationService) readAggregatedJSONMessages(ctx context.Context, rpm readers.JSONPageMetadata) ([]readers.Message, uint64, error) {
	params := map[string]interface{}{
		"limit":     rpm.Limit,
		"offset":    rpm.Offset,
		"subtopic":  rpm.Subtopic,
		"publisher": rpm.Publisher,
		"protocol":  rpm.Protocol,
		"from":      rpm.From,
		"to":        rpm.To,
	}

	qp := QueryParams{
		Table:       jsonTable,
		TimeColumn:  jsonOrder,
		AggField:    rpm.AggField,
		AggInterval: rpm.AggInterval,
		AggValue:    rpm.AggValue,
		AggType:     rpm.AggType,
		Limit:       rpm.Limit,
	}

	conditions := as.getJSONConditions(rpm)
	if len(conditions) > 0 {
		qp.Condition = "WHERE " + strings.Join(conditions, " AND ")
		qp.ConditionForJoin = "AND " + strings.Join(conditions, " AND ")
	}

	strategy := as.getAggregateStrategy(rpm.AggType)
	if strategy == nil {
		return []readers.Message{}, 0, nil
	}

	query := strategy.BuildQuery(qp)
	rows, err := as.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.UndefinedTable {
			return []readers.Message{}, 0, nil
		}
		return []readers.Message{}, 0, errors.Wrap(readers.ErrReadMessages, err)
	}

	if rows == nil {
		return []readers.Message{}, 0, nil
	}
	defer rows.Close()

	messages, err := as.scanAggregatedMessages(rows, jsonTable)
	if err != nil {
		return []readers.Message{}, 0, err
	}

	timeTrunc := buildTruncTimeExpression(rpm.AggValue, rpm.AggInterval, jsonOrder)
	countQuery := fmt.Sprintf(`SELECT COUNT(DISTINCT %s) FROM %s %s`,
		timeTrunc, jsonTable, qp.Condition)

	total, err := dbutil.Total(ctx, as.db, countQuery, params)
	if err != nil {
		return []readers.Message{}, 0, err
	}

	return messages, total, nil
}

func (as *aggregationService) readAggregatedSenMLMessages(ctx context.Context, rpm readers.SenMLPageMetadata) ([]readers.Message, uint64, error) {
	params := map[string]interface{}{
		"limit":        rpm.Limit,
		"offset":       rpm.Offset,
		"subtopic":     rpm.Subtopic,
		"publisher":    rpm.Publisher,
		"name":         rpm.Name,
		"protocol":     rpm.Protocol,
		"value":        rpm.Value,
		"bool_value":   rpm.BoolValue,
		"string_value": rpm.StringValue,
		"data_value":   rpm.DataValue,
		"from":         rpm.From,
		"to":           rpm.To,
	}

	qp := QueryParams{
		Table:       senmlTable,
		TimeColumn:  senmlOrder,
		AggField:    rpm.AggField,
		AggInterval: rpm.AggInterval,
		AggValue:    rpm.AggValue,
		AggType:     rpm.AggType,
		Limit:       rpm.Limit,
	}

	conditions := as.getSenMLConditions(rpm)
	if rpm.AggField != "" {
		conditions = append(conditions, "name = :agg_field")
		params["agg_field"] = rpm.AggField
	}

	if len(conditions) > 0 {
		qp.Condition = "WHERE " + strings.Join(conditions, " AND ")
		qp.ConditionForJoin = "AND " + strings.Join(conditions, " AND ")
	}

	strategy := as.getAggregateStrategy(rpm.AggType)
	if strategy == nil {
		return []readers.Message{}, 0, nil
	}

	query := strategy.BuildQuery(qp)
	rows, err := as.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.UndefinedTable {
			return []readers.Message{}, 0, nil
		}
		return []readers.Message{}, 0, errors.Wrap(readers.ErrReadMessages, err)
	}
	if rows == nil {
		return []readers.Message{}, 0, nil
	}

	defer rows.Close()

	messages, err := as.scanAggregatedMessages(rows, senmlTable)
	if err != nil {
		return []readers.Message{}, 0, err
	}

	timeTrunc := buildTruncTimeExpression(rpm.AggValue, rpm.AggInterval, senmlOrder)
	countQuery := fmt.Sprintf(`SELECT COUNT(DISTINCT %s) FROM %s %s`,
		timeTrunc, senmlTable, qp.Condition)

	total, err := dbutil.Total(ctx, as.db, countQuery, params)
	if err != nil {
		return []readers.Message{}, 0, err
	}

	return messages, total, nil
}

func (as aggregationService) getAggregateStrategy(aggType string) AggStrategy {
	switch aggType {
	case readers.AggregationMax:
		return MaxStrategy{}
	case readers.AggregationMin:
		return MinStrategy{}
	case readers.AggregationAvg:
		return AvgStrategy{}
	case readers.AggregationCount:
		return CountStrategy{}
	default:
		return nil
	}
}

type MaxStrategy struct{}

func (maxStrt MaxStrategy) BuildQuery(qp QueryParams) string {
	tmpl := `
		WITH time_intervals AS (
			{{.TimeIntervals}}
		),
		interval_aggs AS (
			SELECT 
				ti.interval_time,
				{{.AggExpression}} as agg_value
			FROM time_intervals ti
			LEFT JOIN {{.Table}} m ON {{.TimeJoinCondition}}
				{{.ConditionForJoin}}
			GROUP BY ti.interval_time
			HAVING {{.AggExpression}} IS NOT NULL
		)
		SELECT DISTINCT ON (ia.interval_time) {{.SelectedFields}}
		FROM {{.Table}} m
		JOIN interval_aggs ia ON {{.TimeJoinConditionIA}}
			AND {{.ValueCondition}}
		{{.Condition}}
		ORDER BY ia.interval_time DESC, {{.TimeColumn}} DESC;`

	return renderTemplate(tmpl, qp, maxStrt)
}

func (maxStrt MaxStrategy) GetSelectedFields(qp QueryParams) string {
	return "m.*"
}

func (maxStrt MaxStrategy) GetAggregateExpression(qp QueryParams) string {
	switch qp.Table {
	case senmlTable:
		return "MAX(m.value)"
	default:
		jsonPath := buildJSONPath(qp.AggField)
		return fmt.Sprintf("MAX(CAST(m.%s AS float))", jsonPath)
	}
}

type MinStrategy struct{}

func (minStrt MinStrategy) BuildQuery(qp QueryParams) string {
	tmpl := `
		WITH time_intervals AS (
			{{.TimeIntervals}}
		),
		interval_aggs AS (
			SELECT 
				ti.interval_time,
				{{.AggExpression}} as agg_value
			FROM time_intervals ti
			LEFT JOIN {{.Table}} m ON {{.TimeJoinCondition}}
				{{.ConditionForJoin}}
			GROUP BY ti.interval_time
			HAVING {{.AggExpression}} IS NOT NULL
		)
		SELECT DISTINCT ON (ia.interval_time) {{.SelectedFields}}
		FROM {{.Table}} m
		JOIN interval_aggs ia ON {{.TimeJoinConditionIA}}
			AND {{.ValueCondition}}
		{{.Condition}}
		ORDER BY ia.interval_time DESC, {{.TimeColumn}} DESC;`

	return renderTemplate(tmpl, qp, minStrt)
}

func (minStrt MinStrategy) GetSelectedFields(qp QueryParams) string {
	return "m.*"
}

func (minStrt MinStrategy) GetAggregateExpression(qp QueryParams) string {
	switch qp.Table {
	case senmlTable:
		return "MIN(m.value)"
	default:
		jsonPath := buildJSONPath(qp.AggField)
		return fmt.Sprintf("MIN(CAST(m.%s AS float))", jsonPath)
	}
}

type AvgStrategy struct{}

func (avgStrt AvgStrategy) BuildQuery(qp QueryParams) string {
	tmpl := `
		WITH time_intervals AS (
			{{.TimeIntervals}}
		),
		interval_aggs AS (
			SELECT 
				ti.interval_time,
				{{.AggExpression}} as avg_value,
				MAX(m.{{.TimeColumn}}) as max_time  
			FROM time_intervals ti
			LEFT JOIN {{.Table}} m ON {{.TimeJoinCondition}}
				{{.ConditionForJoin}}
			GROUP BY ti.interval_time
			HAVING {{.AggExpression}} IS NOT NULL
		)
		SELECT DISTINCT ON (ia.interval_time) {{.SelectedFields}}
		FROM {{.Table}} m
		JOIN interval_aggs ia ON {{.TimeJoinConditionIA}}
			AND m.{{.TimeColumn}} = ia.max_time
		{{.Condition}}
		ORDER BY ia.interval_time DESC, m.{{.TimeColumn}} DESC;`

	return renderTemplate(tmpl, qp, avgStrt)
}

func (avgStrt AvgStrategy) GetSelectedFields(qp QueryParams) string {
	switch qp.Table {
	case senmlTable:
		return `m.subtopic, m.publisher, m.protocol, m.name, m.unit,
				ia.avg_value as value, 
				m.string_value, m.bool_value, m.data_value, m.sum,
				m.time, m.update_time`
	default:
		return buildAggregatedJSONSelect(qp.AggField, "avg_value")
	}
}

func (avgStrt AvgStrategy) GetAggregateExpression(qp QueryParams) string {
	switch qp.Table {
	case senmlTable:
		return "AVG(m.value)"
	default:
		jsonPath := buildJSONPath(qp.AggField)
		return fmt.Sprintf("AVG(CAST(m.%s AS float))", jsonPath)
	}
}

type CountStrategy struct{}

func (countStrt CountStrategy) BuildQuery(qp QueryParams) string {
	tmpl := `
		WITH time_intervals AS (
			{{.TimeIntervals}}
		),
		interval_aggs AS (
			SELECT 
				ti.interval_time,
				{{.AggExpression}} as sum_value,
				MAX(m.{{.TimeColumn}}) as max_time  
			FROM time_intervals ti
			LEFT JOIN {{.Table}} m ON {{.TimeJoinCondition}}
				{{.ConditionForJoin}}
			GROUP BY ti.interval_time
			HAVING {{.AggExpression}} IS NOT NULL
		)
		SELECT DISTINCT ON (ia.interval_time) {{.SelectedFields}}
		FROM {{.Table}} m
		JOIN interval_aggs ia ON {{.TimeJoinConditionIA}}
			AND m.{{.TimeColumn}} = ia.max_time
		{{.Condition}}
		ORDER BY ia.interval_time DESC, m.{{.TimeColumn}} DESC;`

	return renderTemplate(tmpl, qp, countStrt)
}

func renderTemplate(templateStr string, qp QueryParams, strategy AggStrategy) string {
	data := map[string]string{
		"TimeIntervals":       buildTimeIntervals(qp),
		"AggExpression":       strategy.GetAggregateExpression(qp),
		"Table":               qp.Table,
		"TimeJoinCondition":   buildTimeJoinCondition(qp, timeIntervals),
		"TimeJoinConditionIA": buildTimeJoinCondition(qp, intervalAggregations),
		"ConditionForJoin":    qp.ConditionForJoin,
		"SelectedFields":      strategy.GetSelectedFields(qp),
		"ValueCondition":      buildValueCondition(qp),
		"Condition":           qp.Condition,
		"TimeColumn":          qp.TimeColumn,
	}

	tmpl := template.Must(template.New("query").Parse(templateStr))
	var result strings.Builder
	tmpl.Execute(&result, data)
	return result.String()
}

func (countStrt CountStrategy) GetSelectedFields(qp QueryParams) string {
	switch qp.Table {
	case senmlTable:
		return `m.subtopic, m.publisher, m.protocol, m.name, m.unit,
				ia.sum_value as value, 
				m.string_value, m.bool_value, m.data_value, m.sum,
				m.time, m.update_time`
	default:
		return buildAggregatedJSONSelect(qp.AggField, "sum_value")
	}
}

func (countStrt CountStrategy) GetAggregateExpression(qp QueryParams) string {
	switch qp.Table {
	case senmlTable:
		return "COUNT(m.value)"
	default:
		jsonPath := buildJSONPath(qp.AggField)
		return fmt.Sprintf("COUNT(m.%s)", jsonPath)
	}
}

func buildTimeIntervals(qp QueryParams) string {
	timeTrunc := buildTruncTimeExpression(qp.AggValue, qp.AggInterval, qp.TimeColumn)
	return fmt.Sprintf(`
        SELECT DISTINCT %s as interval_time
        FROM %s 
        %s
        ORDER BY interval_time DESC
        LIMIT %d`,
		timeTrunc, qp.Table, qp.Condition, qp.Limit)
}

func buildTruncTimeExpression(intervalVal int64, intervalUnit string, timeColumn string) string {
	timestamp := fmt.Sprintf("to_timestamp(%s / 1000000000)", timeColumn)

	interval := fmt.Sprintf("%d %s", intervalVal, intervalUnit)
	if isStandardInterval(interval) {
		return fmt.Sprintf("date_trunc('%s', %s)", interval, timestamp)
	}

	return fmt.Sprintf(
		"to_timestamp(floor(extract(epoch from %s) / extract(epoch from interval '%s')) * extract(epoch from interval '%s'))",
		timestamp,
		interval,
		interval,
	)
}

func buildTimeJoinCondition(qp QueryParams, tableAlias string) string {
	timeTrunc := buildTruncTimeExpression(qp.AggValue, qp.AggInterval, "m."+qp.TimeColumn)
	return fmt.Sprintf("%s = %s.interval_time", timeTrunc, tableAlias)
}

func buildValueCondition(qp QueryParams) string {
	switch qp.Table {
	case senmlTable:
		// Always match on 'value' column for SenML
		return "m.value = ia.agg_value"
	default:
		jsonPath := buildJSONPath(qp.AggField)
		return fmt.Sprintf("CAST(m.%s as FLOAT) = ia.agg_value", jsonPath)
	}
}

func (as *aggregationService) scanAggregatedMessages(rows *sqlx.Rows, format string) ([]readers.Message, error) {
	var messages []readers.Message

	switch format {
	case senmlTable:
		for rows.Next() {
			msg := senml.Message{}
			if err := rows.StructScan(&msg); err != nil {
				return nil, errors.Wrap(readers.ErrReadMessages, err)
			}
			messages = append(messages, msg)
		}
	default:
		for rows.Next() {
			msg := mfjson.Message{}
			if err := rows.StructScan(&msg); err != nil {
				return nil, errors.Wrap(readers.ErrReadMessages, err)
			}

			m, err := msg.ToMap()
			if err != nil {
				return nil, errors.Wrap(readers.ErrReadMessages, err)
			}
			messages = append(messages, m)
		}
	}

	return messages, nil
}

func buildJSONPath(field string) string {
	parts := strings.Split(field, ".")
	if len(parts) == 1 {
		return fmt.Sprintf("payload->>'%s'", parts[0])
	}

	var path strings.Builder
	path.WriteString("payload")

	for i, part := range parts {
		if i == len(parts)-1 {
			path.WriteString(fmt.Sprintf("->>'%s'", part))
		} else {
			path.WriteString(fmt.Sprintf("->'%s'", part))
		}
	}

	return path.String()
}

func buildAggregatedJSONSelect(aggField string, aggAlias string) string {
	parts := strings.Split(aggField, ".")
	if len(parts) == 1 {
		return fmt.Sprintf(`m.created, m.subtopic, m.publisher, m.protocol,
				jsonb_set(m.payload, '{%s}', to_jsonb(ia.%s)) as payload`,
			parts[0], aggAlias)
	}

	pathArray := "{" + strings.Join(parts, ",") + "}"
	return fmt.Sprintf(`m.created, m.subtopic, m.publisher, m.protocol,
			jsonb_set(m.payload, '%s', to_jsonb(ia.%s)) as payload`,
		pathArray, aggAlias)
}

func (as *aggregationService) getJSONConditions(rpm readers.JSONPageMetadata) []string {
	var conditions []string

	if rpm.Subtopic != "" {
		conditions = append(conditions, "subtopic = :subtopic")
	}
	if rpm.Publisher != "" {
		conditions = append(conditions, "publisher = :publisher")
	}
	if rpm.Protocol != "" {
		conditions = append(conditions, "protocol = :protocol")
	}
	if rpm.From != 0 {
		conditions = append(conditions, fmt.Sprintf("%s >= :from", jsonOrder))
	}
	if rpm.To != 0 {
		conditions = append(conditions, fmt.Sprintf("%s <= :to", jsonOrder))
	}

	return conditions
}

func (as *aggregationService) getSenMLConditions(rpm readers.SenMLPageMetadata) []string {
	var conditions []string

	if rpm.Subtopic != "" {
		conditions = append(conditions, "subtopic = :subtopic")
	}
	if rpm.Publisher != "" {
		conditions = append(conditions, "publisher = :publisher")
	}
	if rpm.Protocol != "" {
		conditions = append(conditions, "protocol = :protocol")
	}
	if rpm.Name != "" {
		conditions = append(conditions, "name = :name")
	}
	if rpm.Value != 0 {
		comparator := as.parseComparator(rpm.Comparator)
		conditions = append(conditions, fmt.Sprintf("value %s :value", comparator))
	}
	if rpm.BoolValue {
		conditions = append(conditions, "bool_value = :bool_value")
	}
	if rpm.StringValue != "" {
		conditions = append(conditions, "string_value = :string_value")
	}
	if rpm.DataValue != "" {
		conditions = append(conditions, "data_value = :data_value")
	}
	if rpm.From != 0 {
		conditions = append(conditions, fmt.Sprintf("%s >= :from", senmlOrder))
	}
	if rpm.To != 0 {
		conditions = append(conditions, fmt.Sprintf("%s <= :to", senmlOrder))
	}

	return conditions
}

func (as *aggregationService) parseComparator(comparator string) string {
	switch comparator {
	case readers.EqualKey:
		return "="
	case readers.LowerThanKey:
		return "<"
	case readers.LowerThanEqualKey:
		return "<="
	case readers.GreaterThanKey:
		return ">"
	case readers.GreaterThanEqualKey:
		return ">="
	default:
		return "="
	}
}

func isStandardInterval(interval string) bool {
	switch interval {
	case minuteInterval, hourInterval, dayInterval, weekInterval,
		monthInterval, yearInterval:
		return true
	default:
		return false
	}
}
