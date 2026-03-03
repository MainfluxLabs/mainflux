package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
)

var _ alarms.AlarmRepository = (*alarmRepositoryMock)(nil)

type alarmRepositoryMock struct {
	mu     sync.Mutex
	alarms map[string]alarms.Alarm
}

// NewAlarmRepository creates in-memory alarm repository used for testing.
func NewAlarmRepository() alarms.AlarmRepository {
	return &alarmRepositoryMock{
		alarms: make(map[string]alarms.Alarm),
	}
}

func (arm *alarmRepositoryMock) Save(_ context.Context, als ...alarms.Alarm) error {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	for _, a := range als {
		arm.alarms[a.ID] = a
	}

	return nil
}

func (arm *alarmRepositoryMock) RetrieveByID(_ context.Context, id string) (alarms.Alarm, error) {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	a, ok := arm.alarms[id]
	if !ok {
		return alarms.Alarm{}, dbutil.ErrNotFound
	}

	return a, nil
}

func (arm *alarmRepositoryMock) RetrieveByThing(_ context.Context, thingID string, pm apiutil.PageMetadata) (alarms.AlarmsPage, error) {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	var all, items []alarms.Alarm
	first := uint64(pm.Offset) + 1
	last := first + pm.Limit

	for _, a := range arm.alarms {
		if a.ThingID == thingID {
			all = append(all, a)
			id := uuid.ParseID(a.ID)
			if pm.Limit == 0 || (id >= first && id < last) {
				items = append(items, a)
			}
		}
	}

	return alarms.AlarmsPage{
		Total:  uint64(len(all)),
		Alarms: items,
	}, nil
}

func (arm *alarmRepositoryMock) RetrieveByGroup(_ context.Context, groupID string, pm apiutil.PageMetadata) (alarms.AlarmsPage, error) {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	var all, items []alarms.Alarm
	first := uint64(pm.Offset) + 1
	last := first + pm.Limit

	for _, a := range arm.alarms {
		if a.GroupID == groupID {
			all = append(all, a)
			id := uuid.ParseID(a.ID)
			if pm.Limit == 0 || (id >= first && id < last) {
				items = append(items, a)
			}
		}
	}

	return alarms.AlarmsPage{
		Total:  uint64(len(all)),
		Alarms: items,
	}, nil
}

func (arm *alarmRepositoryMock) RetrieveByGroups(_ context.Context, groupIDs []string, pm apiutil.PageMetadata) (alarms.AlarmsPage, error) {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	groupSet := make(map[string]struct{}, len(groupIDs))
	for _, id := range groupIDs {
		groupSet[id] = struct{}{}
	}

	var all, items []alarms.Alarm
	first := uint64(pm.Offset) + 1
	last := first + pm.Limit

	for _, a := range arm.alarms {
		if _, ok := groupSet[a.GroupID]; ok {
			all = append(all, a)
			id := uuid.ParseID(a.ID)
			if pm.Limit == 0 || (id >= first && id < last) {
				items = append(items, a)
			}
		}
	}

	return alarms.AlarmsPage{
		Total:  uint64(len(all)),
		Alarms: items,
	}, nil
}

func (arm *alarmRepositoryMock) Remove(_ context.Context, ids ...string) error {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	for _, id := range ids {
		if _, ok := arm.alarms[id]; !ok {
			return dbutil.ErrNotFound
		}
		delete(arm.alarms, id)
	}

	return nil
}

func (arm *alarmRepositoryMock) RemoveByThing(_ context.Context, thingID string) error {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	for id, a := range arm.alarms {
		if a.ThingID == thingID {
			delete(arm.alarms, id)
		}
	}

	return nil
}

func (arm *alarmRepositoryMock) RemoveByGroup(_ context.Context, groupID string) error {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	for id, a := range arm.alarms {
		if a.GroupID == groupID {
			delete(arm.alarms, id)
		}
	}

	return nil
}

func (arm *alarmRepositoryMock) ExportByThing(_ context.Context, thingID string, pm apiutil.PageMetadata) (alarms.AlarmsPage, error) {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	var all, items []alarms.Alarm
	first := uint64(pm.Offset) + 1
	last := first + pm.Limit

	for _, a := range arm.alarms {
		if a.ThingID == thingID {
			all = append(all, a)
			id := uuid.ParseID(a.ID)
			if pm.Limit == 0 || (id >= first && id < last) {
				items = append(items, a)
			}
		}
	}

	return alarms.AlarmsPage{
		Total:  uint64(len(all)),
		Alarms: items,
	}, nil
}
