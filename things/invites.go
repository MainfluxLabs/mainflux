package things

import (
	"context"
	"database/sql"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/invites"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

type GroupInvite struct {
	invites.InviteCommon
	GroupID   string `db:"group_id"`
	GroupName string
}

func (invite GroupInvite) GetCommon() invites.InviteCommon {
	return invite.InviteCommon
}

func (invite GroupInvite) GetDestinationID() string {
	return invite.GroupID
}

func (invite GroupInvite) ColumnDestinationID() string {
	return "group_id"
}

func (invite GroupInvite) TableName() string {
	return "group_invites"
}

func (invite GroupInvite) ToDBInvite() invites.DbInvite {
	commonDBInvite := invite.InviteCommon.ToDBInvite()
	commonDBInvite.DestinationID = invite.GroupID

	return commonDBInvite
}

type GroupInvitesPage = invites.InvitesPage[GroupInvite]

type Invites interface {
	// CreateGroupInvite creates a pending Invite on behalf of the User authenticated by `token`,
	// towards the user identified by `email`, to join the Group identified by `groupID` with an appropriate role.
	CreateGroupInvite(ctx context.Context, token, email, role, groupID, grRedirectPath string) (GroupInvite, error)

	// CreateDormantGroupInvites creates one or more dormant group invites on behalf of the User authenticated by `token`,
	// tied to a specific pending Org invite denoted by `orgInviteID`. Each membership in `groupMembership` results in one
	// new dormant Group Invite.
	CreateDormantGroupInvites(ctx context.Context, token, orgInviteID string, groupMemberships ...GroupMembership) error

	// RevokeGroupInvite revokes a specific pending Invite. An existing pending Invite can only be revoked
	// by its creator (inviter).
	RevokeGroupInvite(ctx context.Context, token, inviteID string) error

	// RespondGroupInvite responds to a specific invite, either accepting it (after which the invitee
	// is assigned as a member of the appropriate Group), or declining it. An Invite can only be responded
	// to by the invitee that it's directed towards.
	RespondGroupInvite(ctx context.Context, token, inviteID string, accept bool) error

	// ActivateGroupInvites activates all dormant Group invites tied to the Org invite denoted by `orgInviteID`,
	// by setting each invite's `invitee_id` field to the ID of the user authenticated by `token`.
	// The expiration time of each invite is reset, and an e-mail notification is sent to the invitee for each
	// activated invite.
	ActivateGroupInvites(ctx context.Context, token, orgInviteID, invRedirectPath string) error

	// ViewGroupInvite retrieves a single Invite denoted by its ID.  A specific Group Invite can be retrieved
	// by any user with admin privileges within the Group to which the invite belongs,
	// the Invitee towards who it is directed, or the platform Root Admin.
	ViewGroupInvite(ctx context.Context, token, inviteID string) (GroupInvite, error)

	// ListGroupInvitesByUser retrieves a list of invites either directed towards a specific Invitee,
	// or sent out by a specific Inviter, depending on the value of the `userType` argument, which
	// must be either 'invitee' or 'inviter'.
	ListGroupInvitesByUser(ctx context.Context, token, userType, userID string, pm invites.PageMetadataInvites) (GroupInvitesPage, error)

	// ListGroupInvitesByGroup retrieves a list of invites towards any user(s) to join the Group identified
	// by its ID
	ListGroupInvitesByGroup(ctx context.Context, token, groupID string, pm invites.PageMetadataInvites) (GroupInvitesPage, error)

	// SendGroupInviteEmail sends an e-mail notifying the invitee of the corresponding Invite.
	SendGroupInviteEmail(ctx context.Context, invite GroupInvite, email, orgName, invRedirectPath string) error
}

type GroupInviteRepository interface {
	invites.InviteRepository[GroupInvite]

	// SaveDormantInviteRelations saves relations of one or more dormant Group invites to a specific pending Org invite.
	SaveDormantInviteRelations(ctx context.Context, orgInviteID string, groupInviteIDs ...string) error

	// ActivateGroupInvites activates all dormant Group invites corresponding to the Org invite denoted by `orgInviteID` by:
	// - Updating the `invitee_id` and `expires_at` columns of all matching Group Invites to the supplied values
	// - Removing the associated rows from the `dormant_group_invites` table.
	// Returns a slice of the activated Group Invites.
	ActivateGroupInvites(ctx context.Context, orgInviteID, userID string, expirationTime time.Time) ([]GroupInvite, error)
}

func (svc thingsService) CreateGroupInvite(ctx context.Context, token, email, role, groupID, invRedirectPath string) (GroupInvite, error) {
	if err := svc.canAccessGroup(ctx, token, groupID, Admin); err != nil {
		return GroupInvite{}, err
	}

	group, err := svc.groups.RetrieveByID(ctx, groupID)
	if err != nil {
		return GroupInvite{}, err
	}

	identifyRes, err := svc.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return GroupInvite{}, err
	}

	inviterID := identifyRes.GetId()

	muReq := protomfx.UsersByEmailsReq{Emails: []string{email}}
	users, err := svc.users.GetUsersByEmails(ctx, &muReq)

	if err != nil {
		return GroupInvite{}, err
	}

	inviteeID := users.Users[0].Id

	// Make sure invitee is a member of the target Org
	_, err = svc.auth.ViewOrgMembership(ctx, &protomfx.ViewOrgMembershipReq{
		Token:    token,
		MemberID: inviteeID,
		OrgID:    group.OrgID,
	})

	if err != nil {
		return GroupInvite{}, err
	}

	// Make sure invitee isn't already a member of the target Group

	_, err = svc.groupMemberships.RetrieveRole(ctx, GroupMembership{
		GroupID:  groupID,
		MemberID: inviteeID,
	})

	if err != nil && !errors.Contains(err, dbutil.ErrNotFound) {
		return GroupInvite{}, err
	}

	if err == nil {
		return GroupInvite{}, ErrGroupMembershipExists
	}

	createdAt := getTimestamp()
	inviteID, err := svc.idProvider.ID()
	if err != nil {
		return GroupInvite{}, err
	}

	invite := GroupInvite{
		InviteCommon: invites.InviteCommon{
			ID:          inviteID,
			InviteeID:   sql.NullString{Valid: true, String: inviteeID},
			InviterID:   inviterID,
			InviteeRole: role,
			CreatedAt:   createdAt,
			ExpiresAt:   createdAt.Add(svc.inviteDuration),
			State:       invites.InviteStatePending,
		},
		GroupID: groupID,
	}

	if err := svc.groupInvites.SaveInvites(ctx, invite); err != nil {
		return GroupInvite{}, err
	}

	if err := svc.populateInviteInfo(ctx, &invite); err != nil {
		return GroupInvite{}, err
	}

	org, err := svc.auth.ViewOrg(ctx, &protomfx.ViewOrgReq{
		Token: token,
		Id: &protomfx.OrgID{
			Value: group.OrgID,
		},
	})

	if err != nil {
		return GroupInvite{}, err
	}

	go func() {
		svc.SendGroupInviteEmail(ctx, invite, email, org.Name, invRedirectPath)
	}()

	return invite, nil
}

