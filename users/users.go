// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"regexp"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	domainusers "github.com/MainfluxLabs/mainflux/pkg/domain/users"
	"github.com/MainfluxLabs/mainflux/pkg/email"
)

// Metadata is an alias for the shared domain type.
type Metadata = domain.Metadata

// User is an alias for the shared domain type.
type User = domainusers.User

type EmailVerification struct {
	User      User
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type OAuthLoginData struct {
	State       string
	Verifier    string
	RedirectURL string
}

type OAuthCallbackData struct {
	Provider string
	Code     string
	Verifier string
}

type Identity struct {
	UserID         string
	Provider       string
	ProviderUserID string
}

// ValidateUser returns an error if user representation is invalid.
func ValidateUser(u User, passRegex *regexp.Regexp) error {
	if !email.IsEmail(u.Email) {
		return apiutil.ErrMalformedEntity
	}

	if !passRegex.MatchString(u.Password) {
		return ErrPasswordFormat
	}

	return nil
}

type EmailVerificationRepository interface {
	// Save persists the EmailVerification.
	Save(ctx context.Context, verification EmailVerification) (string, error)

	// RetrieveByToken retrieves an EmailVerification based on its token.
	RetrieveByToken(ctx context.Context, token string) (EmailVerification, error)

	// Remove removes an EmailVerification from the database.
	Remove(ctx context.Context, token string) error
}

// UserRepository specifies an account persistence API.
type UserRepository interface {
	// Save persists the user account. A non-nil error is returned to indicate
	// operation failure.
	Save(ctx context.Context, u User) (string, error)

	// Update updates the user.
	Update(ctx context.Context, u User) error

	// UpdateUserMetadata updates the user metadata.
	UpdateUserMetadata(ctx context.Context, u User) error

	// RetrieveByEmail retrieves user by its unique identifier (i.e. email).
	RetrieveByEmail(ctx context.Context, email string) (User, error)

	// RetrieveByID retrieves user by its unique identifier ID.
	RetrieveByID(ctx context.Context, id string) (User, error)

	// RetrieveByIDs retrieves all users for given array of userIDs.
	RetrieveByIDs(ctx context.Context, userIDs []string, pm PageMetadata) (UsersPage, error)

	// UpdatePassword updates password for user with given email
	UpdatePassword(ctx context.Context, email, password string) error

	// ChangeStatus changes users status to enabled or disabled
	ChangeStatus(ctx context.Context, id, status string) error

	// BackupAll retrieves all users.
	BackupAll(ctx context.Context) ([]User, error)
}

type IdentityRepository interface {
	// Save persists an OAuth identity.
	Save(ctx context.Context, identity Identity) error

	// Retrieve fetches an OAuth identity by provider and provider user ID.
	Retrieve(ctx context.Context, provider, providerUserID string) (Identity, error)

	// BackupAll retrieves all OAuth identities.
	BackupAll(ctx context.Context) ([]Identity, error)
}
