package auth

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var (
	// ErrCreateInvite indicates failure to create a new Invite
	ErrCreateInvite = errors.New("error creating invite")

	// ErrUserAlreadyInvited indicates that the invitee already has a pending invitation to join the same Org
	ErrUserAlreadyInvited = errors.New("user already has pending invite to org")

	// ErrInviteExpired indicates that the Invite has expired and cannot be responded to
	ErrInviteExpired = errors.New("invite has expired")
)

type InvitesPage struct {
	Invites []Invite
	apiutil.PageMetadata
}

type Invite struct {
	ID          string
	InviteeID   string
	InviterID   string
	OrgID       string
	InviteeRole string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

type Invites interface {
	// InviteMembers creates pending invitations on behalf of the User authenticated by `token`,
	// towards all members in `oms`, to join the Org identified by `orgID` with an appropriate role.
	InviteMembers(ctx context.Context, token string, orgID string, oms ...OrgMember) error

	// RevokeInvite revokes a specific pending Invite. An existing pending Invite can only be revoked
	// by its original inviter (creator).
	RevokeInvite(ctx context.Context, token string, inviteID string) error

	// InviteRespond responds to a specific invite, either accepting it (after which the invitee
	// is assigned as a member of the appropriate Org), or declining it. In both cases the existing
	// pending Invite is removed.
	InviteRespond(ctx context.Context, token string, inviteID string, accept bool) error

	// ListInvitesByInviteeID retrieves a list of all pending Invites directed towards
	// a particular User, denoted by their User ID.
	ListInvitesByInviteeID(ctx context.Context, token string, userID string, pm apiutil.PageMetadata) (InvitesPage, error)
}

type InvitesRepository interface {
	// Save saves one or more pending invites to the repository.
	Save(ctx context.Context, invites ...Invite) error

	// RetrieveByID retrieves a specific Invite by its ID.
	RetrieveByID(ctx context.Context, inviteID string) (Invite, error)

	// Remove removes a specific pending Invite
	Remove(ctx context.Context, inviteID string) error

	// RetrieveByInviteeID retrieves a list of all pending invites directed towards a particular User,
	// denoted by their User ID.
	RetrieveByInviteeID(ctx context.Context, inviteeID string, pm apiutil.PageMetadata) (InvitesPage, error)
}

func (svc service) InviteMembers(ctx context.Context, token string, orgID string, oms ...OrgMember) error {
	// Check if currently authenticated User has "admin" privileges within Org (required to make invitations)
	if err := svc.canAccessOrg(ctx, token, orgID, Admin); err != nil {
		return err
	}

	// Get userID of inviter
	inviterIdentity, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}

	inviterUserID := inviterIdentity.ID

	// Obtain user IDs of all invited members
	var memberEmails []string
	for _, orgMember := range oms {
		memberEmails = append(memberEmails, orgMember.Email)
	}

	muReq := protomfx.UsersByEmailsReq{Emails: memberEmails}
	users, err := svc.users.GetUsersByEmails(ctx, &muReq)
	if err != nil {
		return err
	}

	// Map user emails to user IDs
	userEmailID := map[string]string{}
	for _, user := range users.Users {
		userEmailID[user.Email] = user.Id
	}

	// Build slice of Invites to save
	invites := make([]Invite, 0, len(oms))

	createdAt := getTimestmap()

	for _, orgMember := range oms {
		inviteId, err := svc.idProvider.ID()
		if err != nil {
			return err
		}

		invite := Invite{
			ID:          inviteId,
			InviteeID:   userEmailID[orgMember.Email],
			InviterID:   inviterUserID,
			OrgID:       orgID,
			InviteeRole: orgMember.Role,
			CreatedAt:   createdAt,
			ExpiresAt:   createdAt.Add(7 * 24 * time.Hour),
		}

		invites = append(invites, invite)
	}

	if err := svc.invites.Save(ctx, invites...); err != nil {
		return err
	}

	return nil
}

func (svc service) RevokeInvite(ctx context.Context, token string, inviteID string) error {
	// Identify User attempting to revoke invite
	currentUser, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}

	currentUserID := currentUser.ID

	// Obtain full invite from db based on inviteID
	invite, err := svc.invites.RetrieveByID(ctx, inviteID)
	if err != nil {
		return err
	}

	// An Invite can only be revoked by its issuer
	if invite.InviterID != currentUserID {
		return errors.ErrAuthorization
	}

	if err := svc.invites.Remove(ctx, inviteID); err != nil {
		return err
	}

	return nil
}

func (svc service) InviteRespond(ctx context.Context, token string, inviteID string, accept bool) error {
	// Identify User attempting to respond to Invite
	currentUser, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}

	currentUserID := currentUser.ID

	// Obtain detailed information about the Invite
	invite, err := svc.invites.RetrieveByID(ctx, inviteID)
	if err != nil {
		return err
	}

	// An Invite can only be responded to by the invitee
	if currentUserID != invite.InviteeID {
		return errors.ErrAuthorization
	}

	// Make sure the Invite hasn't expired
	if time.Now().After(invite.ExpiresAt) {
		// If a response to an expired Invite has been attempted, remove it from the database
		if err := svc.invites.Remove(ctx, inviteID); err != nil {
			return err
		}

		return ErrInviteExpired
	}

	if accept {
		// User has accepted the Invite, assign them as a member of the appropriate Org
		// with the appropriate role
		ts := getTimestmap()

		newOrgMember := OrgMember{
			MemberID:  currentUserID,
			OrgID:     invite.OrgID,
			Role:      invite.InviteeRole,
			CreatedAt: ts,
			UpdatedAt: ts,
		}

		if err := svc.members.Save(ctx, newOrgMember); err != nil {
			return err
		}
	}

	// Remove Invite from database
	if err := svc.invites.Remove(ctx, inviteID); err != nil {
		return err
	}

	return nil
}

func (svc service) ListInvitesByInviteeID(ctx context.Context, token string, userID string, pm apiutil.PageMetadata) (InvitesPage, error) {
	// A specific User's list of pending Invites can only be retrieved by the platform Root Admin
	// or by that specific User themselves:
	if err := svc.isAdmin(ctx, token); err != nil {
		if err != errors.ErrAuthorization {
			return InvitesPage{}, err
		}

		// Current User is not Root Admin - must be the User whose Invites are being requested
		currentUser, err := svc.identify(ctx, token)
		if err != nil {
			return InvitesPage{}, err
		}

		if currentUser.ID != userID {
			return InvitesPage{}, errors.ErrAuthorization
		}
	}

	invitesPage, err := svc.invites.RetrieveByInviteeID(ctx, userID, pm)
	if err != nil {
		return InvitesPage{}, err
	}

	return invitesPage, nil
}
