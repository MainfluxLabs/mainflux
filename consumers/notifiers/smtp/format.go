// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package smtp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
)

// field is a single named value extracted from a message payload.
// Time is set only for SenML records, which carry their own timestamp.
type field struct {
	Name  string
	Value string
	Time  string
}

// formatTime formats a Unix timestamp in nanoseconds as a UTC date and time.
func formatTime(createdNs int64) string {
	return time.Unix(0, createdNs).UTC().Format(time.DateTime) + " UTC"
}

// formatPayload returns the payload as indented JSON, or unchanged when it isn't valid JSON.
func formatPayload(payload []byte) string {
	var indented bytes.Buffer
	if err := json.Indent(&indented, payload, "", "  "); err != nil {
		return string(payload)
	}

	return indented.String()
}

// payloadFields extracts named values from a SenML payload, or from a JSON object or array payload.
func payloadFields(payload []byte, contentType string) ([]field, bool) {
	if contentType == messaging.SenMLContentType {
		return senmlFields(payload)
	}

	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, false
	}

	switch decoded.(type) {
	case map[string]any, []any:
		fields := flattenValue("", decoded)
		return fields, len(fields) > 0
	default:
		return nil, false
	}
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
			Time:  formatTime(message.Time),
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
			elementName := strconv.Itoa(i)
			if name != "" {
				elementName = name + "." + elementName
			}

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
