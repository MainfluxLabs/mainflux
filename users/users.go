// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"


	"github.com/MainfluxLabs/mainflux/pkg/email"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

// Metadata to be used for Mainflux thing or channel for customized
// describing of particular thing or channel.
type Metadata map[string]interface{}

// User represents a Mainflux user account. Each user is identified given its
// email and password.
type User struct {
	ID       string
	Email    string
	Password string
	Metadata Metadata
	Status   string
	Role     string
}

// Validate returns an error if user representation is invalid.
func (u User) Validate() error {
	if !email.IsEmail(u.Email) {
		return errors.ErrMalformedEntity
	}

	if u.Password == "" {
		return errors.ErrMalformedEntity
	}

	return nil
}

// UserRepository specifies an account persistence API.
type UserRepository interface {
	// Save persists the user account. A non-nil error is returned to indicate
	// operation failure.
	Save(ctx context.Context, u User) (string, error)

	// UpdateUser updates the user metadata.
	UpdateUser(ctx context.Context, u User) error

	// RetrieveByEmail retrieves user by its unique identifier (i.e. email).
	RetrieveByEmail(ctx context.Context, email string) (User, error)

	// RetrieveByID retrieves user by its unique identifier ID.
	RetrieveByID(ctx context.Context, id string) (User, error)

	// RetrieveByIDs retrieves all users for given array of userIDs.
	RetrieveByIDs(ctx context.Context, userIDs []string, pm PageMetadata) (UserPage, error)

	// UpdatePassword updates password for user with given email
	UpdatePassword(ctx context.Context, email, password string) error

	// ChangeStatus changes users status to enabled or disabled
	ChangeStatus(ctx context.Context, id, status string) error

	// RetrieveAll retrieves all users.
	RetrieveAll(ctx context.Context) ([]User, error)
}
