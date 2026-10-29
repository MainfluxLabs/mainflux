package cron

import (
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/robfig/cron/v3"
)

var (
	errInitCron       = errors.New("failed to init cron")
	errFormatCronExpr = errors.New("failed to format cron expression")
	errAddFunc        = errors.New("failed to add cron function")
	errParseTime      = errors.New("failed to parse time")
)

type ScheduleManager struct {
	mu        sync.RWMutex
	CronByTZ  map[string]*cron.Cron
	EntryByID map[string]Entry
	TimerByID map[string]*time.Timer
}

// Entry identifies a cron entry together with the time zone of the cron that runs it,
// because entry IDs are only unique within a single cron.
type Entry struct {
	ID       cron.EntryID
	TimeZone string
}

func NewScheduleManager() *ScheduleManager {
	return &ScheduleManager{
		CronByTZ:  make(map[string]*cron.Cron),
		EntryByID: make(map[string]Entry),
		TimerByID: make(map[string]*time.Timer),
	}
}

func (sm *ScheduleManager) RemoveCronEntry(entityID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.removeCronEntry(entityID)
}

func (sm *ScheduleManager) RemoveTimer(entityID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.removeTimer(entityID)
}

func (sm *ScheduleManager) removeCronEntry(entityID string) {
	if entry, ok := sm.EntryByID[entityID]; ok {
		if c, ok := sm.CronByTZ[entry.TimeZone]; ok {
			c.Remove(entry.ID)
		}
		delete(sm.EntryByID, entityID)
	}
}

func (sm *ScheduleManager) removeTimer(entityID string) {
	if t, ok := sm.TimerByID[entityID]; ok {
		t.Stop()
		delete(sm.TimerByID, entityID)
	}
}

func (sm *ScheduleManager) InitCron(timezone string) (*cron.Cron, error) {
	sm.mu.RLock()
	c, exists := sm.CronByTZ[timezone]
	sm.mu.RUnlock()
	if exists {
		return c, nil
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, err
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// double-check under write lock
	if c, exists := sm.CronByTZ[timezone]; exists {
		return c, nil
	}

	newCron := cron.New(cron.WithLocation(location))
	newCron.Start()

	sm.CronByTZ[timezone] = newCron

	return newCron, nil
}

func (sm *ScheduleManager) ScheduleOneTimeTask(task func(), scheduler Scheduler, entityID string) error {
	scheduledDateTime, err := ParseTime(DateTimeLayout, scheduler.DateTime, scheduler.TimeZone)
	if err != nil {
		return errors.Wrap(errParseTime, err)
	}

	now := time.Now().In(scheduledDateTime.Location())
	duration := scheduledDateTime.Sub(now)

	// The lock is held until the timer is stored, because a timer with a past time fires at once.
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Concurrent updates of the same entity can each schedule a task, so the previous one is removed.
	sm.removeCronEntry(entityID)
	sm.removeTimer(entityID)

	var timer *time.Timer
	timer = time.AfterFunc(duration, func() {
		task()
		sm.mu.Lock()
		defer sm.mu.Unlock()
		// A rescheduled entity stores a new timer under the same ID, which must not be removed.
		if sm.TimerByID[entityID] == timer {
			delete(sm.TimerByID, entityID)
		}
	})
	sm.TimerByID[entityID] = timer

	return nil
}

func (sm *ScheduleManager) ScheduleRepeatingTask(task func(), scheduler Scheduler, entityID string) error {
	c, err := sm.InitCron(scheduler.TimeZone)
	if err != nil {
		return errors.Wrap(errInitCron, err)
	}

	cronExpr, err := scheduler.ToCronExpression()
	if err != nil {
		return errors.Wrap(errFormatCronExpr, err)
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	entryID, err := c.AddFunc(cronExpr, task)
	if err != nil {
		return errors.Wrap(errAddFunc, err)
	}

	// Concurrent updates of the same entity can each schedule a task, so the previous one is removed.
	sm.removeCronEntry(entityID)
	sm.removeTimer(entityID)

	sm.EntryByID[entityID] = Entry{ID: entryID, TimeZone: scheduler.TimeZone}

	return nil
}

func (sm *ScheduleManager) Stop() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, c := range sm.CronByTZ {
		c.Stop()
	}

	for _, t := range sm.TimerByID {
		t.Stop()
	}
}
