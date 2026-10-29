// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package smtp

import (
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/stretchr/testify/assert"
)

const (
	thingID     = "thing-id"
	ruleID      = "rule-id"
	createdNs   = 1793103299300713748
	createdTime = "2026-10-27 12:14:59 UTC"
	ruleInfo    = `{
		"conditions": [{"type": "threshold", "field": "temp", "comparator": ">", "threshold": 30}]
	}`
)

func TestPayloadFields(t *testing.T) {
	cases := []struct {
		desc        string
		payload     string
		contentType string
		fields      []field
		ok          bool
	}{
		{
			desc: "SenML records with unit and time",
			payload: `[
				{"name": "main.humidity", "value": 66, "unit": "%", "time": 1793103299300713748},
				{"name": "main.door", "bool_value": true, "time": 1793103000000000000}
			]`,
			contentType: messaging.SenMLContentType,
			fields: []field{
				{
					Name:  "main.humidity",
					Value: "66 %",
					Time:  createdTime,
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
			desc:        "nested JSON object sorted by key",
			payload:     `{"temp":31.2,"main":{"humidity":83}}`,
			contentType: messaging.JSONContentType,
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
			desc:        "JSON integer above 2^53",
			payload:     `{"counter":9007199254740993}`,
			contentType: messaging.JSONContentType,
			fields: []field{
				{
					Name:  "counter",
					Value: "9007199254740993",
				},
			},
			ok: true,
		},
		{
			desc:        "JSON object with an array of objects",
			payload:     `{"weather":[{"main":"Clouds"}]}`,
			contentType: messaging.JSONContentType,
			fields: []field{
				{
					Name:  "weather.0.main",
					Value: "Clouds",
				},
			},
			ok: true,
		},
		{
			desc:        "top-level JSON array",
			payload:     `[{"temp":31},{"temp":32}]`,
			contentType: messaging.JSONContentType,
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
			desc:        "JSON array shaped like SenML",
			payload:     `[{"name":"x","value":1}]`,
			contentType: messaging.JSONContentType,
			fields: []field{
				{
					Name:  "0.name",
					Value: "x",
				},
				{
					Name:  "0.value",
					Value: "1",
				},
			},
			ok: true,
		},
		{
			desc:        "SenML that isn't a list of records",
			payload:     `{"name":"x","value":1}`,
			contentType: messaging.SenMLContentType,
			fields:      nil,
			ok:          false,
		},
		{
			desc:        "empty JSON object",
			payload:     `{}`,
			contentType: messaging.JSONContentType,
			fields:      nil,
			ok:          false,
		},
		{
			desc:        "empty JSON array",
			payload:     `[]`,
			contentType: messaging.JSONContentType,
			fields:      nil,
			ok:          false,
		},
		{
			desc:        "JSON string",
			payload:     `"text"`,
			contentType: messaging.JSONContentType,
			fields:      nil,
			ok:          false,
		},
		{
			desc:        "invalid JSON",
			payload:     `not json`,
			contentType: messaging.JSONContentType,
			fields:      nil,
			ok:          false,
		},
	}

	for _, tc := range cases {
		fields, ok := payloadFields([]byte(tc.payload), tc.contentType)
		assert.Equal(t, tc.ok, ok, fmt.Sprintf("%s: expected ok %v got %v\n", tc.desc, tc.ok, ok))
		if tc.ok {
			fieldsMessage := fmt.Sprintf(
				"%s: expected fields %v got %v\n",
				tc.desc,
				tc.fields,
				fields,
			)
			assert.Equal(t, tc.fields, fields, fieldsMessage)
		}
	}
}

func TestBuildEmail(t *testing.T) {
	cases := []struct {
		desc         string
		notification protomfx.Notification
		email        notificationEmail
		err          error
	}{
		{
			desc: "alarm with subtopic",
			notification: protomfx.Notification{
				Source: &protomfx.Notification_Alarm{
					Alarm: &protomfx.Alarm{
						ThingId:  thingID,
						Subtopic: "room",
						Protocol: "http",
						Created:  createdNs,
						Level:    4,
						RuleId:   ruleID,
						RuleInfo: []byte(ruleInfo),
					},
				},
			},
			email: notificationEmail{
				subject:      "[HIGH] Alarm on room from Thing thing-id",
				templateName: alarmTemplate,
				templateData: map[string]any{
					"ThingID":    thingID,
					"Subtopic":   "room",
					"Protocol":   "http",
					"Time":       createdTime,
					"LevelLabel": "High",
					"LevelClass": "high",
					"RuleID":     ruleID,
					"Conditions": "temp > 30",
				},
			},
			err: nil,
		},
		{
			desc: "alarm without level",
			notification: protomfx.Notification{
				Source: &protomfx.Notification_Alarm{
					Alarm: &protomfx.Alarm{
						ThingId:  thingID,
						Protocol: "http",
						Created:  createdNs,
						RuleId:   ruleID,
					},
				},
			},
			email: notificationEmail{
				subject:      "[UNKNOWN] Alarm from Thing thing-id",
				templateName: alarmTemplate,
				templateData: map[string]any{
					"ThingID":    thingID,
					"Subtopic":   "",
					"Protocol":   "http",
					"Time":       createdTime,
					"LevelLabel": "Unknown",
					"LevelClass": "unknown",
					"RuleID":     ruleID,
					"Conditions": "",
				},
			},
			err: nil,
		},
		{
			desc: "message with fields",
			notification: protomfx.Notification{
				Source: &protomfx.Notification_Message{
					Message: &protomfx.Message{
						ThingId:     thingID,
						Protocol:    "mqtt",
						Created:     createdNs,
						ContentType: messaging.JSONContentType,
						Payload:     []byte(`{"temp":31.2}`),
					},
				},
			},
			email: notificationEmail{
				subject:      "New IoT message from Thing thing-id",
				templateName: messageTemplate,
				templateData: map[string]any{
					"ThingID":  thingID,
					"Subtopic": "",
					"Protocol": "mqtt",
					"Time":     createdTime,
					"Fields": []field{
						{
							Name:  "temp",
							Value: "31.2",
						},
					},
				},
			},
			err: nil,
		},
		{
			desc: "message without fields",
			notification: protomfx.Notification{
				Source: &protomfx.Notification_Message{
					Message: &protomfx.Message{
						ThingId:     thingID,
						Subtopic:    "room",
						Protocol:    "mqtt",
						Created:     createdNs,
						ContentType: messaging.JSONContentType,
						Payload:     []byte(`"text"`),
					},
				},
			},
			email: notificationEmail{
				subject:      "New IoT message on room from Thing thing-id",
				templateName: messageTemplate,
				templateData: map[string]any{
					"ThingID":        thingID,
					"Subtopic":       "room",
					"Protocol":       "mqtt",
					"Time":           createdTime,
					"MessageContent": `"text"`,
				},
			},
			err: nil,
		},
		{
			desc:         "notification without source",
			notification: protomfx.Notification{},
			email:        notificationEmail{},
			err:          notifiers.ErrMissingSource,
		},
	}

	for _, tc := range cases {
		email, err := buildEmail(tc.notification)
		errorMessage := fmt.Sprintf("%s: expected error %v got %v\n", tc.desc, tc.err, err)
		emailMessage := fmt.Sprintf("%s: expected email %v got %v\n", tc.desc, tc.email, email)
		assert.True(t, errors.Contains(err, tc.err), errorMessage)
		assert.Equal(t, tc.email, email, emailMessage)
	}
}
