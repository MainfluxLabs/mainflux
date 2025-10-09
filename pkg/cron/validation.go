package cron

import "time"

const (
	minHour   = 1
	maxHour   = 23
	minMinute = 1
	maxMinute = 59

	TimeLayout     = "15:04"
	DateTimeLayout = "2006-01-02 15:04"
	OnceFreq       = "once"
	WeeklyFreq     = "weekly"
	DailyFreq      = "daily"
	HourlyFreq     = "hourly"
	MinutelyFreq   = "minutely"
)

var WeekDays = []string{"SUN", "MON", "TUE", "WED", "THU", "FRI", "SAT"}

func (s Scheduler) IsValid() bool {
	valid := false

	switch s.Frequency {
	case WeeklyFreq:
		valid = isValidWeek(s.Week, s.TimeZone)
	case DailyFreq:
		valid = isValidTime(s.DayTime, s.TimeZone)
	case HourlyFreq:
		valid = s.Hour >= minHour && s.Hour <= maxHour
	case MinutelyFreq:
		valid = s.Minute >= minMinute && s.Minute <= maxMinute
	case OnceFreq:
		valid = isValidDateTime(s.DateTime, s.TimeZone)
	}

	return valid
}

func isValidWeek(week Week, timezone string) bool {
	return isValidDay(week) && isValidTime(week.Time, timezone)
}

func isValidDay(week Week) bool {
	for _, day := range week.Days {
		valid := false
		for _, validDay := range WeekDays {
			if day == validDay {
				valid = true
				break
			}
		}
		if !valid {
			return false
		}
	}

	return true
}

func isValidTime(time, timezone string) bool {
	_, err := ParseTime(TimeLayout, time, timezone)

	return err == nil
}

func isValidDateTime(dt, timezone string) bool {
	datetime, err := ParseTime(DateTimeLayout, dt, timezone)
	if err != nil {
		return false
	}

	now := time.Now().In(datetime.Location())
	if datetime.Before(now) {
		return false
	}

	duration := datetime.Sub(now)

	return duration >= time.Minute
}

func ParseTime(layout, t, timezone string) (time.Time, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, err
	}

	parsedTime, err := time.ParseInLocation(layout, t, location)
	if err != nil {
		return time.Time{}, err
	}

	return parsedTime, nil
}
