package auth

import "context"

// Client represents gRPC client for Auth service that uses domain types.
type Client interface {
	Issue(ctx context.Context, id, email string, keyType uint32) (string, error)
	Identify(ctx context.Context, token string) (Identity, error)
	Authorize(ctx context.Context, req AuthzReq) error
	GetOwnerIDByOrg(ctx context.Context, orgID string) (string, error)
	RetrieveRole(ctx context.Context, id string) (string, error)
	AssignRole(ctx context.Context, id, role string) error
	CreateDormantOrgInvite(ctx context.Context, token, orgID, inviteeRole string, groupInvites []GroupInvite, platformInviteID string) error
	ActivateOrgInvite(ctx context.Context, platformInviteID, userID, redirectPath string) error
	GetDormantOrgInviteByPlatformInvite(ctx context.Context, platformInviteID string) (OrgInvite, error)
	ViewOrg(ctx context.Context, token, orgID string) (Org, error)
}
