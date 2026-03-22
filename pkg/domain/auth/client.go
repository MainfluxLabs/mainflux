// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

import "context"

// Client specifies the interface that auth gRPC client implementations must fulfill.
// All methods use domain types rather than protobuf-generated types.
type Client interface {
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
