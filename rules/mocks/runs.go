// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

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
