// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package postgres

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

type QueryConfig struct {
	Format           string
	TimeColumn       string
	AggField         string
	Interval         string
	Condition        string
	ConditionForJoin string
	Limit            uint64
	AggType          string
}

type AggStrategy interface {
	// Function that builds the query for aggregation.
	BuildQuery(config QueryConfig) string

	// Function that returns selected strings.
	GetSelectedFields(config QueryConfig) string

	//Function containing aggregation expression.
	GetAggregateExpression(config QueryConfig) string
}

type aggregationService struct {
	db *sqlx.DB
}

func newAggregationService(db *sqlx.DB) *aggregationService {
	return &aggregationService{db: db}
}

func (as *aggregationService) readAggregatedMessages(rpm readers.PageMetadata) ([]readers.Message, error) {
	format, _ := as.getFormatAndOrder(rpm)
	params := as.buildQueryParams(rpm)

	config := QueryConfig{
		Format:     format,
		TimeColumn: as.getTimeColumn(format),
		AggField:   as.getAggregateField(rpm, format),
		Interval:   rpm.AggInterval,
		Limit:      rpm.Limit,
		AggType:    rpm.AggType,
	}

	baseCondition := as.buildBaseCondition(rpm, format)
	nameCondition := as.buildNameCondition(rpm, format)
	config.Condition = as.combineConditions(baseCondition, nameCondition)
	config.ConditionForJoin = strings.Replace(config.Condition, "WHERE", "AND", 1)

	if config.Condition == "" {
		config.ConditionForJoin = ""
	}

	strategy := as.getAggregateStrategy(rpm.AggType)
	if strategy == nil {
		return []readers.Message{}, nil
	}

	query := strategy.BuildQuery(config)
	rows, err := as.executeQuery(query, params)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []readers.Message{}, nil
	}
	defer rows.Close()

	messages, err := as.scanAggregatedMessages(rows, format)
	if err != nil {
		return nil, err
	}

	return messages, nil
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

func (maxStrt MaxStrategy) BuildQuery(config QueryConfig) string {
	tmpl := `
		WITH time_intervals AS (
			{{.TimeIntervals}}
		),
		interval_aggs AS (
			SELECT 
				ti.interval_time,
				{{.AggExpression}} as agg_value
			FROM time_intervals ti
			LEFT JOIN {{.Format}} m ON {{.TimeJoinCondition}}
				{{.ConditionForJoin}}
			GROUP BY ti.interval_time
			HAVING {{.AggExpression}} IS NOT NULL
		)
		SELECT DISTINCT ON (ia.interval_time) {{.SelectedFields}}
		FROM {{.Format}} m
		JOIN interval_aggs ia ON {{.TimeJoinConditionIA}}
			AND {{.ValueCondition}}
		{{.Condition}}
		ORDER BY ia.interval_time DESC, {{.TimeColumn}} DESC;`

	return maxStrt.renderTemplate(tmpl, config)
}

func (maxStrt MaxStrategy) renderTemplate(templateStr string, config QueryConfig) string {
	data := map[string]string{
		"TimeIntervals":       buildTimeIntervals(config),
		"AggExpression":       maxStrt.GetAggregateExpression(config),
		"Format":              config.Format,
		"TimeJoinCondition":   buildTimeJoinCondition(config),
		"TimeJoinConditionIA": buildTimeJoinConditionIA(config),
		"ConditionForJoin":    config.ConditionForJoin,
		"SelectedFields":      maxStrt.GetSelectedFields(config),
		"ValueCondition":      buildValueCondition(config),
		"Condition":           config.Condition,
		"TimeColumn":          config.TimeColumn,
	}

	tmpl := template.Must(template.New("query").Parse(templateStr))
	var result strings.Builder
	tmpl.Execute(&result, data)
	return result.String()
}

func (maxStrt MaxStrategy) GetSelectedFields(config QueryConfig) string {
	return "m.*"
}

