// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"time"
)

// User represents user account information.
type User struct {
	ID       string   `json:"id,omitempty"`
	Email    string   `json:"email,omitempty"`
	Password string   `json:"password,omitempty"`
	Metadata Metadata `json:"metadata,omitempty"`
	Status   string   `json:"status,omitempty"`
	Role     string   `json:"role,omitempty"`
}

// UsersPage contains page metadata and list of users.
type UsersPage struct {
	Total uint64 `json:"total"`
	Users []User `json:"users"`
}

// Status key constants for user filtering.
const (
	EnabledStatusKey  = "enabled"
	DisabledStatusKey = "disabled"
	AllStatusKey      = "all"
)

// UserPageMetadata contains page metadata for list operations.
type UserPageMetadata struct {
	Total    uint64   `json:"total,omitempty"`
	Offset   uint64   `json:"offset,omitempty"`
	Limit    uint64   `json:"limit,omitempty"`
	Email    string   `json:"email,omitempty"`
	Order    string   `json:"order,omitempty"`
	Dir      string   `json:"dir,omitempty"`
	Status   string   `json:"status,omitempty"`
	State    string   `json:"state,omitempty"`
	Metadata Metadata `json:"metadata,omitempty"`
}

// PlatformInvite represents platform invite information.
type PlatformInvite struct {
	ID           string     `json:"id,omitempty"`
	InviteeEmail string     `json:"invitee_email,omitempty"`
	CreatedAt    time.Time  `json:"created_at,omitempty"`
	ExpiresAt    time.Time  `json:"expires_at,omitempty"`
	State        string     `json:"state,omitempty"`
	OrgInvite    *OrgInvite `json:"org_invite"`
}

// PlatformInvitesPage contains page metadata and list of platform invites.
type PlatformInvitesPage struct {
	Total   uint64           `json:"total"`
	Invites []PlatformInvite `json:"invites"`
}
