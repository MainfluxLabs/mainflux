package auth

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var (
	// ErrCreateMembership indicates failure to create org membership.
	ErrCreateMembership = errors.New("failed to create org membership")

	// ErrRemoveMembership indicates failure to remove org membership.
	ErrRemoveMembership = errors.New("failed to remove org membership")

	// ErrMembershipExists indicates that membership already exists.
	ErrMembershipExists = errors.New("membership already exists")
)

type OrgMembership struct {
	MemberID  string
	OrgID     string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
	Email     string
}

// OrgMembershipsPage contains page related metadata as well as list of memberships that
// belong to this page.
type OrgMembershipsPage struct {
	apiutil.PageMetadata
	OrgMemberships []OrgMembership
}

type MembershipsRepository interface {
	// Save saves memberships.
	Save(ctx context.Context, oms ...OrgMembership) error

	// Update updates memberships.
	Update(ctx context.Context, oms ...OrgMembership) error

	// Remove removes memberships.
	Remove(ctx context.Context, orgID string, memberIDs ...string) error

	// RetrieveRole retrieves role of membership specified by memberID and orgID.
	RetrieveRole(ctx context.Context, memberID, orgID string) (string, error)

	// RetrieveByOrgID retrieves org memberships identified by orgID.
	RetrieveByOrgID(ctx context.Context, orgID string, pm apiutil.PageMetadata) (OrgMembershipsPage, error)

	// RetrieveAll retrieves all memberships.
	RetrieveAll(ctx context.Context) ([]OrgMembership, error)
}

// Memberships specify an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Memberships interface {
	// CreateMemberships adds memberships with member emails into the org identified by orgID.
	CreateMemberships(ctx context.Context, token, orgID string, oms ...OrgMembership) error

	// RemoveMemberships removes memberships with member ids from org identified by orgID.
	RemoveMemberships(ctx context.Context, token string, orgID string, memberIDs ...string) error

	// UpdateMemberships updates membership roles in an org.
	UpdateMemberships(ctx context.Context, token, orgID string, oms ...OrgMembership) error

	// ListMembershipsByOrg retrieves memberships created for an org identified by orgID.
	ListMembershipsByOrg(ctx context.Context, token, orgID string, pm apiutil.PageMetadata) (OrgMembershipsPage, error)

	// ViewMembership retrieves membership identified by memberID and orgID.
	ViewMembership(ctx context.Context, token, orgID, memberID string) (OrgMembership, error)
}

func (svc service) CreateMemberships(ctx context.Context, token, orgID string, oms ...OrgMembership) error {
	if err := svc.canAccessOrg(ctx, token, orgID, Admin); err != nil {
		return err
	}

	var memberEmails []string
	var roleByEmail = make(map[string]string)
	for _, om := range oms {
		roleByEmail[om.Email] = om.Role
		memberEmails = append(memberEmails, om.Email)
	}

	muReq := protomfx.UsersByEmailsReq{Emails: memberEmails}
	usr, err := svc.users.GetUsersByEmails(ctx, &muReq)
	if err != nil {
		return err
	}

	timestamp := getTimestmap()
	var memberships []OrgMembership
	for _, user := range usr.Users {
		membership := OrgMembership{
			OrgID:     orgID,
			MemberID:  user.Id,
			Role:      roleByEmail[user.Email],
			UpdatedAt: timestamp,
			CreatedAt: timestamp,
		}

		memberships = append(memberships, membership)
	}

	if err := svc.memberships.Save(ctx, memberships...); err != nil {
		return err
	}

	return nil
}

func (svc service) ViewMembership(ctx context.Context, token, orgID, memberID string) (OrgMembership, error) {
	if err := svc.canAccessOrg(ctx, token, orgID, Viewer); err != nil {
		return OrgMembership{}, err
	}

	usrReq := protomfx.UsersByIDsReq{Ids: []string{memberID}}
	page, err := svc.users.GetUsersByIDs(ctx, &usrReq)
	if err != nil {
		return OrgMembership{}, err
	}

	role, err := svc.memberships.RetrieveRole(ctx, memberID, orgID)
	if err != nil {
		return OrgMembership{}, err
	}

	membership := OrgMembership{
		MemberID: page.Users[0].Id,
		Email:    page.Users[0].Email,
		Role:     role,
	}

	return membership, nil
}