func (maxStrt MaxStrategy) GetAggregateExpression(config QueryConfig) string {
	switch config.Format {
	case defTable:
		return fmt.Sprintf("MAX(m.%s)", config.AggField)
	default:
		return fmt.Sprintf("MAX(CAST(m.payload->>'v' AS float))")
	}
}

type MinStrategy struct{}

func (minStrt MinStrategy) BuildQuery(config QueryConfig) string {
	tmpl := `
		WITH time_intervals AS (
			{{.TimeIntervals}}
		),
		interval_aggs AS (
			SELECT 
				ti.interval_time,
				{{.AggExpression}} as agg_value
			FROM time_intervals ti
			LEFT JOIN {{.Format}} m ON {{.TimeJoinCondition}}
				{{.ConditionForJoin}}
			GROUP BY ti.interval_time
			HAVING {{.AggExpression}} IS NOT NULL
		)
		SELECT DISTINCT ON (ia.interval_time) {{.SelectedFields}}
		FROM {{.Format}} m
		JOIN interval_aggs ia ON {{.TimeJoinConditionIA}}
			AND {{.ValueCondition}}
		{{.Condition}}
		ORDER BY ia.interval_time DESC, {{.TimeColumn}} DESC;`

	return minStrt.renderTemplate(tmpl, config)
}

func (minStrt MinStrategy) renderTemplate(templateStr string, config QueryConfig) string {
	data := map[string]string{
		"TimeIntervals":       buildTimeIntervals(config),
		"AggExpression":       minStrt.GetAggregateExpression(config),
		"Format":              config.Format,
		"TimeJoinCondition":   buildTimeJoinCondition(config),
		"TimeJoinConditionIA": buildTimeJoinConditionIA(config),
		"ConditionForJoin":    config.ConditionForJoin,
		"SelectedFields":      minStrt.GetSelectedFields(config),
		"ValueCondition":      buildValueCondition(config),
		"Condition":           config.Condition,
		"TimeColumn":          config.TimeColumn,
	}

	tmpl := template.Must(template.New("query").Parse(templateStr))
	var result strings.Builder
	tmpl.Execute(&result, data)
	return result.String()
}

func (minStrt MinStrategy) GetSelectedFields(config QueryConfig) string {
	return "m.*"
}

func (minStrt MinStrategy) GetAggregateExpression(config QueryConfig) string {
	switch config.Format {
	case defTable:
		return fmt.Sprintf("MIN(m.%s)", config.AggField)
	default:
		return fmt.Sprintf("MIN(CAST(m.payload->>'v' AS float))")
	}
}

type AvgStrategy struct{}

