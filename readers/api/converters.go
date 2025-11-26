package api

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

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
	"payload",
}

func GenerateCSVFromSenML(page readers.MessagesPage) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	if err := writer.Write(senmlHeader); err != nil {
		return nil, err
	}

	for _, msg := range page.Messages {
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
				fmt.Sprintf("%v", m.Time),
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

func GenerateCSVFromJSON(page readers.MessagesPage) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	if err := writer.Write(jsonHeader); err != nil {
		return nil, err
	}

	for _, msg := range page.Messages {
		if m, ok := msg.(map[string]any); ok {
			created := ""
			if v, ok := m["created"].(int64); ok {
				created = fmt.Sprintf("%v", v)
			}

			subtopic := getStringValue(m, "subtopic")
			publisher := getStringValue(m, "publisher")
			protocol := getStringValue(m, "protocol")

			payload := ""
			if p, ok := m["payload"]; ok {
				if payloadBytes, err := json.Marshal(p); err == nil {
					payload = string(payloadBytes)
				}
			}

			row := []string{
				created,
				subtopic,
				publisher,
				protocol,
				payload,
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

func GenerateJSON(page readers.MessagesPage) ([]byte, error) {
	if page.Total == 0 {
		return []byte("[]"), nil
	}

	data, err := json.Marshal(page.Messages)
	if err != nil {
		return nil, err
	}

	return data, nil
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
