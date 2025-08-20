package auth

import (
	"context"
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

	// ErrInviteExpired indicates that an invite has expired
	ErrInviteExpired = errors.New("invite expired")

	// ErrInviteExpired indicates that an invite is in an invalid state for a certain action to be performed on it
	ErrInvalidInviteState = errors.New("invalid invite state")

	// ErrUserAlreadyInvited indicates that the invitee already has a pending invitation to join the same Org
	ErrUserAlreadyInvited = errors.New("user already has pending invite to org")
)

type OrgInvite struct {
	ID          string
	InviteeID   string
	InviterID   string
	OrgID       string
	InviteeRole string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	State       string
}

type OrgInvitesPage struct {
	Invites []OrgInvite
	apiutil.PageMetadata
}

type PlatformInvite struct {
	ID           string
	InviteeEmail string
	CreatedAt    time.Time
	ExpiresAt    time.Time
	State        string
}

type PlatformInvitesPage struct {
	Invites []PlatformInvite
	apiutil.PageMetadata
}

const (
	UserTypeInvitee = "invitee"
	UserTypeInviter = "inviter"

	InviteStatePending  = "pending"
	InviteStateExpired  = "expired"
	InviteStateRevoked  = "revoked"
	InviteStateAccepted = "accepted"
	InviteStateDeclined = "declined"
)

type Invites interface {
	// InviteOrgMember creates a pending Invite on behalf of the User authenticated by `token`,
	// towards the user in `om`, to join the Org identified by `orgID` with an appropriate role.
	InviteOrgMember(ctx context.Context, token string, orgID string, invRedirectPath string, om OrgMembership) (OrgInvite, error)

	// RevokeOrgInvite revokes a specific pending Invite. An existing pending Invite can only be revoked
	// by its original inviter (creator).
	RevokeOrgInvite(ctx context.Context, token string, inviteID string) error

	// RespondOrgInvite responds to a specific invite, either accepting it (after which the invitee
	// is assigned as a member of the appropriate Org), or declining it. An Invite can only be responded
	// to by the invitee that it's directed towards.
	RespondOrgInvite(ctx context.Context, token string, inviteID string, accept bool) error

	// ViewOrgInvite retrieves a single Invite denoted by its ID. A specific Invite can only be retrieved
	// by its issuer, i.e. inviter, or the invitee towards who it is directed, as well as the platform
	// Root Admin.
	ViewOrgInvite(ctx context.Context, token string, inviteID string) (OrgInvite, error)

	// ListOrgInvitesByUser retrieves a list of invites either directed towards a specific Invitee,
	// or sent out by a specific Inviter, depending on the value of the `userType` argument, which
	// must be either 'invitee' or 'inviter'.
	ListOrgInvitesByUser(ctx context.Context, token string, userType string, userID string, pm apiutil.PageMetadata) (OrgInvitesPage, error)

	// SendOrgInviteEmail sends an e-mail notifying the invitee of the corresponding Invite.
	SendOrgInviteEmail(ctx context.Context, invite OrgInvite, email string, orgName string, invRedirectPath string) error

	// InvitePlatformMember creates a pending platform Invite for the appropriate email address.
	// Only usable by the platform Root Admin.
	InvitePlatformMember(ctx context.Context, token string, redirectPath string, email string) (PlatformInvite, error)

	// RevokePlatformInvite revokes a specific pending PlatformInvite. Only usable by the platform Root Admin.
	RevokePlatformInvite(ctx context.Context, token string, inviteID string) error

	// ViewPlatformInvite retrieves a single PlatformInvite denoted by its ID. Only usable by the platform Root Admin.
	ViewPlatformInvite(ctx context.Context, token string, inviteID string) (PlatformInvite, error)

	// ListPlatformInvites retrieves a list of platform invites. Only usable by the platform Root Admin.
	ListPlatformInvites(ctx context.Context, token string, pm apiutil.PageMetadata) (PlatformInvitesPage, error)

	// ValidatePlatformInvite checks if there exists a valid, pending, non-expired platform invite in the database that matches
	// the passed ID and user e-mail. If so, it marks that invite's state as 'accepted', and returns nil.
	// If no such valid platform invite is found in the database, it instead returns errors.ErrNotFound.
	ValidatePlatformInvite(ctx context.Context, inviteID string, email string) error

	// SendPlatformInviteEmail sends an e-mail notifying the invitee about the corresponding platform invite.
	SendPlatformInviteEmail(ctx context.Context, invite PlatformInvite, redirectPath string) error
}

