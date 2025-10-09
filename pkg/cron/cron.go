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
	EntryByID map[string]cron.EntryID
	TimerByID map[string]*time.Timer
}

func NewScheduleManager() *ScheduleManager {
	return &ScheduleManager{
		CronByTZ:  make(map[string]*cron.Cron),
		EntryByID: make(map[string]cron.EntryID),
		TimerByID: make(map[string]*time.Timer),
	}
}

func (sm *ScheduleManager) RemoveCronEntry(entityID, timezone string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if entryID, ok := sm.EntryByID[entityID]; ok {
		if c, ok := sm.CronByTZ[timezone]; ok {
			c.Remove(entryID)
		}
		delete(sm.EntryByID, entityID)
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

	timer := time.AfterFunc(duration, func() {
		task()
		sm.mu.Lock()
		defer sm.mu.Unlock()
		delete(sm.TimerByID, entityID)
	})

	sm.mu.Lock()
	defer sm.mu.Unlock()
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

	entryID, err := c.AddFunc(cronExpr, task)
	if err != nil {
		return errors.Wrap(errAddFunc, err)
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.EntryByID[entityID] = entryID

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
