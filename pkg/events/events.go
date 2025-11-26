// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
)

// Event represents an event.
type Event interface {
	// Encode encodes event to map.
	Encode() (map[string]any, error)
}

// EventHandler represents event handler for Subscriber.
type EventHandler interface {
	// Handle handles events passed by underlying implementation.
	Handle(ctx context.Context, event Event) error
}

// Subscriber specifies event subscription API.
type Subscriber interface {
	// Subscribe subscribes to the event stream and consumes events.
	Subscribe(ctx context.Context, handler EventHandler) error

	// Close gracefully closes event subscriber's connection.
	Close() error
}
