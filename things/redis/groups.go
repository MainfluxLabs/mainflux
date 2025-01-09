package redis

import (
	"context"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-redis/redis/v8"
)

const (
	grPrefix  = "group"
	orgSuffix = "org"
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

func (gc *groupCache) SaveOrgID(ctx context.Context, groupID, orgID string) error {
	gk := goKey(groupID)
	if err := gc.client.Set(ctx, gk, orgID, 0).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (gc *groupCache) OrgID(ctx context.Context, groupID string) (string, error) {
	gk := goKey(groupID)
	orgID, err := gc.client.Get(ctx, gk).Result()
	if err != nil {
		return "", errors.Wrap(errors.ErrNotFound, err)
	}

	return orgID, nil
}

func (gc *groupCache) RemoveOrgID(ctx context.Context, groupID string) error {
	gk := goKey(groupID)
	if err := gc.client.Del(ctx, gk).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	return nil
}

func (gc *groupCache) SaveRole(ctx context.Context, groupID, memberID, role string) error {
	rk := rKey(groupID, memberID)
	if err := gc.client.Set(ctx, rk, role, 0).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (gc *groupCache) Role(ctx context.Context, groupID, memberID string) (string, error) {
	rk := rKey(groupID, memberID)
	role, err := gc.client.Get(ctx, rk).Result()
	if err != nil {
		return "", errors.Wrap(errors.ErrNotFound, err)
	}

	return role, nil
}

func (gc *groupCache) RemoveRole(ctx context.Context, groupID, memberID string) error {
	// Redis returns Nil Reply when key does not exist.
	rk := rKey(groupID, memberID)
	if err := gc.client.Del(ctx, rk).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}
	return nil
}

func goKey(groupID string) string {
	return fmt.Sprintf("%s:%s:%s", grPrefix, groupID, orgSuffix)
}

func rKey(groupID, memberID string) string {
	return fmt.Sprintf("%s:%s:%s", grPrefix, groupID, memberID)
}
