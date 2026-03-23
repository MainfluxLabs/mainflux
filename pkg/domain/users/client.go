// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import "context"

// Client specifies the interface that users gRPC client implementations must fulfill.
type Client interface {
	// GetUsersByIDs retrieves users by their IDs with optional pagination.
	GetUsersByIDs(ctx context.Context, ids []string, pm PageMetadata) (UsersPage, error)

	// GetUsersByEmails retrieves users by their email addresses.
	GetUsersByEmails(ctx context.Context, emails []string) ([]User, error)
}