type InvitesRepository interface {
	// SaveOrgInvite saves one or more pending org invites to the repository.
	SaveOrgInvite(ctx context.Context, invites ...OrgInvite) error

	// RetrieveOrgInviteByID retrieves a specific OrgInvite by its ID.
	RetrieveOrgInviteByID(ctx context.Context, inviteID string) (OrgInvite, error)

	// RemoveOrgInvite removes a specific pending OrgInvite.
	RemoveOrgInvite(ctx context.Context, inviteID string) error

	// RetrieveOrgInviteByUserID retrieves a list of invites either directed towards a specific Invitee, or sent out by a
	// specific Inviter, depending on the value of the `userType` argument, which must be either 'invitee' or 'inviter'.
	RetrieveOrgInvitesByUserID(ctx context.Context, userType string, userID string, pm apiutil.PageMetadata) (OrgInvitesPage, error)

	// UpdateOrgInviteState updates the state of a specific Invite denoted by its ID.
	UpdateOrgInviteState(ctx context.Context, inviteID string, state string) error

	// SavePlatformInvite saves one or more pending platform invites to the repository.
	SavePlatformInvite(ctx context.Context, invites ...PlatformInvite) error

	// RetrievePlatformInviteByID retrieves a single platform invite by its ID.
	RetrievePlatformInviteByID(ctx context.Context, inviteID string) (PlatformInvite, error)

	// RetrievePlatformInvites retrieves a list of platform invites.
	RetrievePlatformInvites(ctx context.Context, pm apiutil.PageMetadata) (PlatformInvitesPage, error)

	// UpdatePlatformInviteState updates the state of a specific platform invite denoted by its ID.
	UpdatePlatformInviteState(ctx context.Context, inviteID string, state string) error
}

func (svc service) InviteOrgMember(ctx context.Context, token string, orgID string, invRedirectPath string, om OrgMembership) (OrgInvite, error) {
	// Check if currently authenticated User has "admin" or higher privileges within Org (required to make invitations)
	if err := svc.canAccessOrg(ctx, token, orgID, Admin); err != nil {
		return OrgInvite{}, err
	}

	// Get userID of inviter
	inviterIdentity, err := svc.identify(ctx, token)
	if err != nil {
		return OrgInvite{}, err
	}

	org, err := svc.ViewOrg(ctx, token, orgID)
	if err != nil {
		return OrgInvite{}, err
	}

	muReq := protomfx.UsersByEmailsReq{Emails: []string{om.Email}}
	users, err := svc.users.GetUsersByEmails(ctx, &muReq)

	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.NotFound:
				return OrgInvite{}, errors.ErrNotFound
			default:
				return OrgInvite{}, err
			}
		}

		return OrgInvite{}, err
	}

	inviteeID := users.Users[0].Id

	createdAt := getTimestmap()
	inviteID, err := svc.idProvider.ID()
	if err != nil {
		return OrgInvite{}, err
	}

	invite := OrgInvite{
		ID:          inviteID,
		InviteeID:   inviteeID,
		InviterID:   inviterIdentity.ID,
		OrgID:       orgID,
		InviteeRole: om.Role,
		CreatedAt:   createdAt,
		ExpiresAt:   createdAt.Add(svc.inviteDuration),
		State:       InviteStatePending,
	}

	if err := svc.invites.SaveOrgInvite(ctx, invite); err != nil {
		return OrgInvite{}, err
	}

	svc.SendOrgInviteEmail(ctx, invite, om.Email, org.Name, invRedirectPath)

	return invite, nil
}

