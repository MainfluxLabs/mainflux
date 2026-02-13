// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"fmt"
	"strings"

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
	jsonTable  = "json"
	jsonOrder  = "created"
	senmlTable = "senml"
	senmlOrder = "time"
)

// queryParams holds parameters for building aggregation SQL queries.
type queryParams struct {
	table            string
	timeColumn       string
	condition        string
	conditionForJoin string
	limit            uint64
	aggInterval      string
	aggValue         uint64
	aggFields        []string
	aggType          string
	dir              string
}

type aggStrategy interface {
	selectedFields(qp queryParams) string
	aggregateExpr(qp queryParams) string
}

type aggregationService struct {
	db dbutil.Database
}

func newAggregationService(db dbutil.Database) *aggregationService {
	return &aggregationService{db: db}
}

// aggInput holds the data needed to execute an aggregation query.
type aggInput struct {
	params     map[string]any
	qp         queryParams
	conditions []string
}

func (as *aggregationService) readAggregatedJSONMessages(ctx context.Context, rpm readers.JSONPageMetadata) ([]readers.Message, uint64, error) {
	input := aggInput{
		params: map[string]any{
			"limit":     rpm.Limit,
			"offset":    rpm.Offset,
			"subtopic":  rpm.Subtopic,
			"publisher": rpm.Publisher,
			"protocol":  rpm.Protocol,
			"from":      rpm.From,
			"to":        rpm.To,
		},
		qp: queryParams{
			table:       jsonTable,
			timeColumn:  jsonOrder,
			aggFields:   rpm.AggFields,
			aggInterval: rpm.AggInterval,
			aggValue:    rpm.AggValue,
			aggType:     rpm.AggType,
			limit:       rpm.Limit,
			dir:         rpm.Dir,
		},
		conditions: jsonConditions(rpm),
	}

	return as.readAggregatedMessages(ctx, input)
}

func (as *aggregationService) readAggregatedSenMLMessages(ctx context.Context, rpm readers.SenMLPageMetadata) ([]readers.Message, uint64, error) {
	input := aggInput{
		params: map[string]any{
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
		},
		qp: queryParams{
			table:       senmlTable,
			timeColumn:  senmlOrder,
			aggFields:   rpm.AggFields,
			aggInterval: rpm.AggInterval,
			aggValue:    rpm.AggValue,
			aggType:     rpm.AggType,
			limit:       rpm.Limit,
			dir:         rpm.Dir,
		},
		conditions: senmlConditions(rpm),
	}

	return as.readAggregatedMessages(ctx, input)
}

func (as *aggregationService) readAggregatedMessages(ctx context.Context, input aggInput) ([]readers.Message, uint64, error) {
	qp := input.qp
	if len(input.conditions) > 0 {
		qp.condition = "WHERE " + strings.Join(input.conditions, " AND ")
		qp.conditionForJoin = "AND " + strings.Join(input.conditions, " AND ")
	}

	strategy := aggStrategyFor(qp.aggType)
	if strategy == nil {
		return []readers.Message{}, 0, nil
	}

	query := buildAggQuery(qp, strategy)
	rows, err := as.db.NamedQueryContext(ctx, query, input.params)
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

	messages, err := as.scanAggregatedMessages(rows, qp.table)
	if err != nil {
		return []readers.Message{}, 0, err
	}

	cq := buildAggCountQuery(qp)
	total, err := dbutil.Total(ctx, as.db, cq, input.params)
	if err != nil {
		return []readers.Message{}, 0, err
	}

	return messages, total, nil
}

func buildAggQuery(qp queryParams, strategy aggStrategy) string {
	return fmt.Sprintf(`
		WITH time_intervals AS (
			%s
		),
		interval_aggs AS (
			SELECT
				ti.interval_time,
				%s,
				MAX(m.%s) as max_time,
				MAX(CAST(m.subtopic AS text)) as subtopic,
				MAX(CAST(m.publisher AS text)) as publisher,
				MAX(CAST(m.protocol AS text)) as protocol
			FROM time_intervals ti
			LEFT JOIN %s m ON %s
				%s
			GROUP BY ti.interval_time
			HAVING %s
		)
		SELECT %s
		FROM interval_aggs ia
		ORDER BY ia.interval_time %s;`,
		buildTimeIntervals(qp),
		strategy.aggregateExpr(qp),
		qp.timeColumn,
		qp.table,
		buildTimeJoinCondition(qp),
		qp.conditionForJoin,
		buildConditionForCount(qp),
		strategy.selectedFields(qp),
		dbutil.GetDirQuery(qp.dir),
	)
}

func buildAggCountQuery(qp queryParams) string {
	intervals := buildTimeIntervals(qp)
	joinCond := buildTimeJoinCondition(qp)
	havingCond := buildConditionForCount(qp)

	return fmt.Sprintf(`
		WITH time_intervals AS (%s)
		SELECT COUNT(*) FROM (
			SELECT ti.interval_time
			FROM time_intervals ti
			LEFT JOIN %s m ON %s
				%s
			GROUP BY ti.interval_time
			HAVING %s
		) counted`,
		intervals, qp.table, joinCond, qp.conditionForJoin, havingCond)
}

// sqlAggFunc implements aggStrategy for SQL aggregate functions.
type sqlAggFunc string

func aggStrategyFor(aggType string) aggStrategy {
	switch aggType {
	case readers.AggregationMax:
		return sqlAggFunc("MAX")
	case readers.AggregationMin:
		return sqlAggFunc("MIN")
	case readers.AggregationAvg:
		return sqlAggFunc("AVG")
	case readers.AggregationCount:
		return sqlAggFunc("COUNT")
	default:
		return nil
	}
}

