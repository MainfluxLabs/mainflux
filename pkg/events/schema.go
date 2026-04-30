// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"encoding/json"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

// Event is implemented by concrete typed events that can be encoded
// to a type that can be transmitted by a Redis client.
type Event interface {
	Encode() RedisEvent
}

// decodeEvent turns a raw RedisEvent into its typed Event equivalent based
// on the operation field, or returns nil when the operation is unknown.
func decodeEvent(re RedisEvent) Event {
	switch re.Operation() {
	case ThingCreate:
		return DecodeThingCreated(re)
	case ThingUpdate:
		return DecodeThingUpdated(re)
	case ThingUpdateGroupAndProfile:
		return DecodeThingGroupAndProfileUpdated(re)
	case ThingRemove:
		return DecodeThingRemoved(re)
	case ProfileCreate:
		return DecodeProfileCreated(re)
	case ProfileUpdate:
		return DecodeProfileUpdated(re)
	case ProfileRemove:
		return DecodeProfileRemoved(re)
	case GroupRemove:
		return DecodeGroupRemoved(re)
	case OrgCreate:
		return DecodeOrgCreated(re)
	case OrgRemove:
		return DecodeOrgRemoved(re)
	}

	return nil
}

// ThingCreated signals the creation of a thing.
type ThingCreated struct {
	ID        string
	GroupID   string
	ProfileID string
	Name      string
	Metadata  map[string]any
}

func (e ThingCreated) Encode() RedisEvent {
	m := RedisEvent{
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

func DecodeThingCreated(e RedisEvent) ThingCreated {
	t := ThingCreated{
		ID:        e.Field("id", ""),
		GroupID:   e.Field("group_id", ""),
		ProfileID: e.Field("profile_id", ""),
		Name:      e.Field("name", ""),
	}
	if raw := e.Field("metadata", ""); raw != "" {
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

func (e ThingUpdated) Encode() RedisEvent {
	m := RedisEvent{
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

func DecodeThingUpdated(e RedisEvent) ThingUpdated {
	t := ThingUpdated{
		ID:        e.Field("id", ""),
		ProfileID: e.Field("profile_id", ""),
		Name:      e.Field("name", ""),
	}
	if raw := e.Field("metadata", ""); raw != "" {
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

func (e ThingGroupAndProfileUpdated) Encode() RedisEvent {
	m := RedisEvent{
		"operation":  ThingUpdateGroupAndProfile,
		"id":         e.ID,
		"profile_id": e.ProfileID,
	}
	if e.GroupID != "" {
		m["group_id"] = e.GroupID
	}
	return m
}

func DecodeThingGroupAndProfileUpdated(e RedisEvent) ThingGroupAndProfileUpdated {
	return ThingGroupAndProfileUpdated{
		ID:        e.Field("id", ""),
		ProfileID: e.Field("profile_id", ""),
		GroupID:   e.Field("group_id", ""),
	}
}

// ThingRemoved signals that a thing has been removed.
type ThingRemoved struct {
	ID string
}

func (e ThingRemoved) Encode() RedisEvent {
	return RedisEvent{
		"operation": ThingRemove,
		"id":        e.ID,
	}
}

func DecodeThingRemoved(e RedisEvent) ThingRemoved {
	return ThingRemoved{ID: e.Field("id", "")}
}

// ProfileCreated signals the creation of a profile.
type ProfileCreated struct {
	ID       string
	GroupID  string
	Name     string
	Metadata map[string]any
}

func (e ProfileCreated) Encode() RedisEvent {
	m := RedisEvent{
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

func DecodeProfileCreated(e RedisEvent) ProfileCreated {
	p := ProfileCreated{
		ID:      e.Field("id", ""),
		GroupID: e.Field("group_id", ""),
		Name:    e.Field("name", ""),
	}
	if raw := e.Field("metadata", ""); raw != "" {
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

func (e ProfileUpdated) Encode() RedisEvent {
	m := RedisEvent{
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

// ConfigMap returns Config decoded as a generic map, for callers whose
// downstream APIs take map[string]any rather than *domain.ProfileConfig.
func (e ProfileUpdated) ConfigMap() map[string]any {
	if e.Config == nil {
		return nil
	}
	b, err := json.Marshal(e.Config)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	return m
}

func DecodeProfileUpdated(e RedisEvent) ProfileUpdated {
	p := ProfileUpdated{
		ID:   e.Field("id", ""),
		Name: e.Field("name", ""),
	}
	if raw := e.Field("config", ""); raw != "" {
		var cfg domain.ProfileConfig
		if err := json.Unmarshal([]byte(raw), &cfg); err == nil {
			p.Config = &cfg
		}
	}
	if raw := e.Field("metadata", ""); raw != "" {
		_ = json.Unmarshal([]byte(raw), &p.Metadata)
	}
	return p
}

// ProfileRemoved signals that a profile has been removed.
type ProfileRemoved struct {
	ID string
}

func (e ProfileRemoved) Encode() RedisEvent {
	return RedisEvent{
		"operation": ProfileRemove,
		"id":        e.ID,
	}
}

func DecodeProfileRemoved(e RedisEvent) ProfileRemoved {
	return ProfileRemoved{ID: e.Field("id", "")}
}

// GroupRemoved signals that a group has been removed.
type GroupRemoved struct {
	ID string
}

func (e GroupRemoved) Encode() RedisEvent {
	return RedisEvent{
		"operation": GroupRemove,
		"id":        e.ID,
	}
}

func DecodeGroupRemoved(e RedisEvent) GroupRemoved {
	return GroupRemoved{ID: e.Field("id", "")}
}

// OrgCreated signals the creation of an organization.
type OrgCreated struct {
	ID string
}

func (e OrgCreated) Encode() RedisEvent {
	return RedisEvent{
		"operation": OrgCreate,
		"id":        e.ID,
	}
}

func DecodeOrgCreated(e RedisEvent) OrgCreated {
	return OrgCreated{ID: e.Field("id", "")}
}

// OrgRemoved signals that an organization has been removed.
type OrgRemoved struct {
	ID string
}

func (e OrgRemoved) Encode() RedisEvent {
	return RedisEvent{
		"operation": OrgRemove,
		"id":        e.ID,
	}
}

func DecodeOrgRemoved(e RedisEvent) OrgRemoved {
	return OrgRemoved{ID: e.Field("id", "")}
}
