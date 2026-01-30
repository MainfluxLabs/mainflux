package messages

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
)

var senmlHeader = []string{
	"subtopic",
	"publisher",
	"protocol",
	"name",
	"unit",
	"value",
	"string_value",
	"bool_value",
	"data_value",
	"sum",
	"time",
	"update_time",
}

var jsonHeader = []string{
	"created",
	"subtopic",
	"publisher",
	"protocol",
}

func ConvertSenMLToCSVFile(page readers.SenMLMessagesPage, timeFormat string) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	if err := writer.Write(senmlHeader); err != nil {
		return nil, err
	}

	for _, msg := range page.MessagesPage.Messages {
		if m, ok := msg.(senml.Message); ok {
			row := []string{
				m.Subtopic,
				m.Publisher,
				m.Protocol,
				m.Name,
				m.Unit,
				getValue(m.Value, ""),
				getValue(m.StringValue, ""),
				getValue(m.BoolValue, ""),
				getValue(m.DataValue, ""),
				getValue(m.Sum, ""),
				fmt.Sprintf("%v", formatTime(m.Time, timeFormat)),
				fmt.Sprintf("%v", m.UpdateTime),
			}

			if err := writer.Write(row); err != nil {
				return nil, err
			}
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func ConvertJSONToCSVFile(page readers.JSONMessagesPage, timeFormat string) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	flattened := make([]map[string]any, len(page.MessagesPage.Messages))
	payload := []string{}
	added := map[string]bool{}

	for i, raw := range page.MessagesPage.Messages {
		m, ok := raw.(map[string]any)
		if !ok {
			flattened[i] = map[string]any{}
			continue
		}

		p, _ := m["payload"].(map[string]any)
		if p == nil {
			flattened[i] = map[string]any{}
			continue
		}

		flat := Flatten(p, "")
		flattened[i] = flat

		for k := range flat {
			if !added[k] {
				added[k] = true
				payload = append(payload, k)
			}
		}
	}

	header := append(jsonHeader, payload...)
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	for i, raw := range page.MessagesPage.Messages {
		m, _ := raw.(map[string]any)

		created := ""
		if v, ok := m["created"].(int64); ok {
			created = fmt.Sprintf("%v", formatTime(v, timeFormat))
		}

		row := []string{
			created,
			getStringValue(m, "subtopic"),
			getStringValue(m, "publisher"),
			getStringValue(m, "protocol"),
		}

		flat := flattened[i]
		for _, key := range payload {
			val := flat[key]
			if val == nil {
				row = append(row, "")
			} else {
				row = append(row, fmt.Sprint(val))
			}
		}

		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	return buf.Bytes(), writer.Error()
}

func getStringValue(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getValue(ptr any, defaultValue string) string {
	switch v := ptr.(type) {
	case *string:
		if v != nil {
			return *v
		}
	case *float64:
		if v != nil {
			return fmt.Sprintf("%v", *v)
		}
	case *bool:
		if v != nil {
			return fmt.Sprintf("%v", *v)
		}
	}
	return defaultValue
}

func formatTime(ns int64, timeFormat string) any {
	if strings.ToLower(timeFormat) == "rfc3339" {
		return time.Unix(0, ns).UTC().Format(time.RFC3339)
	}
	return ns
}

func Flatten(m map[string]any, prefix string) map[string]any {
	result := make(map[string]any)
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch child := v.(type) {
		case map[string]any:
			nested := Flatten(child, key)
			for nk, nv := range nested {
				result[nk] = nv
			}
		case []any:
			for i, elem := range child {
				indexKey := fmt.Sprintf("%s.%d", key, i)
				switch elemTyped := elem.(type) {
				case map[string]any:
					nested := Flatten(elemTyped, indexKey)
					for nk, nv := range nested {
						result[nk] = nv
					}
				default:
					result[indexKey] = elemTyped
				}
			}
		default:
			result[key] = v
		}
	}
	return result
}

func ConvertJSONToJSONFile(page readers.JSONMessagesPage, timeFormat string) ([]byte, error) {
	if page.MessagesPage.Total == 0 {
		return []byte("[]"), nil
	}

	for _, m := range page.MessagesPage.Messages {
		if msgMap, ok := m.(map[string]any); ok {
			if v, ok := msgMap["created"].(int64); ok {
				msgMap["created"] = formatTime(v, timeFormat)
			}
		}
	}

	data, err := json.Marshal(page.MessagesPage.Messages)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func ConvertSenMLToJSONFile(page readers.SenMLMessagesPage, timeFormat string) ([]byte, error) {
	if page.MessagesPage.Total == 0 {
		return []byte("[]"), nil
	}

	out := make([]map[string]any, 0, len(page.MessagesPage.Messages))
	for _, msg := range page.MessagesPage.Messages {
		if m, ok := msg.(senml.Message); ok {
			msgMap := map[string]any{
				"value":     m.Value,
				"publisher": m.Publisher,
				"protocol":  m.Protocol,
				"name":      m.Name,
				"time":      formatTime(m.Time, timeFormat),
			}

			if m.Subtopic != "" {
				msgMap["subtopic"] = m.Subtopic
			}
			if m.Unit != "" {
				msgMap["unit"] = m.Unit
			}
			if m.UpdateTime != 0 {
				msgMap["update_time"] = m.UpdateTime
			}
			if m.StringValue != nil {
				msgMap["string_value"] = *m.StringValue
			}
			if m.BoolValue != nil {
				msgMap["bool_value"] = *m.BoolValue
			}
			if m.DataValue != nil {
				msgMap["data_value"] = *m.DataValue
			}
			if m.Sum != nil {
				msgMap["sum"] = *m.Sum
			}

			out = append(out, msgMap)
		}
	}

	return json.Marshal(out)
}

func ConvertJSONToJSONMessages(data []byte) ([]mfjson.Message, error) {
	// this was used because mfjson.Message uses []byte but json stores map[string]any
	var tempMessages []struct {
		Created   int64          `json:"created"`
		Subtopic  string         `json:"subtopic"`
		Publisher string         `json:"publisher"`
		Protocol  string         `json:"protocol"`
		Payload   map[string]any `json:"payload"`
	}

	if err := json.Unmarshal(data, &tempMessages); err != nil {
		return nil, err
	}
	messages := make([]mfjson.Message, len(tempMessages))
	for i, temp := range tempMessages {
		payloadBytes, err := json.Marshal(temp.Payload)
		if err != nil {
			return nil, err
		}

		messages[i] = mfjson.Message{
			Created:   temp.Created,
			Subtopic:  temp.Subtopic,
			Publisher: temp.Publisher,
			Protocol:  temp.Protocol,
			Payload:   payloadBytes,
		}
	}

	return messages, nil
}

func ConvertJSONToSenMLMessages(data []byte) ([]senml.Message, error) {
	var messages []senml.Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

func ConvertCSVToSenMLMessages(data []byte) ([]senml.Message, error) {
	reader := csv.NewReader(bytes.NewReader(data))

	header, err := reader.Read()
	if err != nil {
		return nil, err
	}

	var messages []senml.Message
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		msg := senml.Message{}
		for i, value := range record {
			if i >= len(header) {
				continue
			}

			switch header[i] {
			case "subtopic":
				msg.Subtopic = value
			case "publisher":
				msg.Publisher = value
			case "protocol":
				msg.Protocol = value
			case "name":
				msg.Name = value
			case "unit":
				msg.Unit = value
			case "time":
				if v, err := strconv.ParseInt(value, 10, 64); err == nil {
					msg.Time = v
				}
			case "update_time":
				if v, err := strconv.ParseFloat(value, 64); err == nil {
					msg.UpdateTime = v
				}
			case "value":
				if v, err := strconv.ParseFloat(value, 64); err == nil {
					msg.Value = &v
				}
			case "string_value":
				msg.StringValue = &value
			case "data_value":
				msg.DataValue = &value
			case "bool_value":
				if v, err := strconv.ParseBool(value); err == nil {
					msg.BoolValue = &v
				}
			case "sum":
				if v, err := strconv.ParseFloat(value, 64); err == nil {
					msg.Sum = &v
				}
			}
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func ConvertCSVToJSONMessages(csvMessages []byte) ([]mfjson.Message, error) {
	reader := csv.NewReader(bytes.NewReader(csvMessages))
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}

	var messages []mfjson.Message
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		msg := mfjson.Message{}
		for i, value := range record {
			if i >= len(header) {
				continue
			}

			switch header[i] {
			case "created":
				if value != "" {
					if v, err := strconv.ParseInt(value, 10, 64); err == nil {
						msg.Created = v
					}
				}
			case "subtopic":
				msg.Subtopic = value
			case "publisher":
				msg.Publisher = value
			case "protocol":
				msg.Protocol = value
			case "payload":
				msg.Payload = []byte(value)
			}
		}
		messages = append(messages, msg)
	}

	return messages, nil
}