func (f sqlAggFunc) selectedFields(qp queryParams) string {
	if qp.table == senmlTable {
		return `ia.max_time as time, ia.subtopic, ia.publisher, ia.protocol,
		'' as name, '' as unit,
		ia.agg_value as value,
		'' as string_value, false as bool_value, '' as data_value,
		0 as sum, ia.max_time as update_time`
	}
	return buildJSONSelect(qp.aggFields)
}

func (f sqlAggFunc) aggregateExpr(qp queryParams) string {
	if len(qp.aggFields) == 0 {
		return ""
	}

	fn := string(f)
	var exprs []string
	switch qp.table {
	case senmlTable:
		exprs = append(exprs, fmt.Sprintf("%s(m.value) as agg_value", fn))
	default:
		for i, field := range qp.aggFields {
			jsonPath := buildJSONPath(field)
			if fn == "COUNT" {
				exprs = append(exprs, fmt.Sprintf("%s(m.%s) as agg_value_%d", fn, jsonPath, i))
			} else {
				exprs = append(exprs, fmt.Sprintf("%s(CAST(m.%s as FLOAT)) as agg_value_%d", fn, jsonPath, i))
			}
		}
	}
	return strings.Join(exprs, ",\n\t\t\t\t")
}

func buildTimeIntervals(qp queryParams) string {
	timeTrunc := buildTruncTimeExpression(qp.aggValue, qp.aggInterval, qp.timeColumn)
	dir := dbutil.GetDirQuery(qp.dir)

	q := fmt.Sprintf(`
        SELECT DISTINCT %s as interval_time
        FROM %s
        %s
        ORDER BY interval_time %s`,
		timeTrunc, qp.table, qp.condition, dir)

	if qp.limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", qp.limit)
	}
	return q
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

func buildTimeJoinCondition(qp queryParams) string {
	timeTrunc := buildTruncTimeExpression(qp.aggValue, qp.aggInterval, "m."+qp.timeColumn)
	return fmt.Sprintf("%s = ti.interval_time", timeTrunc)
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
			fmt.Fprintf(&path, "->>'%s'", part)
		} else {
			fmt.Fprintf(&path, "->'%s'", part)
		}
	}
	return path.String()
}

// baseFilter holds fields shared between JSON and SenML page metadata.
type baseFilter struct {
	subtopic   string
	publisher  string
	protocol   string
	from       int64
	to         int64
	timeColumn string
}

func (f baseFilter) conditions() []string {
	var conds []string
	if f.subtopic != "" {
		conds = append(conds, "subtopic = :subtopic")
	}
	if f.publisher != "" {
		conds = append(conds, "publisher = :publisher")
	}
	if f.protocol != "" {
		conds = append(conds, "protocol = :protocol")
	}
	if f.from != 0 {
		conds = append(conds, fmt.Sprintf("%s >= :from", f.timeColumn))
	}
	if f.to != 0 {
		conds = append(conds, fmt.Sprintf("%s <= :to", f.timeColumn))
	}
	return conds
}

func jsonConditions(pm readers.JSONPageMetadata) []string {
	return baseFilter{
		subtopic:   pm.Subtopic,
		publisher:  pm.Publisher,
		protocol:   pm.Protocol,
		from:       pm.From,
		to:         pm.To,
		timeColumn: jsonOrder,
	}.conditions()
}

func senmlConditions(pm readers.SenMLPageMetadata) []string {
	conds := baseFilter{
		subtopic:   pm.Subtopic,
		publisher:  pm.Publisher,
		protocol:   pm.Protocol,
		from:       pm.From,
		to:         pm.To,
		timeColumn: senmlOrder,
	}.conditions()

	if pm.Name != "" {
		conds = append(conds, "name = :name")
	}
	if pm.Value != 0 {
		conds = append(conds, fmt.Sprintf("value %s :value", readers.ComparatorSymbol(pm.Comparator)))
	}
	if pm.BoolValue {
		conds = append(conds, "bool_value = :bool_value")
	}
	if pm.StringValue != "" {
		conds = append(conds, "string_value = :string_value")
	}
	if pm.DataValue != "" {
		conds = append(conds, "data_value = :data_value")
	}
	return conds
}

func buildConditionForCount(qp queryParams) string {
	if len(qp.aggFields) == 0 {
		return "1=1"
	}

	var conditions []string
	switch qp.table {
	case senmlTable:
		conditions = append(conditions, "MAX(m.value) IS NOT NULL")
	default:
		for _, field := range qp.aggFields {
			jsonPath := buildJSONPath(field)
			conditions = append(conditions,
				fmt.Sprintf("MAX(CAST(m.%s AS FLOAT)) IS NOT NULL", jsonPath))
		}
	}

	return strings.Join(conditions, " OR ")
}

func buildJSONSelect(aggFields []string) string {
	if len(aggFields) == 0 {
		return "ia.max_time as created, ia.subtopic, ia.publisher, ia.protocol, CAST('{}' AS jsonb) as payload"
	}

	var pairs []string
	for i, field := range aggFields {
		pairs = append(pairs, fmt.Sprintf("'%s', ia.agg_value_%d", field, i))
	}

	return fmt.Sprintf(`ia.max_time as created, ia.subtopic, ia.publisher, ia.protocol,
		jsonb_build_object(%s) as payload`, strings.Join(pairs, ", "))
}