func (svc service) RevokeOrgInvite(ctx context.Context, token string, inviteID string) error {
	// Identify User attempting to revoke invite
	currentUser, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}

	// Obtain full invite from db based on inviteID
	invite, err := svc.invites.RetrieveOrgInviteByID(ctx, inviteID)
	if err != nil {
		return err
	}

	// An Invite can only be revoked by its issuer
	if invite.InviterID != currentUser.ID {
		return errors.ErrAuthorization
	}

	if invite.State != InviteStatePending {
		if invite.State == InviteStateExpired {
			return ErrInviteExpired
		}

		return ErrInvalidInviteState
	}

	if err := svc.invites.UpdateOrgInviteState(ctx, inviteID, InviteStateRevoked); err != nil {
		return err
	}

	return nil
}

func (svc service) ViewOrgInvite(ctx context.Context, token string, inviteID string) (OrgInvite, error) {
	invite, err := svc.invites.RetrieveOrgInviteByID(ctx, inviteID)
	if err != nil {
		return OrgInvite{}, err
	}

	// A specific Invite can only be retrieved by the platform Root Admin, the Invitee towards who
	// the Invite is directed, or the original Inviter (sender)
	if err := svc.isAdmin(ctx, token); err != nil {
		if err != errors.ErrAuthorization {
			return OrgInvite{}, err
		}

		// Current User is not Root Admin - must be either the Invitee or Inviter
		currentUser, err := svc.identify(ctx, token)
		if err != nil {
			return OrgInvite{}, err
		}

		if currentUser.ID != invite.InviteeID && currentUser.ID != invite.InviterID {
			return OrgInvite{}, errors.ErrAuthorization
		}
	}

	return invite, nil
}

func (svc service) RespondOrgInvite(ctx context.Context, token string, inviteID string, accept bool) error {
	// Identify User attempting to respond to Invite
	currentUser, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}

	// Obtain detailed information about the Invite
	invite, err := svc.invites.RetrieveOrgInviteByID(ctx, inviteID)
	if err != nil {
		return err
	}

	if invite.State != "pending" {
		if invite.State == InviteStateExpired {
			return ErrInviteExpired
		}

		return ErrInvalidInviteState
	}

	// An Invite can only be responded to by the invitee
	if currentUser.ID != invite.InviteeID {
		return errors.ErrAuthorization
	}

	newState := InviteStateDeclined

	if accept {
		// User has accepted the Invite, assign them as a member of the appropriate Org
		// with the appropriate role
		newState = InviteStateAccepted
		ts := getTimestmap()

		newOrgMember := OrgMembership{
			MemberID:  currentUser.ID,
			OrgID:     invite.OrgID,
			Role:      invite.InviteeRole,
			CreatedAt: ts,
			UpdatedAt: ts,
		}

		if err := svc.memberships.Save(ctx, newOrgMember); err != nil {
			return err
		}
	}

	if err := svc.invites.UpdateOrgInviteState(ctx, inviteID, newState); err != nil {
		return err
	}

	return nil
}

func (svc service) ListOrgInvitesByUser(ctx context.Context, token string, userType string, userID string, pm apiutil.PageMetadata) (OrgInvitesPage, error) {
	if err := svc.isAdmin(ctx, token); err != nil {
		if err != errors.ErrAuthorization {
			return OrgInvitesPage{}, err
		}

		// Current User is not Root Admin - must be either the Invitee or Inviter
		currentUser, err := svc.identify(ctx, token)
		if err != nil {
			return OrgInvitesPage{}, err
		}

		if currentUser.ID != userID {
			return OrgInvitesPage{}, errors.ErrAuthorization
		}
	}

	invitesPage, err := svc.invites.RetrieveOrgInvitesByUserID(ctx, userType, userID, pm)
	if err != nil {
		return OrgInvitesPage{}, err
	}

	return invitesPage, nil
}

