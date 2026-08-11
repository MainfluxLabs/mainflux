// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package timescale

import (
	"fmt"

	"github.com/MainfluxLabs/mainflux/readers"
)

const (
	jsonTable  = "json"
	jsonOrder  = "created"
	senmlTable = "senml"
	senmlOrder = "time"
)

// baseConditions returns the SQL predicates shared by all message formats.
func baseConditions(pm readers.ReadersMetadata, timeColumn string) []string {
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

// baseQueryParams returns the named query parameters shared by all message formats.
func baseQueryParams(pm readers.ReadersMetadata) map[string]any {
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
