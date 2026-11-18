// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package senml_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
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
			desc: "whole seconds",
			time: 1638310819,
			want: 1638310819000000000,
			err:  nil,
		},
		{
			desc: "fractional seconds",
			time: 1638310819.5,
			want: 1638310819500000000,
			err:  nil,
		},
		{
			desc: "missing time",
			time: 0,
			want: created,
			err:  nil,
		},
		{
			desc: "overflow",
			time: 1e10,
			want: 0,
			err:  transformers.ErrTimestampOutOfRange,
		},
		{
			desc: "underflow",
			time: -1e10,
			want: 0,
			err:  transformers.ErrTimestampOutOfRange,
		},
	}

	for _, tc := range cases {
		records, err := senml.TransformPayload(senml.JSON, senmlPayload(t, tc.time), created)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s, got %s", tc.desc, tc.err, err))
		if tc.err != nil {
			continue
		}

		require.Len(t, records, 1, tc.desc)
		resMsg := fmt.Sprintf("%s: expected %d, got %d", tc.desc, tc.want, records[0].Time)
		assert.Equal(t, tc.want, records[0].Time, resMsg)
	}
}

func TestTransformPayloadTimeOutOfRange(t *testing.T) {
	_, err := senml.TransformPayload(senml.JSON, senmlPayload(t, 1e10), 0)

	assert.True(t, errors.Contains(err, errors.ErrMalformedEntity), "out-of-range timestamp should map to a 400")
	assert.Equal(t, transformers.ErrTimestampOutOfRange.Error(), err.(errors.Error).Msg(), "specific reason should survive to the response body")
}
