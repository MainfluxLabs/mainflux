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
