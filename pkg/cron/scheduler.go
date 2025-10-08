package cron

import (
	"fmt"
	"strings"
)

type Scheduler struct {
	TimeZone  string `json:"time_zone"`
	Frequency string `json:"frequency"`
	DateTime  string `json:"date_time"`
	Week      Week   `json:"week"`
	DayTime   string `json:"day_time"`
	Hour      int    `json:"hour"`
	Minute    int    `json:"minute"`
}

type Week struct {
	Days []string `json:"days"`
	Time string   `json:"time"`
}

func (s Scheduler) ToCronExpression() (string, error) {
	format := "%v %v %v %v %v"

	switch s.Frequency {
	case WeeklyFreq:
		parsedTime, err := ParseTime(TimeLayout, s.Week.Time, s.TimeZone)
		if err != nil {
			return "", err
		}

		hour := parsedTime.Hour()
		minute := parsedTime.Minute()
		formattedDays := strings.Join(s.Week.Days, ",")

		return fmt.Sprintf(format, minute, hour, "*", "*", formattedDays), nil
	case DailyFreq:
		t, err := ParseTime(TimeLayout, s.DayTime, s.TimeZone)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(format, t.Minute(), t.Hour(), "*", "*", "*"), nil
	case HourlyFreq:
		hours := fmt.Sprintf("*/%d", s.Hour)
		return fmt.Sprintf(format, 0, hours, "*", "*", "*"), nil
	case MinutelyFreq:
		minutes := fmt.Sprintf("*/%d", s.Minute)
		return fmt.Sprintf(format, minutes, "*", "*", "*", "*"), nil
	default:
		return "", nil
	}
}

func NormalizeTimezone(scheduler Scheduler) Scheduler {
	if scheduler.TimeZone == "" {
		scheduler.TimeZone = "UTC"
	}
	return scheduler
}
