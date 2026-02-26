// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	prettyjson "github.com/hokaccha/go-prettyjson"
)

var (
	// Limit query parameter
	Limit uint = 10
	// Offset query parameter
	Offset uint = 0
	// Name query parameter
	Name string = ""
	// Email query parameter
	Email string = ""
	// Metadata query parameter
	Metadata string = ""
	// Format query parameter
	Format string = ""
	// Subtopic query parameter
	Subtopic string = ""
	// RawOutput raw output mode
	RawOutput bool = false
	// Publisher query parameter
	Publisher string = ""
	// Protocol query parameter
	Protocol string = ""
	// From timestamp query parameter (milliseconds)
	From int64 = 0
	// To timestamp query parameter (milliseconds)
	To int64 = 0
	// Dir sort direction query parameter (asc/desc)
	Dir string = ""
	// Filter query parameter (JSON messages)
	Filter string = ""
	// AggInterval aggregation interval (minute, hour, day, week, month, year)
	AggInterval string = ""
	// AggValue aggregation value
	AggValue uint = 1
	// AggType aggregation type (min, max, avg, count)
	AggType string = ""
	// AggField aggregation fields (comma-separated)
	AggField string = ""
	// SenMLName SenML name filter
	SenMLName string = ""
	// SenMLValue SenML numeric value filter
	SenMLValue float64 = 0
	// Comparator comparison operator (eq, lt, le, gt, ge)
	Comparator string = ""
	// BoolValue SenML boolean value filter
	BoolValue bool = false
	// StringValue SenML string value filter
	StringValue string = ""
	// DataValue SenML data value filter
	DataValue string = ""
	// ConvertFormat export format (json/csv)
	ConvertFormat string = "json"
	// TimeFormat export time format
	TimeFormat string = ""
)

func logJSON(iList ...any) {
	for _, i := range iList {
		m, err := json.Marshal(i)
		if err != nil {
			logError(err)
			return
		}

		pj, err := prettyjson.Format(m)
		if err != nil {
			logError(err)
			return
		}

		fmt.Printf("\n%s\n\n", string(pj))
	}
}

func logUsage(u string) {
	fmt.Printf(color.YellowString("\nusage: %s\n\n"), u)
}

func logError(err error) {
	boldRed := color.New(color.FgRed, color.Bold)
	boldRed.Print("\nerror: ")

	fmt.Printf("%s\n\n", color.RedString(err.Error()))
}

func logOK() {
	fmt.Printf("\n%s\n\n", color.BlueString("ok"))
}

func logCreated(e string) {
	if RawOutput {
		fmt.Println(e)
	} else {
		fmt.Printf(color.BlueString("\ncreated: %s\n\n"), e)
	}
}

func convertMetadata(m string) (map[string]any, error) {
	var metadata map[string]any
	if m == "" {
		return nil, nil
	}
	if err := json.Unmarshal([]byte(m), &metadata); err != nil {
		return nil, err
	}
	return metadata, nil
}
