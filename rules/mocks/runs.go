// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/rules"
)

func (rrm *ruleRepositoryMock) SaveRuns(_ context.Context, runs ...rules.RuleRun) ([]rules.RuleRun, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	for _, run := range runs {
		rrm.runs[run.ID] = run
	}

	return runs, nil
}

func (rrm *ruleRepositoryMock) RetrieveRunsByIDs(_ context.Context, ids ...string) ([]rules.RuleRun, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	runs := make([]rules.RuleRun, 0, len(ids))
	for _, id := range ids {
		run, ok := rrm.runs[id]
		if !ok {
			return []rules.RuleRun{}, dbutil.ErrNotFound
		}
		runs = append(runs, run)
	}

	return runs, nil
}

func (rrm *ruleRepositoryMock) RetrieveRunsByThing(_ context.Context, thingID string, pm rules.PageMetadata) (rules.RuleRunsPage, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	var all, items []rules.RuleRun
	first := uint64(pm.Offset) + 1
	last := first + pm.Limit

	for _, run := range rrm.runs {
		if run.ThingID == thingID {
			all = append(all, run)
			id := uuid.ParseID(run.ID)
			if pm.Limit == 0 || (id >= first && id < last) {
				items = append(items, run)
			}
		}
	}

	return rules.RuleRunsPage{
		Total: uint64(len(all)),
		Runs:  items,
	}, nil
}

func (rrm *ruleRepositoryMock) RetrieveRunsByRule(_ context.Context, ruleID string, pm rules.PageMetadata) (rules.RuleRunsPage, error) {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	var all, items []rules.RuleRun
	first := uint64(pm.Offset) + 1
	last := first + pm.Limit

	for _, run := range rrm.runs {
		if run.RuleID == ruleID {
			all = append(all, run)
			id := uuid.ParseID(run.ID)
			if pm.Limit == 0 || (id >= first && id < last) {
				items = append(items, run)
			}
		}
	}

	return rules.RuleRunsPage{
		Total: uint64(len(all)),
		Runs:  items,
	}, nil
}

func (rrm *ruleRepositoryMock) RemoveRuns(_ context.Context, ids ...string) error {
	rrm.mu.Lock()
	defer rrm.mu.Unlock()

	for _, id := range ids {
		delete(rrm.runs, id)
	}

	return nil
}
