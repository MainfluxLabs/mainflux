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
	mu       sync.Mutex
	things   map[string]string
	thGroups map[string]string
}

// NewThingCache returns mock cache instance.
func NewThingCache() things.ThingCache {
	return &thingCacheMock{
		things:   make(map[string]string),
		thGroups: make(map[string]string),
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

func (tcm *thingCacheMock) SaveGroupID(_ context.Context, thingID string, groupID string) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	tcm.thGroups[thingID] = groupID
	return nil
}

func (tcm *thingCacheMock) GroupID(_ context.Context, thingID string) (string, error) {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	groupID, ok := tcm.thGroups[thingID]
	if !ok {
		return "", errors.ErrNotFound
	}

	return groupID, nil
}

func (tcm *thingCacheMock) RemoveGroupID(_ context.Context, thingID string) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	delete(tcm.thGroups, thingID)

	return nil
}

type profileCacheMock struct {
	mu       sync.Mutex
	prGroups map[string]string
}

// NewProfileCache returns mock cache instance.
func NewProfileCache() things.ProfileCache {
	return &profileCacheMock{
		prGroups: make(map[string]string),
	}
}

func (ccm *profileCacheMock) SaveGroupID(_ context.Context, profileID string, groupID string) error {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	ccm.prGroups[profileID] = groupID
	return nil
}

func (ccm *profileCacheMock) GroupID(_ context.Context, profileID string) (string, error) {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	groupID, ok := ccm.prGroups[profileID]
	if !ok {
		return "", errors.ErrNotFound
	}

	return groupID, nil
}

func (ccm *profileCacheMock) RemoveGroupID(_ context.Context, profileID string) error {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	delete(ccm.prGroups, profileID)
	return nil
}

type groupCacheMock struct {
	mu     sync.Mutex
	grOrgs map[string]string
	roles  map[string]string
}

// NewGroupCache returns mock cache instance.
func NewGroupCache() things.GroupCache {
	return &groupCacheMock{
		grOrgs: make(map[string]string),
		roles:  make(map[string]string),
	}
}

func (gcm *groupCacheMock) SaveOrgID(_ context.Context, groupID, orgID string) error {
	gcm.mu.Lock()
	defer gcm.mu.Unlock()

	gcm.grOrgs[groupID] = orgID
	return nil
}

func (gcm *groupCacheMock) OrgID(_ context.Context, groupID string) (string, error) {
	gcm.mu.Lock()
	defer gcm.mu.Unlock()

	orgID, ok := gcm.grOrgs[groupID]
	if !ok {
		return "", errors.ErrNotFound
	}

	return orgID, nil
}

func (gcm *groupCacheMock) Remove(_ context.Context, groupID string) error {
	gcm.mu.Lock()
	defer gcm.mu.Unlock()

	delete(gcm.grOrgs, groupID)
	return nil
}

func (gcm *groupCacheMock) SaveRole(_ context.Context, groupID, memberID, role string) error {
	gcm.mu.Lock()
	defer gcm.mu.Unlock()

	key := rKey(groupID, memberID)
	gcm.roles[key] = role
	return nil
}

func (gcm *groupCacheMock) Role(_ context.Context, groupID, memberID string) (string, error) {
	gcm.mu.Lock()
	defer gcm.mu.Unlock()

	key := rKey(groupID, memberID)
	role, ok := gcm.roles[key]
	if !ok {
		return "", errors.ErrNotFound
	}

	return role, nil
}

func (gcm *groupCacheMock) RemoveRole(_ context.Context, groupID, memberID string) error {
	gcm.mu.Lock()
	defer gcm.mu.Unlock()

	key := rKey(groupID, memberID)
	delete(gcm.roles, key)

	return nil
}

func (gcm *groupCacheMock) GroupIDsByMember(_ context.Context, memberID string) ([]string, error) {
	gcm.mu.Lock()
	defer gcm.mu.Unlock()

	groups := []string{}
	for k := range gcm.roles {
		parts := strings.Split(k, ":")
		if parts[1] == memberID {
			groups = append(groups, parts[0])
		}
	}

	return groups, nil
}

func rKey(groupID, memberID string) string {
	return fmt.Sprintf("%s:%s", groupID, memberID)
}
