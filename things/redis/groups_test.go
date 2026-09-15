// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveOrgByGroup(t *testing.T) {
	groupCache := redis.NewGroupCache(redisClient)

	groupID, orgID := "123", "321"

	err := groupCache.SaveOrg(context.Background(), groupID, orgID)
	require.Nil(t, err, fmt.Sprintf("save org by group: expected nil got %s", err))

	err = groupCache.SaveOrg(context.Background(), groupID, orgID)
	assert.Nil(t, err, fmt.Sprintf("re-save org by group: expected nil got %s", err))
}

func TestViewOrgByGroup(t *testing.T) {
	groupCache := redis.NewGroupCache(redisClient)

	groupID, orgID := "123", "321"

	err := groupCache.SaveOrg(context.Background(), groupID, orgID)
	require.Nil(t, err, fmt.Sprintf("save org by group: expected nil got %s", err))

	cases := []struct {
		desc    string
		groupID string
		orgID   string
		err     error
	}{
		{
			desc:    "view cached org by group",
			groupID: groupID,
			orgID:   orgID,
			err:     nil,
		},
		{
			desc:    "view org by non-cached group",
			groupID: "345",
			orgID:   "",
			err:     dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		orgID, err := groupCache.ViewOrg(context.Background(), tc.groupID)
		assert.Equal(t, tc.orgID, orgID, fmt.Sprintf("%s: expected org id %s got %s", tc.desc, tc.orgID, orgID))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestRemoveGroupEntitiesRemovesOrg(t *testing.T) {
	groupCache := redis.NewGroupCache(redisClient)

	groupID, orgID := "123", "321"

	err := groupCache.SaveOrg(context.Background(), groupID, orgID)
	require.Nil(t, err, fmt.Sprintf("save org by group: expected nil got %s", err))

	err = groupCache.RemoveGroupEntities(context.Background(), groupID)
	require.Nil(t, err, fmt.Sprintf("remove group entities: expected nil got %s", err))

	_, err = groupCache.ViewOrg(context.Background(), groupID)
	assert.True(t, errors.Contains(err, dbutil.ErrNotFound), fmt.Sprintf("view org after removing group entities: expected %s got %s", dbutil.ErrNotFound, err))
}
