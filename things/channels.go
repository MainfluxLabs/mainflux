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
	OwnerID  string
	GroupID  string
	Name     string
	Profile  map[string]interface{}
	Metadata map[string]interface{}
}

type Profile struct {
	ContentType string      `json:"content_type"`
	Write       bool        `json:"write"`
	WebhookID   string      `json:"webhook_id"`
	Transformer Transformer `json:"transformer"`
	SmtpID      string      `json:"smtp_id"`
	SmppID      string      `json:"smpp_id"`
}

type Transformer struct {
	ValueFields  []string `json:"value_fields"`
	TimeField    string   `json:"time_field"`
	TimeFormat   string   `json:"time_format"`
	TimeLocation string   `json:"time_location"`
}

type Notifier struct {
	ID       string
	GroupID  string
	Contacts []string
}

// ChannelsPage contains page related metadata as well as list of channels that
// belong to this page.
type ChannelsPage struct {
	PageMetadata
	Channels []Channel
}

// Connection represents a connection between a channel and a thing.
type Connection struct {
	ChannelID string
	ThingID   string
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

	// RetrieveByThing retrieves the channel connected to the given thing id.
	RetrieveByThing(ctx context.Context, thID string) (Channel, error)

	// Remove removes the channels having the provided identifiers, that is owned
	// by the specified user.
	Remove(ctx context.Context, owner string, id ...string) error

	// Connect connects a list of things to a channel.
	Connect(ctx context.Context, chID string, thIDs []string) error

	// Disconnect disconnects a list of things from a channel.
	Disconnect(ctx context.Context, chID string, thIDs []string) error

	// RetrieveConnByThingKey retrieves connections IDs by ThingKey
	RetrieveConnByThingKey(ctx context.Context, key string) (Connection, error)

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
