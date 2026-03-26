// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"time"
)

// Identity contains ID and Email.
type Identity struct {
	ID    string `json:"id,omitempty"`
	Email string `json:"email,omitempty"`
}

// AuthzReq represents an argument struct for authorization requests.
type AuthzReq struct {
	Token   string
	Object  string
	Subject string
	Action  string
}

// OrgsPage contains page metadata and list of orgs.
type OrgsPage struct {
	Total uint64 `json:"total"`
	Orgs  []Org  `json:"orgs"`
}

// Org represents org information.
type Org struct {
	ID          string    `json:"id,omitempty"`
	OwnerID     string    `json:"owner_id,omitempty"`
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	Metadata    Metadata  `json:"metadata,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

// OrgInvite represents org invite information.
type OrgInvite struct {
	ID           string        `json:"id,omitempty"`
	OrgID        string        `json:"org_id,omitempty"`
	OrgName      string        `json:"org_name,omitempty"`
	InviteeRole  string        `json:"invitee_role,omitempty"`
	GroupInvites []GroupInvite `json:"group_invites,omitempty"`
	InviteeID    string        `json:"invitee_id,omitempty"`
	InviteeEmail string        `json:"invitee_email,omitempty"`
	InviterID    string        `json:"inviter_id,omitempty"`
	InviterEmail string        `json:"inviter_email,omitempty"`
	CreatedAt    time.Time     `json:"created_at,omitempty"`
	ExpiresAt    time.Time     `json:"expires_at,omitempty"`
	State        string        `json:"state,omitempty"`
}

// OrgInvitesPage contains page metadata and list of org invites.
type OrgInvitesPage struct {
	Invites []OrgInvite `json:"invites"`
	Total   uint64      `json:"total"`
}

// OrgMembership represents org membership information.
type OrgMembership struct {
	MemberID  string    `json:"member_id,omitempty"`
	OrgID     string    `json:"org_id,omitempty"`
	Role      string    `json:"role,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	Email     string    `json:"email,omitempty"`
}

// OrgMembershipsPage contains page metadata and list of org memberships.
type OrgMembershipsPage struct {
	Total          uint64          `json:"total"`
	OrgMemberships []OrgMembership `json:"org_memberships"`
}

// GroupInvite represents a group membership invite.
type GroupInvite struct {
	GroupID    string `json:"group_id,omitempty"`
	MemberRole string `json:"member_role,omitempty"`
}

// Subject type constants for authorization.
const (
	RootSub = "root"
	OrgSub  = "org"
)

// Org role constants.
const (
	OrgAdmin  = "admin"
	OrgOwner  = "owner"
	OrgEditor = "editor"
	OrgViewer = "viewer"
)

// Platform role constants.
const (
	// RoleRootAdmin is the super admin role.
	RoleRootAdmin = "root"
	// RoleAdmin is the admin role.
	RoleAdmin = "admin"
)

// Key type constants.
const (
	LoginKey uint32 = iota
	RecoveryKey
	APIKey
)

// Key represents API key.
type Key struct {
	ID        string    `json:"id,omitempty"`
	Type      uint32    `json:"type,omitempty"`
	IssuerID  string    `json:"issuer_id,omitempty"`
	Subject   string    `json:"subject,omitempty"`
	IssuedAt  time.Time `json:"issued_at,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// Expired reports whether the key is expired.
func (k Key) Expired() bool {
	if k.Type == APIKey && k.ExpiresAt.IsZero() {
		return false
	}
	return k.ExpiresAt.UTC().Before(time.Now().UTC())
}

// KeysPage contains page metadata and list of keys.
type KeysPage struct {
	Total uint64
	Keys  []Key
}

type AuthClient interface {
	// Issue issues a token for the given id, email and key type.
	Issue(ctx context.Context, id, email string, keyType uint32) (string, error)

	// Identify validates the token and returns the identity.
	Identify(ctx context.Context, token string) (Identity, error)

	// Authorize checks if the subject is authorized to perform the action on the object.
	Authorize(ctx context.Context, ar AuthzReq) error

	// GetOwnerIDByOrg returns the owner ID of the organization.
	GetOwnerIDByOrg(ctx context.Context, orgID string) (string, error)

	// AssignRole assigns a role to a user.
	AssignRole(ctx context.Context, id, role string) error

	// RetrieveRole retrieves the role for a user.
	RetrieveRole(ctx context.Context, id string) (string, error)

	// CreateDormantOrgInvite creates a dormant org invite.
	CreateDormantOrgInvite(ctx context.Context, token, orgID, inviteeRole, platformInviteID string, groupInvites []GroupInvite) error

	// ActivateOrgInvite activates an org invite.
	ActivateOrgInvite(ctx context.Context, platformInviteID, userID, redirectPath string) error

	// GetDormantOrgInviteByPlatformInvite retrieves a dormant org invite by platform invite ID.
	GetDormantOrgInviteByPlatformInvite(ctx context.Context, platformInviteID string) (OrgInvite, error)

	// ViewOrg retrieves organization details.
	ViewOrg(ctx context.Context, token, orgID string) (Org, error)
}
