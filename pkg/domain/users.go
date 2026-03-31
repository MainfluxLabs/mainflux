// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"regexp"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/email"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

// Status key constants for user filtering.
const (
	EnabledStatusKey  = "enabled"
	DisabledStatusKey = "disabled"
	AllStatusKey      = "all"
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

// Validate returns an error if user representation is invalid.
func (u User) Validate(passRegex *regexp.Regexp) error {
	if !email.IsEmail(u.Email) {
		return errors.ErrMalformedEntity
	}

	if !passRegex.MatchString(u.Password) {
		return errors.ErrPasswordFormat
	}

	return nil
}

// UsersPage contains page metadata and list of users.
type UsersPage struct {
	Total uint64 `json:"total"`
	Users []User `json:"users"`
}

// PageMetadata contains page metadata that helps navigation.
type UsersPageMetadata struct {
	Total    uint64   `json:"total,omitempty"`
	Offset   uint64   `json:"offset,omitempty"`
	Limit    uint64   `json:"limit,omitempty"`
	Order    string   `json:"order,omitempty"`
	Dir      string   `json:"dir,omitempty"`
	Email    string   `json:"email,omitempty"`
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

type UsersClient interface {
	// GetUsersByIDs retrieves users by their IDs with optional pagination.
	GetUsersByIDs(ctx context.Context, ids []string, pm UsersPageMetadata) (UsersPage, error)

	// GetUsersByEmails retrieves users by their email addresses.
	GetUsersByEmails(ctx context.Context, emails []string) ([]User, error)
}
