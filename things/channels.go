// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"
)

// Channel represents a Mainflux "communication group". This group contains the
// things that can exchange messages between each other.
type Channel struct {
	ID       string
	Owner    string
	Name     string
	Metadata map[string]interface{}
}

// ChannelsPage contains page related metadata as well as list of channels that
// belong to this page.
type ChannelsPage struct {
	PageMetadata
	Channels []Channel
}

// Connection represents a connection between a channel and a thing.
type Connection struct {
	ChannelID    string
	ChannelOwner string
	ThingID      string
	ThingOwner   string
}

// ChannelRepository specifies a channel persistence API.
type ChannelRepository interface {
	// Save persists multiple channels. Channels are saved using a transaction. If one channel
	// fails then none will be saved. Successful operation is indicated by non-nil
	// error response.
	Save(ctx context.Context, chs ...Channel) ([]Channel, error)

	// Update performs an update to the existing channel. A non-nil error is
	// returned to indicate operation failure.
	Update(ctx context.Context, c Channel) error

	// RetrieveByID retrieves the channel having the provided identifier, that is owned
	// by the specified user.
	RetrieveByID(ctx context.Context, id string) (Channel, error)

	// RetrieveByOwner retrieves the subset of channels owned by the specified user.
	RetrieveByOwner(ctx context.Context, owner string, pm PageMetadata) (ChannelsPage, error)

	// RetrieveByThing retrieves the subset of channels owned by the specified
	// user and have specified thing connected or not connected to them.
	RetrieveByThing(ctx context.Context, owner, thID string, pm PageMetadata) (ChannelsPage, error)

	// RetrieveConns retrieves the subset of channels connected to the specified
	// thing.
	RetrieveConns(ctx context.Context, thID string, pm PageMetadata) (ChannelsPage, error)

	// Remove removes the channel having the provided identifier, that is owned
	// by the specified user.
	Remove(ctx context.Context, owner, id string) error

	// Connect adds things to the channels list of connected things.
	Connect(ctx context.Context, owner string, chIDs, thIDs []string) error

	// Disconnect removes things from the channels list of connected
	// things.
	Disconnect(ctx context.Context, owner string, chIDs, thIDs []string) error

	// HasThing determines whether the thing with the provided access key, is
	// "connected" to the specified channel. If that's the case, it returns
	// thing's ID.
	HasThing(ctx context.Context, chanID, key string) (string, error)

	// HasThingByID determines whether the thing with the provided ID, is
	// "connected" to the specified channel. If that's the case, then
	// returned error will be nil.
	HasThingByID(ctx context.Context, chanID, thingID string) error

	// RetrieveAll retrieves all channels for all users.
	RetrieveAll(ctx context.Context) ([]Channel, error)

	// RetrieveByAdmin  retrieves all channels for all users with pagination.
	RetrieveByAdmin(ctx context.Context, pm PageMetadata) (ChannelsPage, error)

	// RetrieveAllConnections retrieves all connections between channels and things for all users.
	RetrieveAllConnections(ctx context.Context) ([]Connection, error)
}

// ChannelCache contains channel-thing connection caching interface.
type ChannelCache interface {
	// Connect channel thing connection.
	Connect(context.Context, string, string) error

	// HasThing checks if thing is connected to channel.
	HasThing(context.Context, string, string) bool

	// Disconnects thing from channel.
	Disconnect(context.Context, string, string) error

	// Removes channel from cache.
	Remove(context.Context, string) error
}
