package cache

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

const (
	thingByClientPrefix  = "th_by_cl"
	clientsByThingPrefix = "cls_by_th"
)

var _ ConnectionCache = (*mqttCache)(nil)

type mqttCache struct {
	client *redis.Client
}

// NewConnectionCache returns redis mqtt cache implementation.
func NewConnectionCache(client *redis.Client) ConnectionCache {
	return &mqttCache{
		client: client,
	}
}

func (m mqttCache) Connect(ctx context.Context, clientID, thingID string) error {
	tk := thingByClientIDKey(clientID)
	ck := clientsByThingIDKey(thingID)
	pipe := m.client.TxPipeline()

	pipe.Set(ctx, tk, thingID, 0)
	pipe.SAdd(ctx, ck, clientID)

	_, err := pipe.Exec(ctx)
	return err
}

func (m mqttCache) Disconnect(ctx context.Context, clientID string) error {
	key := thingByClientIDKey(clientID)

	thingID, _ := m.client.Get(ctx, key).Result()

	pipe := m.client.TxPipeline()
	pipe.Del(ctx, key)

	if thingID != "" {
		pipe.SRem(ctx, clientsByThingIDKey(thingID), clientID)
	}

	_, err := pipe.Exec(ctx)
	return err
}

func (m mqttCache) DisconnectByThing(ctx context.Context, thingID string) error {
	key := clientsByThingIDKey(thingID)

	clientIDs, err := m.client.SMembers(ctx, key).Result()
	if err != nil {
		return err
	}

	pipe := m.client.TxPipeline()

	for _, clientID := range clientIDs {
		pipe.Del(ctx, thingByClientIDKey(clientID))
	}

	pipe.Del(ctx, key)

	_, err = pipe.Exec(ctx)
	return err
}

func (m mqttCache) RetrieveThingByClient(ctx context.Context, clientID string) string {
	key := thingByClientIDKey(clientID)

	val, err := m.client.Get(ctx, key).Result()
	if err != nil {
		return ""
	}

	return val
}

func thingByClientIDKey(clientID string) string {
	return fmt.Sprintf("%s:%s", thingByClientPrefix, clientID)
}

func clientsByThingIDKey(thingID string) string {
	return fmt.Sprintf("%s:%s", clientsByThingPrefix, thingID)
}
