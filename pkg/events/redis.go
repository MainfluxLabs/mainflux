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

// RedisEvent is the raw payload delivered on a Redis stream.
type RedisEvent map[string]any

// Operation returns the event's operation name, or an empty string if missing.
func (e RedisEvent) Operation() string {
	s, _ := e["operation"].(string)
	return s
}

// Field returns the string value stored under key, or def if missing.
func (e RedisEvent) Field(key, def string) string {
	s, ok := e[key].(string)
	if !ok {
		return def
	}
	return s
}

// EventHandler reacts to a single event delivered by a Subscriber.
type EventHandler interface {
	Handle(ctx context.Context, event Event) error
}

// Subscriber specifies event subscription API.
type Subscriber interface {
	// Subscribe subscribes to the event stream and consumes events.
	Subscribe(ctx context.Context, handler EventHandler) error

	// Close gracefully closes event subscriber's connection.
	Close() error
}
