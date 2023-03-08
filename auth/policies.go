// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
)

// AuthReq represents an argument struct for making an authz related
// function calls.
type AuthzReq struct {
	Email string
}

// Authz represents a authorization service. It exposes
// functionalities through `auth` to perform authorization.
type Authz interface {
	// Authorize checks authorization of the given `subject`. Basically,
	// Authorize verifies that Is `subject` allowed to `relation` on
	// `object`. Authorize returns a non-nil error if the subject has
	// no relation on the object (which simply means the operation is
	// denied).
	Authorize(ctx context.Context, pr AuthzReq) error

	// CanAccessGroup indicates if user can access group for a given token.
	CanAccessGroup(ctx context.Context, token, groupID string) error
}
