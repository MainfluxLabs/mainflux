package auth

import (
	"context"
	"database/sql"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/invites"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrInvalidInviteResponse indicates an invalid Invite response action string.
var ErrInvalidInviteResponse = errors.New("invalid invite response action")

type OrgInvite struct {
	invites.InviteCommon
	OrgID   string `db:"org_id"`
	OrgName string
}

func (invite OrgInvite) GetCommon() invites.InviteCommon {
	return invite.InviteCommon
}

func (invite OrgInvite) GetDestinationID() string {
	return invite.OrgID
}

func (invite OrgInvite) ColumnDestinationID() string {
	return "org_id"
}

func (invite OrgInvite) TableName() string {
	return "org_invites"
}

func (invite OrgInvite) ToDBInvite() invites.DbInvite {
	commonDBInvite := invite.InviteCommon.ToDBInvite()
	commonDBInvite.DestinationID = invite.OrgID

	return commonDBInvite
}

type OrgInvitesPage = invites.InvitesPage[OrgInvite]

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
	ListOrgInvitesByUser(ctx context.Context, token, userType, userID string, pm invites.PageMetadataInvites) (OrgInvitesPage, error)

	// ListOrgInvitesByOrg retrieves a list of invites towards any user(s) to join the org identified
	// by its ID
	ListOrgInvitesByOrg(ctx context.Context, token, orgID string, pm invites.PageMetadataInvites) (OrgInvitesPage, error)

	// SendOrgInviteEmail sends an e-mail notifying the invitee of the corresponding Invite.
	SendOrgInviteEmail(ctx context.Context, invite OrgInvite, email, orgName, invRedirectPath string) error
}

type OrgInviteRepository interface {
	invites.InviteRepository[OrgInvite]
}

func (svc service) CreateOrgInvite(ctx context.Context, token, email, role, orgID, invRedirectPath string) (OrgInvite, error) {
	// Check if currently authenticated User has "admin" or higher privileges within Org
	if err := svc.canAccessOrg(ctx, token, orgID, Admin); err != nil {
		return OrgInvite{}, err
	}

	inviter, err := svc.identify(ctx, token)
	if err != nil {
		return OrgInvite{}, err
	}

	org, err := svc.orgs.RetrieveByID(ctx, orgID)
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

	_, err = svc.memberships.RetrieveRole(ctx, inviteeID, orgID)
	if err != nil && !errors.Contains(err, dbutil.ErrNotFound) {
		return OrgInvite{}, err
	}

	if err == nil {
		return OrgInvite{}, ErrOrgMembershipExists
	}

	createdAt := getTimestamp()
	inviteID, err := svc.idProvider.ID()
	if err != nil {
		return OrgInvite{}, err
	}

	invite := OrgInvite{
		InviteCommon: invites.InviteCommon{
			ID:          inviteID,
			InviteeID:   sql.NullString{Valid: true, String: inviteeID},
			InviterID:   inviter.ID,
			InviteeRole: role,
			CreatedAt:   createdAt,
			ExpiresAt:   createdAt.Add(svc.inviteDuration),
			State:       invites.InviteStatePending,
		},
		OrgID: orgID,
	}

	if err := svc.invites.SaveInvites(ctx, invite); err != nil {
		return OrgInvite{}, err
	}

	go func() {
		svc.SendOrgInviteEmail(ctx, invite, email, org.Name, invRedirectPath)
	}()

	return invite, nil
}

func (svc service) RevokeOrgInvite(ctx context.Context, token, inviteID string) error {
	user, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}

	invite, err := svc.invites.RetrieveInviteByID(ctx, inviteID)
	if err != nil {
		return err
	}

	// An Invite can only be revoked by its issuer
	if invite.InviterID != user.ID {
		return errors.ErrAuthorization
	}

	if invite.State != invites.InviteStatePending {
		if invite.State == invites.InviteStateExpired {
			return apiutil.ErrInviteExpired
		}

		return apiutil.ErrInvalidInviteState
	}

	if err := svc.invites.UpdateInviteState(ctx, inviteID, invites.InviteStateRevoked); err != nil {
		return err
	}

	return nil
}

