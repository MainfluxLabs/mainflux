// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/shadows"
)

var _ shadows.ShadowRepository = (*shadowRepositoryMock)(nil)

type shadowRepositoryMock struct {
	mu      sync.Mutex
	shadows map[string]shadows.Shadow
}

// NewShadowRepository creates an in-memory shadow repository.
func NewShadowRepository() shadows.ShadowRepository {
	return &shadowRepositoryMock{
		shadows: make(map[string]shadows.Shadow),
	}
}

func (srm *shadowRepositoryMock) UpsertDesiredState(_ context.Context, thingID string, desired shadows.State, updatedAt int64) (shadows.Shadow, error) {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	sh := srm.shadows[thingID]
	sh.ThingID = thingID
	sh.Desired = desired
	sh.UpdatedAt = updatedAt
	srm.shadows[thingID] = sh
	return sh, nil
}

func (srm *shadowRepositoryMock) UpsertReportedState(_ context.Context, thingID string, reported shadows.State, reportedAt int64) error {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	sh := srm.shadows[thingID]
	sh.ThingID = thingID
	sh.Reported = reported
	sh.ReportedAt = reportedAt
	srm.shadows[thingID] = sh
	return nil
}

func (srm *shadowRepositoryMock) RetrieveByThing(_ context.Context, thingID string) (shadows.Shadow, error) {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	sh, ok := srm.shadows[thingID]
	if !ok {
		return shadows.Shadow{ThingID: thingID}, nil
	}

	return sh, nil
}

func (srm *shadowRepositoryMock) Remove(_ context.Context, thingID string) error {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	delete(srm.shadows, thingID)
	return nil
}
