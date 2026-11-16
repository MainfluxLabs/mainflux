// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

// EventAction represents an action occurring on the platform.
type EventAction interface {
	// Encode returns a map representation of the event action.
	Encode() map[string]any

	// Operation returns the string representation of the event action operation.
	Operation() string
}

// Event is the type representing a platform event.
type Event struct {
	// The specific EventAction that occurred.
	Action EventAction
	// The identity of the user whose action triggered the event.
	JWTUserIdentity domain.Identity
	// The ID of the organization to which the event belongs. May be empty.
	OrgID string
	// The ID of the group to which the event belongs. May be empty.
	GroupID    string
	OccurredAt time.Time
}

// Encode turns an Event into a redisEvent
func (e Event) Encode() redisEvent {
	re := redisEvent(e.Action.Encode())
	re["operation"] = e.Action.Operation()

	if e.JWTUserIdentity.ID != "" {
		re[jwtIdentityUserID] = e.JWTUserIdentity.ID
		re[jwtIdentityUserEmail] = e.JWTUserIdentity.Email
	}

	if e.JWTUserIdentity.Impersonator != nil {
		re[jwtIdentityImpersonatorID] = e.JWTUserIdentity.Impersonator.ID
		re[jwtIdentityImpersonatorEmail] = e.JWTUserIdentity.Impersonator.Email
	}

	if !e.OccurredAt.IsZero() {
		re[occurredAt] = e.OccurredAt.UTC().Format(time.RFC3339Nano)
	}

	re[evtOrgID] = e.OrgID
	re[evtGroupID] = e.GroupID

	return re
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
	case ThingUpdateMetadata:
		action = decodeThingMetadataUpdated(re)
	case ThingUpdateExternalKey:
		action = decodeThingExternalKeyUpdated(re)
	case ThingRemoveExternalKey:
		action = decodeThingExternalKeyRemoved(re)
	case GroupCreate:
		action = decodeGroupCreated(re)
	case GroupUpdate:
		action = decodeGroupUpdated(re)
	case GroupRemove:
		action = decodeGroupRemoved(re)
	case GroupMembershipCreate:
		action = decodeGroupMembershipCreated(re)
	case GroupMembershipUpdate:
		action = decodeGroupMembershipUpdated(re)
	case GroupMembershipRemove:
		action = decodeGroupMembershipRemoved(re)
	case OrgCreate:
		action = decodeOrgCreated(re)
	case OrgUpdate:
		action = decodeOrgUpdated(re)
	case OrgRemove:
		action = decodeOrgRemoved(re)
	case OrgMembershipCreate:
		action = decodeOrgMembershipCreated(re)
	case OrgMembershipUpdate:
		action = decodeOrgMembershipUpdated(re)
	case OrgMembershipRemove:
		action = decodeOrgMembershipRemoved(re)
	case OrgInviteCreate:
		action = decodeOrgInviteCreated(re)
	case OrgInviteRevoke:
		action = decodeOrgInviteRevoked(re)
	case OrgInviteAccept:
		action = decodeOrgInviteAccepted(re)
	case OrgInviteDecline:
		action = decodeOrgInviteDeclined(re)
	case APIKeyIssue:
		action = decodeAPIKeyIssued(re)
	case APIKeyRevoke:
		action = decodeAPIKeyRevoked(re)
	case SessionLogoutAll:
		action = SessionLoggedOutAll{}
	case UserImpersonate:
		action = decodeUserImpersonated(re)
	default:
		return Event{}, fmt.Errorf("unknown event operation %s", op)
	}

	return Event{
		Action:          action,
		JWTUserIdentity: re.jwtUserIdentity(),
		OccurredAt:      re.occurredAt(),
		OrgID:           re.field(evtOrgID, ""),
		GroupID:         re.field(evtGroupID, ""),
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

// encodeDescribed encodes an entity that has a name, description and metadata.
func encodeDescribed(id, name, description string, metadata map[string]any) map[string]any {
	m := redisEvent{
		"id": id,
	}
	if name != "" {
		m["name"] = name
	}
	if description != "" {
		m["description"] = description
	}
	if metadata != nil {
		if b, err := json.Marshal(metadata); err == nil {
			m["metadata"] = string(b)
		}
	}
	return m
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

// OrgCreated signals the creation of an organization.
type OrgCreated struct {
	ID string
}

func (e OrgCreated) Operation() string {
	return OrgCreate
}

func (e OrgCreated) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
	}
}

