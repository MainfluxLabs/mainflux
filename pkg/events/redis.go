// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/authn"
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

	actorIdentityUserID = "_actor_user_id"
	actorIdentityEmail  = "_actor_user_email"

	ThingsStream = mainfluxPrefix + "things"
	AuthStream   = mainfluxPrefix + "auth"
)

// redisEvent is the raw payload delivered on a Redis stream.
type redisEvent map[string]any

// operation returns the event's operation name, or an empty string if missing.
func (e redisEvent) operation() string {
	return e.field("operation", "")
}

// actorIdentity returns the identity of the actor that initiated the event.
func (e redisEvent) actorIdentity() domain.Identity {
	var identity domain.Identity

	if actorUserID, ok := e[actorIdentityUserID]; ok {
		identity.ID, _ = actorUserID.(string)
	}

	if actorUserEmail, ok := e[actorIdentityEmail]; ok {
		identity.Email, _ = actorUserEmail.(string)
	}

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

// attachActorIdentity extends the RedisEvent with the information from a domain.Identity assocaited with the
// passed context, if it exists.
func (e redisEvent) attachActorIdentity(ctx context.Context) {
	identity, ok := authn.IdentityFromCtx(ctx)
	if !ok {
		return
	}

	e[actorIdentityUserID] = identity.ID
	e[actorIdentityEmail] = identity.Email
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
