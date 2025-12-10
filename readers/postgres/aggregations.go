// Copyright (c) Mainfluxaggregations
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
)

type QueryParams struct {
	Table            string
	TimeColumn       string
	Condition        string
	ConditionForJoin string
	Limit            uint64
	AggInterval      string
	AggValue         uint64
	AggField         []string
	AggType          string
	Dir              string
}

type AggStrategy interface {
	// Function that returns selected strings.
	GetSelectedFields(qp QueryParams) string

	// Function containing aggregation expression.
	GetAggregateExpression(qp QueryParams) string
}

type aggregationService struct {
	db dbutil.Database
}

func newAggregationService(db dbutil.Database) *aggregationService {
	return &aggregationService{db: db}
}

func (as *aggregationService) readAggregatedJSONMessages(ctx context.Context, rpm readers.JSONPageMetadata) ([]readers.Message, uint64, error) {
	params := map[string]any{
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
		Dir:         rpm.Dir,
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

	query := buildAggregationQuery(qp, strategy)
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

	cq := buildAggregationCountQuery(qp)
	total, err := dbutil.Total(ctx, as.db, cq, params)
	if err != nil {
		return []readers.Message{}, 0, err
	}

	return messages, total, nil
}

func (as *aggregationService) readAggregatedSenMLMessages(ctx context.Context, rpm readers.SenMLPageMetadata) ([]readers.Message, uint64, error) {
	params := map[string]any{
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
		AggField:    []string{rpm.AggField},
		AggInterval: rpm.AggInterval,
		AggValue:    rpm.AggValue,
		AggType:     rpm.AggType,
		Limit:       rpm.Limit,
		Dir:         rpm.Dir,
	}

	conditions := as.getSenMLConditions(rpm)
	if len(conditions) > 0 {
		qp.Condition = "WHERE " + strings.Join(conditions, " AND ")
		qp.ConditionForJoin = "AND " + strings.Join(conditions, " AND ")
	}

	strategy := as.getAggregateStrategy(rpm.AggType)
	if strategy == nil {
		return []readers.Message{}, 0, nil
	}

	query := buildAggregationQuery(qp, strategy)
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

	cq := buildAggregationCountQuery(qp)
	total, err := dbutil.Total(ctx, as.db, cq, params)
	if err != nil {
		return []readers.Message{}, 0, err
	}

	return messages, total, nil
}

func buildAggregationQuery(qp QueryParams, strategy AggStrategy) string {
	tmpl := `
		WITH time_intervals AS (
			{{.TimeIntervals}}
		),
		interval_aggs AS (
			SELECT 
				ti.interval_time,
				{{.AggExpression}},
				MAX(m.{{.TimeColumn}}) as max_time,
				MAX(m.subtopic) as subtopic,
				MAX(m.publisher) as publisher,
				MAX(m.protocol) as protocol
			FROM time_intervals ti
			LEFT JOIN {{.Table}} m ON {{.TimeJoinCondition}}
				{{.ConditionForJoin}}
			GROUP BY ti.interval_time
			HAVING {{.HavingCondition}}
		)
		SELECT {{.SelectedFields}}
		FROM interval_aggs ia
		ORDER BY ia.interval_time {{.Dir}};`

	return renderTemplate(tmpl, qp, strategy)
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
		"HavingCondition":     buildHavingCondition(qp),
		"Condition":           qp.Condition,
		"TimeColumn":          qp.TimeColumn,
		"Dir":                 dbutil.GetDirQuery(qp.Dir),
	}

	tmpl := template.Must(template.New("query").Parse(templateStr))
	var result strings.Builder
	tmpl.Execute(&result, data)
	return result.String()
}

