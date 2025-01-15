package redis

import (
	"context"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-redis/redis/v8"
)

const (
	groupOrgPrefix     = "gr_org"
	groupMembersPrefix = "gr_mbs"
	memberGroupsPrefix = "mb_grs"
)

var _ things.GroupCache = (*groupCache)(nil)

type groupCache struct {
	client *redis.Client
}

// NewGroupCache returns redis group cache implementation.
func NewGroupCache(client *redis.Client) things.GroupCache {
	return &groupCache{
		client: client,
	}
}

func (gc *groupCache) SaveOrg(ctx context.Context, groupID, orgID string) error {
	gok := groupOrgKey(groupID)
	if err := gc.client.Set(ctx, gok, orgID, 0).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (gc *groupCache) ViewOrg(ctx context.Context, groupID string) (string, error) {
	gok := groupOrgKey(groupID)
	orgID, err := gc.client.Get(ctx, gok).Result()
	if err != nil {
		return "", errors.Wrap(errors.ErrNotFound, err)
	}

	return orgID, nil
}

func (gc *groupCache) RemoveOrg(ctx context.Context, groupID string) error {
	if err := gc.removeGroupRelations(ctx, groupID); err != nil {
		return err
	}

	return nil
}

func (gc *groupCache) SaveRole(ctx context.Context, groupID, memberID, role string) error {
	mgk := memberGroupsKey(memberID)
	if err := gc.client.HSet(ctx, mgk, groupID, role).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	gmk := groupMembersKey(groupID)
	if err := gc.client.SAdd(ctx, gmk, memberID).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (gc *groupCache) ViewRole(ctx context.Context, groupID, memberID string) (string, error) {
	mgk := memberGroupsKey(memberID)
	role, err := gc.client.HGet(ctx, mgk, groupID).Result()
	if err != nil {
		return "", errors.Wrap(errors.ErrNotFound, err)
	}

	return role, nil
}

func (gc *groupCache) RemoveRole(ctx context.Context, groupID, memberID string) error {
	mgk := memberGroupsKey(memberID)
	if _, err := gc.client.HDel(ctx, mgk, groupID).Result(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	gmk := groupMembersKey(groupID)
	if _, err := gc.client.SRem(ctx, gmk, memberID).Result(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	return nil
}

func (gc *groupCache) GroupMemberships(ctx context.Context, memberID string) ([]string, error) {
	mgk := memberGroupsKey(memberID)
	groups, err := gc.client.HKeys(ctx, mgk).Result()
	if err != nil {
		return nil, errors.Wrap(errors.ErrNotFound, err)
	}

	return groups, nil
}

func groupOrgKey(groupID string) string {
	return fmt.Sprintf("%s:%s", groupOrgPrefix, groupID)
}

func groupMembersKey(groupID string) string {
	return fmt.Sprintf("%s:%s", groupMembersPrefix, groupID)
}

func memberGroupsKey(memberID string) string {
	return fmt.Sprintf("%s:%s", memberGroupsPrefix, memberID)
}

func groupKeys(groupID string) []string {
	return []string{
		groupOrgKey(groupID),
		groupThingsKey(groupID),
		groupProfilesKey(groupID),
		groupMembersKey(groupID),
	}
}

func (gc *groupCache) removeGroupRelations(ctx context.Context, groupID string) error {
	removalKeys := groupKeys(groupID)
	pipe := gc.client.Pipeline()
	prefixes := []string{groupThingsPrefix, groupProfilesPrefix, groupMembersPrefix}

	for _, prefix := range prefixes {
		esKey := fmt.Sprintf("%s:%s", prefix, groupID)
		entities, err := gc.client.SMembers(ctx, esKey).Result()
		if err != nil {
			return errors.Wrap(errors.ErrRemoveEntity, err)
		}

		for _, entityID := range entities {
			switch prefix {
			case groupThingsPrefix:
				tgKey := thingGroupKey(entityID)
				tik := thingIDKey(entityID)
				thingKey, _ := gc.client.Get(ctx, tik).Result()
				tkk := thingKeyKey(thingKey)

				removalKeys = append(removalKeys, tgKey, tik, tkk)
			case groupProfilesPrefix:
				pgKey := profileGroupKey(entityID)
				removalKeys = append(removalKeys, pgKey)
			case groupMembersPrefix:
				mgKey := memberGroupsKey(entityID)
				pipe.HDel(ctx, mgKey, groupID)
			}
		}
	}
	if len(removalKeys) > 0 {
		if err := gc.client.Unlink(ctx, removalKeys...).Err(); err != nil {
			return errors.Wrap(errors.ErrRemoveEntity, err)
		}
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	return nil
}