func decodeOrgCreated(e redisEvent) OrgCreated {
	return OrgCreated{ID: e.field("id", "")}
}

// OrgUpdated signals an organization update.
type OrgUpdated struct {
	ID          string
	Name        string
	Description string
	Metadata    map[string]any
}

func (e OrgUpdated) Operation() string {
	return OrgUpdate
}

func (e OrgUpdated) Encode() map[string]any {
	return encodeDescribed(e.ID, e.Name, e.Description, e.Metadata)
}

func decodeOrgUpdated(e redisEvent) OrgUpdated {
	o := OrgUpdated{
		ID:          e.field("id", ""),
		Name:        e.field("name", ""),
		Description: e.field("description", ""),
	}
	if raw := e.field("metadata", ""); raw != "" {
		_ = json.Unmarshal([]byte(raw), &o.Metadata)
	}
	return o
}

// OrgMembershipCreated signals that a member was added to an organization.
type OrgMembershipCreated struct {
	OrgID    string
	MemberID string
	Role     string
}

func (e OrgMembershipCreated) Operation() string {
	return OrgMembershipCreate
}

func (e OrgMembershipCreated) Encode() map[string]any {
	return redisEvent{
		"org_id":    e.OrgID,
		"member_id": e.MemberID,
		"role":      e.Role,
	}
}

func decodeOrgMembershipCreated(e redisEvent) OrgMembershipCreated {
	return OrgMembershipCreated{
		OrgID:    e.field("org_id", ""),
		MemberID: e.field("member_id", ""),
		Role:     e.field("role", ""),
	}
}

// OrgMembershipUpdated signals that an organization member's role was changed.
type OrgMembershipUpdated struct {
	OrgID    string
	MemberID string
	Role     string
}

func (e OrgMembershipUpdated) Operation() string {
	return OrgMembershipUpdate
}

func (e OrgMembershipUpdated) Encode() map[string]any {
	return redisEvent{
		"org_id":    e.OrgID,
		"member_id": e.MemberID,
		"role":      e.Role,
	}
}

func decodeOrgMembershipUpdated(e redisEvent) OrgMembershipUpdated {
	return OrgMembershipUpdated{
		OrgID:    e.field("org_id", ""),
		MemberID: e.field("member_id", ""),
		Role:     e.field("role", ""),
	}
}

// OrgMembershipRemoved signals that a member was removed from an organization.
type OrgMembershipRemoved struct {
	OrgID    string
	MemberID string
}

func (e OrgMembershipRemoved) Operation() string {
	return OrgMembershipRemove
}

func (e OrgMembershipRemoved) Encode() map[string]any {
	return redisEvent{
		"org_id":    e.OrgID,
		"member_id": e.MemberID,
	}
}

func decodeOrgMembershipRemoved(e redisEvent) OrgMembershipRemoved {
	return OrgMembershipRemoved{
		OrgID:    e.field("org_id", ""),
		MemberID: e.field("member_id", ""),
	}
}

// OrgInviteCreated signals that a user was invited to an organization.
type OrgInviteCreated struct {
	ID           string
	InviteeID    string
	InviteeEmail string
	InviteeRole  string
	GroupInvites []domain.GroupInvite
}

func (e OrgInviteCreated) Operation() string {
	return OrgInviteCreate
}

func (e OrgInviteCreated) Encode() map[string]any {
	m := redisEvent{
		"id":            e.ID,
		"invitee_id":    e.InviteeID,
		"invitee_email": e.InviteeEmail,
		"invitee_role":  e.InviteeRole,
	}
	if len(e.GroupInvites) > 0 {
		if b, err := json.Marshal(e.GroupInvites); err == nil {
			m["group_invites"] = string(b)
		}
	}
	return m
}

