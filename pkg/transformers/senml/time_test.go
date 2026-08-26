// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package senml_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/transformers"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func senmlPayload(t *testing.T, ts float64) []byte {
	t.Helper()

	record := map[string]any{"n": "name", "u": "unit", "v": 42.0}
	if ts != 0 {
		record["t"] = ts
	}

	payload, err := json.Marshal([]map[string]any{record})
	require.Nil(t, err, fmt.Sprintf("marshalling SenML record: %s", err))

	return payload
}

func TestTransformPayloadTime(t *testing.T) {
	const created = int64(1638310819000000000)

	cases := []struct {
		desc string
		time float64
		want int64
		err  error
	}{
		{
			desc: "seconds scaled to nanoseconds",
			time: 1638310819,
			want: 1638310819000000000,
			err:  nil,
		},
		{
			desc: "fractional seconds scaled to nanoseconds",
			time: 1638310819.5,
			want: 1638310819500000000,
			err:  nil,
		},
		{
			desc: "missing time falls back to reception timestamp",
			time: 0,
			want: created,
			err:  nil,
		},
		{
			desc: "seconds beyond int64 nanoseconds",
			time: 1e10,
			want: 0,
			err:  transformers.ErrTimestampOutOfRange,
		},
		{
			desc: "seconds below int64 nanoseconds",
			time: -1e10,
			want: 0,
			err:  transformers.ErrTimestampOutOfRange,
		},
	}

	for _, tc := range cases {
		msg := protomfx.Message{
			Payload:     senmlPayload(t, tc.time),
			ContentType: senml.JSON,
			Created:     created,
		}

		err := senml.TransformPayload(&msg)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s, got %s", tc.desc, tc.err, err))
		if tc.err != nil {
			continue
		}

		var msgs []senml.Message
		require.Nil(t, json.Unmarshal(msg.Payload, &msgs), fmt.Sprintf("%s: unmarshalling transformed payload", tc.desc))
		require.Len(t, msgs, 1, tc.desc)
		assert.Equal(t, tc.want, msgs[0].Time, fmt.Sprintf("%s: expected %d, got %d", tc.desc, tc.want, msgs[0].Time))
	}
}

func TestTransformPayloadTimeOutOfRangeIsMalformedEntity(t *testing.T) {
	msg := protomfx.Message{
		Payload:     senmlPayload(t, 1e10),
		ContentType: senml.JSON,
	}

	err := senml.TransformPayload(&msg)

	assert.True(t, errors.Contains(err, errors.ErrMalformedEntity), "out-of-range timestamp should map to a 400")
	assert.Equal(t, transformers.ErrTimestampOutOfRange.Error(), err.(errors.Error).Msg(), "specific reason should survive to the response body")
}
