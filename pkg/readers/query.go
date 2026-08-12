// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package readers

import (
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

const (
	JsonTable  = "json"
	JsonOrder  = "created"
	SenmlTable = "senml"
	SenmlOrder = "time"

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
)

func BaseConditions(pm domain.ReadersParams, timeColumn string) []string {
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

func BaseQueryParams(pm domain.ReadersParams) map[string]any {
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

// senmlConditions returns the SQL predicates common to every senml read,
// including the aggregated ones.
func SenmlConditions(pm domain.SenMLPageMetadata) []string {
	conds := BaseConditions(pm.ReadersParams, SenmlOrder)

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