func (avgStrt AvgStrategy) BuildQuery(config QueryConfig) string {
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
			LEFT JOIN {{.Format}} m ON {{.TimeJoinCondition}}
				{{.ConditionForJoin}}
			GROUP BY ti.interval_time
			HAVING {{.AggExpression}} IS NOT NULL
		)
		SELECT DISTINCT ON (ia.interval_time) {{.SelectedFields}}
		FROM {{.Format}} m
		JOIN interval_aggs ia ON {{.TimeJoinConditionIA}}
			AND m.{{.TimeColumn}} = ia.max_time
		{{.Condition}}
		ORDER BY ia.interval_time DESC, m.{{.TimeColumn}} DESC;`

	return avgStrt.renderTemplate(tmpl, config)
}

func (avgStrt AvgStrategy) renderTemplate(templateStr string, config QueryConfig) string {
	data := map[string]string{
		"TimeIntervals":       buildTimeIntervals(config),
		"AggExpression":       avgStrt.GetAggregateExpression(config),
		"Format":              config.Format,
		"TimeJoinCondition":   buildTimeJoinCondition(config),
		"TimeJoinConditionIA": buildTimeJoinConditionIA(config),
		"ConditionForJoin":    config.ConditionForJoin,
		"SelectedFields":      avgStrt.GetSelectedFields(config),
		"Condition":           config.Condition,
		"TimeColumn":          config.TimeColumn,
	}

	tmpl := template.Must(template.New("query").Parse(templateStr))
	var result strings.Builder
	tmpl.Execute(&result, data)
	return result.String()
}

func (avgStrt AvgStrategy) GetSelectedFields(config QueryConfig) string {
	switch config.Format {
	case defTable:
		return `m.subtopic, m.publisher, m.protocol, m.name, m.unit,
				ia.avg_value as value, 
				m.string_value, m.bool_value, m.data_value, m.sum,
				m.time, m.update_time`
	default:
		return `m.created, m.subtopic, m.publisher, m.protocol,
				jsonb_build_object(
					'n', m.payload->>'n',
					'v', ia.avg_value  
				) as payload`
	}
}

func (avgStrt AvgStrategy) GetAggregateExpression(config QueryConfig) string {
	switch config.Format {
	case defTable:
		return fmt.Sprintf("AVG(m.%s)", config.AggField)
	default:
		return fmt.Sprintf("AVG(CAST(m.payload->>'v' AS float))")
	}
}

type CountStrategy struct{}

func (countStrt CountStrategy) BuildQuery(config QueryConfig) string {
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
			LEFT JOIN {{.Format}} m ON {{.TimeJoinCondition}}
				{{.ConditionForJoin}}
			GROUP BY ti.interval_time
			HAVING {{.AggExpression}} IS NOT NULL
		)
		SELECT DISTINCT ON (ia.interval_time) {{.SelectedFields}}
		FROM {{.Format}} m
		JOIN interval_aggs ia ON {{.TimeJoinConditionIA}}
			AND m.{{.TimeColumn}} = ia.max_time
		{{.Condition}}
		ORDER BY ia.interval_time DESC, m.{{.TimeColumn}} DESC;`

	return countStrt.renderTemplate(tmpl, config)
}

func (countStrt CountStrategy) renderTemplate(templateStr string, config QueryConfig) string {
	data := map[string]string{
		"TimeIntervals":       buildTimeIntervals(config),
		"AggExpression":       countStrt.GetAggregateExpression(config),
		"Format":              config.Format,
		"TimeJoinCondition":   buildTimeJoinCondition(config),
		"TimeJoinConditionIA": buildTimeJoinConditionIA(config),
		"ConditionForJoin":    config.ConditionForJoin,
		"SelectedFields":      countStrt.GetSelectedFields(config),
		"Condition":           config.Condition,
		"TimeColumn":          config.TimeColumn,
	}

	tmpl := template.Must(template.New("query").Parse(templateStr))
	var result strings.Builder
	tmpl.Execute(&result, data)
	return result.String()
}

func (countStrt CountStrategy) GetSelectedFields(config QueryConfig) string {
	switch config.Format {
	case defTable:
		return `m.subtopic, m.publisher, m.protocol, m.name, m.unit,
				ia.sum_value as value, 
				m.string_value, m.bool_value, m.data_value, m.sum,
				m.time, m.update_time`
	default:
		return `m.created, m.subtopic, m.publisher, m.protocol,
				jsonb_build_object(
					'n', m.payload->>'n',
					'v', ia.sum_value  
				) as payload`
	}
}

func (countStrt CountStrategy) GetAggregateExpression(config QueryConfig) string {
	switch config.Format {
	case defTable:
		return fmt.Sprintf("SUM(m.%s)", config.AggField)
	default:
		return fmt.Sprintf("SUM(CAST(m.payload->>'v' AS float))")
	}
}

func buildTimeIntervals(config QueryConfig) string {
	return fmt.Sprintf(`
		SELECT generate_series(
			date_trunc('%s', NOW() - interval '%d %s'),
			date_trunc('%s', NOW()),
			interval '1 %s'
		) as interval_time
		ORDER BY interval_time DESC
		LIMIT %d`,
		config.Interval, config.Limit, config.Interval, config.Interval, config.Interval, config.Limit)
}

