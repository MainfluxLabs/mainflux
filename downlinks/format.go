// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package downlinks

import (
	"fmt"
	"strings"
	"time"
)

const (
	baseISO8601Format    = "2006-01-02T15:04:05"
	compactISO8601Format = "200601021504"
)

// Layout returns the Go time layout string for the given named format.
// Returns an empty string for unrecognized format names.
func Layout(format string) string {
	switch strings.ToLower(format) {
	case "ansic":
		return time.ANSIC
	case "unixdate":
		return time.UnixDate
	case "rubydate":
		return time.RubyDate
	case "rfc822":
		return time.RFC822
	case "rfc822z":
		return time.RFC822Z
	case "rfc850":
		return time.RFC850
	case "rfc1123":
		return time.RFC1123
	case "rfc1123z":
		return time.RFC1123Z
	case "rfc3339":
		return time.RFC3339
	case "rfc3339nano":
		return time.RFC3339Nano
	case "stamp":
		return time.Stamp
	case "stampmilli":
		return time.StampMilli
	case "stampmicro":
		return time.StampMicro
	case "stampnano":
		return time.StampNano
	case "iso8601":
		return baseISO8601Format
	case "datetime":
		return time.DateTime
	case "compactiso8601":
		return compactISO8601Format
	}

	return ""
}

// IsValidFormat reports whether the given format string is a recognised time format.
func IsValidFormat(format string) bool {
	switch strings.ToLower(format) {
	case "unix", "unix_ms", "unix_us", "unix_ns":
		return true
	default:
		return Layout(format) != ""
	}
}

func formatTime(t time.Time, format string) string {
	switch strings.ToLower(format) {
	case "unix":
		return fmt.Sprintf("%d", t.Unix())
	case "unix_ms":
		return fmt.Sprintf("%d", t.UnixNano()/1e6)
	case "unix_us":
		return fmt.Sprintf("%d", t.UnixNano()/1e3)
	case "unix_ns":
		return fmt.Sprintf("%d", t.UnixNano())
	default:
		return t.Format(Layout(format))
	}
}
