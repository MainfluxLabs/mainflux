// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/things"
)

type thingCacheMock struct {
	mu                  sync.Mutex
	thingsByKey         map[string]string
	thingsByKeyExternal map[string]string
	groups              map[string]string
}

// NewThingCache returns mock cache instance.
func NewThingCache() things.ThingCache {
	return &thingCacheMock{
		thingsByKey:         make(map[string]string),
		thingsByKeyExternal: make(map[string]string),
		groups:              make(map[string]string),
	}
}

func (tcm *thingCacheMock) Save(_ context.Context, key apiutil.ThingKey, id string) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	switch key.Type {
	case things.KeyTypeInternal:
		tcm.thingsByKey[key.Key] = id
	case things.KeyTypeExternal:
		tcm.thingsByKeyExternal[key.Key] = id
	}

	return nil
}

func (tcm *thingCacheMock) ID(_ context.Context, key apiutil.ThingKey) (string, error) {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	switch key.Type {
	case things.KeyTypeInternal:
		if id, ok := tcm.thingsByKey[key.Key]; ok {
			return id, nil
		}
	case things.KeyTypeExternal:
		if id, ok := tcm.thingsByKeyExternal[key.Key]; ok {
			return id, nil
		}
	default:
		return "", apiutil.ErrInvalidThingKeyType
	}

	return "", dbutil.ErrNotFound
}

func (tcm *thingCacheMock) RemoveThing(_ context.Context, id string) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	for key, val := range tcm.thingsByKey {
		if val == id {
			delete(tcm.thingsByKey, key)
		}
	}

	for key, val := range tcm.thingsByKeyExternal {
		if val == id {
			delete(tcm.thingsByKey, key)
		}
	}

	return nil
}

func (tcm *thingCacheMock) RemoveKey(_ context.Context, key apiutil.ThingKey) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	switch key.Type {
	case things.KeyTypeInternal:
		delete(tcm.thingsByKey, key.Key)
	case things.KeyTypeExternal:
		delete(tcm.thingsByKeyExternal, key.Key)
	default:
		return apiutil.ErrInvalidThingKeyType
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
		return "", dbutil.ErrNotFound
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

func (pcm *profileCacheMock) SaveGroup(_ context.Context, profileID string, groupID string) error {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()

	pcm.groups[profileID] = groupID
	return nil
}

func (pcm *profileCacheMock) ViewGroup(_ context.Context, profileID string) (string, error) {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()

	groupID, ok := pcm.groups[profileID]
	if !ok {
		return "", dbutil.ErrNotFound
	}

	return groupID, nil
}

func (pcm *profileCacheMock) RemoveGroup(_ context.Context, profileID string) error {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()

	delete(pcm.groups, profileID)
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

func (gcm *groupCacheMock) SaveGroupMembership(_ context.Context, groupID, memberID, role string) error {
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
		return "", dbutil.ErrNotFound
	}

	return role, nil
}

func (gcm *groupCacheMock) RemoveGroupMembership(_ context.Context, groupID, memberID string) error {
	gcm.mu.Lock()
	defer gcm.mu.Unlock()

	key := mKey(groupID, memberID)
	delete(gcm.members, key)

	return nil
}

func (gcm *groupCacheMock) RetrieveGroupIDsByMember(_ context.Context, memberID string) ([]string, error) {
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