func (svc service) ViewOrgInvite(ctx context.Context, token, inviteID string) (OrgInvite, error) {
	invite, err := svc.invites.RetrieveInviteByID(ctx, inviteID)
	if err != nil {
		return OrgInvite{}, err
	}

	if err := svc.populateInviteInfo(ctx, &invite); err != nil {
		return OrgInvite{}, err
	}

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

	if user.ID == invite.InviteeID.String {
		return invite, nil
	}

	return OrgInvite{}, errors.ErrAuthorization
}

func (svc service) RespondOrgInvite(ctx context.Context, token, inviteID string, accept bool) error {
	user, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}

	invite, err := svc.invites.RetrieveInviteByID(ctx, inviteID)
	if err != nil {
		return err
	}

	if invite.State != invites.InviteStatePending {
		if invite.State == invites.InviteStateExpired {
			return apiutil.ErrInviteExpired
		}

		return apiutil.ErrInvalidInviteState
	}

	// An Invite can only be responded to by the invitee
	if user.ID != invite.InviteeID.String {
		return errors.ErrAuthorization
	}

	newState := invites.InviteStateDeclined

	if accept {
		// User has accepted the Invite, assign them as a member of the appropriate Org
		// with the appropriate role
		newState = invites.InviteStateAccepted
		ts := getTimestamp()

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

	if err := svc.invites.UpdateInviteState(ctx, inviteID, newState); err != nil {
		return err
	}

	return nil
}

func (svc service) ListOrgInvitesByOrg(ctx context.Context, token, orgID string, pm invites.PageMetadataInvites) (OrgInvitesPage, error) {
	if err := svc.canAccessOrg(ctx, token, orgID, Admin); err != nil {
		return OrgInvitesPage{}, err
	}

	page, err := svc.invites.RetrieveInvitesByDestination(ctx, orgID, pm)
	if err != nil {
		return OrgInvitesPage{}, err
	}

	for idx := range page.Invites {
		if err := svc.populateInviteInfo(ctx, &page.Invites[idx]); err != nil {
			return OrgInvitesPage{}, err
		}
	}

	return page, nil
}

func (svc service) ListOrgInvitesByUser(ctx context.Context, token, userType, userID string, pm invites.PageMetadataInvites) (OrgInvitesPage, error) {
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

	invitesPage, err := svc.invites.RetrieveInvitesByUser(ctx, userType, userID, pm)
	if err != nil {
		return OrgInvitesPage{}, err
	}

	for idx := range invitesPage.Invites {
		if err := svc.populateInviteInfo(ctx, &invitesPage.Invites[idx]); err != nil {
			return OrgInvitesPage{}, err
		}
	}

	return invitesPage, nil
}

// Sets the invite.InviterEmail, invite.InviteeEmail and invite.OrgName fields of the passed invite.
func (svc service) populateInviteInfo(ctx context.Context, invite *OrgInvite) error {
	org, err := svc.orgs.RetrieveByID(ctx, invite.OrgID)
	if err != nil {
		return err
	}

	invite.OrgName = org.Name

	usersReq := &protomfx.UsersByIDsReq{Ids: []string{invite.InviterID, invite.InviteeID.String}}
	usersRes, err := svc.users.GetUsersByIDs(ctx, usersReq)
	if err != nil {
		return err
	}

	// Order of results from gRPC call isn't guaranteed to match order of IDs in request
	users := usersRes.GetUsers()

	switch users[0].Id {
	case invite.InviterID:
		invite.InviterEmail = users[0].GetEmail()
		invite.InviteeEmail = users[1].GetEmail()
	default:
		invite.InviterEmail = users[1].GetEmail()
		invite.InviteeEmail = users[0].GetEmail()
	}

	return nil
}

func (svc service) SendOrgInviteEmail(ctx context.Context, invite OrgInvite, email, orgName, invRedirectPath string) error {
	to := []string{email}
	return svc.email.SendOrgInvite(to, invite, orgName, invRedirectPath)
}
