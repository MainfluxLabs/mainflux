package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/mqtt/redis"
)

type CacheMock struct {
	mu             sync.Mutex
	thingsByClient map[string]string
}

// NewCache returns mock cache instance.
func NewCache() redis.Cache {
	return &CacheMock{
		thingsByClient: make(map[string]string),
	}
}

func (c *CacheMock) Connect(_ context.Context, clientID, thingID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.thingsByClient[clientID] = thingID
	return nil
}

func (c *CacheMock) Disconnect(_ context.Context, clientID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.thingsByClient, clientID)
	return nil
}

func (c *CacheMock) RetrieveThingByClient(_ context.Context, clientID string) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.thingsByClient[clientID]
}
