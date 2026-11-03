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
	laterNs     = 1793103000000000000
	laterTime   = "2026-10-27 12:10:00 UTC"
	ruleInfo    = `{
		"conditions": [{"type": "threshold", "field": "temp", "comparator": ">", "threshold": 30}]
	}`
)

func TestPayloadFields(t *testing.T) {
	cases := []struct {
		desc        string
		contentType string
		records     []*protomfx.Record
		fields      []field
		ok          bool
	}{
		{
			desc:        "SenML records with unit and time",
			contentType: messaging.SenMLContentType,
			records: []*protomfx.Record{
				newRecord(`{"name": "main.humidity", "value": 66, "unit": "%"}`, createdNs),
				newRecord(`{"name": "main.door", "bool_value": true}`, laterNs),
			},
			fields: []field{
				{
					Name:  "main.humidity",
					Value: "66 %",
					Time:  createdTime,
				},
				{
					Name:  "main.door",
					Value: "true",
					Time:  laterTime,
				},
			},
			ok: true,
		},
		{
			desc:        "nested JSON object sorted by key",
			contentType: messaging.JSONContentType,
			records: []*protomfx.Record{
				newRecord(`{"temp":31.2,"main":{"humidity":83}}`, createdNs),
			},
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
			contentType: messaging.JSONContentType,
			records: []*protomfx.Record{
				newRecord(`{"counter":9007199254740993}`, createdNs),
			},
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
			contentType: messaging.JSONContentType,
			records: []*protomfx.Record{
				newRecord(`{"weather":[{"main":"Clouds"}]}`, createdNs),
			},
			fields: []field{
				{
					Name:  "weather.0.main",
					Value: "Clouds",
				},
			},
			ok: true,
		},
		{
			desc:        "JSON array records with their own time",
			contentType: messaging.JSONContentType,
			records: []*protomfx.Record{
				newRecord(`{"temp":31}`, createdNs),
				newRecord(`{"temp":32}`, laterNs),
			},
			fields: []field{
				{
					Name:  "0.temp",
					Value: "31",
					Time:  createdTime,
				},
				{
					Name:  "1.temp",
					Value: "32",
					Time:  laterTime,
				},
			},
			ok: true,
		},
		{
			desc:        "JSON array in a single record",
			contentType: messaging.JSONContentType,
			records: []*protomfx.Record{
				newRecord(`[{"temp":31},{"temp":32}]`, createdNs),
			},
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
			desc:        "JSON object shaped like SenML",
			contentType: messaging.JSONContentType,
			records: []*protomfx.Record{
				newRecord(`{"name":"x","value":1}`, createdNs),
			},
			fields: []field{
				{
					Name:  "name",
					Value: "x",
				},
				{
					Name:  "value",
					Value: "1",
				},
			},
			ok: true,
		},
		{
			desc:        "SenML record without a name",
			contentType: messaging.SenMLContentType,
			records: []*protomfx.Record{
				newRecord(`{"value":1}`, createdNs),
			},
			fields: nil,
			ok:     false,
		},
		{
			desc:        "empty JSON object",
			contentType: messaging.JSONContentType,
			records: []*protomfx.Record{
				newRecord(`{}`, createdNs),
			},
			fields: nil,
			ok:     false,
		},
		{
			desc:        "JSON string",
			contentType: messaging.JSONContentType,
			records: []*protomfx.Record{
				newRecord(`"text"`, createdNs),
			},
			fields: nil,
			ok:     false,
		},
		{
			desc:        "invalid JSON",
			contentType: messaging.JSONContentType,
			records: []*protomfx.Record{
				newRecord(`not json`, createdNs),
			},
			fields: nil,
			ok:     false,
		},
		{
			desc:        "message without records",
			contentType: messaging.JSONContentType,
			records:     nil,
			fields:      nil,
			ok:          false,
		},
	}

	for _, tc := range cases {
		message := &protomfx.Message{
			ContentType: tc.contentType,
			Records:     tc.records,
		}

		fields, ok := payloadFields(message)
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
						Records: []*protomfx.Record{
							newRecord(`{"temp":31.2}`, createdNs),
						},
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
						Records: []*protomfx.Record{
							newRecord(`"text"`, createdNs),
						},
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

func newRecord(payload string, time int64) *protomfx.Record {
	return &protomfx.Record{
		Payload: []byte(payload),
		Time:    time,
	}
}
