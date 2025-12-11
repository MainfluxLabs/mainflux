// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
)

const (
	thingPrefix    = "thing."
	profilePrefix  = "profile."
	groupPrefix    = "group."
	orgPrefix      = "org."
	mainfluxPrefix = "mainflux."

	ThingCreate                = thingPrefix + "create"
	ThingUpdate                = thingPrefix + "update"
	ThingUpdateGroupAndProfile = thingPrefix + "update_group_and_profile"
	ThingRemove                = thingPrefix + "remove"

	ProfileCreate = profilePrefix + "create"
	ProfileUpdate = profilePrefix + "update"
	ProfileRemove = profilePrefix + "remove"

	GroupRemove = groupPrefix + "remove"

	OrgCreate = orgPrefix + "create"
	OrgRemove = orgPrefix + "remove"

	ThingsStream = mainfluxPrefix + "things"
	AuthStream   = mainfluxPrefix + "auth"
)

// Event represents an event.
type Event interface {
	// Encode encodes event to map.
	Encode() (map[string]interface{}, error)
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

// ReadField returns the string value stored under the given map key.
// If the key is missing or not a string, it returns the provided default.
func ReadField(event map[string]any, key, def string) string {
	val, ok := event[key].(string)
	if !ok {
		return def
	}

	return val
}
