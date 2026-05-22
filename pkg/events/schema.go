// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

// EventAction represents an platform action that can be encoded to a redisEvent.
type EventAction interface {
	Encode() redisEvent
}

// Event is the type representing a platform event, composed of an action and the actor's identity.
type Event struct {
	Action        EventAction
	ActorIdentity domain.Identity
}

// decodeEvent turns a raw RedisEvent into an Event.
func decodeEvent(re redisEvent) (Event, error) {
	var action EventAction

	op := re.operation()

	switch op {
	case ThingCreate:
		action = decodeThingCreated(re)
	case ThingUpdate:
		action = decodeThingUpdated(re)
	case ThingUpdateGroupAndProfile:
		action = decodeThingGroupAndProfileUpdated(re)
	case ThingRemove:
		action = decodeThingRemoved(re)
	case ProfileCreate:
		action = decodeProfileCreated(re)
	case ProfileUpdate:
		action = decodeProfileUpdated(re)
	case ProfileRemove:
		action = decodeProfileRemoved(re)
	case GroupRemove:
		action = decodeGroupRemoved(re)
	case OrgCreate:
		action = decodeOrgCreated(re)
	case OrgRemove:
		action = decodeOrgRemoved(re)
	default:
		return Event{}, fmt.Errorf("unknown event operation %s", op)
	}

	return Event{
		Action:        action,
		ActorIdentity: re.actorIdentity(),
	}, nil
}

// ThingCreated signals the creation of a thing.
type ThingCreated struct {
	ID        string
	GroupID   string
	ProfileID string
	Name      string
	Metadata  map[string]any
}

func (e ThingCreated) Encode() redisEvent {
	m := redisEvent{
		"operation":  ThingCreate,
		"id":         e.ID,
		"group_id":   e.GroupID,
		"profile_id": e.ProfileID,
	}
	if e.Name != "" {
		m["name"] = e.Name
	}
	if e.Metadata != nil {
		if b, err := json.Marshal(e.Metadata); err == nil {
			m["metadata"] = string(b)
		}
	}
	return m
}

func decodeThingCreated(e redisEvent) ThingCreated {
	t := ThingCreated{
		ID:        e.field("id", ""),
		GroupID:   e.field("group_id", ""),
		ProfileID: e.field("profile_id", ""),
		Name:      e.field("name", ""),
	}
	if raw := e.field("metadata", ""); raw != "" {
		_ = json.Unmarshal([]byte(raw), &t.Metadata)
	}
	return t
}

// ThingUpdated signals a thing update.
type ThingUpdated struct {
	ID        string
	ProfileID string
	Name      string
	Metadata  map[string]any
}

func (e ThingUpdated) Encode() redisEvent {
	m := redisEvent{
		"operation":  ThingUpdate,
		"id":         e.ID,
		"profile_id": e.ProfileID,
	}
	if e.Name != "" {
		m["name"] = e.Name
	}
	if e.Metadata != nil {
		if b, err := json.Marshal(e.Metadata); err == nil {
			m["metadata"] = string(b)
		}
	}
	return m
}

func decodeThingUpdated(e redisEvent) ThingUpdated {
	t := ThingUpdated{
		ID:        e.field("id", ""),
		ProfileID: e.field("profile_id", ""),
		Name:      e.field("name", ""),
	}
	if raw := e.field("metadata", ""); raw != "" {
		_ = json.Unmarshal([]byte(raw), &t.Metadata)
	}
	return t
}

// ThingGroupAndProfileUpdated signals a thing's group/profile reassignment.
type ThingGroupAndProfileUpdated struct {
	ID        string
	ProfileID string
	GroupID   string
}

func (e ThingGroupAndProfileUpdated) Encode() redisEvent {
	m := redisEvent{
		"operation":  ThingUpdateGroupAndProfile,
		"id":         e.ID,
		"profile_id": e.ProfileID,
	}
	if e.GroupID != "" {
		m["group_id"] = e.GroupID
	}
	return m
}

func decodeThingGroupAndProfileUpdated(e redisEvent) ThingGroupAndProfileUpdated {
	return ThingGroupAndProfileUpdated{
		ID:        e.field("id", ""),
		ProfileID: e.field("profile_id", ""),
		GroupID:   e.field("group_id", ""),
	}
}

// ThingRemoved signals that a thing has been removed.
type ThingRemoved struct {
	ID string
}

