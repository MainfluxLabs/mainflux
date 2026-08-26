// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package json_test

import (
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/transformers"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/stretchr/testify/assert"
)

func TestParseTimestamp(t *testing.T) {
	const expected = int64(1638310819000000000)
	cases := []struct {
		desc   string
		format string
		val    any
		ns     int64
		err    error
	}{
		{
			desc:   "unix seconds",
			format: "unix",
			val:    "1638310819",
			ns:     expected,
			err:    nil,
		},
		{
			desc:   "unix milliseconds",
			format: "unix_ms",
			val:    "1638310819000",
			ns:     expected,
			err:    nil,
		},
		{
			desc:   "unix microseconds",
			format: "unix_us",
			val:    "1638310819000000",
			ns:     expected,
			err:    nil,
		},
		{
			desc:   "unix nanoseconds",
			format: "unix_ns",
			val:    "1638310819000000000",
			ns:     expected,
			err:    nil,
		},
		{
			desc:   "rfc3339",
			format: "rfc3339",
			val:    "2021-11-30T22:20:19Z",
			ns:     expected,
			err:    nil,
		},
		{
			desc:   "unix seconds beyond int64 nanoseconds",
			format: "unix",
			val:    "10000000000",
			ns:     0,
			err:    transformers.ErrTimestampOutOfRange,
		},
		{
			desc:   "unix milliseconds beyond int64 nanoseconds",
			format: "unix_ms",
			val:    "10000000000000",
			ns:     0,
			err:    transformers.ErrTimestampOutOfRange,
		},
		{
			desc:   "unix microseconds beyond int64 nanoseconds",
			format: "unix_us",
			val:    "10000000000000000",
			ns:     0,
			err:    transformers.ErrTimestampOutOfRange,
		},
		{
			desc:   "rfc3339 beyond int64 nanoseconds",
			format: "rfc3339",
			val:    "9999-12-31T00:00:00Z",
			ns:     0,
			err:    transformers.ErrTimestampOutOfRange,
		},
		{
			desc:   "float seconds beyond int64 nanoseconds",
			format: "unix",
			val:    float64(1e30),
			ns:     0,
			err:    transformers.ErrTimestampOutOfRange,
		},
	}

	for _, tc := range cases {
		tr := domain.Transformer{TimeFormat: tc.format}
		ns, err := mfjson.ParseTimestamp(tr, tc.val)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.ns, ns, fmt.Sprintf("%s: expected %d, got %d", tc.desc, tc.ns, ns))
	}
}

func TestParseTimestampOutOfRangeIsMalformedEntity(t *testing.T) {
	tr := domain.Transformer{TimeFormat: "unix"}
	_, err := mfjson.ParseTimestamp(tr, "10000000000")

	assert.True(t, errors.Contains(err, errors.ErrMalformedEntity), "out-of-range timestamp should map to a 400")
	assert.Equal(t, transformers.ErrTimestampOutOfRange.Error(), err.(errors.Error).Msg(), "specific reason should survive to the response body")
}
