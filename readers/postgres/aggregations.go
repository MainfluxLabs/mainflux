// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package postgres

import (
	"fmt"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

type aggregationService struct {
	db *sqlx.DB
}

func newAggregationService(db *sqlx.DB) *aggregationService {
	return &aggregationService{db: db}
}

func (as *aggregationService) readAggregatedMessages(rpm readers.PageMetadata) ([]readers.Message, error) {
	format, _ := as.getFormatAndOrder(rpm)
	params := as.buildQueryParams(rpm)

	query := as.buildAggregationQuery(rpm, format)
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

func (as *aggregationService) buildAggregationQuery(rpm readers.PageMetadata, format string) string {
	interval := rpm.AggInterval
	aggField := as.getAggregateField(rpm, format)
	timeColumn := as.getTimeColumn(format)
	condition := as.buildCondition(rpm, format)
	limit := rpm.Limit
	conditionForJoin := strings.Replace(condition, "WHERE", "AND", 1)
	if condition == "" {
		conditionForJoin = ""
	}

	switch rpm.AggType {
	case readers.AggregationMax:
		return as.buildMinMaxQuery(format, timeColumn, aggField, condition, conditionForJoin, interval, limit, strings.ToUpper(readers.AggregationMax))

	case readers.AggregationMin:
		return as.buildMinMaxQuery(format, timeColumn, aggField, condition, conditionForJoin, interval, limit, strings.ToUpper(readers.AggregationMin))

	case readers.AggregationAvg:
		return as.buildAvgQuery(format, timeColumn, aggField, condition, conditionForJoin, interval, limit)

	case readers.AggregationCount:
		return as.buildCountQuery(format, timeColumn, aggField, condition, conditionForJoin, interval, limit)
	}

	return ""
}

func (as *aggregationService) buildMinMaxQuery(format, timeColumn, aggField, condition, conditionForJoin, interval string, limit uint64, aggFunc string) string {
	switch format {
	case defTable:
		return fmt.Sprintf(`
			WITH time_intervals AS (
				SELECT generate_series(
					date_trunc('%s', NOW() - interval '%d %s'),
					date_trunc('%s', NOW()),
					interval '1 %s'
				) as interval_time
				ORDER BY interval_time DESC
				LIMIT %d
			),
			interval_aggs AS (
				SELECT 
					ti.interval_time,
					%s(m.%s) as agg_value
				FROM time_intervals ti
				LEFT JOIN %s m ON date_trunc('%s', to_timestamp(m.%s / 1000000000)) = ti.interval_time
					%s
				GROUP BY ti.interval_time
				HAVING %s(m.%s) IS NOT NULL
			)
			SELECT DISTINCT ON (ia.interval_time) m.*
			FROM %s m
			JOIN interval_aggs ia ON date_trunc('%s', to_timestamp(m.%s / 1000000000)) = ia.interval_time
				AND m.%s = ia.agg_value
			%s
			ORDER BY ia.interval_time DESC, m.%s DESC;`,
			interval, limit, interval, interval, interval, limit,
			aggFunc, aggField, format, interval, timeColumn, conditionForJoin,
			aggFunc, aggField, format, interval, timeColumn, aggField, condition, timeColumn)
	default:
		return fmt.Sprintf(`
			WITH time_intervals AS (
				SELECT generate_series(
					date_trunc('%s', NOW() - interval '%d %s'),
					date_trunc('%s', NOW()),
					interval '1 %s'
				) as interval_time
				ORDER BY interval_time DESC
				LIMIT %d
			),
			interval_aggs AS (
				SELECT 
					ti.interval_time,
					%s((m.payload->>'%s')::float) as agg_value
				FROM time_intervals ti
				LEFT JOIN %s m ON date_trunc('%s', to_timestamp(m.created / 1000000000)) = ti.interval_time
					%s
				GROUP BY ti.interval_time
				HAVING %s((m.payload->>'%s')::float) IS NOT NULL
			)
			SELECT DISTINCT ON (ia.interval_time) m.*
			FROM %s m
			JOIN interval_aggs ia ON date_trunc('%s', to_timestamp(m.created / 1000000000)) = ia.interval_time
				AND (m.payload->>'%s')::float = ia.agg_value
			%s
			ORDER BY ia.interval_time DESC, m.created DESC;`,
			interval, limit, interval, interval, interval, limit,
			aggFunc, aggField, format, interval, conditionForJoin,
			aggFunc, aggField, format, interval, aggField, condition)
	}
}

func (as *aggregationService) buildAvgQuery(format, timeColumn, aggField, condition, conditionForJoin, interval string, limit uint64) string {
	switch format {
	case defTable:
		return fmt.Sprintf(`
			WITH time_intervals AS (
				SELECT generate_series(
					date_trunc('%s', NOW() - interval '%d %s'),
					date_trunc('%s', NOW()),
					interval '1 %s'
				) as interval_time
				ORDER BY interval_time DESC
				LIMIT %d
			),
			interval_aggs AS (
				SELECT 
					ti.interval_time,
					AVG(m.%s) as avg_value
				FROM time_intervals ti
				LEFT JOIN %s m ON date_trunc('%s', to_timestamp(m.%s / 1000000000)) = ti.interval_time
					%s
				GROUP BY ti.interval_time
				HAVING AVG(m.%s) IS NOT NULL
			)
			SELECT DISTINCT ON (ia.interval_time) m.*
			FROM %s m
			JOIN interval_aggs ia ON date_trunc('%s', to_timestamp(m.%s / 1000000000)) = ia.interval_time
			%s
			ORDER BY ia.interval_time DESC, ABS(m.%s - ia.avg_value) ASC;`,
			interval, limit, interval, interval, interval, limit,
			aggField, format, interval, timeColumn, conditionForJoin,
			aggField, format, interval, timeColumn, condition, aggField)
	default:
		return fmt.Sprintf(`
			WITH time_intervals AS (
				SELECT generate_series(
					date_trunc('%s', NOW() - interval '%d %s'),
					date_trunc('%s', NOW()),
					interval '1 %s'
				) as interval_time
				ORDER BY interval_time DESC
				LIMIT %d
			),
			interval_aggs AS (
				SELECT 
					ti.interval_time,
					AVG((m.payload->>'%s')::float) as avg_value
				FROM time_intervals ti
				LEFT JOIN %s m ON date_trunc('%s', to_timestamp(m.created / 1000000000)) = ti.interval_time
					%s
				GROUP BY ti.interval_time
				HAVING AVG((m.payload->>'%s')::float) IS NOT NULL
			)
			SELECT DISTINCT ON (ia.interval_time) m.*
			FROM %s m
			JOIN interval_aggs ia ON date_trunc('%s', to_timestamp(m.created / 1000000000)) = ia.interval_time
			%s
			ORDER BY ia.interval_time DESC, ABS((m.payload->>'%s')::float - ia.avg_value) ASC;`,
			interval, limit, interval, interval, interval, limit,
			aggField, format, interval, conditionForJoin,
			aggField, format, interval, condition, aggField)
	}
}

func (as *aggregationService) buildCountQuery(format, timeColumn, aggField, condition, conditionForJoin, interval string, limit uint64) string {
	switch format {
	case defTable:
		return fmt.Sprintf(`
			WITH time_intervals AS (
				SELECT generate_series(
					date_trunc('%s', NOW() - interval '%d %s'),
					date_trunc('%s', NOW()),
					interval '1 %s'
				) as interval_time
				ORDER BY interval_time DESC
				LIMIT %d
			),
			interval_aggs AS (
				SELECT 
					ti.interval_time,
					SUM(m.%s) as sum_value
				FROM time_intervals ti
				LEFT JOIN %s m ON date_trunc('%s', to_timestamp(m.%s / 1000000000)) = ti.interval_time
					%s
				GROUP BY ti.interval_time
				HAVING SUM(m.%s) IS NOT NULL
			)
			SELECT DISTINCT ON (ia.interval_time) m.*
			FROM %s m
			JOIN interval_aggs ia ON date_trunc('%s', to_timestamp(m.%s / 1000000000)) = ia.interval_time
			%s
			ORDER BY ia.interval_time DESC, ABS(m.%s - ia.sum_value) ASC;`,
			interval, limit, interval, interval, interval, limit,
			aggField, format, interval, timeColumn, conditionForJoin,
			aggField, format, interval, timeColumn, condition, aggField)
	default:
		return fmt.Sprintf(`
			WITH time_intervals AS (
				SELECT generate_series(
					date_trunc('%s', NOW() - interval '%d %s'),
					date_trunc('%s', NOW()),
					interval '1 %s'
				) as interval_time
				ORDER BY interval_time DESC
				LIMIT %d
			),
			interval_aggs AS (
				SELECT 
					ti.interval_time,
					SUM((m.payload->>'%s')::float) as sum_value
				FROM time_intervals ti
				LEFT JOIN %s m ON date_trunc('%s', to_timestamp(m.created / 1000000000)) = ti.interval_time
					%s
				GROUP BY ti.interval_time
				HAVING SUM((m.payload->>'%s')::float) IS NOT NULL
			)
			SELECT DISTINCT ON (ia.interval_time) m.*
			FROM %s m
			JOIN interval_aggs ia ON date_trunc('%s', to_timestamp(m.created / 1000000000)) = ia.interval_time
			%s
			ORDER BY ia.interval_time DESC, ABS((m.payload->>'%s')::float - ia.sum_value) ASC;`,
			interval, limit, interval, interval, interval, limit,
			aggField, format, interval, conditionForJoin,
			aggField, format, interval, condition, aggField)
	}
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

func (as *aggregationService) buildCondition(rpm readers.PageMetadata, table string) string {
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
	if rpm.Name != "" {
		condition = fmt.Sprintf(`%s %s name = :name`, condition, op)
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
