// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"encoding/json"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

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

func decodeSessionLoggedOutAll(_ redisEvent) SessionLoggedOutAll {
	return SessionLoggedOutAll{}
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
