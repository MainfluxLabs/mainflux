// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package readers

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const (
	JSONTable  = "json"
	JSONOrder  = "created"
	SenMLTable = "senml"
	SenMLOrder = "time"

	// EqualKey represents the equal comparison operator key.
	EqualKey = "eq"
	// LowerThanKey represents the lower-than comparison operator key.
	LowerThanKey = "lt"
	// LowerThanEqualKey represents the lower-than-or-equal comparison operator key.
	LowerThanEqualKey = "le"
	// GreaterThanKey represents the greater-than-or-equal comparison operator key.
	GreaterThanKey = "gt"
	// GreaterThanEqualKey represents the greater-than-or-equal comparison operator key.
	GreaterThanEqualKey = "ge"
	// StartsWithKey represents the case-insensitive starts-with comparison operator key.
	StartsWithKey = "starts_with"
	// ContainsKey represents the case-insensitive contains comparison operator key.
	ContainsKey = "contains"

	// MicrosecondInterval represents the microsecond aggregation interval unit.
	MicrosecondInterval = "microsecond"
	// MillisecondInterval represents the millisecond aggregation interval unit.
	MillisecondInterval = "millisecond"
	// SecondInterval represents the second aggregation interval unit.
	SecondInterval = "second"
	// MinuteInterval represents the minute aggregation interval unit.
	MinuteInterval = "minute"
	// HourInterval represents the hour aggregation interval unit.
	HourInterval = "hour"
	// DayInterval represents the day aggregation interval unit.
	DayInterval = "day"
	// WeekInterval represents the week aggregation interval unit.
	WeekInterval = "week"
	// MonthInterval represents the month aggregation interval unit.
	MonthInterval = "month"
	// YearInterval represents the year aggregation interval unit.
	YearInterval = "year"
)

const (
	payloadKey      = "CAST(:payload_key AS jsonpath)"
	anyPayloadPath  = "'strict $.**'"
	jsonScalarTypes = "('string', 'number', 'boolean')"
	// unwrapArrayFilter is a no-op filter; in lax mode it unwraps an array at the end of the key into its items.
	unwrapArrayFilter = " ? (1 == 1)"
)

var (
	// ErrInvalidPayloadKey indicates a malformed JSON payload key.
	ErrInvalidPayloadKey = errors.New("invalid payload key")

	payloadKeyPartRegexp = regexp.MustCompile(`^([^\[\]\x00]+)((?:\[\d+\])*)$`)
	likeEscaper          = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
)

func BaseConditions(pm domain.MessagesPageMetadata, timeColumn string) []string {
	var conds []string
	if pm.Subtopic != "" {
		conds = append(conds, "subtopic = :subtopic")
	}
	if pm.Publisher != "" {
		conds = append(conds, "publisher = :publisher")
	}
	if pm.Protocol != "" {
		conds = append(conds, "protocol = :protocol")
	}
	if pm.From != 0 {
		conds = append(conds, fmt.Sprintf("%s >= :from", timeColumn))
	}
	if pm.To != 0 {
		conds = append(conds, fmt.Sprintf("%s < :to", timeColumn))
	}

	return conds
}

func BaseQueryParams(pm domain.MessagesPageMetadata) map[string]any {
	return map[string]any{
		"limit":     pm.Limit,
		"offset":    pm.Offset,
		"subtopic":  pm.Subtopic,
		"publisher": pm.Publisher,
		"protocol":  pm.Protocol,
		"from":      pm.From,
		"to":        pm.To,
	}
}

