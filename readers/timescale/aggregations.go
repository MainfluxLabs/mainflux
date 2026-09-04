// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package timescale

import (
	"context"
	"fmt"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	mfreaders "github.com/MainfluxLabs/mainflux/pkg/readers"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

func escapeFieldName(field string) (string, error) {
	if field == "" || strings.ContainsRune(field, 0) {
		return "", fmt.Errorf("invalid field name: %s", field)
	}
	return strings.ReplaceAll(field, "'", "''"), nil
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

	condition := dbutil.BuildWhereClause(mfreaders.BaseConditions(rpm.MessagesPageMetadata, mfreaders.JSONOrder)...)
	bucket := timeBucketExpr(rpm.AggValue, rpm.AggInterval, mfreaders.JSONOrder)
	aggExpr, err := jsonAggExpr(rpm.AggType, rpm.AggFields)
	if err != nil {
		return []readers.Message{}, 0, errors.Wrap(readers.ErrReadMessages, err)
	}

	if aggExpr == "" {
		return []readers.Message{}, 0, nil
	}

	selectFields, err := jsonSelectFields(rpm.AggFields, rpm.AggType)
	if err != nil {
		return []readers.Message{}, 0, errors.Wrap(readers.ErrReadMessages, err)
	}

	having, err := jsonFilterNullFields(rpm.AggFields, rpm.AggType)
	if err != nil {
		return []readers.Message{}, 0, errors.Wrap(readers.ErrReadMessages, err)
	}

	dir := dbutil.GetDirQuery(rpm.Dir)
	olq := dbutil.GetOffsetLimitQuery(rpm.Limit)

	subquery := fmt.Sprintf(`SELECT %s AS bucket, %s%s
          FROM %s %s
          GROUP BY bucket
          HAVING %s`,
		bucket, aggExpr, jsonBucketColumns(rpm.AggType), mfreaders.JSONTable, condition, having)

	query := fmt.Sprintf(`SELECT %s FROM (%s ORDER BY bucket %s) agg %s;`, selectFields, subquery, dir, olq)

	messages, err := as.executeAggQuery(ctx, query, params, mfreaders.JSONTable)
	if err != nil {
		return []readers.Message{}, 0, err
	}

	if rpm.NoTotal {
		return messages, 0, nil
	}

	total, err := as.countAgg(ctx, subquery, params)
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

	condition := dbutil.BuildWhereClause(mfreaders.SenMLConditions(rpm)...)
	bucket := timeBucketExpr(rpm.AggValue, rpm.AggInterval, mfreaders.SenMLOrder)
	aggFunc := sqlAggFunc(rpm.AggType)
	if aggFunc == "" && !isFirstLast(rpm.AggType) {
		return []readers.Message{}, 0, nil
	}
	dir := dbutil.GetDirQuery(rpm.Dir)
	olq := dbutil.GetOffsetLimitQuery(rpm.Limit)

	subquery := senmlAggSubquery(aggFunc, condition, bucket)
	if isFirstLast(rpm.AggType) {
		subquery = senmlFirstLastSubquery(aggOrderDir(rpm.AggType), condition, bucket)
	}

	query := fmt.Sprintf(`%s ORDER BY %s %s %s;`, subquery, bucket, dir, olq)

	messages, err := as.executeAggQuery(ctx, query, params, mfreaders.SenMLTable)
	if err != nil {
		return []readers.Message{}, 0, err
	}

	if rpm.NoTotal {
		return messages, 0, nil
	}

	total, err := as.countAgg(ctx, subquery, params)
	if err != nil {
		return []readers.Message{}, 0, err
	}

	return messages, total, nil
}

func senmlAggSubquery(aggFunc, condition, bucket string) string {
	return fmt.Sprintf(`SELECT
          MAX(time) AS time, MAX(CAST(subtopic AS text)) AS subtopic,
          MAX(CAST(publisher AS text)) AS publisher, MAX(CAST(protocol AS text)) AS protocol,
          '' AS name, '' AS unit,
          %s(value) AS value,
          CAST(NULL AS text) AS string_value, CAST(NULL AS bool) AS bool_value, CAST(NULL AS text) AS data_value,
          CAST(NULL AS float) AS sum, MAX(update_time) AS update_time
          FROM %s %s
          GROUP BY %s
          HAVING MAX(value) IS NOT NULL`,
		aggFunc, mfreaders.SenMLTable, condition, bucket)
}

func senmlFirstLastSubquery(dir, condition, bucket string) string {
	col := mfreaders.SenMLOrder
	pick := func(field string) string {
		return fmt.Sprintf("(array_agg(%s ORDER BY %s %s))[1]", field, col, dir)
	}

	return fmt.Sprintf(`SELECT
          %s AS time, %s AS subtopic,
          %s AS publisher, %s AS protocol,
          %s AS name, COALESCE(%s, '') AS unit,
          %s AS value,
          %s AS string_value, %s AS bool_value, %s AS data_value,
          %s AS sum, COALESCE(%s, 0) AS update_time
          FROM %s %s
          GROUP BY %s`,
		pick(col), pick("subtopic"), pick("publisher"), pick("protocol"),
		pick("name"), pick("unit"), pick("value"),
		pick("string_value"), pick("bool_value"), pick("data_value"),
		pick("sum"), pick("update_time"),
		mfreaders.SenMLTable, condition, bucket)
}

func (as *aggregationService) countAgg(ctx context.Context, subquery string, params map[string]any) (uint64, error) {
	cq := fmt.Sprintf(`SELECT COUNT(*) FROM (%s) agg;`, subquery)
	return dbutil.Total(ctx, as.db, cq, params)
}

func (as *aggregationService) executeAggQuery(ctx context.Context, query string, params map[string]any, table string) ([]readers.Message, error) {
	rows, err := as.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.UndefinedTable {
			return []readers.Message{}, nil
		}
		return []readers.Message{}, errors.Wrap(readers.ErrReadMessages, err)
	}

	if rows == nil {
		return []readers.Message{}, nil
	}
	defer rows.Close()

	return scanAggregatedMessages(rows, table)
}

