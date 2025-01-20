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
	keyByIDPrefix       = "key_by_id"
	idByKeyPrefix       = "id_by_key"
	groupByThingPrefix  = "gr_by_th"
	thingsByGroupPrefix = "ths_by_gr"
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
	ik := idByThingKeyKey(thingKey)
	if err := tc.client.Set(ctx, ik, thingID, 0).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	kk := keyByThingIDKey(thingID)
	if err := tc.client.Set(ctx, kk, thingKey, 0).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}
	return nil
}

func (tc *thingCache) ID(ctx context.Context, thingKey string) (string, error) {
	ik := idByThingKeyKey(thingKey)
	thingID, err := tc.client.Get(ctx, ik).Result()
	if err != nil {
		return "", errors.Wrap(errors.ErrNotFound, err)
	}

	return thingID, nil
}

func (tc *thingCache) Remove(ctx context.Context, thingID string) error {
	kk := keyByThingIDKey(thingID)
	thingKey, err := tc.client.Get(ctx, kk).Result()
	// Redis returns Nil Reply when key does not exist.
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	ik := idByThingKeyKey(thingKey)
	if err := tc.client.Del(ctx, ik, kk).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}
	return nil
}

func (tc *thingCache) SaveGroup(ctx context.Context, thingID string, groupID string) error {
	gk := groupByThingIDKey(thingID)
	if err := tc.client.Set(ctx, gk, groupID, 0).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	tk := thingsByGroupIDKey(groupID)
	if err := tc.client.SAdd(ctx, tk, thingID).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (tc *thingCache) ViewGroup(ctx context.Context, thingID string) (string, error) {
	gk := groupByThingIDKey(thingID)
	groupID, err := tc.client.Get(ctx, gk).Result()
	if err != nil {
		return "", errors.Wrap(errors.ErrNotFound, err)
	}

	return groupID, nil
}

func (tc *thingCache) RemoveGroup(ctx context.Context, thingID string) error {
	gk := groupByThingIDKey(thingID)

	groupID, err := tc.client.Get(ctx, gk).Result()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	if err := tc.client.Del(ctx, gk).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	tk := thingsByGroupIDKey(groupID)
	if err := tc.client.SRem(ctx, tk, thingID).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	return nil
}

func idByThingKeyKey(thingKey string) string {
	return fmt.Sprintf("%s:%s", idByKeyPrefix, thingKey)
}

func keyByThingIDKey(thingID string) string {
	return fmt.Sprintf("%s:%s", keyByIDPrefix, thingID)
}

func groupByThingIDKey(thingID string) string {
	return fmt.Sprintf("%s:%s", groupByThingPrefix, thingID)
}

func thingsByGroupIDKey(groupID string) string {
	return fmt.Sprintf("%s:%s", thingsByGroupPrefix, groupID)
}
