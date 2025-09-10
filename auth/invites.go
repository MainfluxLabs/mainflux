package auth

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrInvalidInviteResponse indicates an invalid Invite response action string.
var ErrInvalidInviteResponse = errors.New("invalid invite response action")

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

type PageMetadataInvites struct {
	apiutil.PageMetadata
	State string `json:"state,omitempty"`
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
	// CreateOrgInvite creates a pending Invite on behalf of the User authenticated by `token`,
	// towards the user identified by `email`, to join the Org identified by `orgID` with an appropriate role.
	CreateOrgInvite(ctx context.Context, token, email, role, orgID, invRedirectPath string) (OrgInvite, error)

	// RevokeOrgInvite revokes a specific pending Invite. An existing pending Invite can only be revoked
	// by its original inviter (creator).
	RevokeOrgInvite(ctx context.Context, token, inviteID string) error

	// RespondOrgInvite responds to a specific invite, either accepting it (after which the invitee
	// is assigned as a member of the appropriate Org), or declining it. An Invite can only be responded
	// to by the invitee that it's directed towards.
	RespondOrgInvite(ctx context.Context, token, inviteID string, accept bool) error

	// ViewOrgInvite retrieves a single Invite denoted by its ID.  A specific Org Invite can be retrieved
	// by any user with admin privileges within the Org to which the invite belongs,
	// the Invitee towards who it is directed, or the platform Root Admin.
	ViewOrgInvite(ctx context.Context, token, inviteID string) (OrgInvite, error)

	// ListOrgInvitesByUser retrieves a list of invites either directed towards a specific Invitee,
	// or sent out by a specific Inviter, depending on the value of the `userType` argument, which
	// must be either 'invitee' or 'inviter'.
	ListOrgInvitesByUser(ctx context.Context, token, userType, userID string, pm PageMetadataInvites) (OrgInvitesPage, error)

	// ListOrgInvitesByOrg retrieves a list of invites towards any user(s) to join the org identified
	// by its ID
	ListOrgInvitesByOrg(ctx context.Context, token, orgID string, pm PageMetadataInvites) (OrgInvitesPage, error)

	// SendOrgInviteEmail sends an e-mail notifying the invitee of the corresponding Invite.
	SendOrgInviteEmail(ctx context.Context, invite OrgInvite, email, orgName, invRedirectPath string) error
}

type OrgInvitesRepository interface {
	// SaveOrgInvite saves one or more pending org invites to the repository.
	SaveOrgInvite(ctx context.Context, invites ...OrgInvite) error

	// RetrieveOrgInviteByID retrieves a specific OrgInvite by its ID.
	RetrieveOrgInviteByID(ctx context.Context, inviteID string) (OrgInvite, error)

	// RemoveOrgInvite removes a specific pending OrgInvite.
	RemoveOrgInvite(ctx context.Context, inviteID string) error

	// RetrieveOrgInviteByUserID retrieves a list of invites either directed towards a specific Invitee, or sent out by a
	// specific Inviter, depending on the value of the `userType` argument, which must be either 'invitee' or 'inviter'.
	RetrieveOrgInvitesByUser(ctx context.Context, userType, userID string, pm PageMetadataInvites) (OrgInvitesPage, error)

	// RetrieveOrgInvitesByOrg retrieves a list of invites towards any user(s) to join the Org identified
	// by its ID.
	RetrieveOrgInvitesByOrg(ctx context.Context, orgID string, pm PageMetadataInvites) (OrgInvitesPage, error)

	// UpdateOrgInviteState updates the state of a specific Invite denoted by its ID.
	UpdateOrgInviteState(ctx context.Context, inviteID, state string) error
}

func (svc service) CreateOrgInvite(ctx context.Context, token, email, role, orgID, invRedirectPath string) (OrgInvite, error) {
	// Check if currently authenticated User has "admin" or higher privileges within Org
	if err := svc.canAccessOrg(ctx, token, orgID, Admin); err != nil {
		return OrgInvite{}, err
	}

	// Get userID of inviter
	inviter, err := svc.identify(ctx, token)
	if err != nil {
		return OrgInvite{}, err
	}

	org, err := svc.ViewOrg(ctx, token, orgID)
	if err != nil {
		return OrgInvite{}, err
	}

	muReq := protomfx.UsersByEmailsReq{Emails: []string{email}}
	users, err := svc.users.GetUsersByEmails(ctx, &muReq)

	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.NotFound:
				return OrgInvite{}, dbutil.ErrNotFound
			default:
				return OrgInvite{}, err
			}
		}

		return OrgInvite{}, err
	}

	inviteeID := users.Users[0].Id

	_, err = svc.ViewOrgMembership(ctx, token, orgID, inviteeID)
	if err != nil && !errors.Contains(err, dbutil.ErrNotFound) {
		return OrgInvite{}, err
	}

	if err == nil {
		return OrgInvite{}, ErrOrgMembershipExists
	}

	createdAt := getTimestmap()
	inviteID, err := svc.idProvider.ID()
	if err != nil {
		return OrgInvite{}, err
	}

	invite := OrgInvite{
		ID:          inviteID,
		InviteeID:   inviteeID,
		InviterID:   inviter.ID,
		OrgID:       orgID,
		InviteeRole: role,
		CreatedAt:   createdAt,
		ExpiresAt:   createdAt.Add(svc.inviteDuration),
		State:       InviteStatePending,
	}

	if err := svc.invites.SaveOrgInvite(ctx, invite); err != nil {
		return OrgInvite{}, err
	}

	go func() {
		svc.SendOrgInviteEmail(ctx, invite, email, org.Name, invRedirectPath)
	}()

	return invite, nil
}