func buildAggregationCountQuery(qp QueryParams) string {
	timeTrunc := buildTruncTimeExpression(qp.AggValue, qp.AggInterval, qp.TimeColumn)
	havingCondition := buildHavingConditionForCount(qp.AggField, qp.Table)

	timeTruncWithAlias := strings.Replace(timeTrunc, qp.TimeColumn, "m."+qp.TimeColumn, 1)

	return fmt.Sprintf(`
		SELECT COUNT(*) FROM (
			SELECT ti.interval_time
			FROM (
				SELECT DISTINCT %s as interval_time
				FROM %s
				%s
			) ti
			LEFT JOIN %s m ON %s = ti.interval_time
			GROUP BY ti.interval_time
			HAVING %s
		) counted`,
		timeTrunc, qp.Table, qp.Condition,
		qp.Table,
		timeTruncWithAlias,
		havingCondition)
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

func (maxStrt MaxStrategy) GetSelectedFields(qp QueryParams) string {
	switch qp.Table {
	case senmlTable:
		return `ia.max_time as time, ia.subtopic, ia.publisher, ia.protocol, 
				'' as name, '' as unit,
				ia.agg_value as value,
				'' as string_value, false as bool_value, '' as data_value, 
				0 as sum, ia.max_time as update_time`
	default:
		return buildAggregatedJSONSelectMultipleFromIA(qp.AggField, "agg_value")
	}
}

func (maxStrt MaxStrategy) GetAggregateExpression(qp QueryParams) string {
	if len(qp.AggField) == 0 {
		return ""
	}

	var expressions []string

	switch qp.Table {
	case senmlTable:
		expressions = append(expressions, "MAX(m.value) as agg_value")
	default:
		for i, field := range qp.AggField {
			jsonPath := buildJSONPath(field)
			expressions = append(expressions,
				fmt.Sprintf("MAX(CAST(m.%s as FLOAT)) as agg_value_%d", jsonPath, i))
		}
	}

	return strings.Join(expressions, ",\n\t\t\t\t")
}

type MinStrategy struct{}

func (minStrt MinStrategy) GetSelectedFields(qp QueryParams) string {
	switch qp.Table {
	case senmlTable:
		return `ia.max_time as time, ia.subtopic, ia.publisher, ia.protocol, 
				'' as name, '' as unit,
				ia.agg_value as value,
				'' as string_value, false as bool_value, '' as data_value, 
				0 as sum, ia.max_time as update_time`
	default:
		return buildAggregatedJSONSelectMultipleFromIA(qp.AggField, "agg_value")
	}
}

func (minStrt MinStrategy) GetAggregateExpression(qp QueryParams) string {
	if len(qp.AggField) == 0 {
		return ""
	}

	var expressions []string
	switch qp.Table {
	case senmlTable:
		expressions = append(expressions, "MIN(m.value) as agg_value")
	default:
		for i, field := range qp.AggField {
			jsonPath := buildJSONPath(field)
			expressions = append(expressions,
				fmt.Sprintf("MIN(CAST(m.%s as FLOAT)) as agg_value_%d", jsonPath, i))
		}
	}
	return strings.Join(expressions, ",\n\t\t\t\t")
}

type AvgStrategy struct{}

func (avgStrt AvgStrategy) GetSelectedFields(qp QueryParams) string {
	switch qp.Table {
	case senmlTable:
		return `ia.max_time as time, ia.subtopic, ia.publisher, ia.protocol, 
				'' as name, '' as unit,
				ia.avg_value_0 as value,
				'' as string_value, false as bool_value, '' as data_value, 
				0 as sum, ia.max_time as update_time`
	default:
		return buildAggregatedJSONSelectMultipleFromIA(qp.AggField, "avg_value")
	}
}

func (avgStrt AvgStrategy) GetAggregateExpression(qp QueryParams) string {
	if len(qp.AggField) == 0 {
		return ""
	}

	var expressions []string
	switch qp.Table {
	case senmlTable:
		expressions = append(expressions, "AVG(m.value) as avg_value_0")
	default:
		for i, field := range qp.AggField {
			jsonPath := buildJSONPath(field)
			expressions = append(expressions,
				fmt.Sprintf("AVG(CAST(m.%s AS float)) as avg_value_%d", jsonPath, i))
		}
	}
	return strings.Join(expressions, ",\n\t\t\t\t")
}

type CountStrategy struct{}

func (countStrt CountStrategy) GetSelectedFields(qp QueryParams) string {
	switch qp.Table {
	case senmlTable:
		return `ia.max_time as time, ia.subtopic, ia.publisher, ia.protocol, 
				'' as name, '' as unit,
				ia.sum_value_0 as value,
				'' as string_value, false as bool_value, '' as data_value, 
				0 as sum, ia.max_time as update_time`
	default:
		return buildAggregatedJSONSelectMultipleFromIA(qp.AggField, "sum_value")
	}
}

func (countStrt CountStrategy) GetAggregateExpression(qp QueryParams) string {
	if len(qp.AggField) == 0 {
		return ""
	}

	var expressions []string
	switch qp.Table {
	case senmlTable:
		expressions = append(expressions, "COUNT(m.value) as sum_value_0")
	default:
		for i, field := range qp.AggField {
			jsonPath := buildJSONPath(field)
			expressions = append(expressions,
				fmt.Sprintf("COUNT(m.%s) as sum_value_%d", jsonPath, i))
		}
	}
	return strings.Join(expressions, ",\n\t\t\t\t")
}

func buildTimeIntervals(qp QueryParams) string {
	dq := dbutil.GetDirQuery(qp.Dir)
	lq := fmt.Sprintf("LIMIT %d", qp.Limit)
	if qp.Limit <= 0 {
		lq = ""
	}
	timeTrunc := buildTruncTimeExpression(qp.AggValue, qp.AggInterval, qp.TimeColumn)
	return fmt.Sprintf(`
        SELECT DISTINCT %s as interval_time
        FROM %s 
        %s
        ORDER BY interval_time %s 
		%s`,
		timeTrunc, qp.Table, qp.Condition, dq, lq)
}

func buildTruncTimeExpression(intervalVal uint64, intervalUnit string, timeColumn string) string {
	timestamp := fmt.Sprintf("to_timestamp(%s / 1000000000)", timeColumn)

	if intervalVal == 1 {
		return fmt.Sprintf("date_trunc('%s', %s)", intervalUnit, timestamp)
	}

	interval := fmt.Sprintf("%d %s", intervalVal, intervalUnit+"s")

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

func buildHavingCondition(qp QueryParams) string {
	if len(qp.AggField) == 0 {
		return "1=1"
	}

	var conditions []string
	switch qp.Table {
	case senmlTable:
		conditions = append(conditions, "MAX(m.value) IS NOT NULL")
	default:
		for _, field := range qp.AggField {
			jsonPath := buildJSONPath(field)
			conditions = append(conditions,
				fmt.Sprintf("MAX(CAST(m.%s AS FLOAT)) IS NOT NULL", jsonPath))
		}
	}

	return strings.Join(conditions, " OR ")
}

func buildHavingConditionForCount(aggFields []string, table string) string {
	if len(aggFields) == 0 {
		return "1=1"
	}

	var conditions []string
	switch table {
	case senmlTable:
		conditions = append(conditions, "MAX(m.value) IS NOT NULL")
	default:
		for _, field := range aggFields {
			jsonPath := buildJSONPath(field)
			conditions = append(conditions,
				fmt.Sprintf("MAX(CAST(m.%s AS FLOAT)) IS NOT NULL", jsonPath))
		}
	}

	return strings.Join(conditions, " OR ")
}

func buildAggregatedJSONSelectMultipleFromIA(aggFields []string, aggPrefix string) string {
	if len(aggFields) == 0 {
		return "ia.max_time as created, ia.subtopic, ia.publisher, ia.protocol, '{}'::jsonb as payload"
	}

	var jsonbPairs []string
	for i, field := range aggFields {
		parts := strings.Split(field, ".")
		key := parts[len(parts)-1]
		jsonbPairs = append(jsonbPairs, fmt.Sprintf("'%s', ia.%s_%d", key, aggPrefix, i))
	}

	return fmt.Sprintf(`ia.max_time as created, ia.subtopic, ia.publisher, ia.protocol,
		jsonb_build_object(%s) as payload`, strings.Join(jsonbPairs, ", "))
}
