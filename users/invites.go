package users

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

const (
	UserTypeInvitee = "invitee"
	UserTypeInviter = "inviter"

	InviteStatePending  = "pending"
	InviteStateExpired  = "expired"
	InviteStateRevoked  = "revoked"
	InviteStateAccepted = "accepted"
	InviteStateDeclined = "declined"
)

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

type PageMetadataInvites struct {
	apiutil.PageMetadata
	State string `json:"state,omitempty"`
}

type PlatformInvites interface {
	// CreatePlatformInvite creates a pending platform Invite for the appropriate email address.
	// The user can optionally also be invited to an Organization with a certain role by supplying the `orgInvite` argument - the invite
	// becomes visible once the user completes registration via the platform invite.
	// temp:orgid, role, gis
	CreatePlatformInvite(ctx context.Context, token, redirectPath, email string, orgInvite auth.OrgInvite) (PlatformInvite, error)

	// RevokePlatformInvite revokes a specific pending PlatformInvite. Only usable by the platform Root Admin.
	RevokePlatformInvite(ctx context.Context, token, inviteID string) error

	// ViewPlatformInvite retrieves a single PlatformInvite denoted by its ID. Only usable by the platform Root Admin.
	ViewPlatformInvite(ctx context.Context, token, inviteID string) (PlatformInvite, error)

	// ListPlatformInvites retrieves a list of platform invites. Only usable by the platform Root Admin.
	ListPlatformInvites(ctx context.Context, token string, pm PageMetadataInvites) (PlatformInvitesPage, error)

	// ValidatePlatformInvite checks if there exists a valid, pending, non-expired platform invite in the database that matches
	// the passed ID and user e-mail. If so, it marks that invite's state as 'accepted', and returns nil.
	// If no such valid platform invite is found in the database, it instead returns errors.ErrAuthorization.
	ValidatePlatformInvite(ctx context.Context, inviteID, email string) error

	// SendPlatformInviteEmail sends an e-mail notifying the invitee about the corresponding platform invite.
	SendPlatformInviteEmail(ctx context.Context, invite PlatformInvite, redirectPath string) error
}

type PlatformInvitesRepository interface {
	// SavePlatformInvite saves one or more pending platform invites to the repository.
	SavePlatformInvite(ctx context.Context, invites ...PlatformInvite) error

	// RetrievePlatformInviteByID retrieves a single platform invite by its ID.
	RetrievePlatformInviteByID(ctx context.Context, inviteID string) (PlatformInvite, error)

	// RetrievePlatformInvites retrieves a list of platform invites.
	RetrievePlatformInvites(ctx context.Context, pm PageMetadataInvites) (PlatformInvitesPage, error)

	// UpdatePlatformInviteState updates the state of a specific platform invite denoted by its ID.
	UpdatePlatformInviteState(ctx context.Context, inviteID, state string) error
}

func (svc usersService) CreatePlatformInvite(ctx context.Context, token, redirectPath, email string, orgInvite auth.OrgInvite) (PlatformInvite, error) {
	if err := svc.isAdmin(ctx, token); err != nil {
		return PlatformInvite{}, err
	}

	_, err := svc.ListUsersByEmails(ctx, []string{email})

	if err != nil && !errors.Contains(err, dbutil.ErrNotFound) {
		return PlatformInvite{}, err
	}

	// User with e-mail already registered
	if err == nil {
		return PlatformInvite{}, dbutil.ErrConflict
	}

	createdAt := time.Now()
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

	if orgInvite.OrgID != "" {
		var reqGroupInvites []*protomfx.GroupInvite
		for _, group := range orgInvite.GroupInvites {
			reqGroupInvites = append(reqGroupInvites, &protomfx.GroupInvite{
				GroupID:    group.GroupID,
				MemberRole: group.MemberRole,
			})
		}

		dormantInviteReq := &protomfx.CreateDormantOrgInviteReq{
			Token:            token,
			OrgID:            orgInvite.OrgID,
			InviteeRole:      orgInvite.InviteeRole,
			GroupInvites:     reqGroupInvites,
			PlatformInviteID: inviteID,
		}

		if _, err := svc.auth.CreateDormantOrgInvite(ctx, dormantInviteReq); err != nil {
			return PlatformInvite{}, err
		}
	}

	go func() {
		svc.SendPlatformInviteEmail(ctx, invite, redirectPath)
	}()

	return invite, nil
}

func (svc usersService) RevokePlatformInvite(ctx context.Context, token, inviteID string) error {
	if err := svc.isAdmin(ctx, token); err != nil {
		return err
	}

	invite, err := svc.invites.RetrievePlatformInviteByID(ctx, inviteID)
	if err != nil {
		return err
	}

	if invite.State != InviteStatePending {
		if invite.State == InviteStateExpired {
			return apiutil.ErrInviteExpired
		}

		return apiutil.ErrInvalidInviteState
	}

	if err := svc.invites.UpdatePlatformInviteState(ctx, inviteID, InviteStateRevoked); err != nil {
		return err
	}

	return nil
}

func (svc usersService) ViewPlatformInvite(ctx context.Context, token, inviteID string) (PlatformInvite, error) {
	if err := svc.isAdmin(ctx, token); err != nil {
		return PlatformInvite{}, err
	}

	invite, err := svc.invites.RetrievePlatformInviteByID(ctx, inviteID)
	if err != nil {
		return PlatformInvite{}, err
	}

	return invite, nil
}

func (svc usersService) ListPlatformInvites(ctx context.Context, token string, pm PageMetadataInvites) (PlatformInvitesPage, error) {
	if err := svc.isAdmin(ctx, token); err != nil {
		return PlatformInvitesPage{}, err
	}

	invitesPage, err := svc.invites.RetrievePlatformInvites(ctx, pm)
	if err != nil {
		return PlatformInvitesPage{}, err
	}

	return invitesPage, nil
}

func (svc usersService) ValidatePlatformInvite(ctx context.Context, inviteID, email string) error {
	invite, err := svc.invites.RetrievePlatformInviteByID(ctx, inviteID)
	if err != nil {
		if errors.Contains(err, dbutil.ErrNotFound) {
			return errors.Wrap(errors.ErrAuthorization, err)
		}

		return err
	}

	if invite.InviteeEmail != email {
		return errors.ErrAuthorization
	}

	if invite.State != InviteStatePending {
		if invite.State == InviteStateExpired {
			return errors.Wrap(errors.ErrAuthorization, apiutil.ErrInviteExpired)
		}

		return errors.Wrap(errors.ErrAuthorization, apiutil.ErrInvalidInviteState)
	}

	if err := svc.invites.UpdatePlatformInviteState(ctx, inviteID, InviteStateAccepted); err != nil {
		return err
	}

	return nil
}

func (svc usersService) SendPlatformInviteEmail(ctx context.Context, invite PlatformInvite, redirectPath string) error {
	to := []string{invite.InviteeEmail}
	return svc.email.SendPlatformInvite(to, invite, redirectPath)
}
