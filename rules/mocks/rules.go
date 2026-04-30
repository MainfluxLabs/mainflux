// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"slices"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/rules"
)

var _ rules.Repository = (*ruleRepositoryMock)(nil)

type ruleRepositoryMock struct {
	mu                sync.Mutex
	rules             map[string]rules.Rule
	ruleAssignments   map[string][]string // thingID -> []ruleID
	scripts           map[string]rules.LuaScript
	scriptAssignments map[string][]string // thingID -> []scriptID
	scriptRuns        map[string]rules.ScriptRun
}

// NewRuleRepository creates in-memory rule repository used for testing.
func NewRuleRepository() rules.Repository {
	return &ruleRepositoryMock{
		rules:             make(map[string]rules.Rule),
		ruleAssignments:   make(map[string][]string),
		scripts:           make(map[string]rules.LuaScript),
		scriptAssignments: make(map[string][]string),
		scriptRuns:        make(map[string]rules.ScriptRun),
	}
}

func (rrm *ruleRepositoryMock) Save(_ context.Context, rs ...rules.Rule) ([]rules.Rule, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	for _, r := range rs {
		rrm.rules[r.ID] = r

		for _, thingID := range r.Input.ThingIDs {
			rrm.ruleAssignments[thingID] = append(rrm.ruleAssignments[thingID], r.ID)
		}
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

	var thingIDs []string
	for thingID, ruleIDs := range rrm.ruleAssignments {
		if slices.Contains(ruleIDs, id) {
			thingIDs = append(thingIDs, thingID)
		}
	}
	r.Input.ThingIDs = thingIDs

	return r, nil
}

func (rrm *ruleRepositoryMock) RetrieveByThing(_ context.Context, thingID string, pm rules.PageMetadata) (rules.RulesPage, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	ruleIDs := rrm.ruleAssignments[thingID]

	var all, items []rules.Rule
	first := uint64(pm.Offset) + 1
	last := first + pm.Limit

	for _, ruleID := range ruleIDs {
		r, ok := rrm.rules[ruleID]
		if !ok {
			continue
		}
		if pm.InputType != "" && r.Input.Type != pm.InputType {
			continue
		}
		all = append(all, r)
		id := uuid.ParseID(r.ID)
		if pm.Limit == 0 || (id >= first && id < last) {
			items = append(items, r)
		}
	}

	return rules.RulesPage{
		Total: uint64(len(all)),
		Rules: items,
	}, nil
}

func (rrm *ruleRepositoryMock) RetrieveByGroup(_ context.Context, groupID string, pm rules.PageMetadata) (rules.RulesPage, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	var all, items []rules.Rule
	first := uint64(pm.Offset) + 1
	last := first + pm.Limit

	for _, r := range rrm.rules {
		if r.GroupID == groupID && (pm.InputType == "" || r.Input.Type == pm.InputType) {
			all = append(all, r)
			id := uuid.ParseID(r.ID)
			if pm.Limit == 0 || (id >= first && id < last) {
				items = append(items, r)
			}
		}
	}

	return rules.RulesPage{
		Total: uint64(len(all)),
		Rules: items,
	}, nil
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

func (rrm *ruleRepositoryMock) AssignThings(_ context.Context, ruleID string, thingIDs ...string) error {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	if _, ok := rrm.rules[ruleID]; !ok {
		return dbutil.ErrNotFound
	}

	for _, thingID := range thingIDs {
		if slices.Contains(rrm.ruleAssignments[thingID], ruleID) {
			return dbutil.ErrConflict
		}
		rrm.ruleAssignments[thingID] = append(rrm.ruleAssignments[thingID], ruleID)
	}

	return nil
}

func (rrm *ruleRepositoryMock) UnassignThings(_ context.Context, ruleID string, thingIDs ...string) error {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	for _, thingID := range thingIDs {
		var filtered []string
		for _, id := range rrm.ruleAssignments[thingID] {
			if id != ruleID {
				filtered = append(filtered, id)
			}
		}
		rrm.ruleAssignments[thingID] = filtered
	}

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
		rrm.unassignThingsFromRule(id)
	}

	return nil
}

func (rrm *ruleRepositoryMock) RemoveByGroup(_ context.Context, groupID string) error {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	for id, r := range rrm.rules {
		if r.GroupID == groupID {
			delete(rrm.rules, id)
			rrm.unassignThingsFromRule(id)
		}
	}

	return nil
}

func (rrm *ruleRepositoryMock) UnassignRulesFromThing(_ context.Context, thingID string) error {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	delete(rrm.ruleAssignments, thingID)

	return nil
}

func (rrm *ruleRepositoryMock) unassignThingsFromRule(ruleID string) {
	for thingID, ruleIDs := range rrm.ruleAssignments {
		var filtered []string
		for _, id := range ruleIDs {
			if id != ruleID {
				filtered = append(filtered, id)
			}
		}
		rrm.ruleAssignments[thingID] = filtered
	}
}
