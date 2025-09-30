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

func TestThingSave(t *testing.T) {
	thingCache := redis.NewThingCache(redisClient)
	existingKey, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	externalKey := "external_key"

	id := "123"
	id2 := "124"

	err = thingCache.Save(context.Background(), things.KeyTypeInline, existingKey, id2)
	require.Nil(t, err, fmt.Sprintf("save thing to cache: expected nil got %s", err))

	cases := []struct {
		desc    string
		ID      string
		key     string
		keyType string
	}{
		{
			desc:    "save inline key to thing cache",
			ID:      id,
			key:     existingKey,
			keyType: things.KeyTypeInline,
		},
		{
			desc:    "save already cached inline key thing to cache",
			ID:      id2,
			key:     existingKey,
			keyType: things.KeyTypeInline,
		},
		{
			desc:    "save external key to thing cache",
			ID:      id,
			key:     externalKey,
			keyType: things.KeyTypeExternal,
		},
		{
			desc:    "save already cached external key thing to cache",
			ID:      id,
			key:     externalKey,
			keyType: things.KeyTypeExternal,
		},
	}

	for _, tc := range cases {
		err := thingCache.Save(context.Background(), tc.keyType, tc.key, tc.ID)
		assert.Nil(t, err, fmt.Sprintf("%s: expected nil got %s", tc.desc, err))
	}
}

func TestThingID(t *testing.T) {
	thingCache := redis.NewThingCache(redisClient)

	key, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	id := "123"
	err = thingCache.Save(context.Background(), things.KeyTypeInline, key, id)
	require.Nil(t, err, fmt.Sprintf("save thing to cache: expected nil got %s", err))

	externalKey := "external_key"
	err = thingCache.Save(context.Background(), things.KeyTypeExternal, externalKey, id)
	require.Nil(t, err, fmt.Sprintf("save thing to cache: expected nil got %s", err))

	cases := map[string]struct {
		ID      string
		key     string
		keyType string
		err     error
	}{
		"get ID by existing inline thing-key": {
			ID:      id,
			key:     key,
			keyType: things.KeyTypeInline,
			err:     nil,
		},
		"get ID by non-existing inline thing-key": {
			ID:      "",
			key:     wrongValue,
			keyType: things.KeyTypeInline,
			err:     r.Nil,
		},
		"get ID by existing external thing-key": {
			ID:      id,
			key:     externalKey,
			keyType: things.KeyTypeExternal,
			err:     nil,
		},
		"get ID by non-existing external thing-key": {
			ID:      "",
			key:     wrongValue,
			keyType: things.KeyTypeExternal,
			err:     r.Nil,
		},
	}

	for desc, tc := range cases {
		cacheID, err := thingCache.ID(context.Background(), tc.keyType, tc.key)
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
	err = thingCache.Save(context.Background(), things.KeyTypeInline, key, id)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	externalKey := "external_key"
	err = thingCache.Save(context.Background(), things.KeyTypeExternal, externalKey, id)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc string
		ID   string
		err  error
	}{
		{
			desc: "remove existing thing from cache",
			ID:   id,
			err:  nil,
		},
		{
			desc: "remove non-existing thing from cache",
			ID:   id2,
			err:  nil,
		},
	}

	for _, tc := range cases {
		err := thingCache.RemoveThing(context.Background(), tc.ID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}

}
