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
func (pc profileCache) Save(ctx context.Context, profileID string, groupID string) error {
	pk := pidKey(profileID)
	if err := pc.client.Set(ctx, pk, groupID, 0).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (pc profileCache) Remove(ctx context.Context, profileID string) error {
	pk := pidKey(profileID)
	if err := pc.client.Del(ctx, pk).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	return nil
}

func (pc profileCache) GroupID(ctx context.Context, profileID string) (string, error) {
	pk := pidKey(profileID)
	groupID, err := pc.client.Get(ctx, pk).Result()
	if err != nil {
		return "", errors.Wrap(errors.ErrNotFound, err)
	}

	return groupID, nil
}

func pidKey(profileID string) string {
	return fmt.Sprintf("%s:%s", prPrefix, profileID)
}
