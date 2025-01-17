package redis

import (
	"context"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-redis/redis/v8"
)

const (
	orgByGroupPrefix     = "org_by_gr"
	membersByGroupPrefix = "mbs_by_gr"
	groupsByMemberPrefix = "grs_by_mb"
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
	ok := orgByGroupIDKey(groupID)
	if err := gc.client.Set(ctx, ok, orgID, 0).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (gc *groupCache) ViewOrg(ctx context.Context, groupID string) (string, error) {
	ok := orgByGroupIDKey(groupID)
	orgID, err := gc.client.Get(ctx, ok).Result()
	if err != nil {
		return "", errors.Wrap(errors.ErrNotFound, err)
	}

	return orgID, nil
}

func (gc *groupCache) RemoveOrg(ctx context.Context, groupID string) error {
	removalKeys := []string{
		orgByGroupIDKey(groupID),
		thingsByGroupIDKey(groupID),
		profilesByGroupIDKey(groupID),
		membersByGroupIDKey(groupID),
	}
	pipe := gc.client.Pipeline()
	prefixes := []string{thingsByGroupPrefix, profilesByGroupPrefix, membersByGroupPrefix}

	for _, prefix := range prefixes {
		esKey := fmt.Sprintf("%s:%s", prefix, groupID)
		entities, err := gc.client.SMembers(ctx, esKey).Result()
		if err != nil {
			return errors.Wrap(errors.ErrRemoveEntity, err)
		}

		for _, entityID := range entities {
			switch prefix {
			case thingsByGroupPrefix:
				gk := groupByThingIDKey(entityID)
				kk := keyByThingIDKey(entityID)
				removalKeys = append(removalKeys, gk, kk)

				if thingKey, err := gc.client.Get(ctx, kk).Result(); err == nil {
					ik := idByThingKeyKey(thingKey)
					removalKeys = append(removalKeys, ik)
				}
			case profilesByGroupPrefix:
				gk := groupByProfileIDKey(entityID)
				removalKeys = append(removalKeys, gk)
			case membersByGroupPrefix:
				gk := groupsByMemberIDKey(entityID)
				pipe.HDel(ctx, gk, groupID)
			}
		}
	}

	if err := gc.client.Unlink(ctx, removalKeys...).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	return nil
}

func (gc *groupCache) SaveRole(ctx context.Context, groupID, memberID, role string) error {
	gk := groupsByMemberIDKey(memberID)
	if err := gc.client.HSet(ctx, gk, groupID, role).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	mk := membersByGroupIDKey(groupID)
	if err := gc.client.SAdd(ctx, mk, memberID).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (gc *groupCache) ViewRole(ctx context.Context, groupID, memberID string) (string, error) {
	gk := groupsByMemberIDKey(memberID)
	role, err := gc.client.HGet(ctx, gk, groupID).Result()
	if err != nil {
		return "", errors.Wrap(errors.ErrNotFound, err)
	}

	return role, nil
}

func (gc *groupCache) RemoveRole(ctx context.Context, groupID, memberID string) error {
	gk := groupsByMemberIDKey(memberID)
	if _, err := gc.client.HDel(ctx, gk, groupID).Result(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	mk := membersByGroupIDKey(groupID)
	if _, err := gc.client.SRem(ctx, mk, memberID).Result(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	return nil
}

func (gc *groupCache) GroupMemberships(ctx context.Context, memberID string) ([]string, error) {
	gk := groupsByMemberIDKey(memberID)
	groups, err := gc.client.HKeys(ctx, gk).Result()
	if err != nil {
		return nil, errors.Wrap(errors.ErrNotFound, err)
	}

	return groups, nil
}

func orgByGroupIDKey(groupID string) string {
	return fmt.Sprintf("%s:%s", orgByGroupPrefix, groupID)
}

func membersByGroupIDKey(groupID string) string {
	return fmt.Sprintf("%s:%s", membersByGroupPrefix, groupID)
}

func groupsByMemberIDKey(memberID string) string {
	return fmt.Sprintf("%s:%s", groupsByMemberPrefix, memberID)
}
