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
	client  *redis.Client
	thCache things.ThingCache
}

// NewGroupCache returns redis group cache implementation.
func NewGroupCache(client *redis.Client, thCache things.ThingCache) things.GroupCache {
	return &groupCache{
		client:  client,
		thCache: thCache,
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
	var (
		cursor uint64
		keys   []string
		err    error
	)

	if err = gc.clearGroup(ctx, groupID, groupThingsPrefix, groupProfilesPrefix, groupMembersPrefix); err != nil {
		return err
	}

	// Using this key pattern will delete keys whose prefix starts with 'gr' and the suffix is '<group_id>', such as:
	// gr_org:<group_id>, gr_ths:<group_id>, gr_prs:<group_id>, gr_mbs:<group_id>
	key := fmt.Sprintf("gr_*:%s", groupID)
	for {
		keys, cursor, err = gc.client.Scan(ctx, cursor, key, 100).Result()
		if err != nil {
			return errors.Wrap(errors.ErrRemoveEntity, err)
		}

		if len(keys) > 0 {
			if err = gc.client.Unlink(ctx, keys...).Err(); err != nil {
				return errors.Wrap(errors.ErrRemoveEntity, err)
			}
		}

		if cursor == 0 {
			break
		}
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

func (gc *groupCache) clearGroup(ctx context.Context, groupID string, keyPrefixes ...string) error {
	var gKey string
	for _, prefix := range keyPrefixes {
		esKey := fmt.Sprintf("%s:%s", prefix, groupID)
		entities, err := gc.client.SMembers(ctx, esKey).Result()
		if err != nil {
			return errors.Wrap(errors.ErrRemoveEntity, err)
		}

		for _, entityID := range entities {
			switch prefix {
			case groupThingsPrefix:
				gKey = thingGroupKey(entityID)
				if err := gc.client.Del(ctx, gKey).Err(); err != nil {
					return errors.Wrap(errors.ErrRemoveEntity, err)
				}

				if err := gc.thCache.Remove(ctx, entityID); err != nil {
					return err
				}
			case groupProfilesPrefix:
				gKey = profileGroupKey(entityID)
				if err := gc.client.Del(ctx, gKey).Err(); err != nil {
					return errors.Wrap(errors.ErrRemoveEntity, err)
				}
			case groupMembersPrefix:
				gKey = memberGroupsKey(entityID)
				if _, err := gc.client.HDel(ctx, gKey, groupID).Result(); err != nil {
					return errors.Wrap(errors.ErrRemoveEntity, err)
				}
			}
		}
	}

	return nil
}
