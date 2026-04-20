package http

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
)

func ConvertToJSONFile(page alarms.AlarmsPage, timeFormat string) ([]byte, error) {
	if page.Total == 0 {
		return []byte("[]"), nil
	}

	result := make([]map[string]any, 0, len(page.Alarms))

	for _, a := range page.Alarms {
		item := map[string]any{
			"thing_id":  a.ThingID,
			"group_id":  a.GroupID,
			"rule_id":   a.RuleID,
			"script_id": a.ScriptID,
			"subtopic":  a.Subtopic,
			"protocol":  a.Protocol,
			"rule":      a.Rule,
		}

		item["created"] = formatTimeNs(a.Created, timeFormat)
		result = append(result, item)
	}

	return json.Marshal(result)
}

func ConvertToCSVFile(page alarms.AlarmsPage, timeFormat string) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	if page.Total == 0 {
		return []byte{}, nil
	}

	header := []string{
		"created",
		"thing_id",
		"group_id",
		"rule_id",
		"script_id",
		"subtopic",
		"protocol",
		"rule",
	}

	if err := writer.Write(header); err != nil {
		return nil, err
	}

	for _, alarm := range page.Alarms {
		rule := ""
		if alarm.Rule != nil {
			b, err := json.Marshal(alarm.Rule)
			if err != nil {
				return nil, err
			}
			rule = string(b)
		}

		row := []string{
			formatTimeNs(alarm.Created, timeFormat),
			alarm.ThingID,
			alarm.GroupID,
			alarm.RuleID,
			alarm.ScriptID,
			alarm.Subtopic,
			alarm.Protocol,
			rule,
		}

		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	return buf.Bytes(), writer.Error()
}

func formatTimeNs(ns int64, timeFormat string) string {
	if strings.ToLower(timeFormat) == "rfc3339" {
		return time.Unix(0, ns).UTC().Format(time.RFC3339)
	}
	return fmt.Sprintf("%v", ns)
}

