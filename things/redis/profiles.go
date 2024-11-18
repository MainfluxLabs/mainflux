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

const prPrefix = "profile"

var _ things.ProfileCache = (*profileCache)(nil)

type profileCache struct {
	client *redis.Client
}

// NewProfileCache returns redis profile cache implementation.
func NewProfileCache(client *redis.Client) things.ProfileCache {
	return profileCache{client: client}
}

func (cc profileCache) Connect(ctx context.Context, profileID, thingID string) error {
	cid, tid := kv(profileID, thingID)
	if err := cc.client.SAdd(ctx, cid, tid).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}
	return nil
}

func (cc profileCache) HasThing(ctx context.Context, profileID, thingID string) bool {
	cid, tid := kv(profileID, thingID)
	return cc.client.SIsMember(ctx, cid, tid).Val()
}

func (cc profileCache) Disconnect(ctx context.Context, profileID, thingID string) error {
	cid, tid := kv(profileID, thingID)
	if err := cc.client.SRem(ctx, cid, tid).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}
	return nil
}

func (cc profileCache) Remove(ctx context.Context, profileID string) error {
	cid, _ := kv(profileID, "0")
	if err := cc.client.Del(ctx, cid).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}
	return nil
}

// Generates key-value pair
func kv(profileID, thingID string) (string, string) {
	cid := fmt.Sprintf("%s:%s", prPrefix, profileID)
	return cid, thingID
}
