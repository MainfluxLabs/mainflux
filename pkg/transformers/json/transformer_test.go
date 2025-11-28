// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package json_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	validPayload      = `{"key1": "val1", "key2": 123, "key3": {"key4": "val4"}, "key5": "val5"}`
	tsPayload         = `{"custom_ts_key": "1638310819", "key1": "val1", "key2": 123, "key3": {"key4": "val4"}, "key5": "val5"}`
	microsPayload     = `{"custom_ts_micro_key": "1638310819000000", "key1": "val1", "key2": 123, "key3": {"key4": "val4"}, "key5": "val5"}`
	invalidTsPayload  = `{"custom_ts_key": "abc", "key1": "val1", "key2": 123, "key3": {"key4": "val4"}, "key5": "val5"}`
	listPayload       = `[{"key1": "val1", "key2": 123, "key3": {"key4": "val4"}, "key5": "val5"}, {"key1": "val1", "key2": 123, "key3": {"key4": "val4"}, "key5": "val5"}]`
	invalidPayload    = `{"key1": }`
	subtopic          = "subtopic"
	format            = "format"
	timeFieldLocation = "UTC"
	timeFieldFormat   = "unix"
	timeFieldName     = "custom_ts_key"
)

var (
	dataFilters = []string{"key1", "key2", "key3"}
)

var transformer = protomfx.Transformer{DataFilters: dataFilters, TimeField: "nanos_key", TimeFormat: timeFieldFormat, TimeLocation: timeFieldLocation}

func TestTransformJSON(t *testing.T) {
	now := time.Now().UnixNano()

	msg := protomfx.Message{
		Subtopic:  subtopic,
		Publisher: "publisher-1",
		Protocol:  "protocol",
		Payload:   []byte(validPayload),
		Created:   now,
	}
	invalid := msg
	invalid.Payload = []byte(invalidPayload)

	listMsg := msg
	listMsg.Payload = []byte(listPayload)

	tsMsg := msg
	tsMsg.Payload = []byte(tsPayload)
	tsMsgTr := protomfx.Transformer{DataFilters: dataFilters, TimeField: timeFieldName, TimeFormat: timeFieldFormat, TimeLocation: timeFieldLocation}

	microsMsg := msg
	microsMsg.Payload = []byte(microsPayload)
	microsMsgTr := protomfx.Transformer{DataFilters: dataFilters, TimeField: "custom_ts_micro_key", TimeFormat: "unix_us", TimeLocation: timeFieldLocation}

	invalidTimeField := msg
	invalidTimeField.Payload = []byte(invalidTsPayload)
	invalidTimeFieldTr := protomfx.Transformer{DataFilters: dataFilters, TimeField: timeFieldName, TimeFormat: timeFieldName, TimeLocation: timeFieldLocation}

	pyd := map[string]any{
		"key1": "val1",
		"key2": float64(123),
		"key3": map[string]any{
			"key4": "val4",
		},
	}

	payload, err := json.Marshal(pyd)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	msgs := []protomfx.Message{
		{
			Subtopic:  subtopic,
			Publisher: msg.Publisher,
			Protocol:  msg.Protocol,
			Created:   msg.Created,
			Payload:   payload,
		},
	}

	tsMsgs := []protomfx.Message{
		{
			Subtopic:  subtopic,
			Publisher: msg.Publisher,
			Protocol:  msg.Protocol,
			Created:   int64(1638310819000000000),
			Payload:   payload,
		},
	}

	microsMsgs := []protomfx.Message{
		{
			Subtopic:  subtopic,
			Publisher: msg.Publisher,
			Protocol:  msg.Protocol,
			Created:   int64(1638310819000000000),
			Payload:   payload,
		},
	}

	listMsgs := []protomfx.Message{
		{
			Subtopic:  subtopic,
			Publisher: msg.Publisher,
			Protocol:  msg.Protocol,
			Created:   msg.Created,
			Payload:   payload,
		},
		{
			Subtopic:  subtopic,
			Publisher: msg.Publisher,
			Protocol:  msg.Protocol,
			Created:   msg.Created,
			Payload:   payload,
		},
	}

	cases := []struct {
		desc        string
		transformer protomfx.Transformer
		msg         protomfx.Message
		msgs        []protomfx.Message
		err         error
	}{
		{
			desc:        "test transform JSON",
			msg:         msg,
			transformer: transformer,
			msgs:        msgs,
			err:         nil,
		},
		{
			desc:        "test transform JSON array",
			msg:         listMsg,
			transformer: transformer,
			msgs:        listMsgs,
			err:         nil,
		},
		{
			desc: "test transform JSON with invalid payload",
			msg:  invalid,
			msgs: nil,
			err:  mfjson.ErrTransform,
		},
		{
			desc:        "test transform JSON with timestamp transformation",
			msg:         tsMsg,
			transformer: tsMsgTr,
			msgs:        tsMsgs,
			err:         nil,
		},
		{
			desc:        "test transform JSON with timestamp transformation in micros",
			transformer: microsMsgTr,
			msg:         microsMsg,
			msgs:        microsMsgs,
			err:         nil,
		},
		{
			desc:        "test transform JSON with invalid timestamp transformation in micros",
			transformer: invalidTimeFieldTr,
			msg:         invalidTimeField,
			msgs:        nil,
			err:         mfjson.ErrInvalidTimeField,
		},
	}

	for _, tc := range cases {
		err := mfjson.TransformPayload(tc.transformer, &tc.msg)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))
	}
}
