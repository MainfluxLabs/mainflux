// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
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

// Status key constants for user filtering.
const (
	EnabledStatusKey  = "enabled"
	DisabledStatusKey = "disabled"
	AllStatusKey      = "all"
)

// AllowedOrders defines the valid order-by fields for user list queries.
var AllowedOrders = map[string]string{
	"id":            "id",
	"email":         "email",
	"invitee_email": "invitee_email",
	"state":         "state",
	"created_at":    "created_at",
}

// PageMetadata contains page metadata for list operations.
type PageMetadata struct {
	Total    uint64          `json:"total,omitempty"`
	Offset   uint64          `json:"offset,omitempty"`
	Limit    uint64          `json:"limit,omitempty"`
	Email    string          `json:"email,omitempty"`
	Order    string          `json:"order,omitempty"`
	Dir      string          `json:"dir,omitempty"`
	Status   string          `json:"status,omitempty"`
	State    string          `json:"state,omitempty"`
	Metadata domain.Metadata `json:"metadata,omitempty"`
}

// Validate validates the page metadata.
func (pm PageMetadata) Validate(maxLimitSize, maxEmailSize int) error {
	if len(pm.Email) > maxEmailSize {
		return apiutil.ErrEmailSize
	}

	if pm.Status != "" {
		if pm.Status != AllStatusKey &&
			pm.Status != EnabledStatusKey &&
			pm.Status != DisabledStatusKey {
			return apiutil.ErrInvalidStatus
		}
	}

	common := apiutil.PageMetadata{Offset: pm.Offset, Limit: pm.Limit, Order: pm.Order, Dir: pm.Dir}
	return common.Validate(maxLimitSize, AllowedOrders)
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
