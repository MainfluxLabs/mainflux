// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package readers

import (
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
	payloadKeyPath  = "CAST(:payload_key_path AS text[])"
	jsonScalarTypes = "('string', 'number', 'boolean')"
)

var (
	// ErrInvalidPayloadKey indicates a malformed JSON payload key.
	ErrInvalidPayloadKey = errors.New("invalid payload key")

	payloadKeyPartRegexp  = regexp.MustCompile(`^([^\[\]\x00]+)((?:\[\d+\])*)$`)
	payloadKeyIndexRegexp = regexp.MustCompile(`\d+`)
	likeEscaper           = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
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

// JSONConditions returns the SQL predicates common to every JSON read.
func JSONConditions(pm domain.JSONPageMetadata) []string {
	conds := BaseConditions(pm.MessagesPageMetadata, JSONOrder)

	switch {
	case pm.PayloadKey != "" && pm.PayloadValue != "":
		keyNode := fmt.Sprintf("(payload #> %s)", payloadKeyPath)
		valueCond := payloadValueCondition(keyNode, pm.Comparator)
		return append(conds, valueCond)
	case pm.PayloadKey != "":
		keyCond := fmt.Sprintf("payload #>> %s IS NOT NULL", payloadKeyPath)
		return append(conds, keyCond)
	case pm.PayloadValue != "":
		valueCond := payloadValueCondition("node", pm.Comparator)
		anyNodeCond := fmt.Sprintf("EXISTS (SELECT 1 FROM jsonb_path_query(payload, 'strict $.**') AS node WHERE %s)", valueCond)
		return append(conds, anyNodeCond)
	default:
		return conds
	}
}

// JSONQueryParams returns the named parameters referenced by JSONConditions.
func JSONQueryParams(pm domain.JSONPageMetadata) map[string]any {
	params := BaseQueryParams(pm.MessagesPageMetadata)

	if pm.PayloadKey != "" {
		keyPath, _ := ParsePayloadKey(pm.PayloadKey)
		params["payload_key_path"] = keyPath
	}

	if pm.PayloadValue != "" {
		params["payload_value"] = payloadValueParam(pm.PayloadValue, pm.Comparator)
	}

	return params
}

// ParsePayloadKey splits a dot-separated JSON payload key with optional array indexes.
func ParsePayloadKey(key string) ([]string, error) {
	var segments []string

	for _, part := range strings.Split(key, ".") {
		matches := payloadKeyPartRegexp.FindStringSubmatch(part)
		if matches == nil {
			return nil, ErrInvalidPayloadKey
		}

		name := matches[1]
		indexes := payloadKeyIndexRegexp.FindAllString(matches[2], -1)

		segments = append(segments, name)
		segments = append(segments, indexes...)
	}

	return segments, nil
}

// payloadValueCondition matches a jsonb node against :payload_value when the node is a scalar.
func payloadValueCondition(node, comparator string) string {
	scalarCond := fmt.Sprintf("jsonb_typeof(%s) IN %s", node, jsonScalarTypes)
	nodeText := fmt.Sprintf("%s #>> '{}'", node)

	valueCond := fmt.Sprintf("%s = :payload_value", nodeText)
	if comparator == StartsWithKey || comparator == ContainsKey {
		valueCond = fmt.Sprintf("LOWER(%s) LIKE :payload_value", nodeText)
	}

	return fmt.Sprintf("%s AND %s", scalarCond, valueCond)
}

func payloadValueParam(value, comparator string) string {
	pattern := likeEscaper.Replace(strings.ToLower(value))

	switch comparator {
	case StartsWithKey:
		return pattern + "%"
	case ContainsKey:
		return "%" + pattern + "%"
	default:
		return value
	}
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
