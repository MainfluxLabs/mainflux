// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

import "context"

// AuthServiceClient specifies the subset of the auth service API exposed over gRPC.
// All methods use plain domain types so callers have no dependency on protobuf.
type AuthServiceClient interface {
	// Issue issues a new key of the given type for the user identified by id/email
	// and returns the token string.
	Issue(ctx context.Context, id, email string, keyType uint32) (string, error)

	// Identify validates a token and returns the corresponding Identity.
	Identify(ctx context.Context, token string) (Identity, error)

	// Authorize checks whether the action described by ar is permitted.
	Authorize(ctx context.Context, ar AuthzReq) error

	// GetOwnerIDByOrg returns the owner ID of the given org.
	GetOwnerIDByOrg(ctx context.Context, orgID string) (string, error)

	// AssignRole assigns a role to the user identified by id.
	AssignRole(ctx context.Context, id, role string) error

	// RetrieveRole returns the role of the user identified by id.
	RetrieveRole(ctx context.Context, id string) (string, error)

	// CreateDormantOrgInvite creates a dormant org invite tied to the given
	// platform invite ID.
	CreateDormantOrgInvite(ctx context.Context, token string, oi OrgInvite, platformInviteID string) error

	// ActivateOrgInvite activates all dormant org invites tied to the given
	// platform invite ID for the given user.
	ActivateOrgInvite(ctx context.Context, platformInviteID, userID, redirectPath string) error

	// ViewOrg retrieves the org identified by orgID.
	ViewOrg(ctx context.Context, token, orgID string) (Org, error)
}