func scanAggregatedMessages(rows *sqlx.Rows, table string) ([]readers.Message, error) {
	messages := []readers.Message{}

	switch table {
	case mfreaders.SenMLTable:
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

func isFirstLast(aggType string) bool {
	return aggType == readers.AggregationFirst || aggType == readers.AggregationLast
}

func aggOrderDir(aggType string) string {
	if aggType == readers.AggregationLast {
		return "DESC"
	}
	return "ASC"
}

func jsonBucketColumns(aggType string) string {
	if isFirstLast(aggType) {
		return ""
	}

	return fmt.Sprintf(`,
                  MAX(%s) AS max_time,
                  MAX(CAST(subtopic AS text)) AS subtopic,
                  MAX(CAST(publisher AS text)) AS publisher,
                  MAX(CAST(protocol AS text)) AS protocol`, mfreaders.JSONOrder)
}

func jsonAggExpr(aggType string, aggFields []string) (string, error) {
	// first and last pick a whole row, so they stay valid without agg fields.
	if isFirstLast(aggType) {
		col, dir := mfreaders.JSONOrder, aggOrderDir(aggType)
		pick := func(field string) string {
			return fmt.Sprintf("(array_agg(%s ORDER BY %s %s))[1]", field, col, dir)
		}
		payload := fmt.Sprintf("(array_agg(COALESCE(payload, CAST('{}' AS jsonb)) ORDER BY %s %s))[1]", col, dir)

		return fmt.Sprintf(
			`%s AS agg_payload, %s AS agg_time, `+
				`%s AS agg_subtopic, %s AS agg_publisher, %s AS agg_protocol`,
			payload, pick(col), pick("subtopic"), pick("publisher"), pick("protocol")), nil
	}

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

func jsonSelectFields(aggFields []string, aggType string) (string, error) {
	if isFirstLast(aggType) {
		const head = `agg.agg_time AS created, agg.agg_subtopic AS subtopic,
          agg.agg_publisher AS publisher, agg.agg_protocol AS protocol,`
		if len(aggFields) == 0 {
			return head + " agg.agg_payload AS payload", nil
		}

		var pairs []string
		for _, field := range aggFields {
			escaped, err := escapeFieldName(field)
			if err != nil {
				return "", err
			}
			// -> not ->>, so numbers, bools and objects keep their JSON type.
			pairs = append(pairs, fmt.Sprintf("'%s', agg.agg_payload->'%s'", escaped, escaped))
		}
		return fmt.Sprintf("%s\n          jsonb_build_object(%s) AS payload", head, strings.Join(pairs, ", ")), nil
	}

	var pairs []string
	for i, field := range aggFields {
		escaped, err := escapeFieldName(field)
		if err != nil {
			return "", err
		}
		pairs = append(pairs, fmt.Sprintf("'%s', agg.agg_value_%d", escaped, i))
	}

	return fmt.Sprintf(`agg.max_time AS created, agg.subtopic, agg.publisher, agg.protocol,
          jsonb_build_object(%s) AS payload`, strings.Join(pairs, ", ")), nil
}

func jsonFilterNullFields(aggFields []string, aggType string) (string, error) {
	if len(aggFields) == 0 {
		return "1=1", nil
	}

	var conditions []string
	for _, field := range aggFields {
		jsonPath, err := buildJSONPath(field)
		if err != nil {
			return "", err
		}
		// first and last also serve string, bool and object payloads, so they
		// test for presence rather than for a numeric value.
		if isFirstLast(aggType) {
			conditions = append(conditions, fmt.Sprintf("MAX(%s) IS NOT NULL", jsonPath))
		} else {
			conditions = append(conditions, fmt.Sprintf("MAX(CAST(%s AS FLOAT)) IS NOT NULL", jsonPath))
		}
	}
	return strings.Join(conditions, " OR "), nil
}

func buildJSONPath(field string) (string, error) {
	if _, err := escapeFieldName(field); err != nil {
		return "", err
	}

	parts := strings.Split(field, ".")
	if len(parts) == 1 {
		escaped, _ := escapeFieldName(parts[0])
		return fmt.Sprintf("payload->>'%s'", escaped), nil
	}

	var path strings.Builder
	path.WriteString("payload")
	for i, part := range parts {
		escaped, _ := escapeFieldName(part)
		if i == len(parts)-1 {
			fmt.Fprintf(&path, "->>'%s'", escaped)
		} else {
			fmt.Fprintf(&path, "->'%s'", escaped)
		}
	}
	return path.String(), nil
}
