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
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
)

type field struct {
	Name  string
	Value string
	Time  string
}

// formatTime formats a Unix timestamp in nanoseconds as a UTC date and time.
func formatTime(timestampNs int64) string {
	return time.Unix(0, timestampNs).UTC().Format(time.DateTime) + " UTC"
}

// formatPayload returns the payload as indented JSON, or unchanged when it isn't valid JSON.
func formatPayload(payload []byte) string {
	var indented bytes.Buffer
	if err := json.Indent(&indented, payload, "", "  "); err != nil {
		return string(payload)
	}

	return indented.String()
}

// payloadFields extracts named values from the records of a SenML or JSON message.
func payloadFields(message *protomfx.Message) ([]field, bool) {
	if message.ContentType == messaging.SenMLContentType {
		return senmlFields(message.Records)
	}

	return jsonFields(message.Records)
}

// jsonFields flattens the records of a JSON message. The records of a JSON array
// are prefixed with their index and carry their own time.
func jsonFields(records []*protomfx.Record) ([]field, bool) {
	isArray := len(records) > 1

	fields := []field{}
	for i, record := range records {
		decoded, ok := decodeJSON(record.Payload)
		if !ok {
			return nil, false
		}

		prefix := ""
		recordTime := ""
		if isArray {
			prefix = strconv.Itoa(i)
			recordTime = formatTime(record.Time)
		}

		for _, recordField := range flattenValue(prefix, decoded) {
			recordField.Time = recordTime
			fields = append(fields, recordField)
		}
	}

	return fields, len(fields) > 0
}

// decodeJSON decodes a JSON object or array, returning false for any other payload.
func decodeJSON(payload []byte) (any, bool) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()

	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, false
	}

	switch decoded.(type) {
	case map[string]any, []any:
		return decoded, true
	default:
		return nil, false
	}
}

// senmlFields converts normalized SenML records to fields, returning false
// if any of the records isn't a SenML message.
func senmlFields(records []*protomfx.Record) ([]field, bool) {
	fields := make([]field, 0, len(records))
	for _, record := range records {
		var message senml.Message
		if err := json.Unmarshal(record.Payload, &message); err != nil {
			return nil, false
		}

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
			Time:  formatTime(record.Time),
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
