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
	tkey := tkKey(thingKey)
	if err := tc.client.Set(ctx, tkey, thingID, 0).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	tid := tidKey(thingID)
	if err := tc.client.Set(ctx, tid, thingKey, 0).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}
	return nil
}

func (tc *thingCache) ID(ctx context.Context, thingKey string) (string, error) {
	tkey := tkKey(thingKey)
	thingID, err := tc.client.Get(ctx, tkey).Result()
	if err != nil {
		return "", errors.Wrap(errors.ErrNotFound, err)
	}

	return thingID, nil
}

func (tc *thingCache) Remove(ctx context.Context, thingID string) error {
	tid := tidKey(thingID)
	key, err := tc.client.Get(ctx, tid).Result()
	// Redis returns Nil Reply when key does not exist.
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	tkey := tkKey(key)
	if err := tc.client.Del(ctx, tkey, tid).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}
	return nil
}

func (tc *thingCache) SaveGroupID(ctx context.Context, thingID string, groupID string) error {
	gk := tgKey(thingID)
	if err := tc.client.Set(ctx, gk, groupID, 0).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (tc *thingCache) GroupID(ctx context.Context, thingID string) (string, error) {
	gk := tgKey(thingID)
	groupID, err := tc.client.Get(ctx, gk).Result()
	if err != nil {
		return "", errors.Wrap(errors.ErrNotFound, err)
	}

	return groupID, nil
}

func (tc *thingCache) RemoveGroupID(ctx context.Context, thingID string) error {
	gk := tgKey(thingID)

	if err := tc.client.Del(ctx, gk).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	return nil
}

func tkKey(thingKey string) string {
	return fmt.Sprintf("%s:%s", keyPrefix, thingKey)
}

func tidKey(thingID string) string {
	return fmt.Sprintf("%s:%s", thPrefix, thingID)
}

func tgKey(thingID string) string {
	return fmt.Sprintf("%s:%s:%s", thPrefix, thingID, grPrefix)
}
