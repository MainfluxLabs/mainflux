// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package transformers_test

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/transformers"
	"github.com/stretchr/testify/assert"
)

func TestSecondsToNanos(t *testing.T) {
	cases := []struct {
		desc string
		sec  float64
		want int64
		err  error
	}{
		{
			desc: "whole seconds",
			sec:  1638310819,
			want: 1638310819000000000,
			err:  nil,
		},
		{
			desc: "fractional seconds",
			sec:  1638310819.5,
			want: 1638310819500000000,
			err:  nil,
		},
		{
			desc: "zero",
			sec:  0,
			want: 0,
			err:  nil,
		},
		{
			desc: "negative seconds",
			sec:  -1638310819,
			want: -1638310819000000000,
			err:  nil,
		},
		{
			desc: "overflow",
			sec:  1e10,
			want: 0,
			err:  transformers.ErrTimestampOutOfRange,
		},
		{
			desc: "underflow",
			sec:  -1e10,
			want: 0,
			err:  transformers.ErrTimestampOutOfRange,
		},
		{
			desc: "NaN",
			sec:  math.NaN(),
			want: 0,
			err:  transformers.ErrTimestampOutOfRange,
		},
		{
			desc: "+Inf",
			sec:  math.Inf(1),
			want: 0,
			err:  transformers.ErrTimestampOutOfRange,
		},
		{
			desc: "-Inf",
			sec:  math.Inf(-1),
			want: 0,
			err:  transformers.ErrTimestampOutOfRange,
		},
	}

	for _, tc := range cases {
		ns, err := transformers.SecondsToNanos(tc.sec)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.want, ns, fmt.Sprintf("%s: expected %d, got %d", tc.desc, tc.want, ns))
	}
}

func TestTimeToNanos(t *testing.T) {
	cases := []struct {
		desc string
		val  time.Time
		want int64
		err  error
	}{
		{
			desc: "in range",
			val:  time.Unix(1638310819, 0).UTC(),
			want: 1638310819000000000,
			err:  nil,
		},
		{
			desc: "max",
			val:  time.Unix(0, math.MaxInt64).UTC(),
			want: math.MaxInt64,
			err:  nil,
		},
		{
			desc: "min",
			val:  time.Unix(0, math.MinInt64).UTC(),
			want: math.MinInt64,
			err:  nil,
		},
		{
			desc: "one nano past max",
			val:  time.Unix(0, math.MaxInt64).UTC().Add(time.Nanosecond),
			want: 0,
			err:  transformers.ErrTimestampOutOfRange,
		},
		{
			desc: "one nano before min",
			val:  time.Unix(0, math.MinInt64).UTC().Add(-time.Nanosecond),
			want: 0,
			err:  transformers.ErrTimestampOutOfRange,
		},
		{
			desc: "year 9999",
			val:  time.Date(9999, time.December, 31, 0, 0, 0, 0, time.UTC),
			want: 0,
			err:  transformers.ErrTimestampOutOfRange,
		},
	}

	for _, tc := range cases {
		ns, err := transformers.TimeToNanos(tc.val)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.want, ns, fmt.Sprintf("%s: expected %d, got %d", tc.desc, tc.want, ns))
	}
}

func TestSecondsToNanosBoundaries(t *testing.T) {
	cases := []struct {
		desc string
		sec  float64
		want int64
		err  error
	}{
		{
			desc: "max whole second",
			sec:  9223372036,
			want: 9223372036000000000,
			err:  nil,
		},
		{
			desc: "max sub-second",
			sec:  9223372036.5,
			want: 9223372036499999744,
			err:  nil,
		},
		{
			desc: "one second past max",
			sec:  9223372037,
			want: 0,
			err:  transformers.ErrTimestampOutOfRange,
		},
		{
			desc: "2^63 nanoseconds",
			sec:  math.Pow(2, 63) / 1e9,
			want: 0,
			err:  transformers.ErrTimestampOutOfRange,
		},
		{
			desc: "min whole second",
			sec:  -9223372036,
			want: -9223372036000000000,
			err:  nil,
		},
		{
			desc: "min sub-second",
			sec:  -9223372036.5,
			want: -9223372036499999744,
			err:  nil,
		},
	}

	for _, tc := range cases {
		ns, err := transformers.SecondsToNanos(tc.sec)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.want, ns, fmt.Sprintf("%s: expected %d, got %d", tc.desc, tc.want, ns))
	}
}

func TestSecondsToNanosOverflow(t *testing.T) {
	_, err := transformers.SecondsToNanos(1e10)

	assert.True(t, errors.Contains(err, transformers.ErrTimestampOutOfRange), "expected an out-of-range error")
	assert.True(t, errors.Contains(err, errors.ErrMalformedEntity), "out-of-range timestamp should map to a 400")
}
