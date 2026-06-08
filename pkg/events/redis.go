// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
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

	jwtIdentityUserID    = "jwt_identity_user_id"
	jwtIdentityUserEmail = "jwt_identity_user_email"

	ThingsStream = mainfluxPrefix + "things"
	AuthStream   = mainfluxPrefix + "auth"
)

// redisEvent is the raw payload delivered on a Redis stream.
type redisEvent map[string]any

// operation returns the event's operation name, or an empty string if missing.
func (e redisEvent) operation() string {
	return e.field("operation", "")
}

// jwtUserIdentity returns the identity of the user that initiated the event.
func (e redisEvent) jwtUserIdentity() domain.Identity {
	var identity domain.Identity

	identity.ID, _ = e[jwtIdentityUserID].(string)
	identity.Email, _ = e[jwtIdentityUserEmail].(string)

	return identity
}

// field returns the string value stored under key, or def if missing.
func (e redisEvent) field(key, def string) string {
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

// cursorKey is the Redis key where a named subscriber stores its last
// processed message ID for the given stream.
func cursorKey(name, stream string) string {
	return fmt.Sprintf("mainflux:cursor:%s:%s", name, stream)
}
