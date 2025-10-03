// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-redis/redis/v8"
)

const (
	keysByIDPrefix      = "keys_by_id"
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

func (tc *thingCache) Save(ctx context.Context, keyType, thingKey string, thingID string) error {
	// Associate the given thing key with the given thing ID
	idKey := idByThingKeyKey(keyType, thingKey)
	if err := tc.client.Set(ctx, idKey, thingID, 0).Err(); err != nil {
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	// Add the given thing key to the set containing thing keys associated with
	// this particular thing
	keysSetKey := keysByThingIDKey(thingID)
	thingKeyVal := fmt.Sprintf("%s:%s", keyType, thingKey)
	if err := tc.client.SAdd(ctx, keysSetKey, thingKeyVal).Err(); err != nil {
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return nil
}

func (tc *thingCache) ID(ctx context.Context, keyType, thingKey string) (string, error) {
	ik := idByThingKeyKey(keyType, thingKey)
	thingID, err := tc.client.Get(ctx, ik).Result()
	if err != nil {
		return "", errors.Wrap(dbutil.ErrNotFound, err)
	}

	return thingID, nil
}

func (tc *thingCache) RemoveThing(ctx context.Context, thingID string) error {
	// Retrieve all thing keys associated with the given thing ID
	keysSetKey := keysByThingIDKey(thingID)
	thingKeys, err := tc.client.SMembers(ctx, keysSetKey).Result()

	if err == redis.Nil {
		return nil
	}

	if err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	if len(thingKeys) == 0 {
		return nil
	}

	// Append prefix to each key
	for idx, keyVal := range thingKeys {
		thingKeys[idx] = fmt.Sprintf("%s:%s", idByKeyPrefix, keyVal)
	}

	if err := tc.client.Del(ctx, thingKeys...).Err(); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	if err := tc.client.Del(ctx, keysSetKey).Err(); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}
	return nil
}

func (tc *thingCache) RemoveKey(ctx context.Context, keyType, thingKey string) error {
	// Obtain id of thing represented by this particular key
	thingIdKey := idByThingKeyKey(keyType, thingKey)
	thingID, err := tc.client.Get(ctx, thingIdKey).Result()

	if err == redis.Nil {
		return nil
	}

	if err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	// Remove thing key key from cache
	if err := tc.client.Del(ctx, thingIdKey).Err(); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	// Remove thing key from set associating thing ID with all of its keys
	keysSetKey := keysByThingIDKey(thingID)
	thingKeyVal := fmt.Sprintf("%s:%s", keyType, thingKey)
	if err := tc.client.SRem(ctx, keysSetKey, thingKeyVal).Err(); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (tc *thingCache) SaveGroup(ctx context.Context, thingID string, groupID string) error {
	gk := groupByThingIDKey(thingID)
	if err := tc.client.Set(ctx, gk, groupID, 0).Err(); err != nil {
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	tk := thingsByGroupIDKey(groupID)
	if err := tc.client.SAdd(ctx, tk, thingID).Err(); err != nil {
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return nil
}

func (tc *thingCache) ViewGroup(ctx context.Context, thingID string) (string, error) {
	gk := groupByThingIDKey(thingID)
	groupID, err := tc.client.Get(ctx, gk).Result()
	if err != nil {
		return "", errors.Wrap(dbutil.ErrNotFound, err)
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
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	if err := tc.client.Del(ctx, gk).Err(); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	tk := thingsByGroupIDKey(groupID)
	if err := tc.client.SRem(ctx, tk, thingID).Err(); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func idByThingKeyKey(keyType, thingKey string) string {
	return fmt.Sprintf("%s:%s:%s", idByKeyPrefix, keyType, thingKey)
}

func keysByThingIDKey(thingID string) string {
	return fmt.Sprintf("%s:%s", keysByIDPrefix, thingID)
}

func groupByThingIDKey(thingID string) string {
	return fmt.Sprintf("%s:%s", groupByThingPrefix, thingID)
}

func thingsByGroupIDKey(groupID string) string {
	return fmt.Sprintf("%s:%s", thingsByGroupPrefix, groupID)
}
