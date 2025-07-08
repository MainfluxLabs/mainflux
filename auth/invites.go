package auth

import (
	"context"
	"log"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var (
	// ErrCreateInvite indicates failure to create a new invite
	ErrCreateInvite       = errors.New("error creating invite")
	ErrUserAlreadyInvited = errors.New("user already has pending invite to org")
)

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
}

type InvitesRepository interface {
	// Save saves one or more pending invites to the repository
	Save(ctx context.Context, invites ...Invite) error
}

func (svc service) InviteMembers(ctx context.Context, token string, orgID string, oms ...OrgMember) error {
	log.Printf("InviteMembers: orgID: %s, token: %s\n", orgID, token)
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
			// TODO: Make invite expiry time configurable (env. var + default value in service)
			CreatedAt: createdAt,
			ExpiresAt: createdAt.Add(7 * 24 * time.Hour),
		}

		invites = append(invites, invite)
	}

	if err := svc.invites.Save(ctx, invites...); err != nil {
		return err
	}

	return nil
}
