// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"
	"fmt"

	"github.com/MainfluxLabs/mainflux/lora"
	"github.com/go-redis/redis/v8"
)

var _ lora.RouteMapRepository = (*routerMap)(nil)

type routerMap struct {
	client *redis.Client
	prefix string
}

// NewRouteMapRepository returns redis thing cache implementation.
func NewRouteMapRepository(client *redis.Client, prefix string) lora.RouteMapRepository {
	return &routerMap{
		client: client,
		prefix: prefix,
	}
}

func (rm *routerMap) Save(ctx context.Context, mfxID, loraID string) error {
	tkey := fmt.Sprintf("%s:%s", rm.prefix, mfxID)
	if err := rm.client.Set(ctx, tkey, loraID, 0).Err(); err != nil {
		return err
	}

	lkey := fmt.Sprintf("%s:%s", rm.prefix, loraID)
	if err := rm.client.Set(ctx, lkey, mfxID, 0).Err(); err != nil {
		return err
	}

	return nil
}

func (rm *routerMap) Get(ctx context.Context, id string) (string, error) {
	lKey := fmt.Sprintf("%s:%s", rm.prefix, id)
	mval, err := rm.client.Get(ctx, lKey).Result()
	if err != nil {
		return "", err
	}

	return mval, nil
}

func (rm *routerMap) Remove(ctx context.Context, mfxID string) error {
	mkey := fmt.Sprintf("%s:%s", rm.prefix, mfxID)
	lval, err := rm.client.Get(ctx, mkey).Result()
	if err != nil {
		return err
	}

	lkey := fmt.Sprintf("%s:%s", rm.prefix, lval)
	return rm.client.Del(ctx, mkey, lkey).Err()
}