// JSONConditions returns the SQL predicates common to every json read.
func JSONConditions(pm domain.JSONPageMetadata) []string {
	conds := BaseConditions(pm.MessagesPageMetadata, JSONOrder)

	switch {
	case pm.PayloadKey != "" && pm.PayloadValue != "":
		valueCond := payloadValueCondition(pm.Comparator)
		keyValueCond := anyPayloadNodeCondition(payloadKey, valueCond)
		return append(conds, keyValueCond)
	case pm.PayloadKey != "":
		keyCond := anyPayloadNodeCondition(payloadKey, "jsonb_typeof(node) <> 'null'")
		return append(conds, keyCond)
	case pm.PayloadValue != "":
		valueCond := payloadValueCondition(pm.Comparator)
		anyValueCond := anyPayloadNodeCondition(anyPayloadPath, valueCond)
		return append(conds, anyValueCond)
	default:
		return conds
	}
}

// JSONQueryParams returns the named parameters referenced by JSONConditions.
func JSONQueryParams(pm domain.JSONPageMetadata) map[string]any {
	params := BaseQueryParams(pm.MessagesPageMetadata)

	if pm.PayloadKey != "" {
		params["payload_key"] = payloadKeyParam(pm.PayloadKey, pm.PayloadValue)
	}

	if pm.PayloadValue != "" {
		params["payload_value"] = payloadValueParam(pm.PayloadValue, pm.Comparator)
	}

	return params
}

// ParsePayloadKey converts a dot-separated JSON payload key into a JSON path.
func ParsePayloadKey(key string) (string, error) {
	var path strings.Builder
	path.WriteString("lax $")

	for _, part := range strings.Split(key, ".") {
		matches := payloadKeyPartRegexp.FindStringSubmatch(part)
		if matches == nil {
			return "", ErrInvalidPayloadKey
		}

		name, _ := json.Marshal(matches[1])

		path.WriteString(".")
		path.Write(name)
		path.WriteString(matches[2])
	}

	return path.String(), nil
}

// payloadKeyParam returns the JSON path bound as :payload_key.
func payloadKeyParam(key, value string) any {
	path, err := ParsePayloadKey(key)
	if err != nil {
		return nil
	}

	if value == "" {
		return path
	}

	return path + unwrapArrayFilter
}

// anyPayloadNodeCondition matches messages where any jsonb node returned by
// the path satisfies nodeCond.
func anyPayloadNodeCondition(path, nodeCond string) string {
	return fmt.Sprintf("EXISTS (SELECT 1 FROM jsonb_path_query(payload, %s) AS node WHERE %s)", path, nodeCond)
}

// payloadValueCondition matches node against :payload_value when node is a scalar.
func payloadValueCondition(comparator string) string {
	scalarCond := fmt.Sprintf("jsonb_typeof(node) IN %s", jsonScalarTypes)

	valueCond := "node #>> '{}' = :payload_value"
	if comparator == StartsWithKey || comparator == ContainsKey {
		valueCond = "LOWER(node #>> '{}') LIKE :payload_value"
	}

	return fmt.Sprintf("%s AND %s", scalarCond, valueCond)
}

func payloadValueParam(value, comparator string) string {
	if comparator != StartsWithKey && comparator != ContainsKey {
		return value
	}

	lowerValue := strings.ToLower(value)
	pattern := likeEscaper.Replace(lowerValue)

	if comparator == StartsWithKey {
		return pattern + "%"
	}

	return "%" + pattern + "%"
}

// senmlConditions returns the SQL predicates common to every senml read,
// including the aggregated ones.
func SenMLConditions(pm domain.SenMLPageMetadata) []string {
	conds := BaseConditions(pm.MessagesPageMetadata, SenMLOrder)

	if pm.Name != "" {
		conds = append(conds, "name = :name")
	}
	if pm.Value != 0 {
		conds = append(conds, fmt.Sprintf("value %s :value", ComparatorSymbol(pm.Comparator)))
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

// ComparatorSymbol converts a comparison operator key into its SQL symbol.
func ComparatorSymbol(key string) string {
	switch key {
	case EqualKey:
		return "="
	case LowerThanKey:
		return "<"
	case LowerThanEqualKey:
		return "<="
	case GreaterThanKey:
		return ">"
	case GreaterThanEqualKey:
		return ">="
	default:
		return "="
	}
}
