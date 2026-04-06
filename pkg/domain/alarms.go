// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package domain

const (
	// AlarmOriginRule indicates an alarm was triggered by a rule.
	AlarmOriginRule = "rule"
	// AlarmOriginScript indicates an alarm was triggered by a Lua script.
	AlarmOriginScript = "script"
)

const (
	AlarmLevelNameInfo     = "info"
	AlarmLevelNameWarning  = "warning"
	AlarmLevelNameMinor    = "minor"
	AlarmLevelNameMajor    = "major"
	AlarmLevelNameCritical = "critical"
)

var alarmLevelNames = map[int]string{
	1: AlarmLevelNameInfo,
	2: AlarmLevelNameWarning,
	3: AlarmLevelNameMinor,
	4: AlarmLevelNameMajor,
	5: AlarmLevelNameCritical,
}

var alarmLevelValues = map[string]int{
	AlarmLevelNameInfo:     1,
	AlarmLevelNameWarning:  2,
	AlarmLevelNameMinor:    3,
	AlarmLevelNameMajor:    4,
	AlarmLevelNameCritical: 5,
}

// AlarmLevelName returns the string name for a numeric alarm level.
func AlarmLevelName(level int) (string, bool) {
	name, ok := alarmLevelNames[level]
	return name, ok
}

// ParseAlarmLevel returns the numeric level for a string alarm level name.
func ParseAlarmLevel(name string) (int, bool) {
	level, ok := alarmLevelValues[name]
	return level, ok
}

// Condition represents a single evaluatable condition used in rules and recorded on alarms.
type Condition struct {
	Field      string   `json:"field"`
	Comparator string   `json:"comparator"`
	Threshold  *float64 `json:"threshold"`
}
