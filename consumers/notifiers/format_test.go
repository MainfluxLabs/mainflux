// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package notifiers_test

import (
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	"github.com/stretchr/testify/assert"
)

func TestFormatConditions(t *testing.T) {
	cases := []struct {
		desc     string
		ruleInfo string
		want     string
	}{
		{
			desc: "single threshold condition",
			ruleInfo: `{
				"conditions": [
					{
						"type": "threshold",
						"field": "main.humidity",
						"comparator": ">",
						"threshold": 60
					}
				]
			}`,
			want: "main.humidity > 60",
		},
		{
			desc: "threshold and script conditions joined by operator",
			ruleInfo: `{
				"conditions": [
					{"type": "threshold", "field": "temp", "comparator": ">=", "threshold": 30.5},
					{"type": "script", "script_id": "script-id"}
				],
				"operator": "AND"
			}`,
			want: "temp >= 30.5 AND script script-id",
		},
		{
			desc:     "invalid rule info",
			ruleInfo: `not json`,
			want:     "",
		},
	}

	for _, tc := range cases {
		got := notifiers.FormatConditions([]byte(tc.ruleInfo))
		message := fmt.Sprintf("%s: expected %q got %q\n", tc.desc, tc.want, got)
		assert.Equal(t, tc.want, got, message)
	}
}
