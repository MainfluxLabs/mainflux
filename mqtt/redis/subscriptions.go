package redis

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

const (
	thingByClientPrefix = "th_by_cl"
)

var _ Cache = (*mqttCache)(nil)

type Cache interface {
	// Connect stores a mapping between an MQTT client ID and a Thing ID.
	Connect(ctx context.Context, clientID, thingID string) error

	// Disconnect removes the cached mapping for the given MQTT client ID.
	Disconnect(ctx context.Context, clientID string) error

	// RetrieveThingByClient returns the Thing ID associated with the given MQTT client ID.
	// If no mapping exists, an empty string is returned.
	RetrieveThingByClient(ctx context.Context, clientID string) string
}

type mqttCache struct {
	client *redis.Client
}

// NewCache returns redis mqtt cache implementation.
func NewCache(client *redis.Client) Cache {
	return &mqttCache{
		client: client,
	}
}

func (m mqttCache) Connect(ctx context.Context, clientID, thingID string) error {
	key := thingByClientIDKey(clientID)

	return m.client.Set(ctx, key, thingID, 0).Err()
}

func (m mqttCache) Disconnect(ctx context.Context, clientID string) error {
	key := thingByClientIDKey(clientID)

	return m.client.Del(ctx, key).Err()
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