func (svc thingsService) CreateDormantGroupInvites(ctx context.Context, token, orgInviteID string, groupMemberships ...GroupMembership) error {
	identifyRes, err := svc.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return err
	}

	inviterID := identifyRes.GetId()

	var invs []GroupInvite
	var inviteIDs []string

	for _, grm := range groupMemberships {
		// Make sure authenticated user has admin privileges in the target Group
		if err := svc.canAccessGroup(ctx, token, grm.GroupID, Admin); err != nil {
			return err
		}

		createdAt := getTimestamp()
		inviteID, err := svc.idProvider.ID()
		if err != nil {
			return err
		}

		inv := GroupInvite{
			InviteCommon: invites.InviteCommon{
				ID:          inviteID,
				InviteeID:   sql.NullString{Valid: false},
				InviterID:   inviterID,
				InviteeRole: grm.Role,
				CreatedAt:   createdAt,
				ExpiresAt:   createdAt.Add(svc.inviteDuration),
				State:       invites.InviteStatePending,
			},
			GroupID: grm.GroupID,
		}

		invs = append(invs, inv)
		inviteIDs = append(inviteIDs, inviteID)
	}

	if err := svc.groupInvites.SaveInvites(ctx, invs...); err != nil {
		return err
	}

	if err := svc.groupInvites.SaveDormantInviteRelations(ctx, orgInviteID, inviteIDs...); err != nil {
		return err
	}

	return nil
}

func (svc thingsService) ActivateGroupInvites(ctx context.Context, token, orgInviteID, invRedirectPath string) error {
	newExpirationTime := getTimestamp().Add(svc.inviteDuration)

	identifyRes, err := svc.auth.Identify(ctx, &protomfx.Token{
		Value: token,
	})

	if err != nil {
		return err
	}

	inviteeID := identifyRes.Id

	activatedInvites, err := svc.groupInvites.ActivateGroupInvites(ctx, orgInviteID, inviteeID, newExpirationTime)
	if err != nil {
		return err
	}

	// Send e-mail notification for each activated Group Invite
	for _, invite := range activatedInvites {
		if invite.State != invites.InviteStatePending {
			continue
		}

		if err := svc.populateInviteInfo(ctx, &invite); err != nil {
			continue
		}

		group, err := svc.groups.RetrieveByID(ctx, invite.GroupID)
		if err != nil {
			return err
		}

		org, err := svc.auth.ViewOrg(ctx, &protomfx.ViewOrgReq{
			Token: token,
			Id: &protomfx.OrgID{
				Value: group.OrgID,
			},
		})

		if err != nil {
			return err
		}

		go func() {
			svc.SendGroupInviteEmail(ctx, invite, invite.InviteeEmail, org.Name, invRedirectPath)
		}()
	}

	return nil
}