func decodeOrgInviteCreated(e redisEvent) OrgInviteCreated {
	oi := OrgInviteCreated{
		ID:           e.field("id", ""),
		InviteeID:    e.field("invitee_id", ""),
		InviteeEmail: e.field("invitee_email", ""),
		InviteeRole:  e.field("invitee_role", ""),
	}
	if raw := e.field("group_invites", ""); raw != "" {
		_ = json.Unmarshal([]byte(raw), &oi.GroupInvites)
	}
	return oi
}

// OrgInviteRevoked signals that an organization invite was revoked by its inviter.
type OrgInviteRevoked struct {
	ID string
}

func (e OrgInviteRevoked) Operation() string {
	return OrgInviteRevoke
}

func (e OrgInviteRevoked) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
	}
}

func decodeOrgInviteRevoked(e redisEvent) OrgInviteRevoked {
	return OrgInviteRevoked{ID: e.field("id", "")}
}

// OrgInviteAccepted signals that an organization invite was accepted by its invitee.
type OrgInviteAccepted struct {
	ID string
}

func (e OrgInviteAccepted) Operation() string {
	return OrgInviteAccept
}

func (e OrgInviteAccepted) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
	}
}

func decodeOrgInviteAccepted(e redisEvent) OrgInviteAccepted {
	return OrgInviteAccepted{ID: e.field("id", "")}
}

// OrgInviteDeclined signals that an organization invite was declined by its invitee.
type OrgInviteDeclined struct {
	ID string
}

func (e OrgInviteDeclined) Operation() string {
	return OrgInviteDecline
}

func (e OrgInviteDeclined) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
	}
}

func decodeOrgInviteDeclined(e redisEvent) OrgInviteDeclined {
	return OrgInviteDeclined{ID: e.field("id", "")}
}

// APIKeyIssued signals that a user issued an API key, without the key secret.
type APIKeyIssued struct {
	ID        string
	ExpiresAt time.Time
}

func (e APIKeyIssued) Operation() string {
	return APIKeyIssue
}

func (e APIKeyIssued) Encode() map[string]any {
	m := redisEvent{
		"id": e.ID,
	}
	if !e.ExpiresAt.IsZero() {
		m["expires_at"] = e.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	return m
}

func decodeAPIKeyIssued(e redisEvent) APIKeyIssued {
	k := APIKeyIssued{ID: e.field("id", "")}
	if raw := e.field("expires_at", ""); raw != "" {
		if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			k.ExpiresAt = t
		}
	}
	return k
}

// APIKeyRevoked signals that a user revoked an API key.
type APIKeyRevoked struct {
	ID string
}

func (e APIKeyRevoked) Operation() string {
	return APIKeyRevoke
}

func (e APIKeyRevoked) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
	}
}

func decodeAPIKeyRevoked(e redisEvent) APIKeyRevoked {
	return APIKeyRevoked{ID: e.field("id", "")}
}

// SessionLoggedOutAll signals that a user ended all of their sessions.
type SessionLoggedOutAll struct{}

func (e SessionLoggedOutAll) Operation() string {
	return SessionLogoutAll
}

func (e SessionLoggedOutAll) Encode() map[string]any {
	return redisEvent{}
}

// OrgRemoved signals that an organization has been removed.
type OrgRemoved struct {
	ID string
}

func (e OrgRemoved) Operation() string {
	return OrgRemove
}

func (e OrgRemoved) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
	}
}

func decodeOrgRemoved(e redisEvent) OrgRemoved {
	return OrgRemoved{ID: e.field("id", "")}
}

// UserImpersonated signals that a platform admin started an impersonated session of a user.
type UserImpersonated struct {
	ID    string
	Email string
}

func (e UserImpersonated) Operation() string {
	return UserImpersonate
}

func (e UserImpersonated) Encode() map[string]any {
	return redisEvent{
		"id":    e.ID,
		"email": e.Email,
	}
}

func decodeUserImpersonated(e redisEvent) UserImpersonated {
	return UserImpersonated{
		ID:    e.field("id", ""),
		Email: e.field("email", ""),
	}
}
