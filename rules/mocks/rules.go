// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/rules"
)

var _ rules.RuleRepository = (*ruleRepositoryMock)(nil)

type ruleRepositoryMock struct {
	mu          sync.Mutex
	rules       map[string]rules.Rule
	assignments map[string][]string // thingID -> []ruleID
}

// NewRuleRepository creates in-memory rule repository used for testing.
func NewRuleRepository() rules.RuleRepository {
	return &ruleRepositoryMock{
		rules:       make(map[string]rules.Rule),
		assignments: make(map[string][]string),
	}
}

func (rrm *ruleRepositoryMock) Save(_ context.Context, rs ...rules.Rule) ([]rules.Rule, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	for _, r := range rs {
		rrm.rules[r.ID] = r
	}

	return rs, nil
}

func (rrm *ruleRepositoryMock) RetrieveByID(_ context.Context, id string) (rules.Rule, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	r, ok := rrm.rules[id]
	if !ok {
		return rules.Rule{}, dbutil.ErrNotFound
	}

	return r, nil
}

func (rrm *ruleRepositoryMock) RetrieveByThing(_ context.Context, thingID string, pm apiutil.PageMetadata) (rules.RulesPage, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	ruleIDs := rrm.assignments[thingID]
	var items []rules.Rule

	first := uint64(pm.Offset) + 1
	last := first + pm.Limit

	for _, rID := range ruleIDs {
		r, ok := rrm.rules[rID]
		if !ok {
			continue
		}
		id := uuid.ParseID(r.ID)
		if id >= first && id < last || pm.Limit == 0 {
			items = append(items, r)
		}
	}

	return rules.RulesPage{
		Total: uint64(len(items)),
		Rules: items,
	}, nil
}

func (rrm *ruleRepositoryMock) RetrieveByGroup(_ context.Context, groupID string, pm apiutil.PageMetadata) (rules.RulesPage, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	var items []rules.Rule
	first := uint64(pm.Offset) + 1
	last := first + pm.Limit

	for _, r := range rrm.rules {
		if r.GroupID == groupID {
			id := uuid.ParseID(r.ID)
			if id >= first && id < last || pm.Limit == 0 {
				items = append(items, r)
			}
		}
	}

	return rules.RulesPage{
		Total: uint64(len(items)),
		Rules: items,
	}, nil
}

func (rrm *ruleRepositoryMock) RetrieveThingIDsByRule(_ context.Context, ruleID string) ([]string, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	var thingIDs []string
	for thingID, rIDs := range rrm.assignments {
		for _, rID := range rIDs {
			if rID == ruleID {
				thingIDs = append(thingIDs, thingID)
				break
			}
		}
	}

	return thingIDs, nil
}

func (rrm *ruleRepositoryMock) Update(_ context.Context, r rules.Rule) error {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	if _, ok := rrm.rules[r.ID]; !ok {
		return dbutil.ErrNotFound
	}

	rrm.rules[r.ID] = r

	return nil
}

func (rrm *ruleRepositoryMock) Remove(_ context.Context, ids ...string) error {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	for _, id := range ids {
		if _, ok := rrm.rules[id]; !ok {
			return dbutil.ErrNotFound
		}
		delete(rrm.rules, id)
	}

	return nil
}

func (rrm *ruleRepositoryMock) RemoveByGroup(_ context.Context, groupID string) error {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	for id, r := range rrm.rules {
		if r.GroupID == groupID {
			delete(rrm.rules, id)
		}
	}

	return nil
}

func (rrm *ruleRepositoryMock) Assign(_ context.Context, thingID string, ruleIDs ...string) error {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	existing := make(map[string]struct{})
	for _, id := range rrm.assignments[thingID] {
		existing[id] = struct{}{}
	}

	for _, id := range ruleIDs {
		if _, ok := existing[id]; !ok {
			rrm.assignments[thingID] = append(rrm.assignments[thingID], id)
		}
	}

	return nil
}

func (rrm *ruleRepositoryMock) Unassign(_ context.Context, thingID string, ruleIDs ...string) error {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	remove := make(map[string]struct{})
	for _, id := range ruleIDs {
		remove[id] = struct{}{}
	}

	var remaining []string
	for _, id := range rrm.assignments[thingID] {
		if _, ok := remove[id]; !ok {
			remaining = append(remaining, id)
		}
	}
	rrm.assignments[thingID] = remaining

	return nil
}

func (rrm *ruleRepositoryMock) UnassignByThing(_ context.Context, thingID string) error {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	delete(rrm.assignments, thingID)

	return nil
}
