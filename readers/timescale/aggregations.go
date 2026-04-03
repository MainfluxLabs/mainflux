// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package timescale

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

var validFieldName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.]*$`)

const (
	jsonTable  = "json"
	jsonOrder  = "created"
	senmlTable = "senml"
	senmlOrder = "time"
)

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

	condition := dbutil.BuildWhereClause(jsonConditions(rpm)...)
	bucket := timeBucketExpr(rpm.AggValue, rpm.AggInterval, jsonOrder)
	aggExpr, err := jsonAggExpr(rpm.AggType, rpm.AggFields)

	if err != nil {
		return []readers.Message{}, 0, errors.Wrap(readers.ErrReadMessages, err)
	}

	if aggExpr == "" {
		return []readers.Message{}, 0, nil
	}

	selectFields, err := jsonSelectFields(rpm.AggFields)
	if err != nil {
		return []readers.Message{}, 0, errors.Wrap(readers.ErrReadMessages, err)
	}

	having, err := jsonHaving(rpm.AggFields)
	if err != nil {
		return []readers.Message{}, 0, errors.Wrap(readers.ErrReadMessages, err)
	}

	dir := dbutil.GetDirQuery(rpm.Dir)
	olq := dbutil.GetOffsetLimitQuery(rpm.Limit)

	query := fmt.Sprintf(`SELECT %s FROM (
          SELECT %s AS bucket, %s,
                  MAX(%s) AS max_time,
                  MAX(CAST(subtopic AS text)) AS subtopic,
                  MAX(CAST(publisher AS text)) AS publisher,
                  MAX(CAST(protocol AS text)) AS protocol
          FROM %s %s
          GROUP BY bucket
          HAVING %s
          ORDER BY bucket %s) agg %s;`,
		selectFields, bucket, aggExpr, jsonOrder, jsonTable, condition, having, dir, olq)

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM (
          SELECT %s AS bucket
          FROM %s %s
          GROUP BY bucket
          HAVING %s) counted;`,
		bucket, jsonTable, condition, having)

	return as.executeAggQuery(ctx, query, countQuery, params, jsonTable)
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

	condition := dbutil.BuildWhereClause(senmlConditions(rpm)...)
	bucket := timeBucketExpr(rpm.AggValue, rpm.AggInterval, senmlOrder)
	aggFunc := sqlAggFunc(rpm.AggType)
	if aggFunc == "" {
		return []readers.Message{}, 0, nil
	}
	dir := dbutil.GetDirQuery(rpm.Dir)
	olq := dbutil.GetOffsetLimitQuery(rpm.Limit)

	query := fmt.Sprintf(`SELECT
          MAX(time) AS time, MAX(CAST(subtopic AS text)) AS subtopic,
          MAX(CAST(publisher AS text)) AS publisher, MAX(CAST(protocol AS text)) AS protocol,
          '' AS name, '' AS unit,
          %s(value) AS value,
          CAST(NULL AS text) AS string_value, CAST(NULL AS bool) AS bool_value, CAST(NULL AS text) AS data_value,
          CAST(NULL AS float) AS sum, MAX(update_time) AS update_time
          FROM %s %s
          GROUP BY %s
          HAVING MAX(value) IS NOT NULL
          ORDER BY %s %s %s;`,
		aggFunc, senmlTable, condition, bucket, bucket, dir, olq)

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM (
          SELECT %s AS bucket
          FROM %s %s
          GROUP BY bucket
          HAVING MAX(value) IS NOT NULL) counted;`,
		bucket, senmlTable, condition)

	return as.executeAggQuery(ctx, query, countQuery, params, senmlTable)
}

func (as *aggregationService) executeAggQuery(ctx context.Context, query, countQuery string, params map[string]any, table string) ([]readers.Message, uint64, error) {
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

	messages, err := scanAggregatedMessages(rows, table)
	if err != nil {
		return []readers.Message{}, 0, err
	}

	total, err := dbutil.Total(ctx, as.db, countQuery, params)
	if err != nil {
		return []readers.Message{}, 0, err
	}

	return messages, total, nil
}

func scanAggregatedMessages(rows *sqlx.Rows, table string) ([]readers.Message, error) {
	messages := []readers.Message{}

	switch table {
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
			msg := json.Message{}
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

func timeBucketExpr(intervalVal uint64, intervalUnit, timeColumn string) string {
	interval := fmt.Sprintf("%d %s", intervalVal, intervalUnit)
	return fmt.Sprintf("time_bucket('%s', to_timestamp(%s / 1000000000))", interval, timeColumn)
}

func sqlAggFunc(aggType string) string {
	switch aggType {
	case readers.AggregationMax:
		return strings.ToUpper(readers.AggregationMax)
	case readers.AggregationMin:
		return strings.ToUpper(readers.AggregationMin)
	case readers.AggregationAvg:
		return strings.ToUpper(readers.AggregationAvg)
	case readers.AggregationCount:
		return strings.ToUpper(readers.AggregationCount)
	default:
		return ""
	}
}

func jsonAggExpr(aggType string, aggFields []string) (string, error) {
	fn := sqlAggFunc(aggType)
	if fn == "" || len(aggFields) == 0 {
		return "", nil
	}

	var exprs []string
	for i, field := range aggFields {
		jsonPath, err := buildJSONPath(field)
		if err != nil {
			return "", err

		}
		if fn == strings.ToUpper(readers.AggregationCount) {
			exprs = append(exprs, fmt.Sprintf("%s(%s) AS agg_value_%d", fn, jsonPath, i))
		} else {
			exprs = append(exprs, fmt.Sprintf("%s(CAST(%s AS FLOAT)) AS agg_value_%d", fn, jsonPath, i))
		}
	}
	return strings.Join(exprs, ", "), nil
}

func jsonSelectFields(aggFields []string) (string, error) {
	var pairs []string
	for i, field := range aggFields {
		if !validFieldName.MatchString(field) {
			return "", fmt.Errorf("invalid field name: %s", field)
		}
		pairs = append(pairs, fmt.Sprintf("'%s', agg.agg_value_%d", field, i))
	}

	return fmt.Sprintf(`agg.max_time AS created, agg.subtopic, agg.publisher, agg.protocol,
          jsonb_build_object(%s) AS payload`, strings.Join(pairs, ", ")), nil
}

func jsonHaving(aggFields []string) (string, error) {
	if len(aggFields) == 0 {
		return "1=1", nil
	}

	var conditions []string
	for _, field := range aggFields {
		jsonPath, err := buildJSONPath(field)
		if err != nil {
			return "", err
		}
		conditions = append(conditions, fmt.Sprintf("MAX(CAST(%s AS FLOAT)) IS NOT NULL", jsonPath))
	}
	return strings.Join(conditions, " OR "), nil
}

func buildJSONPath(field string) (string, error) {
	if !validFieldName.MatchString(field) {
		return "", fmt.Errorf("invalid field name: %s", field)
	}

	parts := strings.Split(field, ".")
	if len(parts) == 1 {
		return fmt.Sprintf("payload->>'%s'", parts[0]), nil
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
	return path.String(), nil
}

func jsonConditions(rpm readers.JSONPageMetadata) []string {
	return baseConditions(rpm.Subtopic, rpm.Publisher, rpm.Protocol, rpm.From, rpm.To, jsonOrder)
}

func senmlConditions(rpm readers.SenMLPageMetadata) []string {
	conds := baseConditions(rpm.Subtopic, rpm.Publisher, rpm.Protocol, rpm.From, rpm.To, senmlOrder)

	if rpm.Name != "" {
		conds = append(conds, "name = :name")
	}
	if rpm.Value != 0 {
		conds = append(conds, fmt.Sprintf("value %s :value", readers.ComparatorSymbol(rpm.Comparator)))
	}
	if rpm.BoolValue {
		conds = append(conds, "bool_value = :bool_value")
	}
	if rpm.StringValue != "" {
		conds = append(conds, "string_value = :string_value")
	}
	if rpm.DataValue != "" {
		conds = append(conds, "data_value = :data_value")
	}
	return conds

}

func baseConditions(subtopic, publisher, protocol string, from, to int64, timeColumn string) []string {
	var conds []string
	if subtopic != "" {
		conds = append(conds, "subtopic = :subtopic")
	}
	if publisher != "" {
		conds = append(conds, "publisher = :publisher")
	}
	if protocol != "" {
		conds = append(conds, "protocol = :protocol")
	}
	if from != 0 {
		conds = append(conds, fmt.Sprintf("%s >= :from", timeColumn))
	}
	if to != 0 {
		conds = append(conds, fmt.Sprintf("%s <= :to", timeColumn))
	}
	return conds
}