func buildTimeJoinCondition(config QueryConfig) string {
	return fmt.Sprintf("date_trunc('%s', to_timestamp(m.%s / 1000000000)) = ti.interval_time",
		config.Interval, config.TimeColumn)
}

func buildTimeJoinConditionIA(config QueryConfig) string {
	return fmt.Sprintf("date_trunc('%s', to_timestamp(m.%s / 1000000000)) = ia.interval_time",
		config.Interval, config.TimeColumn)
}

func buildValueCondition(config QueryConfig) string {
	switch config.Format {
	case defTable:
		return fmt.Sprintf("m.%s = ia.agg_value", config.AggField)
	default:
		return "CAST(m.payload->>'v' as FLOAT) = ia.agg_value"
	}
}

func (as *aggregationService) buildNameCondition(rpm readers.PageMetadata, format string) string {
	if rpm.Name == "" {
		return ""
	}

	switch format {
	case defTable:
		return "WHERE name = :name"
	default:
		return "WHERE payload->>'n' = :name"
	}
}

func (as *aggregationService) combineConditions(condition1, condition2 string) string {
	if condition1 == "" && condition2 == "" {
		return ""
	}
	if condition1 == "" {
		return condition2
	}
	if condition2 == "" {
		return condition1
	}

	condition2 = strings.Replace(condition2, "WHERE", "AND", 1)
	return condition1 + " " + condition2
}

func (as *aggregationService) buildBaseCondition(rpm readers.PageMetadata, table string) string {
	condition := ""
	op := "WHERE"
	timeColumn := as.getTimeColumn(table)

	if rpm.Subtopic != "" {
		condition = fmt.Sprintf(`%s %s subtopic = :subtopic`, condition, op)
		op = "AND"
	}
	if rpm.Publisher != "" {
		condition = fmt.Sprintf(`%s %s publisher = :publisher`, condition, op)
		op = "AND"
	}
	if rpm.Protocol != "" {
		condition = fmt.Sprintf(`%s %s protocol = :protocol`, condition, op)
		op = "AND"
	}
	if rpm.From != 0 {
		condition = fmt.Sprintf(`%s %s %s >= :from`, condition, op, timeColumn)
		op = "AND"
	}
	if rpm.To != 0 {
		condition = fmt.Sprintf(`%s %s %s <= :to`, condition, op, timeColumn)
		op = "AND"
	}

	return condition
}
func (as *aggregationService) scanAggregatedMessages(rows *sqlx.Rows, format string) ([]readers.Message, error) {
	var messages []readers.Message

	switch format {
	case defTable:
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

func (as *aggregationService) executeQuery(query string, params map[string]interface{}) (*sqlx.Rows, error) {
	rows, err := as.db.NamedQuery(query, params)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.UndefinedTable {
			return nil, nil
		}
		return nil, errors.Wrap(readers.ErrReadMessages, err)
	}
	return rows, nil
}

func (as *aggregationService) getFormatAndOrder(rpm readers.PageMetadata) (format, order string) {
	format = defTable
	order = "time"

	if rpm.Format == jsonTable {
		format = jsonTable
		order = "created"
	}
	return format, order
}

func (as *aggregationService) getTimeColumn(table string) string {
	if table == jsonTable {
		return "created"
	}
	return "time"
}

func (as *aggregationService) getAggregateField(rpm readers.PageMetadata, format string) string {
	switch rpm.AggField {
	case "":
		if format == jsonTable {
			return "created"
		} else {
			return "value"
		}
	default:
		return rpm.AggField
	}
}

func (as *aggregationService) buildQueryParams(rpm readers.PageMetadata) map[string]interface{} {
	return map[string]interface{}{
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
}
