package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	ID           string
	InviteeID    string
	InviteeEmail string
	InviterID    string
	OrgID        string
	InviteeRole  string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

type Invites interface {
	// InviteMembers creates a pending Invite on behalf of the User authenticated by `token`,
	// towards the user in `om`, to join the Org identified by `orgID` with an appropriate role.
	// To support inviting unregistered users, `om.Email` may belong to a person without a registered acount:
	// in that case, the created Invite is considered 'inactive', and becomes a proper invite only once
	// a new user with the that e-mail address is registered.
	InviteMember(ctx context.Context, token string, orgID string, redirectPath string, om OrgMembership) (Invite, error)

	// RevokeInvite revokes a specific pending Invite. An existing pending Invite can only be revoked
	// by its original inviter (creator).
	RevokeInvite(ctx context.Context, token string, inviteID string) error

	// InviteRespond responds to a specific invite, either accepting it (after which the invitee
	// is assigned as a member of the appropriate Org), or declining it. In both cases the existing
	// pending Invite is removed.
	InviteRespond(ctx context.Context, token string, inviteID string, accept bool) error

	// ViewInvite retrieves a single Invite denoted by its ID.
	ViewInvite(ctx context.Context, token string, inviteID string) (Invite, error)

	// ListInvitesByInviteeID retrieves a list of all pending Invites directed towards
	// a particular User, denoted by their User ID.
	ListInvitesByInviteeID(ctx context.Context, token string, userID string, pm apiutil.PageMetadata) (InvitesPage, error)

	// FlipInactiveInvites 'activates' all existing invites towards an unregistered user with the provided email,
	// by setting the invitee_email columns to NULL, and the invitee_id columns to the provided `inviteeID`, which indicates
	// an active Invite. Returns the number of activated Invites.
	FlipInactiveInvites(ctx context.Context, email string, inviteeID string) (uint32, error)

	// SendOrgInviteEmail sends an e-mail representing a certain Invite to a corresponding end User.
	SendOrgInviteEmail(ctx context.Context, invite Invite, orgName string, redirectPath string) error
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

	FlipInactiveInvites(ctx context.Context, email string, inviteeID string) (uint32, error)
}

func (svc service) InviteMember(ctx context.Context, token string, orgID string, redirectPath string, om OrgMembership) (Invite, error) {
	// Check if currently authenticated User has "admin" privileges within Org (required to make invitations)
	if err := svc.canAccessOrg(ctx, token, orgID, Admin); err != nil {
		return Invite{}, err
	}

	// Get userID of inviter
	inviterIdentity, err := svc.identify(ctx, token)
	if err != nil {
		return Invite{}, err
	}

	inviterUserID := inviterIdentity.ID

	org, err := svc.ViewOrg(ctx, token, orgID)
	if err != nil {
		return Invite{}, err
	}

	// To support inviting unregistered users, when preserving an Invite we either provide
	// an e-mail and no ID (invite towards unregistered user) or _no_ email and a valid ID
	// (invite towards registered user).
	var inviteeID string
	var inviteeEmail string

	muReq := protomfx.UsersByEmailsReq{Emails: []string{om.Email}}
	users, err := svc.users.GetUsersByEmails(ctx, &muReq)

	if err != nil {
		// Error getting user based on e-mail: either the user is not registered, or some other error occurred
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.NotFound:
				// User is not registered, set inviteeEmail to indicate that we're inviting
				// an unregistered user
				inviteeEmail = om.Email
			default:
				// Some other gRPC error
				return Invite{}, err
			}
		} else {
			return Invite{}, err
		}
	} else {
		// No error - user is registered and we can obtain their User ID
		inviteeID = users.Users[0].Id
	}

	if inviteeID != "" {
		_, err := svc.ViewOrgMembership(ctx, token, orgID, inviteeID)

		if err == nil {
			return Invite{}, ErrOrgMembershipExists
		}

		if err != errors.ErrNotFound {
			return Invite{}, err
		}
	}

	createdAt := getTimestmap()
	inviteID, err := svc.idProvider.ID()
	if err != nil {
		return Invite{}, err
	}

	invite := Invite{
		ID:           inviteID,
		InviteeID:    inviteeID,
		InviteeEmail: inviteeEmail,
		InviterID:    inviterUserID,
		OrgID:        orgID,
		InviteeRole:  om.Role,
		CreatedAt:    createdAt,
		ExpiresAt:    createdAt.Add(svc.inviteDuration),
	}

	if err := svc.invites.Save(ctx, invite); err != nil {
		return Invite{}, err
	}

	svc.SendOrgInviteEmail(ctx, invite, org.Name, redirectPath)

	return invite, nil
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

func (svc service) ViewInvite(ctx context.Context, token string, inviteID string) (Invite, error) {
	invite, err := svc.invites.RetrieveByID(ctx, inviteID)
	if err != nil {
		return Invite{}, err
	}

	// A specific Invite can only be retrieved by the platform Root Admin, the Invitee towards who
	// the Invite is directed, or the original Inviter (sender)
	if err := svc.isAdmin(ctx, token); err != nil {
		if err != errors.ErrAuthorization {
			return Invite{}, err
		}

		// Current User is not Root Admin - must be the Invitee
		currentUser, err := svc.identify(ctx, token)
		if err != nil {
			return Invite{}, err
		}

		if currentUser.ID != invite.InviteeID && currentUser.ID != invite.InviterID {
			return Invite{}, errors.ErrAuthorization
		}
	}

	return invite, nil
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

		newOrgMember := OrgMembership{
			MemberID:  currentUserID,
			OrgID:     invite.OrgID,
			Role:      invite.InviteeRole,
			CreatedAt: ts,
			UpdatedAt: ts,
		}

		if err := svc.memberships.Save(ctx, newOrgMember); err != nil {
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

func (svc service) FlipInactiveInvites(ctx context.Context, email string, inviteeID string) (uint32, error) {
	cnt, err := svc.invites.FlipInactiveInvites(ctx, email, inviteeID)
	if err != nil {
		return 0, err
	}

	return cnt, nil
}

func (svc service) SendOrgInviteEmail(ctx context.Context, invite Invite, orgName string, redirectPath string) error {
	to := []string{invite.InviteeEmail}
	return svc.email.SendOrgInvite(to, invite.ID, orgName, invite.InviteeRole, redirectPath)
}