func (svc thingsService) RevokeGroupInvite(ctx context.Context, token, inviteID string) error {
	user, err := svc.auth.Identify(ctx, &protomfx.Token{
		Value: token,
	})
	if err != nil {
		return err
	}

	userID := user.GetId()

	invite, err := svc.groupInvites.RetrieveInviteByID(ctx, inviteID)
	if err != nil {
		return err
	}

	// An Invite can only be revoked by its issuer
	if invite.InviterID != userID {
		return errors.ErrAuthorization
	}

	if invite.State != invites.InviteStatePending {
		if invite.State == invites.InviteStateExpired {
			return apiutil.ErrInviteExpired
		}

		return apiutil.ErrInvalidInviteState
	}

	if err := svc.groupInvites.UpdateInviteState(ctx, inviteID, invites.InviteStateRevoked); err != nil {
		return err
	}

	return nil
}

func (svc thingsService) RespondGroupInvite(ctx context.Context, token, inviteID string, accept bool) error {
	user, err := svc.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return err
	}

	userID := user.GetId()

	invite, err := svc.groupInvites.RetrieveInviteByID(ctx, inviteID)
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
	if userID != invite.InviteeID.String {
		return errors.ErrAuthorization
	}

	newState := invites.InviteStateDeclined

	if accept {
		// User has accepted the Invite, assign them as a member of the appropriate Group
		// with the appropriate role
		newState = invites.InviteStateAccepted

		membership := GroupMembership{
			MemberID: userID,
			GroupID:  invite.GroupID,
			Role:     invite.InviteeRole,
		}

		if err := svc.groupMemberships.Save(ctx, membership); err != nil {
			return err
		}
	}

	if err := svc.groupInvites.UpdateInviteState(ctx, inviteID, newState); err != nil {
		return err
	}

	return nil
}

func (svc thingsService) ViewGroupInvite(ctx context.Context, token, inviteID string) (GroupInvite, error) {
	invite, err := svc.groupInvites.RetrieveInviteByID(ctx, inviteID)
	if err != nil {
		return GroupInvite{}, err
	}

	if err := svc.populateInviteInfo(ctx, &invite); err != nil {
		return GroupInvite{}, err
	}

	if err := svc.isAdmin(ctx, token); err == nil {
		return invite, nil
	}

	if err := svc.canAccessGroup(ctx, token, invite.GroupID, Admin); err == nil {
		return invite, nil
	}

	user, err := svc.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return GroupInvite{}, err
	}

	userID := user.GetId()

	if userID == invite.InviteeID.String {
		return invite, nil
	}

	return GroupInvite{}, errors.ErrAuthorization
}

func (svc thingsService) ListGroupInvitesByUser(ctx context.Context, token, userType, userID string, pm invites.PageMetadataInvites) (GroupInvitesPage, error) {
	if err := svc.isAdmin(ctx, token); err != nil {
		// Current User is not Root Admin - must be either the Invitee or Inviter
		user, err := svc.auth.Identify(ctx, &protomfx.Token{Value: token})
		if err != nil {
			return GroupInvitesPage{}, err
		}

		authUserID := user.GetId()

		if authUserID != userID {
			return GroupInvitesPage{}, errors.ErrAuthorization
		}
	}

	invitesPage, err := svc.groupInvites.RetrieveInvitesByUser(ctx, userType, userID, pm)
	if err != nil {
		return GroupInvitesPage{}, err
	}

	for idx := range invitesPage.Invites {
		if err := svc.populateInviteInfo(ctx, &invitesPage.Invites[idx]); err != nil {
			return GroupInvitesPage{}, err
		}
	}

	return invitesPage, nil
}

func (svc thingsService) ListGroupInvitesByGroup(ctx context.Context, token, groupID string, pm invites.PageMetadataInvites) (GroupInvitesPage, error) {
	if err := svc.canAccessGroup(ctx, token, groupID, Admin); err != nil {
		return GroupInvitesPage{}, err
	}

	page, err := svc.groupInvites.RetrieveInvitesByDestination(ctx, groupID, pm)
	if err != nil {
		return GroupInvitesPage{}, err
	}

	for idx := range page.Invites {
		if err := svc.populateInviteInfo(ctx, &page.Invites[idx]); err != nil {
			return GroupInvitesPage{}, err
		}
	}

	return page, nil
}

func (svc thingsService) SendGroupInviteEmail(ctx context.Context, invite GroupInvite, email, orgName, invRedirectPath string) error {
	to := []string{email}

	return svc.email.SendGroupInvite(to, invite, orgName, invRedirectPath)
}

func (svc thingsService) populateInviteInfo(ctx context.Context, invite *GroupInvite) error {
	group, err := svc.groups.RetrieveByID(ctx, invite.GroupID)
	if err != nil {
		return err
	}

	invite.GroupName = group.Name

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
