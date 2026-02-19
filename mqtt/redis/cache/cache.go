package cache

import (
	"context"
)

type ConnectionCache interface {
	// Connect stores a mapping between an MQTT client ID and a Thing ID.
	Connect(ctx context.Context, clientID, thingID string) error

	// Disconnect removes the cached mapping for the given MQTT client ID.
	Disconnect(ctx context.Context, clientID string) error

	// DisconnectByThing removes all cached mappings associated with the specified thing ID.
	DisconnectByThing(ctx context.Context, thingID string) error

	// RetrieveThingByClient returns the Thing ID associated with the given MQTT client ID.
	// If no mapping exists, an empty string is returned.
	RetrieveThingByClient(ctx context.Context, clientID string) string
}
