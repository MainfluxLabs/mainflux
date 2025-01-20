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
	groupByProfilePrefix  = "gr_by_pr"
	profilesByGroupPrefix = "prs_by_gr"
)

var _ things.ProfileCache = (*profileCache)(nil)

type profileCache struct {
	client *redis.Client
}

// NewProfileCache returns redis profile cache implementation.
func NewProfileCache(client *redis.Client) things.ProfileCache {
	return &profileCache{client: client}
}
func (pc *profileCache) SaveGroup(ctx context.Context, profileID string, groupID string) error {
	gk := groupByProfileIDKey(profileID)
	if err := pc.client.Set(ctx, gk, groupID, 0).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	pk := profilesByGroupIDKey(groupID)
	if err := pc.client.SAdd(ctx, pk, profileID).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (pc *profileCache) RemoveGroup(ctx context.Context, profileID string) error {
	gk := groupByProfileIDKey(profileID)
	groupID, err := pc.client.Get(ctx, gk).Result()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	if err := pc.client.Del(ctx, gk).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	pk := profilesByGroupIDKey(groupID)
	if err := pc.client.SRem(ctx, pk, profileID).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	return nil
}

func (pc *profileCache) ViewGroup(ctx context.Context, profileID string) (string, error) {
	gk := groupByProfileIDKey(profileID)
	groupID, err := pc.client.Get(ctx, gk).Result()
	if err != nil {
		return "", errors.Wrap(errors.ErrNotFound, err)
	}

	return groupID, nil
}

func groupByProfileIDKey(profileID string) string {
	return fmt.Sprintf("%s:%s", groupByProfilePrefix, profileID)
}

func profilesByGroupIDKey(groupID string) string {
	return fmt.Sprintf("%s:%s", profilesByGroupPrefix, groupID)
}
