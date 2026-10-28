// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package smtp

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPayloadFields(t *testing.T) {
	cases := []struct {
		desc    string
		payload string
		fields  []field
		ok      bool
	}{
		{
			desc:    "SenML records with unit and time",
			payload: `[{"name":"main.humidity","value":66,"unit":"%","time":1793103299300713748},{"name":"main.door","bool_value":true,"time":1793103000000000000}]`,
			fields: []field{
				{
					Name:  "main.humidity",
					Value: "66 %",
					Time:  "2026-10-27 12:14:59 UTC",
				},
				{
					Name:  "main.door",
					Value: "true",
					Time:  "2026-10-27 12:10:00 UTC",
				},
			},
			ok: true,
		},
		{
			desc:    "nested JSON object sorted by key",
			payload: `{"temp":31.2,"main":{"humidity":83}}`,
			fields: []field{
				{
					Name:  "main.humidity",
					Value: "83",
				},
				{
					Name:  "temp",
					Value: "31.2",
				},
			},
			ok: true,
		},
		{
			desc:    "JSON object with an array of objects",
			payload: `{"weather":[{"main":"Clouds"}]}`,
			fields: []field{
				{
					Name:  "weather.0.main",
					Value: "Clouds",
				},
			},
			ok: true,
		},
		{
			desc:    "top-level JSON array",
			payload: `[{"temp":31},{"temp":32}]`,
			fields: []field{
				{
					Name:  "0.temp",
					Value: "31",
				},
				{
					Name:  "1.temp",
					Value: "32",
				},
			},
			ok: true,
		},
		{
			desc:    "empty JSON object",
			payload: `{}`,
			fields:  nil,
			ok:      false,
		},
		{
			desc:    "empty JSON array",
			payload: `[]`,
			fields:  nil,
			ok:      false,
		},
		{
			desc:    "JSON string",
			payload: `"text"`,
			fields:  nil,
			ok:      false,
		},
		{
			desc:    "invalid JSON",
			payload: `not json`,
			fields:  nil,
			ok:      false,
		},
	}

	for _, tc := range cases {
		fields, ok := payloadFields([]byte(tc.payload))
		assert.Equal(t, tc.ok, ok, fmt.Sprintf("%s: expected ok %v got %v\n", tc.desc, tc.ok, ok))
		if tc.ok {
			assert.Equal(t, tc.fields, fields, fmt.Sprintf("%s: expected fields %v got %v\n", tc.desc, tc.fields, fields))
		}
	}
}

func TestFormatConditions(t *testing.T) {
	cases := []struct {
		desc     string
		ruleInfo string
		want     string
	}{
		{
			desc:     "single threshold condition",
			ruleInfo: `{"conditions":[{"type":"threshold","field":"main.humidity","comparator":">","threshold":60}]}`,
			want:     "main.humidity > 60",
		},
		{
			desc:     "threshold and script conditions joined by operator",
			ruleInfo: `{"conditions":[{"type":"threshold","field":"temp","comparator":">=","threshold":30.5},{"type":"script","script_id":"script-id"}],"operator":"AND"}`,
			want:     "temp >= 30.5 AND script script-id",
		},
		{
			desc:     "invalid rule info",
			ruleInfo: `not json`,
			want:     "",
		},
	}

	for _, tc := range cases {
		got := formatConditions([]byte(tc.ruleInfo))
		assert.Equal(t, tc.want, got, fmt.Sprintf("%s: expected %q got %q\n", tc.desc, tc.want, got))
	}
}
