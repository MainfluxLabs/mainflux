// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package smtp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
)

var levelLabels = map[int32]string{
	1: "Info",
	2: "Low",
	3: "Medium",
	4: "High",
	5: "Critical",
}

// field is a single named value extracted from a message payload.
type field struct {
	Name  string
	Value string
}

// formatTime formats a Unix timestamp in nanoseconds as a UTC date and time.
func formatTime(createdNs int64) string {
	return time.Unix(0, createdNs).UTC().Format(time.DateTime) + " UTC"
}

// formatConditions renders the conditions of a rule as a single readable expression.
func formatConditions(ruleInfo []byte) string {
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
		threshold = formatFloat(*condition.Threshold)
	}

	return fmt.Sprintf("%s %s %s", condition.Field, condition.Comparator, threshold)
}

// formatPayload returns the payload as indented JSON, or unchanged when it isn't valid JSON.
func formatPayload(payload []byte) string {
	var indented bytes.Buffer
	if err := json.Indent(&indented, payload, "", "  "); err != nil {
		return string(payload)
	}

	return indented.String()
}

// payloadFields extracts named values from a SenML or JSON object payload.
func payloadFields(payload []byte) ([]field, bool) {
	if fields, ok := senmlFields(payload); ok {
		return fields, true
	}

	var object map[string]any
	if err := json.Unmarshal(payload, &object); err != nil {
		return nil, false
	}

	fields := flattenObject("", object)

	return fields, len(fields) > 0
}

// senmlFields converts normalized SenML messages to fields, returning false
// if any of the items isn't a SenML message.
func senmlFields(payload []byte) ([]field, bool) {
	var messages []senml.Message
	if err := json.Unmarshal(payload, &messages); err != nil {
		return nil, false
	}

	fields := make([]field, 0, len(messages))
	for _, message := range messages {
		value, ok := senmlValue(message)
		if message.Name == "" || !ok {
			return nil, false
		}

		if message.Unit != "" {
			value = fmt.Sprintf("%s %s", value, message.Unit)
		}

		fields = append(fields, field{
			Name:  message.Name,
			Value: value,
		})
	}

	return fields, len(fields) > 0
}

func senmlValue(message senml.Message) (string, bool) {
	switch {
	case message.Value != nil:
		return formatFloat(*message.Value), true
	case message.StringValue != nil:
		return *message.StringValue, true
	case message.BoolValue != nil:
		return strconv.FormatBool(*message.BoolValue), true
	case message.DataValue != nil:
		return *message.DataValue, true
	case message.Sum != nil:
		return formatFloat(*message.Sum), true
	default:
		return "", false
	}
}

// flattenObject returns the values of a JSON object as fields, sorted by key.
func flattenObject(prefix string, object map[string]any) []field {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	fields := []field{}
	for _, key := range keys {
		name := key
		if prefix != "" {
			name = prefix + "." + key
		}

		fields = append(fields, flattenValue(name, object[key])...)
	}

	return fields
}

// flattenValue returns a JSON value as fields, joining the keys of nested objects
// and the indexes of array elements with dots.
func flattenValue(name string, value any) []field {
	switch typed := value.(type) {
	case map[string]any:
		return flattenObject(name, typed)
	case []any:
		fields := []field{}
		for i, element := range typed {
			elementName := fmt.Sprintf("%s.%d", name, i)
			fields = append(fields, flattenValue(elementName, element)...)
		}

		return fields
	default:
		valueField := field{
			Name:  name,
			Value: formatJSONValue(value),
		}

		return []field{valueField}
	}
}

// formatJSONValue returns strings as they are and encodes any other value as JSON.
func formatJSONValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}

	return string(encoded)
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
