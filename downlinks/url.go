// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package downlinks

import (
	"fmt"
	"net/url"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var errUnknownInterval = errors.New("unknown time filter interval")

// formatURL appends time-filter query params (start/end) to the downlink URL.
func formatURL(d Downlink) (string, error) {
	u, err := url.Parse(d.Url)
	if err != nil {
		return "", err
	}

	startTime, endTime, err := calculateTimeRange(d.Scheduler.TimeZone, d.TimeFilter)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set(d.TimeFilter.StartParam, formatTime(startTime, d.TimeFilter.Format))
	q.Set(d.TimeFilter.EndParam, formatTime(endTime, d.TimeFilter.Format))
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func calculateTimeRange(timezone string, filter TimeFilter) (time.Time, time.Time, error) {
	var duration time.Duration

	switch filter.Interval {
	case MinuteInterval:
		duration = time.Duration(filter.Value) * time.Minute
	case HourInterval:
		duration = time.Duration(filter.Value) * time.Hour
	case DayInterval:
		duration = time.Duration(filter.Value*24) * time.Hour
	default:
		return time.Time{}, time.Time{}, errUnknownInterval
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	now := time.Now().In(loc)

	if filter.Forecast {
		return now, now.Add(duration), nil
	}
	return now.Add(-duration), now, nil
}

// getBaseURL returns scheme://host/path from a full URL (no query or fragment).
func getBaseURL(path string) (string, error) {
	u, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, u.Path), nil
}
