// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"encoding/json"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

// ThingCreated signals the creation of a thing.
type ThingCreated struct {
	ID        string
	GroupID   string
	ProfileID string
	Name      string
	Metadata  map[string]any
}

func (e ThingCreated) Encode() map[string]any {
	m := map[string]any{
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

func (e ThingCreated) Operation() string {
	return ThingCreate
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

func (e ThingUpdated) Operation() string {
	return ThingUpdate
}

func (e ThingUpdated) Encode() map[string]any {
	m := map[string]any{
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

func (e ThingGroupAndProfileUpdated) Operation() string {
	return ThingUpdateGroupAndProfile
}

func (e ThingGroupAndProfileUpdated) Encode() map[string]any {
	m := map[string]any{
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

func (e ThingRemoved) Operation() string {
	return ThingRemove
}

func (e ThingRemoved) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
	}
}

func decodeThingRemoved(e redisEvent) ThingRemoved {
	return ThingRemoved{ID: e.field("id", "")}
}

// ThingMetadataUpdated signals a merge of metadata keys into a thing's metadata.
type ThingMetadataUpdated struct {
	ID       string
	Metadata map[string]any
}

func (e ThingMetadataUpdated) Operation() string {
	return ThingUpdateMetadata
}

func (e ThingMetadataUpdated) Encode() map[string]any {
	m := map[string]any{
		"id": e.ID,
	}
	if e.Metadata != nil {
		if b, err := json.Marshal(e.Metadata); err == nil {
			m["metadata"] = string(b)
		}
	}
	return m
}

func decodeThingMetadataUpdated(e redisEvent) ThingMetadataUpdated {
	t := ThingMetadataUpdated{ID: e.field("id", "")}
	if raw := e.field("metadata", ""); raw != "" {
		_ = json.Unmarshal([]byte(raw), &t.Metadata)
	}
	return t
}

// ThingExternalKeyUpdated signals that a thing's external key was set, without the key itself.
type ThingExternalKeyUpdated struct {
	ID string
}

func (e ThingExternalKeyUpdated) Operation() string {
	return ThingUpdateExternalKey
}

func (e ThingExternalKeyUpdated) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
	}
}

func decodeThingExternalKeyUpdated(e redisEvent) ThingExternalKeyUpdated {
	return ThingExternalKeyUpdated{ID: e.field("id", "")}
}

// ThingExternalKeyRemoved signals that a thing's external key was removed.
type ThingExternalKeyRemoved struct {
	ID string
}

func (e ThingExternalKeyRemoved) Operation() string {
	return ThingRemoveExternalKey
}

func (e ThingExternalKeyRemoved) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
	}
}

func decodeThingExternalKeyRemoved(e redisEvent) ThingExternalKeyRemoved {
	return ThingExternalKeyRemoved{ID: e.field("id", "")}
}

// ProfileCreated signals the creation of a profile.
type ProfileCreated struct {
	ID       string
	GroupID  string
	Name     string
	Metadata map[string]any
}

func (e ProfileCreated) Operation() string {
	return ProfileCreate
}

func (e ProfileCreated) Encode() map[string]any {
	m := redisEvent{
		"id":       e.ID,
		"group_id": e.GroupID,
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

func (e ProfileUpdated) Operation() string {
	return ProfileUpdate
}

func (e ProfileUpdated) Encode() map[string]any {
	m := redisEvent{
		"id": e.ID,
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

func (e ProfileRemoved) Operation() string {
	return ProfileRemove
}

func (e ProfileRemoved) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
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

func (e GroupRemoved) Operation() string {
	return GroupRemove
}

func (e GroupRemoved) Encode() map[string]any {
	m := redisEvent{
		"id": e.ID,
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

// GroupCreated signals the creation of a group.
type GroupCreated struct {
	ID          string
	Name        string
	Description string
	Metadata    map[string]any
}

func (e GroupCreated) Operation() string {
	return GroupCreate
}

func (e GroupCreated) Encode() map[string]any {
	return encodeDescribed(e.ID, e.Name, e.Description, e.Metadata)
}

func decodeGroupCreated(e redisEvent) GroupCreated {
	g := GroupCreated{
		ID:          e.field("id", ""),
		Name:        e.field("name", ""),
		Description: e.field("description", ""),
	}
	if raw := e.field("metadata", ""); raw != "" {
		_ = json.Unmarshal([]byte(raw), &g.Metadata)
	}
	return g
}

// GroupUpdated signals a group update.
type GroupUpdated struct {
	ID          string
	Name        string
	Description string
	Metadata    map[string]any
}

func (e GroupUpdated) Operation() string {
	return GroupUpdate
}

func (e GroupUpdated) Encode() map[string]any {
	return encodeDescribed(e.ID, e.Name, e.Description, e.Metadata)
}

func decodeGroupUpdated(e redisEvent) GroupUpdated {
	g := GroupUpdated{
		ID:          e.field("id", ""),
		Name:        e.field("name", ""),
		Description: e.field("description", ""),
	}
	if raw := e.field("metadata", ""); raw != "" {
		_ = json.Unmarshal([]byte(raw), &g.Metadata)
	}
	return g
}

// GroupMembershipCreated signals that a member was added to a group.
type GroupMembershipCreated struct {
	GroupID  string
	MemberID string
	Role     string
}

func (e GroupMembershipCreated) Operation() string {
	return GroupMembershipCreate
}

func (e GroupMembershipCreated) Encode() map[string]any {
	return redisEvent{
		"group_id":  e.GroupID,
		"member_id": e.MemberID,
		"role":      e.Role,
	}
}

func decodeGroupMembershipCreated(e redisEvent) GroupMembershipCreated {
	return GroupMembershipCreated{
		GroupID:  e.field("group_id", ""),
		MemberID: e.field("member_id", ""),
		Role:     e.field("role", ""),
	}
}

// GroupMembershipUpdated signals that a group member's role was changed.
type GroupMembershipUpdated struct {
	GroupID  string
	MemberID string
	Role     string
}

func (e GroupMembershipUpdated) Operation() string {
	return GroupMembershipUpdate
}

func (e GroupMembershipUpdated) Encode() map[string]any {
	return redisEvent{
		"group_id":  e.GroupID,
		"member_id": e.MemberID,
		"role":      e.Role,
	}
}

func decodeGroupMembershipUpdated(e redisEvent) GroupMembershipUpdated {
	return GroupMembershipUpdated{
		GroupID:  e.field("group_id", ""),
		MemberID: e.field("member_id", ""),
		Role:     e.field("role", ""),
	}
}

// GroupMembershipRemoved signals that a member was removed from a group.
type GroupMembershipRemoved struct {
	GroupID  string
	MemberID string
}

func (e GroupMembershipRemoved) Operation() string {
	return GroupMembershipRemove
}

func (e GroupMembershipRemoved) Encode() map[string]any {
	return redisEvent{
		"group_id":  e.GroupID,
		"member_id": e.MemberID,
	}
}

func decodeGroupMembershipRemoved(e redisEvent) GroupMembershipRemoved {
	return GroupMembershipRemoved{
		GroupID:  e.field("group_id", ""),
		MemberID: e.field("member_id", ""),
	}
}
