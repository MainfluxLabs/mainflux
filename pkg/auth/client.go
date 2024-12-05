// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/go-redis/redis/v8"
)

// Client represents Auth cache.
type Client interface {
	Identify(ctx context.Context, thingKey string) (string, error)
	GetPubConfByKey(ctx context.Context, thingKey string) (protomfx.PubConfByKeyRes, error)
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

func (c client) Identify(ctx context.Context, thingKey string) (string, error) {
	tkey := keyPrefix + ":" + thingKey
	thingID, err := c.redisClient.Get(ctx, tkey).Result()
	if err != nil {
		t := &protomfx.Token{
			Value: string(thingKey),
		}

		thid, err := c.things.Identify(context.TODO(), t)
		if err != nil {
			return "", err
		}
		return thid.GetValue(), nil
	}
	return thingID, nil
}

func (c client) GetPubConfByKey(ctx context.Context, thingKey string) (protomfx.PubConfByKeyRes, error) {
	req := &protomfx.PubConfByKeyReq{
		Key: thingKey,
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
