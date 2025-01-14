// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/things/redis"
	"github.com/stretchr/testify/assert"
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
