// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sort"
	"sync"

	"github.com/MainfluxLabs/mainflux/audit"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
)

var _ audit.EventRepository = (*eventRepositoryMock)(nil)

type eventRepositoryMock struct {
	mu     sync.Mutex
	events map[string]audit.Event
}

func NewEventRepository() audit.EventRepository {
	return &eventRepositoryMock{
		events: make(map[string]audit.Event),
	}
}

func (m *eventRepositoryMock) SaveEvent(_ context.Context, e audit.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.events[e.ID]; ok {
		return dbutil.ErrConflict
	}

	m.events[e.ID] = e
	return nil
}

func (m *eventRepositoryMock) RetrieveEvents(_ context.Context, pm audit.PageMetadata) (audit.EventsPage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	matched := make([]audit.Event, 0, len(m.events))
	for _, e := range m.events {
		if pm.Email != "" && e.ActorUserEmail != pm.Email {
			continue
		}
		if pm.Operation != "" && e.Operation != pm.Operation {
			continue
		}
		if pm.OrgID != "" && e.OrgID != pm.OrgID {
			continue
		}
		if pm.GroupID != "" && e.GroupID != pm.GroupID {
			continue
		}
		if !containsJSON(e.Data, pm.Data) {
			continue
		}
		matched = append(matched, e)
	}

	total := uint64(len(matched))
	sortEvents(matched, pm.Order, pm.Dir)
	matched = paginate(matched, pm.Offset, pm.Limit)

	out := pm
	out.Total = total
	return audit.EventsPage{
		PageMetadata: out,
		Events:       matched,
	}, nil
}

func containsJSON(haystack, needle map[string]any) bool {
	if len(needle) == 0 {
		return true
	}
	for k, nv := range needle {
		hv, ok := haystack[k]
		if !ok {
			return false
		}
		if !valueContains(hv, nv) {
			return false
		}
	}
	return true
}

func valueContains(a, b any) bool {
	am, aOk := a.(map[string]any)
	bm, bOk := b.(map[string]any)
	if aOk && bOk {
		return containsJSON(am, bm)
	}

	aArr, aIsArr := a.([]any)
	bArr, bIsArr := b.([]any)
	if aIsArr && bIsArr {
		for _, bv := range bArr {
			found := false
			for _, av := range aArr {
				if valueContains(av, bv) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}

	return a == b
}

func sortEvents(events []audit.Event, order, dir string) {
	less := func(i, j int) bool {
		var a, b string
		switch order {
		case "operation":
			a, b = events[i].Operation, events[j].Operation
		case "actor_user_email":
			a, b = events[i].ActorUserEmail, events[j].ActorUserEmail
		case "org_id":
			a, b = events[i].OrgID, events[j].OrgID
		case "group_id":
			a, b = events[i].GroupID, events[j].GroupID
		case "id":
			a, b = events[i].ID, events[j].ID
		default:
			ti, tj := events[i].OccurredAt, events[j].OccurredAt
			if dir == "asc" {
				return ti.Before(tj)
			}
			return ti.After(tj)
		}
		if dir == "asc" {
			return a < b
		}
		return a > b
	}
	sort.SliceStable(events, less)
}

func paginate(events []audit.Event, offset, limit uint64) []audit.Event {
	if offset >= uint64(len(events)) {
		return []audit.Event{}
	}
	end := offset + limit
	if limit == 0 || end > uint64(len(events)) {
		end = uint64(len(events))
	}
	return events[offset:end]
}
