// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package notifiers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

const unknownLevel = "Unknown"

var levelLabels = map[int32]string{
	1: "Info",
	2: "Low",
	3: "Medium",
	4: "High",
	5: "Critical",
}

// AlarmLevelLabel returns a human-readable label for the alarm level.
func AlarmLevelLabel(level int32) string {
	if label, ok := levelLabels[level]; ok {
		return label
	}

	return unknownLevel
}

// FormatConditions renders the conditions of a rule as a single readable expression.
func FormatConditions(ruleInfo []byte) string {
	var info domain.RuleInfo
	if err := json.Unmarshal(ruleInfo, &info); err != nil {
		return ""
	}

	conditions := make([]string, 0, len(info.Conditions))
	for _, condition := range info.Conditions {
		conditions = append(conditions, formatCondition(condition))
	}

	separator := fmt.Sprintf(" %s ", info.Operator)

	return strings.Join(conditions, separator)
}

func formatCondition(condition domain.Condition) string {
	if condition.ScriptID != "" {
		return fmt.Sprintf("script %s", condition.ScriptID)
	}

	threshold := ""
	if condition.Threshold != nil {
		threshold = strconv.FormatFloat(*condition.Threshold, 'f', -1, 64)
	}

	return fmt.Sprintf("%s %s %s", condition.Field, condition.Comparator, threshold)
}
