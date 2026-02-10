package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/mqtt/redis/cache"
)

type CacheMock struct {
	mu             sync.Mutex
	thingsByClient map[string]string
	clientsByThing map[string][]string
}

// NewCache returns mock cache instance.
func NewCache() cache.ConnectionCache {
	return &CacheMock{
		thingsByClient: make(map[string]string),
		clientsByThing: make(map[string][]string),
	}
}

func (c *CacheMock) Connect(_ context.Context, clientID, thingID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.thingsByClient[clientID] = thingID
	c.clientsByThing[thingID] = append(c.clientsByThing[thingID], clientID)
	return nil
}

func (c *CacheMock) Disconnect(_ context.Context, clientID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.thingsByClient, clientID)

	for _, thingID := range c.clientsByThing[clientID] {
		delete(c.clientsByThing, thingID)
	}

	return nil
}

func (c *CacheMock) DisconnectByThing(_ context.Context, thingID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, clientID := range c.clientsByThing[thingID] {
		delete(c.thingsByClient, clientID)
	}

	return nil
}

func (c *CacheMock) RetrieveThingByClient(_ context.Context, clientID string) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.thingsByClient[clientID]
}
