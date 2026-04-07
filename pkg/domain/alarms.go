// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package domain

const (
	// AlarmOriginRule indicates an alarm was triggered by a rule.
	AlarmOriginRule = "rule"
	// AlarmOriginScript indicates an alarm was triggered by a Lua script.
	AlarmOriginScript = "script"
)

// Condition represents a single evaluatable condition used in rules and recorded on alarms.
type Condition struct {
	Field      string   `json:"field"`
	Comparator string   `json:"comparator"`
	Threshold  *float64 `json:"threshold"`
}
