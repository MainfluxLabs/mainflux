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
// Type selects how the condition is evaluated: "threshold" compares Field against Threshold
// using Comparator; "script" runs the Lua script identified by ScriptID and uses its boolean
// return value.
type Condition struct {
	Type       string   `json:"type"`
	Field      string   `json:"field,omitempty"`
	Comparator string   `json:"comparator,omitempty"`
	Threshold  *float64 `json:"threshold,omitempty"`
	ScriptID   string   `json:"script_id,omitempty"`
}

// RuleInfo captures the evaluation logic of the rule that triggered an alarm.
type RuleInfo struct {
	Conditions []Condition `json:"conditions"`
	Operator   string      `json:"operator,omitempty"`
}
