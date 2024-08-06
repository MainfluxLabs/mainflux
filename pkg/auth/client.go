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
	GetConnByKey(ctx context.Context, thingKey string) (protomfx.ConnByKeyRes, error)
}

const (
	chanPrefix = "channel"
	keyPrefix  = "thing_key"
)

type client struct {
	redisClient *redis.Client
	things      protomfx.ThingsServiceClient
}

// New returns redis channel cache implementation.
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

func (c client) GetConnByKey(ctx context.Context, thingKey string) (protomfx.ConnByKeyRes, error) {
	req := &protomfx.ConnByKeyReq{
		Key: thingKey,
	}

	conn, err := c.things.GetConnByKey(ctx, req)
	if err != nil {
		return protomfx.ConnByKeyRes{}, err
	}

	if conn != nil {
		return *conn, nil
	}

	return protomfx.ConnByKeyRes{}, nil
}
