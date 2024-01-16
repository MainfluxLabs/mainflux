// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package json_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/stretchr/testify/assert"
)

const (
	validPayload      = `{"key1": "val1", "key2": 123, "key3": "val3", "key4": {"key5": "val5"}}`
	tsPayload         = `{"custom_ts_key": "1638310819", "key1": "val1", "key2": 123, "key3": "val3", "key4": {"key5": "val5"}}`
	microsPayload     = `{"custom_ts_micro_key": "1638310819000000", "key1": "val1", "key2": 123, "key3": "val3", "key4": {"key5": "val5"}}`
	invalidTsPayload  = `{"custom_ts_key": "abc", "key1": "val1", "key2": 123, "key3": "val3", "key4": {"key5": "val5"}}`
	listPayload       = `[{"key1": "val1", "key2": 123, "keylist3": "val3", "key4": {"key5": "val5"}}, {"key1": "val1", "key2": 123, "key3": "val3", "key4": {"key5": "val5"}}]`
	invalidPayload    = `{"key1": }`
	subtopic          = "subtopic"
	format            = "format"
	timeFieldLocation = "UTC"
	timeFieldFormat   = "unix"
	timeFieldName     = "custom_ts_key"
)

var profile = &messaging.Profile{TimeField: &messaging.TimeField{Name: "nanos_key", Format: timeFieldFormat, Location: timeFieldLocation}}

func TestTransformJSON(t *testing.T) {
	now := time.Now().Unix()

	tr := json.New()
	msg := messaging.Message{
		Channel:   "channel-1",
		Subtopic:  subtopic + "." + format,
		Publisher: "publisher-1",
		Protocol:  "protocol",
		Payload:   []byte(validPayload),
		Created:   now,
		Profile:   profile,
	}
	invalid := msg
	invalid.Payload = []byte(invalidPayload)

	listMsg := msg
	listMsg.Payload = []byte(listPayload)

	tsMsg := msg
	tsMsg.Payload = []byte(tsPayload)
	tsMsg.Profile.TimeField.Name = timeFieldName

	microsMsg := msg
	microsMsg.Payload = []byte(microsPayload)
	microsMsg.Profile = &messaging.Profile{TimeField: &messaging.TimeField{Name: "custom_ts_micro_key", Format: "unix_us", Location: timeFieldLocation}}

	invalidFmt := msg
	invalidFmt.Subtopic = ""

	invalidTimeField := msg
	invalidTimeField.Payload = []byte(invalidTsPayload)
	invalidTimeField.Profile.TimeField.Name = timeFieldName

	jsonMsgs := json.Messages{
		Data: []json.Message{
			{
				Channel:   msg.Channel,
				Subtopic:  subtopic,
				Publisher: msg.Publisher,
				Protocol:  msg.Protocol,
				Created:   msg.Created,
				Payload: map[string]interface{}{
					"key1": "val1",
					"key2": float64(123),
					"key3": "val3",
					"key4": map[string]interface{}{
						"key5": "val5",
					},
				},
			},
		},
		Format: format,
	}

	jsonTsMsgs := json.Messages{
		Data: []json.Message{
			{
				Channel:   msg.Channel,
				Subtopic:  subtopic,
				Publisher: msg.Publisher,
				Protocol:  msg.Protocol,
				Created:   int64(1638310819000000000),
				Payload: map[string]interface{}{
					timeFieldName: "1638310819",
					"key1":        "val1",
					"key2":        float64(123),
					"key3":        "val3",
					"key4": map[string]interface{}{
						"key5": "val5",
					},
				},
			},
		},
		Format: format,
	}

	jsonMicrosMsgs := json.Messages{
		Data: []json.Message{
			{
				Channel:   msg.Channel,
				Subtopic:  subtopic,
				Publisher: msg.Publisher,
				Protocol:  msg.Protocol,
				Created:   int64(1638310819000000000),
				Payload: map[string]interface{}{
					"custom_ts_micro_key": "1638310819000000",
					"key1":                "val1",
					"key2":                float64(123),
					"key3":                "val3",
					"key4": map[string]interface{}{
						"key5": "val5",
					},
				},
			},
		},
		Format: format,
	}

	listJSON := json.Messages{
		Data: []json.Message{
			{
				Channel:   msg.Channel,
				Subtopic:  subtopic,
				Publisher: msg.Publisher,
				Protocol:  msg.Protocol,
				Created:   msg.Created,
				Payload: map[string]interface{}{
					"key1":     "val1",
					"key2":     float64(123),
					"keylist3": "val3",
					"key4": map[string]interface{}{
						"key5": "val5",
					},
				},
			},
			{
				Channel:   msg.Channel,
				Subtopic:  subtopic,
				Publisher: msg.Publisher,
				Protocol:  msg.Protocol,
				Created:   msg.Created,
				Payload: map[string]interface{}{
					"key1": "val1",
					"key2": float64(123),
					"key3": "val3",
					"key4": map[string]interface{}{
						"key5": "val5",
					},
				},
			},
		},
		Format: format,
	}

	cases := []struct {
		desc string
		msg  messaging.Message
		json interface{}
		err  error
	}{
		{
			desc: "test transform JSON",
			msg:  msg,
			json: jsonMsgs,
			err:  nil,
		},
		{
			desc: "test transform JSON with an invalid subtopic",
			msg:  invalidFmt,
			json: nil,
			err:  json.ErrTransform,
		},
		{
			desc: "test transform JSON array",
			msg:  listMsg,
			json: listJSON,
			err:  nil,
		},
		{
			desc: "test transform JSON with invalid payload",
			msg:  invalid,
			json: nil,
			err:  json.ErrTransform,
		},
		{
			desc: "test transform JSON with timestamp transformation",
			msg:  tsMsg,
			json: jsonTsMsgs,
			err:  nil,
		},
		{
			desc: "test transform JSON with timestamp transformation in micros",
			msg:  microsMsg,
			json: jsonMicrosMsgs,
			err:  nil,
		},
		{
			desc: "test transform JSON with invalid timestamp transformation in micros",
			msg:  invalidTimeField,
			json: nil,
			err:  json.ErrInvalidTimeField,
		},
	}

	for _, tc := range cases {
		m, err := tr.Transform(tc.msg)
		assert.Equal(t, tc.json, m, fmt.Sprintf("%s expected %v, got %v", tc.desc, tc.json, m))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))
	}
}
