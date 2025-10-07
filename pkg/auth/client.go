// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-redis/redis/v8"
)

// Client represents Auth cache.
type Client interface {
	Identify(ctx context.Context, key things.ThingKey) (string, error)
	GetPubConfByKey(ctx context.Context, key things.ThingKey) (protomfx.PubConfByKeyRes, error)
}

const (
	profilePrefix = "profile"
	keyPrefix     = "thing_key"
)

type client struct {
	redisClient *redis.Client
	things      protomfx.ThingsServiceClient
}

// New returns redis profile cache implementation.
func New(redisClient *redis.Client, things protomfx.ThingsServiceClient) Client {
	return client{
		redisClient: redisClient,
		things:      things,
	}
}

func (c client) Identify(ctx context.Context, key things.ThingKey) (string, error) {
	tkey := keyPrefix + ":" + key.Value
	thingID, err := c.redisClient.Get(ctx, tkey).Result()
	if err != nil {
		t := &protomfx.ThingKey{
			Value: string(key.Value),
			Type:  key.Type,
		}

		thid, err := c.things.Identify(context.TODO(), t)
		if err != nil {
			return "", err
		}
		return thid.GetValue(), nil
	}
	return thingID, nil
}

func (c client) GetPubConfByKey(ctx context.Context, key things.ThingKey) (protomfx.PubConfByKeyRes, error) {
	req := &protomfx.ThingKey{
		Value: key.Value,
		Type:  key.Type,
	}

	pc, err := c.things.GetPubConfByKey(ctx, req)
	if err != nil {
		return protomfx.PubConfByKeyRes{}, err
	}

	if pc != nil {
		return *pc, nil
	}

	return protomfx.PubConfByKeyRes{}, nil
}