func (svc service) ListMembershipsByOrg(ctx context.Context, token string, orgID string, pm apiutil.PageMetadata) (OrgMembershipsPage, error) {
	if err := svc.canAccessOrg(ctx, token, orgID, Viewer); err != nil {
		return OrgMembershipsPage{}, err
	}

	omp, err := svc.memberships.RetrieveByOrgID(ctx, orgID, pm)
	if err != nil {
		return OrgMembershipsPage{}, errors.Wrap(ErrRetrieveMembershipsByOrg, err)
	}

	var oms []OrgMembership
	var page *protomfx.UsersRes
	if len(omp.OrgMemberships) > 0 {
		var memberIDs []string
		var roleByMemberID = make(map[string]string)
		for _, m := range omp.OrgMemberships {
			roleByMemberID[m.MemberID] = m.Role
			memberIDs = append(memberIDs, m.MemberID)
		}

		usrReq := protomfx.UsersByIDsReq{Ids: memberIDs, Email: pm.Email, Order: pm.Order, Dir: pm.Dir}
		page, err = svc.users.GetUsersByIDs(ctx, &usrReq)
		if err != nil {
			return OrgMembershipsPage{}, err
		}

		for _, user := range page.Users {
			om := OrgMembership{
				MemberID: user.Id,
				Email:    user.Email,
				Role:     roleByMemberID[user.Id],
			}
			oms = append(oms, om)
		}
	}

	mpg := OrgMembershipsPage{
		OrgMemberships: oms,
		PageMetadata: apiutil.PageMetadata{
			Total:  omp.Total,
			Offset: omp.Offset,
			Limit:  omp.Limit,
			Email:  omp.Email,
		},
	}

	return mpg, nil
}

func (svc service) UpdateMemberships(ctx context.Context, token, orgID string, members ...OrgMembership) error {
	if err := svc.canAccessOrg(ctx, token, orgID, Admin); err != nil {
		return err
	}

	org, err := svc.orgs.RetrieveByID(ctx, orgID)
	if err != nil {
		return err
	}

	var memberEmails []string
	var roleByEmail = make(map[string]string)
	for _, m := range members {
		roleByEmail[m.Email] = m.Role
		memberEmails = append(memberEmails, m.Email)
	}

	muReq := protomfx.UsersByEmailsReq{Emails: memberEmails}
	usr, err := svc.users.GetUsersByEmails(ctx, &muReq)
	if err != nil {
		return err
	}

	var oms []OrgMembership
	for _, user := range usr.Users {
		if user.Id == org.OwnerID {
			return errors.ErrAuthorization
		}

		om := OrgMembership{
			OrgID:     orgID,
			MemberID:  user.Id,
			Role:      roleByEmail[user.Email],
			UpdatedAt: getTimestmap(),
		}

		oms = append(oms, om)
	}

	if err := svc.memberships.Update(ctx, oms...); err != nil {
		return err
	}

	return nil
}

func (svc service) RemoveMemberships(ctx context.Context, token string, orgID string, memberIDs ...string) error {
	if err := svc.canRemoveMemberships(ctx, token, orgID, memberIDs...); err != nil {
		return err
	}

	if err := svc.memberships.Remove(ctx, orgID, memberIDs...); err != nil {
		return err
	}

	return nil
}

func (svc service) canRemoveMemberships(ctx context.Context, token, orgID string, memberIDs ...string) error {
	if err := svc.canAccessOrg(ctx, token, orgID, Admin); err != nil {
		return err
	}

	for _, memberID := range memberIDs {
		role, err := svc.memberships.RetrieveRole(ctx, memberID, orgID)
		if err != nil {
			return err
		}

		if role == Owner {
			return errors.ErrAuthorization
		}
	}

	return nil
}
