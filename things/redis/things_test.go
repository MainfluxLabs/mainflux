// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/things/redis"
	r "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var idProvider = uuid.New()

// TODO: update these tests to also test against external things keys

func TestThingSave(t *testing.T) {
	thingCache := redis.NewThingCache(redisClient)
	key, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	id := "123"
	id2 := "124"

	err = thingCache.Save(context.Background(), things.KeyTypeInline, key, id2)
	require.Nil(t, err, fmt.Sprintf("Save thing to cache: expected nil got %s", err))

	cases := []struct {
		desc string
		ID   string
		key  string
		err  error
	}{
		{
			desc: "Save thing to cache",
			ID:   id,
			key:  key,
			err:  nil,
		},
		{
			desc: "Save already cached thing to cache",
			ID:   id2,
			key:  key,
			err:  nil,
		},
	}

	for _, tc := range cases {
		err := thingCache.Save(context.Background(), things.KeyTypeInline, tc.key, tc.ID)
		assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))

	}
}

func TestThingID(t *testing.T) {
	thingCache := redis.NewThingCache(redisClient)

	key, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	id := "123"
	err = thingCache.Save(context.Background(), things.KeyTypeInline, key, id)
	require.Nil(t, err, fmt.Sprintf("Save thing to cache: expected nil got %s", err))

	cases := map[string]struct {
		ID  string
		key string
		err error
	}{
		"Get ID by existing thing-key": {
			ID:  id,
			key: key,
			err: nil,
		},
		"Get ID by non-existing thing-key": {
			ID:  "",
			key: wrongValue,
			err: r.Nil,
		},
	}

	for desc, tc := range cases {
		cacheID, err := thingCache.ID(context.Background(), things.KeyTypeInline, tc.key)
		assert.Equal(t, tc.ID, cacheID, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.ID, cacheID))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestThingRemove(t *testing.T) {
	thingCache := redis.NewThingCache(redisClient)

	key, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	id := "123"
	id2 := "321"
	thingCache.Save(context.Background(), things.KeyTypeInline, key, id)

	cases := []struct {
		desc string
		ID   string
		err  error
	}{
		{
			desc: "Remove existing thing from cache",
			ID:   id,
			err:  nil,
		},
		{
			desc: "Remove non-existing thing from cache",
			ID:   id2,
			err:  nil,
		},
	}

	for _, tc := range cases {
		err := thingCache.Remove(context.Background(), tc.ID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}

}
