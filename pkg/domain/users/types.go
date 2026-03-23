// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	domainauth "github.com/MainfluxLabs/mainflux/pkg/domain/auth"
)

// User represents user account information.
type User struct {
	ID       string          `json:"id,omitempty"`
	Email    string          `json:"email,omitempty"`
	Password string          `json:"password,omitempty"`
	Metadata domain.Metadata `json:"metadata,omitempty"`
	Status   string          `json:"status,omitempty"`
	Role     string          `json:"role,omitempty"`
}

// UsersPage contains page metadata and list of users.
type UsersPage struct {
	Total uint64 `json:"total"`
	Users []User `json:"users"`
}

// PageMetadata contains page metadata for list operations (matches proto PageMetadata).
type PageMetadata struct {
	Total  uint64 `json:"total,omitempty"`
	Offset uint64 `json:"offset,omitempty"`
	Limit  uint64 `json:"limit,omitempty"`
	Email  string `json:"email,omitempty"`
	Order  string `json:"order,omitempty"`
	Dir    string `json:"dir,omitempty"`
}

// PlatformInvite represents platform invite information.
type PlatformInvite struct {
	ID           string                `json:"id,omitempty"`
	InviteeEmail string                `json:"invitee_email,omitempty"`
	CreatedAt    time.Time             `json:"created_at,omitempty"`
	ExpiresAt    time.Time             `json:"expires_at,omitempty"`
	State        string                `json:"state,omitempty"`
	OrgInvite    *domainauth.OrgInvite `json:"org_invite"`
}

// PlatformInvitesPage contains page metadata and list of platform invites.
type PlatformInvitesPage struct {
	Total   uint64           `json:"total"`
	Invites []PlatformInvite `json:"invites"`
}
