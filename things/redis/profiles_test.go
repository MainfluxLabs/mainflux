// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/things/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemove(t *testing.T) {
	profileCache := redis.NewProfileCache(redisClient)

	pid := "123"
	pid2 := "124"
	tid := "321"

	cases := []struct {
		desc      string
		pid       string
		tid       string
		err       error
		hasAccess bool
	}{
		{
			desc:      "Remove profile group from cache",
			pid:       pid,
			tid:       tid,
			err:       nil,
			hasAccess: false,
		},
		{
			desc:      "Remove non-cached profile group from cache",
			pid:       pid2,
			tid:       tid,
			err:       nil,
			hasAccess: false,
		},
	}

	for _, tc := range cases {
		err := profileCache.RemoveGroup(context.Background(), tc.pid)
		assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewConfigByProfile(t *testing.T) {
	profileCache := redis.NewProfileCache(redisClient)

	profileID := "123"
	config := &things.Config{ContentType: messaging.SenMLContentType}

	err := profileCache.SaveConfig(context.Background(), profileID, config)
	require.Nil(t, err, fmt.Sprintf("save config by profile: expected nil got %s", err))

	cases := []struct {
		desc      string
		profileID string
		config    *things.Config
		err       error
	}{
		{
			desc:      "view cached config by profile",
			profileID: profileID,
			config:    config,
			err:       nil,
		},
		{
			desc:      "view config by non-cached profile",
			profileID: wrongValue,
			config:    nil,
			err:       dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		config, err := profileCache.ViewConfig(context.Background(), tc.profileID)
		assert.Equal(t, tc.config, config,
			fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.config, config))
		assert.True(t, errors.Contains(err, tc.err),
			fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveConfigByProfile(t *testing.T) {
	profileCache := redis.NewProfileCache(redisClient)

	profileID := "123"
	config := &things.Config{ContentType: messaging.SenMLContentType}

	err := profileCache.SaveConfig(context.Background(), profileID, config)
	require.Nil(t, err, fmt.Sprintf("save config by profile: expected nil got %s", err))

	cases := []struct {
		desc      string
		profileID string
		err       error
	}{
		{
			desc:      "remove cached config by profile",
			profileID: profileID,
			err:       nil,
		},
		{
			desc:      "remove config by non-cached profile",
			profileID: wrongValue,
			err:       nil,
		},
	}

	for _, tc := range cases {
		err := profileCache.RemoveConfig(context.Background(), tc.profileID)
		assert.True(t, errors.Contains(err, tc.err),
			fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		_, err = profileCache.ViewConfig(context.Background(), tc.profileID)
		assert.True(t, errors.Contains(err, dbutil.ErrNotFound),
			fmt.Sprintf("%s: expected %s got %s\n", tc.desc, dbutil.ErrNotFound, err))
	}
}