func (svc service) SendOrgInviteEmail(ctx context.Context, invite OrgInvite, email string, orgName string, invRedirectPath string) error {
	to := []string{email}
	return svc.email.SendOrgInvite(to, invite, orgName, invRedirectPath)
}

func (svc service) InvitePlatformMember(ctx context.Context, token string, redirectPath string, email string) (PlatformInvite, error) {
	if err := svc.isAdmin(ctx, token); err != nil {
		return PlatformInvite{}, err
	}

	muReq := protomfx.UsersByEmailsReq{Emails: []string{email}}
	_, err := svc.users.GetUsersByEmails(ctx, &muReq)

	// User with e-mail already registered
	if err == nil {
		return PlatformInvite{}, errors.ErrConflict
	} else {
		st, ok := status.FromError(err)
		if ok && st.Code() != codes.NotFound {
			return PlatformInvite{}, err
		}
	}

	createdAt := getTimestmap()
	inviteID, err := svc.idProvider.ID()
	if err != nil {
		return PlatformInvite{}, err
	}

	invite := PlatformInvite{
		ID:           inviteID,
		InviteeEmail: email,
		CreatedAt:    createdAt,
		ExpiresAt:    createdAt.Add(svc.inviteDuration),
		State:        InviteStatePending,
	}

	if err := svc.invites.SavePlatformInvite(ctx, invite); err != nil {
		return PlatformInvite{}, err
	}

	svc.SendPlatformInviteEmail(ctx, invite, redirectPath)

	return invite, nil
}

func (svc service) RevokePlatformInvite(ctx context.Context, token string, inviteID string) error {
	if err := svc.isAdmin(ctx, token); err != nil {
		return err
	}

	invite, err := svc.invites.RetrievePlatformInviteByID(ctx, inviteID)
	if err != nil {
		return err
	}

	if invite.State != InviteStatePending {
		if invite.State == InviteStateExpired {
			return ErrInviteExpired
		}

		return ErrInvalidInviteState
	}

	if err := svc.invites.UpdatePlatformInviteState(ctx, inviteID, InviteStateRevoked); err != nil {
		return err
	}

	return nil
}

func (svc service) ViewPlatformInvite(ctx context.Context, token string, inviteID string) (PlatformInvite, error) {
	if err := svc.isAdmin(ctx, token); err != nil {
		return PlatformInvite{}, err
	}

	invite, err := svc.invites.RetrievePlatformInviteByID(ctx, inviteID)
	if err != nil {
		return PlatformInvite{}, err
	}

	return invite, nil
}

func (svc service) ListPlatformInvites(ctx context.Context, token string, pm apiutil.PageMetadata) (PlatformInvitesPage, error) {
	if err := svc.isAdmin(ctx, token); err != nil {
		return PlatformInvitesPage{}, err
	}

	invitesPage, err := svc.invites.RetrievePlatformInvites(ctx, pm)
	if err != nil {
		return PlatformInvitesPage{}, err
	}

	return invitesPage, nil
}

func (svc service) ValidatePlatformInvite(ctx context.Context, inviteID string, email string) error {
	invite, err := svc.invites.RetrievePlatformInviteByID(ctx, inviteID)
	if err != nil {
		return err
	}

	if invite.InviteeEmail != email {
		return errors.ErrAuthorization
	}

	if err := svc.invites.UpdatePlatformInviteState(ctx, inviteID, InviteStateAccepted); err != nil {
		return err
	}

	return nil
}

func (svc service) SendPlatformInviteEmail(ctx context.Context, invite PlatformInvite, redirectPath string) error {
	to := []string{invite.InviteeEmail}
	return svc.email.SendPlatformInvite(to, invite, redirectPath)
}
