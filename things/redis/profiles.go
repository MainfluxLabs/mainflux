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
	prKey  = "profile"
	prsKey = "profiles"
)

var _ things.ProfileCache = (*profileCache)(nil)

type profileCache struct {
	client *redis.Client
}

// NewProfileCache returns redis profile cache implementation.
func NewProfileCache(client *redis.Client) things.ProfileCache {
	return profileCache{client: client}
}
func (pc profileCache) SaveGroupID(ctx context.Context, profileID string, groupID string) error {
	pgk := profileGroupKey(profileID)
	if err := pc.client.Set(ctx, pgk, groupID, 0).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	gpk := groupProfilesKey(groupID)
	if err := pc.client.SAdd(ctx, gpk, profileID).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (pc profileCache) RemoveGroupID(ctx context.Context, profileID string) error {
	pgk := profileGroupKey(profileID)
	groupID, err := pc.client.Get(ctx, pgk).Result()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	if err := pc.client.Del(ctx, pgk).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	gpk := groupProfilesKey(groupID)
	if err := pc.client.SRem(ctx, gpk, profileID).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	return nil
}

func (pc profileCache) GroupID(ctx context.Context, profileID string) (string, error) {
	pgk := profileGroupKey(profileID)
	groupID, err := pc.client.Get(ctx, pgk).Result()
	if err != nil {
		return "", errors.Wrap(errors.ErrNotFound, err)
	}

	return groupID, nil
}

func profileGroupKey(profileID string) string {
	return fmt.Sprintf("%s:%s:%s", prKey, profileID, grKey)
}

func groupProfilesKey(groupID string) string {
	return fmt.Sprintf("%s:%s:%s", grKey, groupID, prsKey)
}