func (svc service) RevokeOrgInvite(ctx context.Context, token, inviteID string) error {
	// Identify User attempting to revoke invite
	user, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}

	invite, err := svc.invites.RetrieveOrgInviteByID(ctx, inviteID)
	if err != nil {
		return err
	}

	// An Invite can only be revoked by its issuer
	if invite.InviterID != user.ID {
		return errors.ErrAuthorization
	}

	if invite.State != InviteStatePending {
		if invite.State == InviteStateExpired {
			return apiutil.ErrInviteExpired
		}

		return apiutil.ErrInvalidInviteState
	}

	if err := svc.invites.UpdateOrgInviteState(ctx, inviteID, InviteStateRevoked); err != nil {
		return err
	}

	return nil
}

func (svc service) ViewOrgInvite(ctx context.Context, token, inviteID string) (OrgInvite, error) {
	invite, err := svc.invites.RetrieveOrgInviteByID(ctx, inviteID)
	if err != nil {
		return OrgInvite{}, err
	}

	// A specific Invite can only be retrieved by the platform Root Admin, the Invitee towards who
	// the Invite is directed, or any person with admin (or higher) rights in the Org to which
	// the invite belongs
	if err := svc.isAdmin(ctx, token); err == nil {
		return invite, nil
	}

	if err := svc.canAccessOrg(ctx, token, invite.OrgID, Admin); err == nil {
		return invite, nil
	}

	user, err := svc.identify(ctx, token)
	if err != nil {
		return OrgInvite{}, err
	}

	if user.ID == invite.InviteeID {
		return invite, nil
	}

	return OrgInvite{}, errors.ErrAuthorization
}

func (svc service) RespondOrgInvite(ctx context.Context, token, inviteID string, accept bool) error {
	user, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}

	// Obtain detailed information about the Invite
	invite, err := svc.invites.RetrieveOrgInviteByID(ctx, inviteID)
	if err != nil {
		return err
	}

	if invite.State != InviteStatePending {
		if invite.State == InviteStateExpired {
			return apiutil.ErrInviteExpired
		}

		return apiutil.ErrInvalidInviteState
	}

	// An Invite can only be responded to by the invitee
	if user.ID != invite.InviteeID {
		return errors.ErrAuthorization
	}

	newState := InviteStateDeclined

	if accept {
		// User has accepted the Invite, assign them as a member of the appropriate Org
		// with the appropriate role
		newState = InviteStateAccepted
		ts := getTimestmap()

		membership := OrgMembership{
			MemberID:  user.ID,
			OrgID:     invite.OrgID,
			Role:      invite.InviteeRole,
			CreatedAt: ts,
			UpdatedAt: ts,
		}

		if err := svc.memberships.Save(ctx, membership); err != nil {
			return err
		}
	}

	if err := svc.invites.UpdateOrgInviteState(ctx, inviteID, newState); err != nil {
		return err
	}

	return nil
}

func (svc service) ListOrgInvitesByOrg(ctx context.Context, token, orgID string, pm PageMetadataInvites) (OrgInvitesPage, error) {
	if err := svc.canAccessOrg(ctx, token, orgID, Admin); err != nil {
		return OrgInvitesPage{}, err
	}

	page, err := svc.invites.RetrieveOrgInvitesByOrg(ctx, orgID, pm)
	if err != nil {
		return OrgInvitesPage{}, err
	}

	return page, nil
}

func (svc service) ListOrgInvitesByUser(ctx context.Context, token, userType, userID string, pm PageMetadataInvites) (OrgInvitesPage, error) {
	if err := svc.isAdmin(ctx, token); err != nil {
		if err != errors.ErrAuthorization {
			return OrgInvitesPage{}, err
		}

		// Current User is not Root Admin - must be either the Invitee or Inviter
		user, err := svc.identify(ctx, token)
		if err != nil {
			return OrgInvitesPage{}, err
		}

		if user.ID != userID {
			return OrgInvitesPage{}, errors.ErrAuthorization
		}
	}

	invitesPage, err := svc.invites.RetrieveOrgInvitesByUser(ctx, userType, userID, pm)
	if err != nil {
		return OrgInvitesPage{}, err
	}

	return invitesPage, nil
}

func (svc service) SendOrgInviteEmail(ctx context.Context, invite OrgInvite, email, orgName, invRedirectPath string) error {
	to := []string{email}
	return svc.email.SendOrgInvite(to, invite, orgName, invRedirectPath)
}
