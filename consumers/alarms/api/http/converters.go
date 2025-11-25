package http

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	"github.com/jung-kurt/gofpdf"
)

func GenerateJSON(page alarms.AlarmsPage, timeFormat string) ([]byte, error) {
	if page.Total == 0 {
		return []byte("[]"), nil
	}

	result := make([]map[string]any, 0, len(page.Alarms))

	for _, a := range page.Alarms {
		item := map[string]any{
			"thing_id": a.ThingID,
			"group_id": a.GroupID,
			"rule_id":  a.RuleID,
			"subtopic": a.Subtopic,
			"protocol": a.Protocol,
			"payload":  a.Payload,
		}

		item["created"] = formatTimeNs(a.Created, timeFormat)

		result = append(result, item)
	}

	return json.Marshal(result)
}

func GenerateCSV(page alarms.AlarmsPage, timeFormat string) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	if page.Total == 0 {
		return []byte{}, nil
	}

	flattened := make([]map[string]any, len(page.Alarms))
	payload := []string{}
	added := map[string]bool{}

	for i, alarm := range page.Alarms {

		p := alarm.Payload
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

	header := []string{
		"created",
		"thing_id",
		"group_id",
		"rule_id",
		"subtopic",
		"protocol",
	}

	header = append(header, payload...)

	if err := writer.Write(header); err != nil {
		return nil, err
	}

	for i, alarm := range page.Alarms {

		created := formatTimeNs(alarm.Created, timeFormat)

		row := []string{
			created,
			alarm.ThingID,
			alarm.GroupID,
			alarm.RuleID,
			alarm.Subtopic,
			alarm.Protocol,
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

func GeneratePDF(page alarms.AlarmsPage, timeFormat string) ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetFont("Arial", "", 10)
	pdf.AddPage()

	pdf.CellFormat(0, 10, "Alarms Backup", "", 1, "C", false, 0, "")

	headers := []string{"Created", "ThingID", "GroupID", "RuleID", "Subtopic", "Protocol"}
	colWidths := []float64{35, 40, 40, 40, 60, 50}

	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	for _, a := range page.Alarms {
		row := []string{
			formatTimeNs(a.Created, timeFormat),
			a.ThingID,
			a.GroupID,
			a.RuleID,
			a.Subtopic,
			a.Protocol,
		}

		maxLines := 1
		for i, cell := range row {
			lines := pdf.SplitLines([]byte(cell), colWidths[i])
			if len(lines) > maxLines {
				maxLines = len(lines)
			}
		}
		rowHeight := float64(maxLines*6 + 1)

		for i, cell := range row {
			x, y := pdf.GetX(), pdf.GetY()
			pdf.MultiCell(colWidths[i], 6, cell, "1", "L", false)
			pdf.SetXY(x+colWidths[i], y)
		}
		pdf.Ln(rowHeight)
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}

func formatTimeNs(ns int64, timeFormat string) string {
	if strings.ToLower(timeFormat) == "rfc3339" {
		return time.Unix(0, ns).UTC().Format(time.RFC3339)
	}
	return fmt.Sprintf("%v", ns)
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
		default:
			result[key] = v
		}
	}
	return result
}
