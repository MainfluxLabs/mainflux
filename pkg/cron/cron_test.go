// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cron

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduleOneTimeTask(t *testing.T) {
	sm := NewScheduleManager()
	now := time.Now().UTC()

	expired := Scheduler{
		Frequency: OnceFreq,
		DateTime:  now.Format(DateTimeLayout),
		TimeZone:  "UTC",
	}
	pending := Scheduler{
		Frequency: OnceFreq,
		DateTime:  now.Add(time.Hour).Format(DateTimeLayout),
		TimeZone:  "UTC",
	}

	cases := []struct {
		desc      string
		id        string
		scheduler Scheduler
	}{
		{
			desc:      "reschedule timer while previous task runs",
			id:        "rescheduled",
			scheduler: pending,
		},
	}

	for _, tc := range cases {
		started := make(chan struct{})
		release := make(chan struct{})
		task := func() {
			close(started)
			<-release
		}
		err := sm.ScheduleOneTimeTask(task, expired, tc.id)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		<-started

		sm.RemoveTimer(tc.id)
		err = sm.ScheduleOneTimeTask(func() {}, tc.scheduler, tc.id)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		close(release)

		removed := func() bool {
			sm.mu.RLock()
			defer sm.mu.RUnlock()
			_, ok := sm.TimerByID[tc.id]
			return !ok
		}
		assert.Never(t, removed, 100*time.Millisecond, 10*time.Millisecond, fmt.Sprintf("%s: expected new timer to remain stored", tc.desc))
	}

	replaced := []struct {
		desc     string
		previous Scheduler
	}{
		{
			desc:     "reschedule one-time task over pending timer",
			previous: pending,
		},
		{
			desc: "reschedule repeating task as one-time task",
			previous: Scheduler{
				Frequency: MinutelyFreq,
				Minute:    5,
				TimeZone:  "UTC",
			},
		},
	}

	for _, tc := range replaced {
		sm := NewScheduleManager()
		id := "entity-id"

		var err error
		if tc.previous.Frequency == OnceFreq {
			err = sm.ScheduleOneTimeTask(func() {}, tc.previous, id)
		} else {
			err = sm.ScheduleRepeatingTask(func() {}, tc.previous, id)
		}
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		previous := sm.TimerByID[id]

		err = sm.ScheduleOneTimeTask(func() {}, pending, id)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))

		entries := 0
		for _, c := range sm.CronByTZ {
			entries += len(c.Entries())
		}
		assert.Equal(t, 0, entries, fmt.Sprintf("%s: expected previous cron entry to be removed", tc.desc))
		if previous != nil {
			assert.False(t, previous.Stop(), fmt.Sprintf("%s: expected previous timer to be stopped", tc.desc))
		}
		sm.Stop()
	}
}

func TestScheduleRepeatingTask(t *testing.T) {
	minutely := Scheduler{
		Frequency: MinutelyFreq,
		Minute:    5,
		TimeZone:  "UTC",
	}
	hourly := Scheduler{
		Frequency: HourlyFreq,
		Hour:      1,
		TimeZone:  "UTC",
	}
	otherZone := Scheduler{
		Frequency: MinutelyFreq,
		Minute:    5,
		TimeZone:  "Europe/Belgrade",
	}
	once := Scheduler{
		Frequency: OnceFreq,
		DateTime:  time.Now().UTC().Add(time.Hour).Format(DateTimeLayout),
		TimeZone:  "UTC",
	}

	cases := []struct {
		desc      string
		previous  Scheduler
		scheduler Scheduler
	}{
		{
			desc:      "reschedule repeating task in the same time zone",
			previous:  minutely,
			scheduler: hourly,
		},
		{
			desc:      "reschedule repeating task in another time zone",
			previous:  minutely,
			scheduler: otherZone,
		},
		{
			desc:      "reschedule one-time task as repeating task",
			previous:  once,
			scheduler: minutely,
		},
	}

	for _, tc := range cases {
		sm := NewScheduleManager()
		id := "entity-id"

		var err error
		if tc.previous.Frequency == OnceFreq {
			err = sm.ScheduleOneTimeTask(func() {}, tc.previous, id)
		} else {
			err = sm.ScheduleRepeatingTask(func() {}, tc.previous, id)
		}
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))

		err = sm.ScheduleRepeatingTask(func() {}, tc.scheduler, id)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))

		entries := 0
		for _, c := range sm.CronByTZ {
			entries += len(c.Entries())
		}
		_, timer := sm.TimerByID[id]
		assert.Equal(t, 1, entries, fmt.Sprintf("%s: expected one cron entry", tc.desc))
		assert.Equal(t, tc.scheduler.TimeZone, sm.EntryByID[id].TimeZone, fmt.Sprintf("%s: unexpected time zone", tc.desc))
		assert.False(t, timer, fmt.Sprintf("%s: expected previous timer to be removed", tc.desc))
		sm.Stop()
	}
}

func TestRemoveTimer(t *testing.T) {
	sm := NewScheduleManager()
	now := time.Now().UTC()

	cases := []struct {
		desc      string
		id        string
		scheduler Scheduler
	}{
		{
			desc: "remove pending timer",
			id:   "pending",
			scheduler: Scheduler{
				Frequency: OnceFreq,
				DateTime:  now.Add(time.Hour).Format(DateTimeLayout),
				TimeZone:  "UTC",
			},
		},
		{
			desc: "remove expired timer",
			id:   "expired",
			scheduler: Scheduler{
				Frequency: OnceFreq,
				DateTime:  now.Format(DateTimeLayout),
				TimeZone:  "UTC",
			},
		},
	}

	for _, tc := range cases {
		err := sm.ScheduleOneTimeTask(func() {}, tc.scheduler, tc.id)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))

		sm.RemoveTimer(tc.id)

		sm.mu.RLock()
		_, ok := sm.TimerByID[tc.id]
		sm.mu.RUnlock()
		assert.False(t, ok, fmt.Sprintf("%s: expected timer to be removed", tc.desc))
	}
}
