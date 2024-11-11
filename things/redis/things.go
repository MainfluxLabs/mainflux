// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-redis/redis/v8"
)

const (
	keyPrefix = "thing_key"
	thPrefix  = "thing"
	grPrefix  = "group"
)

var _ things.ThingCache = (*thingCache)(nil)

type thingCache struct {
	client *redis.Client
}

// NewThingCache returns redis thing cache implementation.
func NewThingCache(client *redis.Client) things.ThingCache {
	return &thingCache{
		client: client,
	}
}

func (tc *thingCache) Save(ctx context.Context, thingKey string, thingID string) error {
	tkey := fmt.Sprintf("%s:%s", keyPrefix, thingKey)
	if err := tc.client.Set(ctx, tkey, thingID, 0).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	tid := fmt.Sprintf("%s:%s", thPrefix, thingID)
	if err := tc.client.Set(ctx, tid, thingKey, 0).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}
	return nil
}

func (tc *thingCache) ID(ctx context.Context, thingKey string) (string, error) {
	tkey := fmt.Sprintf("%s:%s", keyPrefix, thingKey)
	thingID, err := tc.client.Get(ctx, tkey).Result()
	if err != nil {
		return "", errors.Wrap(errors.ErrNotFound, err)
	}

	return thingID, nil
}

func (tc *thingCache) Remove(ctx context.Context, thingID string) error {
	tid := fmt.Sprintf("%s:%s", thPrefix, thingID)
	key, err := tc.client.Get(ctx, tid).Result()
	// Redis returns Nil Reply when key does not exist.
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	tkey := fmt.Sprintf("%s:%s", keyPrefix, key)
	if err := tc.client.Del(ctx, tkey, tid).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}
	return nil
}

func (tc *thingCache) SaveRole(ctx context.Context, groupID, memberID, role string) (error) {
	rkey := fmt.Sprintf("%s:%s:%s", grPrefix, groupID, memberID)
	if err := tc.client.Set(ctx, rkey, role, 0).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (tc *thingCache) Role(ctx context.Context, groupID, memberID string) (string, error) {
	rkey := fmt.Sprintf("%s:%s:%s", grPrefix, groupID, memberID)
	role, err := tc.client.Get(ctx, rkey).Result()
	if err != nil {
		return "", errors.Wrap(errors.ErrNotFound, err)
	}

	return role, nil
}

func (tc *thingCache) RemoveRole(ctx context.Context, groupID, memberID string) error {
	// Redis returns Nil Reply when key does not exist.
	rKey := fmt.Sprintf("%s:%s:%s", grPrefix, groupID, memberID)
	if err := tc.client.Del(ctx, rKey).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}
	return nil
}
