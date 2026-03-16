// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/rules"
)

func (rrm *ruleRepositoryMock) SaveScripts(_ context.Context, scripts ...rules.LuaScript) ([]rules.LuaScript, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	for _, s := range scripts {
		rrm.scripts[s.ID] = s
	}

	return scripts, nil
}

func (rrm *ruleRepositoryMock) RetrieveScriptByID(_ context.Context, id string) (rules.LuaScript, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	s, ok := rrm.scripts[id]
	if !ok {
		return rules.LuaScript{}, dbutil.ErrNotFound
	}

	return s, nil
}

func (rrm *ruleRepositoryMock) RetrieveScriptsByThing(_ context.Context, thingID string, pm rules.PageMetadata) (rules.LuaScriptsPage, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	scriptIDs := rrm.scriptAssignments[thingID]
	var all, items []rules.LuaScript

	first := uint64(pm.Offset) + 1
	last := first + pm.Limit

	for _, sID := range scriptIDs {
		s, ok := rrm.scripts[sID]
		if !ok {
			continue
		}
		all = append(all, s)
		id := uuid.ParseID(s.ID)
		if pm.Limit == 0 || (id >= first && id < last) {
			items = append(items, s)
		}
	}

	return rules.LuaScriptsPage{
		Total:   uint64(len(all)),
		Scripts: items,
	}, nil
}

func (rrm *ruleRepositoryMock) RetrieveScriptsByGroup(_ context.Context, groupID string, pm rules.PageMetadata) (rules.LuaScriptsPage, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	var all, items []rules.LuaScript
	first := uint64(pm.Offset) + 1
	last := first + pm.Limit

	for _, s := range rrm.scripts {
		if s.GroupID == groupID {
			all = append(all, s)
			id := uuid.ParseID(s.ID)
			if pm.Limit == 0 || (id >= first && id < last) {
				items = append(items, s)
			}
		}
	}

	return rules.LuaScriptsPage{
		Total:   uint64(len(all)),
		Scripts: items,
	}, nil
}

func (rrm *ruleRepositoryMock) RetrieveThingIDsByScript(_ context.Context, scriptID string) ([]string, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	var thingIDs []string
	for thingID, sIDs := range rrm.scriptAssignments {
		for _, sID := range sIDs {
			if sID == scriptID {
				thingIDs = append(thingIDs, thingID)
				break
			}
		}
	}

	return thingIDs, nil
}

func (rrm *ruleRepositoryMock) UpdateScript(_ context.Context, script rules.LuaScript) error {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	if _, ok := rrm.scripts[script.ID]; !ok {
		return dbutil.ErrNotFound
	}

	rrm.scripts[script.ID] = script

	return nil
}

func (rrm *ruleRepositoryMock) RemoveScripts(_ context.Context, ids ...string) error {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	for _, id := range ids {
		delete(rrm.scripts, id)
	}

	return nil
}

func (rrm *ruleRepositoryMock) RemoveScriptsByGroup(_ context.Context, groupID string) error {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	for id, s := range rrm.scripts {
		if s.GroupID == groupID {
			delete(rrm.scripts, id)
		}
	}

	return nil
}

func (rrm *ruleRepositoryMock) AssignScripts(_ context.Context, thingID string, scriptIDs ...string) error {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	existing := make(map[string]struct{})
	for _, id := range rrm.scriptAssignments[thingID] {
		existing[id] = struct{}{}
	}

	for _, id := range scriptIDs {
		if _, ok := existing[id]; !ok {
			rrm.scriptAssignments[thingID] = append(rrm.scriptAssignments[thingID], id)
		}
	}

	return nil
}

func (rrm *ruleRepositoryMock) UnassignScripts(_ context.Context, thingID string, scriptIDs ...string) error {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	remove := make(map[string]struct{})
	for _, id := range scriptIDs {
		remove[id] = struct{}{}
	}

	var remaining []string
	for _, id := range rrm.scriptAssignments[thingID] {
		if _, ok := remove[id]; !ok {
			remaining = append(remaining, id)
		}
	}
	rrm.scriptAssignments[thingID] = remaining

	return nil
}

func (rrm *ruleRepositoryMock) UnassignScriptsFromThing(_ context.Context, thingID string) error {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	delete(rrm.scriptAssignments, thingID)

	return nil
}

func (rrm *ruleRepositoryMock) SaveScriptRuns(_ context.Context, runs ...rules.ScriptRun) ([]rules.ScriptRun, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	for _, run := range runs {
		rrm.scriptRuns[run.ID] = run
	}

	return runs, nil
}

func (rrm *ruleRepositoryMock) RetrieveScriptRunByID(_ context.Context, id string) (rules.ScriptRun, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	run, ok := rrm.scriptRuns[id]
	if !ok {
		return rules.ScriptRun{}, dbutil.ErrNotFound
	}

	return run, nil
}

func (rrm *ruleRepositoryMock) RetrieveScriptRunsByThing(_ context.Context, thingID string, pm rules.PageMetadata) (rules.ScriptRunsPage, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	var all, items []rules.ScriptRun
	first := uint64(pm.Offset) + 1
	last := first + pm.Limit

	for _, run := range rrm.scriptRuns {
		if run.ThingID == thingID {
			all = append(all, run)
			id := uuid.ParseID(run.ID)
			if pm.Limit == 0 || (id >= first && id < last) {
				items = append(items, run)
			}
		}
	}

	return rules.ScriptRunsPage{
		Total: uint64(len(all)),
		Runs:  items,
	}, nil
}

func (rrm *ruleRepositoryMock) RemoveScriptRuns(_ context.Context, ids ...string) error {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	for _, id := range ids {
		delete(rrm.scriptRuns, id)
	}

	return nil
}
