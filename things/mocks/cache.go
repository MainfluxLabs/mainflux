// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"fmt"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
)

type thingCacheMock struct {
	mu        sync.Mutex
	things    map[string]string
	thsGroups map[string]string
	roles     map[string]string
}

// NewThingCache returns mock cache instance.
func NewThingCache() things.ThingCache {
	return &thingCacheMock{
		things:    make(map[string]string),
		thsGroups: make(map[string]string),
		roles:     make(map[string]string),
	}
}

func (tcm *thingCacheMock) Save(_ context.Context, key, id string) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	tcm.things[key] = id
	return nil
}

func (tcm *thingCacheMock) ID(_ context.Context, key string) (string, error) {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	id, ok := tcm.things[key]
	if !ok {
		return "", errors.ErrNotFound
	}

	return id, nil
}

func (tcm *thingCacheMock) Remove(_ context.Context, id string) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	for key, val := range tcm.things {
		if val == id {
			delete(tcm.things, key)
			return nil
		}
	}

	return nil
}

func (tcm *thingCacheMock) SaveGroupID(ctx context.Context, thingID string, groupID string) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	tcm.thsGroups[thingID] = groupID
	return nil
}

func (tcm *thingCacheMock) GroupID(ctx context.Context, thingID string) (string, error) {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	groupID, ok := tcm.thsGroups[thingID]
	if !ok {
		return "", errors.ErrNotFound
	}

	return groupID, nil
}

func (tcm *thingCacheMock) RemoveGroupID(ctx context.Context, thingID string) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	delete(tcm.thsGroups, thingID)

	return nil
}

func (tcm *thingCacheMock) SaveRole(_ context.Context, groupID, memberID, role string) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	key := fmt.Sprintf("%s:%s", groupID, memberID)
	tcm.roles[key] = role
	return nil
}

func (tcm *thingCacheMock) Role(_ context.Context, groupID, memberID string) (string, error) {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	key := fmt.Sprintf("%s:%s", groupID, memberID)
	role, ok := tcm.roles[key]
	if !ok {
		return "", errors.ErrNotFound
	}

	return role, nil
}

func (tcm *thingCacheMock) RemoveRole(_ context.Context, groupID, memberID string) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	key := fmt.Sprintf("%s:%s", groupID, memberID)
	delete(tcm.roles, key)

	return nil
}

type profileCacheMock struct {
	mu       sync.Mutex
	profiles map[string]string
}

// NewProfileCache returns mock cache instance.
func NewProfileCache() things.ProfileCache {
	return &profileCacheMock{
		profiles: make(map[string]string),
	}
}

func (ccm *profileCacheMock) Save(ctx context.Context, profileID string, groupID string) error {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	ccm.profiles[profileID] = groupID
	return nil
}

func (ccm *profileCacheMock) GroupID(ctx context.Context, profileID string) (string, error) {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	groupID, ok := ccm.profiles[profileID]
	if !ok {
		return "", errors.ErrNotFound
	}

	return groupID, nil
}

func (ccm *profileCacheMock) Remove(_ context.Context, profileID string) error {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	delete(ccm.profiles, profileID)
	return nil
}
