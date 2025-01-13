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
	thkKey = "thing_key"
	thKey  = "thing"
	thsKey = "things"
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
	tkk := tkKey(thingKey)
	if err := tc.client.Set(ctx, tkk, thingID, 0).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	tik := thingIDKey(thingID)
	if err := tc.client.Set(ctx, tik, thingKey, 0).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}
	return nil
}

func (tc *thingCache) ID(ctx context.Context, thingKey string) (string, error) {
	tkk := tkKey(thingKey)
	thingID, err := tc.client.Get(ctx, tkk).Result()
	if err != nil {
		return "", errors.Wrap(errors.ErrNotFound, err)
	}

	return thingID, nil
}

func (tc *thingCache) Remove(ctx context.Context, thingID string) error {
	tik := thingIDKey(thingID)
	key, err := tc.client.Get(ctx, tik).Result()
	// Redis returns Nil Reply when key does not exist.
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	tkk := tkKey(key)
	if err := tc.client.Del(ctx, tkk, tik).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}
	return nil
}

func (tc *thingCache) SaveGroupID(ctx context.Context, thingID string, groupID string) error {
	tgk := thingGroupKey(thingID)
	if err := tc.client.Set(ctx, tgk, groupID, 0).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	gtk := groupThingsKey(groupID)
	if err := tc.client.SAdd(ctx, gtk, thingID).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (tc *thingCache) GroupID(ctx context.Context, thingID string) (string, error) {
	tgk := thingGroupKey(thingID)
	groupID, err := tc.client.Get(ctx, tgk).Result()
	if err != nil {
		return "", errors.Wrap(errors.ErrNotFound, err)
	}

	return groupID, nil
}

func (tc *thingCache) RemoveGroupID(ctx context.Context, thingID string) error {
	tgk := thingGroupKey(thingID)

	groupID, err := tc.client.Get(ctx, tgk).Result()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	if err := tc.client.Del(ctx, tgk).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	gtk := groupThingsKey(groupID)
	if err := tc.client.SRem(ctx, gtk, thingID).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	return nil
}

func tkKey(thingKey string) string {
	return fmt.Sprintf("%s:%s", thkKey, thingKey)
}

func thingIDKey(thingID string) string {
	return fmt.Sprintf("%s:%s", thKey, thingID)
}

func thingGroupKey(thingID string) string {
	return fmt.Sprintf("%s:%s:%s", thKey, thingID, grKey)
}

func groupThingsKey(groupID string) string {
	return fmt.Sprintf("%s:%s:%s", grKey, groupID, thsKey)
}
