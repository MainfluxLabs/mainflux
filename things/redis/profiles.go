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

func (cc profileCache) Remove(ctx context.Context, profileID string) error {
	pid := key(profileID)
	if err := cc.client.Del(ctx, pid).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}
	return nil
}

func key(profileID string) string {
	pid := fmt.Sprintf("%s:%s", prPrefix, profileID)
	return pid
}
