// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pyscripts"
)

var _ pyscripts.ScriptRepository = (*scriptRepositoryMock)(nil)

type scriptRepositoryMock struct {
	mu      sync.Mutex
	scripts map[string]pyscripts.Script
	runs    map[string]pyscripts.ScriptRun
}

func NewScriptRepository() pyscripts.ScriptRepository {
	return &scriptRepositoryMock{
		scripts: make(map[string]pyscripts.Script),
		runs:    make(map[string]pyscripts.ScriptRun),
	}
}

func (srm *scriptRepositoryMock) SaveScripts(_ context.Context, scripts ...pyscripts.Script) ([]pyscripts.Script, error) {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	for _, s := range scripts {
		if _, ok := srm.scripts[s.ID]; ok {
			return []pyscripts.Script{}, dbutil.ErrConflict
		}
	}

	now := time.Now()
	saved := make([]pyscripts.Script, 0, len(scripts))
	for i := range scripts {
		scripts[i].CreatedAt = now
		scripts[i].UpdatedAt = now
		srm.scripts[scripts[i].ID] = scripts[i]
		saved = append(saved, scripts[i])
	}

	return saved, nil
}

func (srm *scriptRepositoryMock) RetrieveScriptByID(_ context.Context, id string) (pyscripts.Script, error) {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	s, ok := srm.scripts[id]
	if !ok {
		return pyscripts.Script{}, dbutil.ErrNotFound
	}

	return s, nil
}

func (srm *scriptRepositoryMock) RetrieveScriptsByGroup(_ context.Context, groupID string, pm pyscripts.PageMetadata) (pyscripts.ScriptsPage, error) {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	var matched []pyscripts.Script
	for _, s := range srm.scripts {
		if s.GroupID != groupID {
			continue
		}
		if pm.Name != "" && !strings.Contains(strings.ToLower(s.Name), strings.ToLower(pm.Name)) {
			continue
		}
		matched = append(matched, s)
	}

	sort.Slice(matched, func(i, j int) bool { return matched[i].ID < matched[j].ID })

	total := uint64(len(matched))
	if pm.Offset < total {
		matched = matched[pm.Offset:]
	} else {
		matched = nil
	}
	if pm.Limit > 0 && pm.Limit < uint64(len(matched)) {
		matched = matched[:pm.Limit]
	}

	return pyscripts.ScriptsPage{Total: total, Scripts: matched}, nil
}

func (srm *scriptRepositoryMock) UpdateScript(_ context.Context, script pyscripts.Script) error {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	existing, ok := srm.scripts[script.ID]
	if !ok {
		return dbutil.ErrNotFound
	}

	existing.Name = script.Name
	existing.Description = script.Description
	existing.Source = script.Source
	existing.Checksum = script.Checksum
	existing.UpdatedAt = time.Now()
	srm.scripts[script.ID] = existing

	return nil
}

func (srm *scriptRepositoryMock) RemoveScripts(_ context.Context, ids ...string) error {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	for _, id := range ids {
		delete(srm.scripts, id)
		for rid, run := range srm.runs {
			if run.ScriptID == id {
				delete(srm.runs, rid)
			}
		}
	}

	return nil
}

func (srm *scriptRepositoryMock) RemoveScriptsByGroup(_ context.Context, groupID string) error {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	for id, s := range srm.scripts {
		if s.GroupID == groupID {
			delete(srm.scripts, id)
		}
	}

	return nil
}

func (srm *scriptRepositoryMock) SaveRuns(_ context.Context, runs ...pyscripts.ScriptRun) ([]pyscripts.ScriptRun, error) {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	for _, r := range runs {
		if _, ok := srm.scripts[r.ScriptID]; !ok {
			return []pyscripts.ScriptRun{}, dbutil.ErrNotFound
		}
	}

	saved := make([]pyscripts.ScriptRun, 0, len(runs))
	for _, r := range runs {
		srm.runs[r.ID] = r
		saved = append(saved, r)
	}

	return saved, nil
}

func (srm *scriptRepositoryMock) RetrieveRunByID(_ context.Context, id string) (pyscripts.ScriptRun, error) {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	r, ok := srm.runs[id]
	if !ok {
		return pyscripts.ScriptRun{}, dbutil.ErrNotFound
	}

	return r, nil
}

func (srm *scriptRepositoryMock) RetrieveRunsByScript(_ context.Context, scriptID string, pm pyscripts.PageMetadata) (pyscripts.ScriptRunsPage, error) {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	var matched []pyscripts.ScriptRun
	for _, r := range srm.runs {
		if r.ScriptID != scriptID {
			continue
		}
		if pm.Status != "" && r.Status != pm.Status {
			continue
		}
		matched = append(matched, r)
	}

	sort.Slice(matched, func(i, j int) bool { return matched[i].ID < matched[j].ID })

	total := uint64(len(matched))
	if pm.Offset < total {
		matched = matched[pm.Offset:]
	} else {
		matched = nil
	}
	if pm.Limit > 0 && pm.Limit < uint64(len(matched)) {
		matched = matched[:pm.Limit]
	}

	return pyscripts.ScriptRunsPage{Total: total, Runs: matched}, nil
}

func (srm *scriptRepositoryMock) RemoveRuns(_ context.Context, ids ...string) error {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	for _, id := range ids {
		delete(srm.runs, id)
	}

	return nil
}
