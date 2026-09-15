// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package transformers

import (
	"math"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var ErrTimestampOutOfRange = errors.New("timestamp out of representable range")

const (
	maxUnixNanoseconds = math.MaxInt64
	minUnixNanoseconds = math.MinInt64
	nanosPerSecond     = 1_000_000_000
)

var (
	minNanoTime = time.Unix(0, minUnixNanoseconds).UTC()
	maxNanoTime = time.Unix(0, maxUnixNanoseconds).UTC()
)

func SecondsToNanos(sec float64) (int64, error) {
	ns := sec * nanosPerSecond
	if err := validateNanos(ns); err != nil {
		return 0, err
	}

	return int64(ns), nil
}

func TimeToNanos(t time.Time) (int64, error) {
	if t.Before(minNanoTime) || t.After(maxNanoTime) {
		return 0, errors.Wrap(ErrTimestampOutOfRange, errors.ErrMalformedEntity)
	}

	return t.UnixNano(), nil
}

func validateNanos(ns float64) error {
	if math.IsNaN(ns) || ns >= float64(maxUnixNanoseconds) || ns < float64(minUnixNanoseconds) {
		return errors.Wrap(ErrTimestampOutOfRange, errors.ErrMalformedEntity)
	}

	return nil
}