func (e ThingRemoved) Encode() redisEvent {
	return redisEvent{
		"operation": ThingRemove,
		"id":        e.ID,
	}
}

func decodeThingRemoved(e redisEvent) ThingRemoved {
	return ThingRemoved{ID: e.field("id", "")}
}

// ProfileCreated signals the creation of a profile.
type ProfileCreated struct {
	ID       string
	GroupID  string
	Name     string
	Metadata map[string]any
}

func (e ProfileCreated) Encode() redisEvent {
	m := redisEvent{
		"operation": ProfileCreate,
		"id":        e.ID,
		"group_id":  e.GroupID,
	}
	if e.Name != "" {
		m["name"] = e.Name
	}
	if e.Metadata != nil {
		if b, err := json.Marshal(e.Metadata); err == nil {
			m["metadata"] = string(b)
		}
	}
	return m
}

func decodeProfileCreated(e redisEvent) ProfileCreated {
	p := ProfileCreated{
		ID:      e.field("id", ""),
		GroupID: e.field("group_id", ""),
		Name:    e.field("name", ""),
	}
	if raw := e.field("metadata", ""); raw != "" {
		_ = json.Unmarshal([]byte(raw), &p.Metadata)
	}
	return p
}

// ProfileUpdated signals a profile update.
type ProfileUpdated struct {
	ID       string
	Name     string
	Config   *domain.ProfileConfig
	Metadata map[string]any
}

func (e ProfileUpdated) Encode() redisEvent {
	m := redisEvent{
		"operation": ProfileUpdate,
		"id":        e.ID,
	}
	if e.Name != "" {
		m["name"] = e.Name
	}
	if e.Config != nil {
		if b, err := json.Marshal(e.Config); err == nil {
			m["config"] = string(b)
		}
	}
	if e.Metadata != nil {
		if b, err := json.Marshal(e.Metadata); err == nil {
			m["metadata"] = string(b)
		}
	}
	return m
}

func decodeProfileUpdated(e redisEvent) ProfileUpdated {
	p := ProfileUpdated{
		ID:   e.field("id", ""),
		Name: e.field("name", ""),
	}
	if raw := e.field("config", ""); raw != "" {
		var cfg domain.ProfileConfig
		if err := json.Unmarshal([]byte(raw), &cfg); err == nil {
			p.Config = &cfg
		}
	}
	if raw := e.field("metadata", ""); raw != "" {
		_ = json.Unmarshal([]byte(raw), &p.Metadata)
	}
	return p
}

// ProfileRemoved signals that a profile has been removed.
type ProfileRemoved struct {
	ID string
}

func (e ProfileRemoved) Encode() redisEvent {
	return redisEvent{
		"operation": ProfileRemove,
		"id":        e.ID,
	}
}

func decodeProfileRemoved(e redisEvent) ProfileRemoved {
	return ProfileRemoved{ID: e.field("id", "")}
}

// GroupRemoved signals that a group has been removed.
type GroupRemoved struct {
	ID       string
	ThingIDs []string
}

func (e GroupRemoved) Encode() redisEvent {
	m := redisEvent{
		"operation": GroupRemove,
		"id":        e.ID,
	}
	if len(e.ThingIDs) > 0 {
		m["thing_ids"] = strings.Join(e.ThingIDs, ",")
	}
	return m
}

func decodeGroupRemoved(e redisEvent) GroupRemoved {
	g := GroupRemoved{ID: e.field("id", "")}
	if raw := e.field("thing_ids", ""); raw != "" {
		g.ThingIDs = strings.Split(raw, ",")
	}
	return g
}

// OrgCreated signals the creation of an organization.
type OrgCreated struct {
	ID string
}

func (e OrgCreated) Encode() redisEvent {
	return redisEvent{
		"operation": OrgCreate,
		"id":        e.ID,
	}
}

func decodeOrgCreated(e redisEvent) OrgCreated {
	return OrgCreated{ID: e.field("id", "")}
}

// OrgRemoved signals that an organization has been removed.
type OrgRemoved struct {
	ID string
}

func (e OrgRemoved) Encode() redisEvent {
	return redisEvent{
		"operation": OrgRemove,
		"id":        e.ID,
	}
}

func decodeOrgRemoved(e redisEvent) OrgRemoved {
	return OrgRemoved{ID: e.field("id", "")}
}
