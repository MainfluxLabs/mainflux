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

// RuleInfo captures the evaluation logic of the rule that triggered an alarm.
type RuleInfo struct {
	Conditions []Condition `json:"conditions"`
	Operator   string      `json:"operator,omitempty"`
}
