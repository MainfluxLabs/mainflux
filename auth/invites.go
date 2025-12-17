package auth

import (
	"context"
	"fmt"
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
var ErrGroupsDifferingOrgs = errors.New("groups belong to differing organizations")

type OrgInvite struct {
	ID           string
	InviteeID    string
	InviteeEmail string
	InviterID    string
	InviterEmail string
	OrgID        string
	OrgName      string
	InviteeRole  string
	// Map of Group ID to proposed role in the group
	Groups    map[string]string
	CreatedAt time.Time
	ExpiresAt time.Time
	State     string
}

type OrgInvitesPage struct {
	Invites []OrgInvite
	Total   uint64
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
	// `groups` is an optional mapping of Group IDs to a role within that Group. If present, the invitee will be additionally
	// be assigned as a member of each of groups after they accept the Org invite.
	CreateOrgInvite(ctx context.Context, token, email, role, orgID string, groups map[string]string, invRedirectPath string) (OrgInvite, error)

	// CreateDormantOrgInvite creates a pending, dormant Org Invite associated with a specfic Platform Invite
	// denoted by `platformInviteID`.
	// `groups` is an optional mapping of Group IDs to a role within that Group. If present, the invitee will be additionally
	// be assigned as a member of each of groups after they accept the Org invite.
	CreateDormantOrgInvite(ctx context.Context, token, orgID, role string, groups map[string]string, platformInviteID string) (OrgInvite, error)

	// RevokeOrgInvite revokes a specific pending Invite. An existing pending Invite can only be revoked
	// by its original inviter (creator).
	RevokeOrgInvite(ctx context.Context, token, inviteID string) error

	// RespondOrgInvite responds to a specific invite, either accepting it (after which the invitee
	// is assigned as a member of the appropriate Org), or declining it. An Invite can only be responded
	// to by the invitee that it's directed towards.
	RespondOrgInvite(ctx context.Context, token, inviteID string, accept bool) error

	// ActivateOrgInvite activates all dormant Org Invites associated with the specific Platform Invite.
	// The expiration time of the invites is reset. An e-mail notification is sent to the invitee for each
	// activated invite.
	ActivateOrgInvite(ctx context.Context, platformInviteID, userID, invRedirectPath string) error

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

	// SaveDormantInviteRelation saves a relation of a dormant Org Invite with a specific Platform Invite.
	SaveDormantInviteRelation(ctx context.Context, orgInviteID, platformInviteID string) error

	// ActivateOrgInvite activates all dormant Org Invites corresponding to the specified Platform Invite by:
	// - Updating the "invitee_id" and "expires_at" columns of all matching Org Invites to the supplied values
	// - Removing the associated rows from the "dormant_org_invites" table
	// Returns slice of activated Org Invites.
	ActivateOrgInvite(ctx context.Context, platformInviteID, userID string, expirationTime time.Time) ([]OrgInvite, error)

	// RetrieveOrgInviteByID retrieves a specific OrgInvite by its ID.
	RetrieveOrgInviteByID(ctx context.Context, inviteID string) (OrgInvite, error)

	// RemoveOrgInvite removes a specific pending OrgInvite.
	RemoveOrgInvite(ctx context.Context, inviteID string) error

	// RetrieveOrgInviteByUser retrieves a list of invites either directed towards a specific Invitee, or sent out by a
	// specific Inviter, depending on the value of the `userType` argument, which must be either 'invitee' or 'inviter'.
	RetrieveOrgInvitesByUser(ctx context.Context, userType, userID string, pm PageMetadataInvites) (OrgInvitesPage, error)

	// RetrieveOrgInvitesByOrg retrieves a list of invites towards any user(s) to join the Org identified
	// by its ID.
	RetrieveOrgInvitesByOrg(ctx context.Context, orgID string, pm PageMetadataInvites) (OrgInvitesPage, error)

	// UpdateOrgInviteState updates the state of a specific Invite denoted by its ID.
	UpdateOrgInviteState(ctx context.Context, inviteID, state string) error
}

func (svc service) CreateOrgInvite(ctx context.Context, token, email, role, orgID string, groups map[string]string, invRedirectPath string) (OrgInvite, error) {
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

	// If the invite is associated with one or more Groups, make sure that they all belong to the target Org
	if groups != nil {
		groupIDs := make([]string, 0, len(groups))
		for groupID := range groups {
			groupIDs = append(groupIDs, groupID)
		}

		if err := svc.validateGroupsSameOrg(ctx, orgID, groupIDs); err != nil {
			return OrgInvite{}, err
		}
	}

	createdAt := getTimestamp()
	inviteID, err := svc.idProvider.ID()
	if err != nil {
		return OrgInvite{}, err
	}

	invite := OrgInvite{
		ID:          inviteID,
		InviteeID:   inviteeID,
		InviterID:   inviter.ID,
		OrgID:       orgID,
		Groups:      groups,
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

func (svc service) CreateDormantOrgInvite(ctx context.Context, token, orgID, role string, groups map[string]string, platformInviteID string) (OrgInvite, error) {
	if err := svc.canAccessOrg(ctx, token, orgID, Admin); err != nil {
		return OrgInvite{}, err
	}

	// If the invite is associated with one or more Groups, make sure that they all belong to the target Org
	if groups != nil {
		groupIDs := make([]string, 0, len(groups))
		for groupID := range groups {
			groupIDs = append(groupIDs, groupID)
		}

		if err := svc.validateGroupsSameOrg(ctx, orgID, groupIDs); err != nil {
			return OrgInvite{}, err
		}
	}

	inviter, err := svc.identify(ctx, token)
	if err != nil {
		return OrgInvite{}, err
	}

	createdAt := getTimestamp()

	inviteID, err := svc.idProvider.ID()
	if err != nil {
		return OrgInvite{}, err
	}

	invite := OrgInvite{
		ID:          inviteID,
		InviteeID:   "",
		InviterID:   inviter.ID,
		OrgID:       orgID,
		InviteeRole: role,
		Groups:      groups,
		CreatedAt:   createdAt,
		ExpiresAt:   createdAt.Add(svc.inviteDuration),
		State:       InviteStatePending,
	}

	fmt.Printf("CreateDormantOrgInvite: groups arg: %+v\n", groups)

	if err := svc.invites.SaveOrgInvite(ctx, invite); err != nil {
		return OrgInvite{}, err
	}

	if err := svc.invites.SaveDormantInviteRelation(ctx, inviteID, platformInviteID); err != nil {
		if err := svc.invites.RemoveOrgInvite(ctx, inviteID); err != nil {
			return OrgInvite{}, err
		}

		return OrgInvite{}, err
	}

	return invite, nil
}

func (svc service) RevokeOrgInvite(ctx context.Context, token, inviteID string) error {
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

func (svc service) ActivateOrgInvite(ctx context.Context, platformInviteID, userID, orgInviteRedirectPath string) error {
	newExpirationTime := getTimestamp().Add(svc.inviteDuration)

	invites, err := svc.invites.ActivateOrgInvite(ctx, platformInviteID, userID, newExpirationTime)
	if err != nil {
		return err
	}

	// Send e-mail notification for each activated Org Invite
	for _, invite := range invites {
		if invite.State != InviteStatePending {
			continue
		}

		if err := svc.populateInviteInfo(ctx, &invite); err != nil {
			continue
		}

		go func() {
			svc.SendOrgInviteEmail(ctx, invite, invite.InviteeEmail, invite.OrgName, orgInviteRedirectPath)
		}()
	}

	return nil
}

func (svc service) ViewOrgInvite(ctx context.Context, token, inviteID string) (OrgInvite, error) {
	invite, err := svc.invites.RetrieveOrgInviteByID(ctx, inviteID)
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

		// Create one group membership in the things service for each group the invite was associated with
		if len(invite.Groups) > 0 {
			grpcReq := &protomfx.CreateGroupMembershipsReq{
				Memberships: make([]*protomfx.GroupMembership, 0, len(invite.Groups)),
			}

			for groupID, role := range invite.Groups {
				grpcReq.Memberships = append(grpcReq.Memberships, &protomfx.GroupMembership{
					UserID:  invite.InviteeID,
					GroupID: groupID,
					Role:    role,
				})
			}

			if _, err := svc.things.CreateGroupMemberships(ctx, grpcReq); err != nil {
				return err
			}
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

	for idx := range page.Invites {
		if err := svc.populateInviteInfo(ctx, &page.Invites[idx]); err != nil {
			return OrgInvitesPage{}, err
		}
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

	userIDs := []string{invite.InviterID}
	if invite.InviteeID != "" {
		userIDs = append(userIDs, invite.InviteeID)
	}

	usersRes, err := svc.users.GetUsersByIDs(ctx, &protomfx.UsersByIDsReq{Ids: userIDs})
	if err != nil {
		return err
	}

	for _, user := range usersRes.GetUsers() {
		switch user.GetId() {
		case invite.InviterID:
			invite.InviterEmail = user.GetEmail()
		case invite.InviteeID:
			invite.InviteeEmail = user.GetEmail()
		}
	}

	return nil
}

// Validates that all passed Groups (denoted by their IDs) belong to the same Organization denoted by `orgID`. Returns ErrGroupsDifferingOrgs
// if at least one of the Groups belongs to a different Org, and nil otherwise.
func (svc service) validateGroupsSameOrg(ctx context.Context, orgID string, groupIDs []string) error {
	for _, groupID := range groupIDs {
		group, err := svc.things.ViewGroup(ctx, &protomfx.ViewGroupReq{
			GroupID: groupID,
		})

		if err != nil {
			return err
		}

		if group.OrgID != orgID {
			return ErrGroupsDifferingOrgs
		}
	}

	return nil
}

func (svc service) SendOrgInviteEmail(ctx context.Context, invite OrgInvite, email, orgName, invRedirectPath string) error {
	to := []string{email}

	var groupNames map[string]string

	if invite.Groups != nil {
		groupNames = make(map[string]string, len(invite.Groups))

		for groupID := range invite.Groups {
			group, err := svc.things.ViewGroup(context.Background(), &protomfx.ViewGroupReq{GroupID: groupID})
			if err != nil {
				return err
			}

			groupNames[groupID] = group.GetName()
		}
	}

	return svc.email.SendOrgInvite(to, invite, orgName, invRedirectPath, groupNames)
}
