// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package json

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/transformers"
)

var errUnsupportedFormat = errors.New("unsupported time format")

const baseISO8601Format = "2006-01-02T15:04:05"

// ParseTimestamp parses v using the transformer's time_format and time_location and returns nanoseconds.
func ParseTimestamp(tr domain.Transformer, v any) (int64, error) {
	t, err := parseTimestamp(tr.TimeFormat, v, tr.TimeLocation)
	if err != nil {
		return 0, err
	}
	return transformers.TimeToNanos(t)
}

func parseTimestamp(format string, timestamp any, location string) (time.Time, error) {
	switch format {
	case "unix", "unix_ms", "unix_us", "unix_ns":
		return parseUnix(format, timestamp)
	default:
		if location == "" {
			location = "UTC"
		}
		return parseTime(format, timestamp, location)
	}
}

func parseUnix(format string, timestamp any) (time.Time, error) {
	integer, fractional, err := parseComponents(timestamp)
	if err != nil {
		return time.Unix(0, 0), err
	}

	switch strings.ToLower(format) {
	case "unix":
		return time.Unix(integer, fractional).UTC(), nil
	case "unix_ms":
		return time.UnixMilli(integer).UTC(), nil
	case "unix_us":
		return time.UnixMicro(integer).UTC(), nil
	case "unix_ns":
		return time.Unix(0, integer).UTC(), nil
	default:
		return time.Unix(0, 0), errUnsupportedFormat
	}
}

func parseComponents(timestamp any) (int64, int64, error) {
	switch ts := timestamp.(type) {
	case string:
		parts := strings.SplitN(ts, ".", 2)
		if len(parts) == 2 {
			return parseUnixTimeComponents(parts[0], parts[1])
		}

		parts = strings.SplitN(ts, ",", 2)
		if len(parts) == 2 {
			return parseUnixTimeComponents(parts[0], parts[1])
		}

		integer, err := strconv.ParseInt(ts, 10, 64)
		if err != nil {
			return 0, 0, err
		}
		return integer, 0, nil
	case int8:
		return int64(ts), 0, nil
	case int16:
		return int64(ts), 0, nil
	case int32:
		return int64(ts), 0, nil
	case int64:
		return ts, 0, nil
	case uint8:
		return int64(ts), 0, nil
	case uint16:
		return int64(ts), 0, nil
	case uint32:
		return int64(ts), 0, nil
	case uint64:
		return int64(ts), 0, nil
	case float32:
		return parseFloatComponents(float64(ts))
	case float64:
		return parseFloatComponents(ts)
	default:
		return 0, 0, errUnsupportedFormat
	}
}

func parseFloatComponents(ts float64) (int64, int64, error) {
	integer, fractional := math.Modf(ts)
	if _, err := transformers.SecondsToNanos(integer); err != nil {
		return 0, 0, err
	}
	return int64(integer), int64(fractional * 1e9), nil
}

func parseUnixTimeComponents(first, second string) (int64, int64, error) {
	integer, err := strconv.ParseInt(first, 10, 64)
	if err != nil {
		return 0, 0, err
	}

	// Convert to nanoseconds, dropping any greater precision.
	buf := []byte("000000000")
	copy(buf, second)

	fractional, err := strconv.ParseInt(string(buf), 10, 64)
	if err != nil {
		return 0, 0, err
	}
	return integer, fractional, nil
}

func parseTime(format string, timestamp any, location string) (time.Time, error) {
	switch ts := timestamp.(type) {
	case string:
		loc, err := time.LoadLocation(location)
		if err != nil {
			return time.Unix(0, 0), err
		}
		switch strings.ToLower(format) {
		case "ansic":
			format = time.ANSIC
		case "unixdate":
			format = time.UnixDate
		case "rubydate":
			format = time.RubyDate
		case "rfc822":
			format = time.RFC822
		case "rfc822z":
			format = time.RFC822Z
		case "rfc850":
			format = time.RFC850
		case "rfc1123":
			format = time.RFC1123
		case "rfc1123z":
			format = time.RFC1123Z
		case "rfc3339":
			format = time.RFC3339
		case "rfc3339nano":
			format = time.RFC3339Nano
		case "stamp":
			format = time.Stamp
		case "stampmilli":
			format = time.StampMilli
		case "stampmicro":
			format = time.StampMicro
		case "stampnano":
			format = time.StampNano
		case "iso8601":
			format = baseISO8601Format
		}
		return time.ParseInLocation(format, ts, loc)
	default:
		return time.Unix(0, 0), errUnsupportedFormat
	}
}
