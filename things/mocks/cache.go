// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
)

type thingCacheMock struct {
	mu     sync.Mutex
	things map[string]string
	groups map[string]string
}

// NewThingCache returns mock cache instance.
func NewThingCache() things.ThingCache {
	return &thingCacheMock{
		things: make(map[string]string),
		groups: make(map[string]string),
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

func (tcm *thingCacheMock) SaveGroup(_ context.Context, thingID string, groupID string) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	tcm.groups[thingID] = groupID
	return nil
}

func (tcm *thingCacheMock) ViewGroup(_ context.Context, thingID string) (string, error) {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	groupID, ok := tcm.groups[thingID]
	if !ok {
		return "", errors.ErrNotFound
	}

	return groupID, nil
}

func (tcm *thingCacheMock) RemoveGroup(_ context.Context, thingID string) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	delete(tcm.groups, thingID)

	return nil
}

type profileCacheMock struct {
	mu     sync.Mutex
	groups map[string]string
}

// NewProfileCache returns mock cache instance.
func NewProfileCache() things.ProfileCache {
	return &profileCacheMock{
		groups: make(map[string]string),
	}
}

func (ccm *profileCacheMock) SaveGroup(_ context.Context, profileID string, groupID string) error {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	ccm.groups[profileID] = groupID
	return nil
}

func (ccm *profileCacheMock) ViewGroup(_ context.Context, profileID string) (string, error) {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	groupID, ok := ccm.groups[profileID]
	if !ok {
		return "", errors.ErrNotFound
	}

	return groupID, nil
}

func (ccm *profileCacheMock) RemoveGroup(_ context.Context, profileID string) error {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	delete(ccm.groups, profileID)
	return nil
}

type groupCacheMock struct {
	mu      sync.Mutex
	members map[string]string
}

// NewGroupCache returns mock cache instance.
func NewGroupCache() things.GroupCache {
	return &groupCacheMock{
		members: make(map[string]string),
	}
}

func (gcm *groupCacheMock) RemoveGroupEntities(_ context.Context, groupID string) error {
	return nil
}

func (gcm *groupCacheMock) SaveGroupMember(_ context.Context, groupID, memberID, role string) error {
	gcm.mu.Lock()
	defer gcm.mu.Unlock()

	key := mKey(groupID, memberID)
	gcm.members[key] = role
	return nil
}

func (gcm *groupCacheMock) ViewRole(_ context.Context, groupID, memberID string) (string, error) {
	gcm.mu.Lock()
	defer gcm.mu.Unlock()

	key := mKey(groupID, memberID)
	role, ok := gcm.members[key]
	if !ok {
		return "", errors.ErrNotFound
	}

	return role, nil
}

func (gcm *groupCacheMock) RemoveGroupMember(_ context.Context, groupID, memberID string) error {
	gcm.mu.Lock()
	defer gcm.mu.Unlock()

	key := mKey(groupID, memberID)
	delete(gcm.members, key)

	return nil
}

func (gcm *groupCacheMock) GroupMemberships(_ context.Context, memberID string) ([]string, error) {
	gcm.mu.Lock()
	defer gcm.mu.Unlock()

	groups := []string{}
	for k := range gcm.members {
		parts := strings.Split(k, ":")
		if parts[1] == memberID {
			groups = append(groups, parts[0])
		}
	}

	return groups, nil
}

func mKey(groupID, memberID string) string {
	return fmt.Sprintf("%s:%s", groupID, memberID)
}
