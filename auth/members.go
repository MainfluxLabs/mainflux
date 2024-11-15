package auth

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

type OrgMember struct {
	MemberID  string
	OrgID     string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
	Email     string
}

// OrgMembersPage contains page related metadata as well as list of members that
// belong to this page.
type OrgMembersPage struct {
	PageMetadata
	OrgMembers []OrgMember
}

type MembersRepository interface {
	// Save saves membershipa.
	Save(ctx context.Context, oms ...OrgMember) error

	// Update updates memberships.
	Update(ctx context.Context, oms ...OrgMember) error

	// Remove removes memberships.
	Remove(ctx context.Context, orgID string, memberIDs ...string) error

	// RetrieveRole retrieves role of membership specified by memberID and orgID.
	RetrieveRole(ctx context.Context, memberID, orgID string) (string, error)

	// RetrieveByOrgID retrieves members assigned to an org identified by orgID.
	RetrieveByOrgID(ctx context.Context, orgID string, pm PageMetadata) (OrgMembersPage, error)

	// RetrieveAll retrieves all members.
	RetrieveAll(ctx context.Context) ([]OrgMember, error)
}

// Memberships specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Members interface {
	// AssignMembers adds members with member emails into the org identified by orgID.
	AssignMembers(ctx context.Context, token, orgID string, oms ...OrgMember) error

	// UnassignMembers removes members with member ids from org identified by orgID.
	UnassignMembers(ctx context.Context, token string, orgID string, memberIDs ...string) error

	// UpdateMembers updates members role in an org.
	UpdateMembers(ctx context.Context, token, orgID string, oms ...OrgMember) error

	// ListMembersByOrg retrieves members assigned to an org identified by orgID.
	ListMembersByOrg(ctx context.Context, token, orgID string, pm PageMetadata) (OrgMembersPage, error)

	// ViewMember retrieves member identified by memberID in org identified by orgID.
	ViewMember(ctx context.Context, token, orgID, memberID string) (OrgMember, error)
}

func (svc service) AssignMembers(ctx context.Context, token, orgID string, oms ...OrgMember) error {
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
	var members []OrgMember
	for _, user := range usr.Users {
		member := OrgMember{
			OrgID:     orgID,
			MemberID:  user.Id,
			Role:      roleByEmail[user.Email],
			UpdatedAt: timestamp,
			CreatedAt: timestamp,
		}

		members = append(members, member)
	}

	if err := svc.members.Save(ctx, members...); err != nil {
		return err
	}

	return nil
}

func (svc service) UnassignMembers(ctx context.Context, token string, orgID string, memberIDs ...string) error {
	if err := svc.canAssignMembers(ctx, token, orgID, memberIDs...); err != nil {
		return err
	}

	if err := svc.members.Remove(ctx, orgID, memberIDs...); err != nil {
		return err
	}

	return nil
}

func (svc service) ViewMember(ctx context.Context, token, orgID, memberID string) (OrgMember, error) {
	if err := svc.canAccessOrg(ctx, token, orgID, Viewer); err != nil {
		return OrgMember{}, err
	}

	usrReq := protomfx.UsersByIDsReq{Ids: []string{memberID}}
	page, err := svc.users.GetUsersByIDs(ctx, &usrReq)
	if err != nil {
		return OrgMember{}, err
	}

	role, err := svc.members.RetrieveRole(ctx, memberID, orgID)
	if err != nil {
		return OrgMember{}, err
	}

	member := OrgMember{
		MemberID: page.Users[0].Id,
		Email:    page.Users[0].Email,
		Role:     role,
	}

	return member, nil
}

func (svc service) UpdateMembers(ctx context.Context, token, orgID string, members ...OrgMember) error {
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

	var oms []OrgMember
	for _, user := range usr.Users {
		if user.Id == org.OwnerID {
			return errors.ErrAuthorization
		}

		om := OrgMember{
			OrgID:     orgID,
			MemberID:  user.Id,
			Role:      roleByEmail[user.Email],
			UpdatedAt: getTimestmap(),
		}

		oms = append(oms, om)
	}

	if err := svc.members.Update(ctx, oms...); err != nil {
		return err
	}

	return nil
}

func (svc service) ListMembersByOrg(ctx context.Context, token string, orgID string, pm PageMetadata) (OrgMembersPage, error) {
	if err := svc.canAccessOrg(ctx, token, orgID, Viewer); err != nil {
		return OrgMembersPage{}, err
	}

	omp, err := svc.members.RetrieveByOrgID(ctx, orgID, pm)
	if err != nil {
		return OrgMembersPage{}, errors.Wrap(ErrRetrieveMembersByOrg, err)
	}

	var oms []OrgMember
	if len(omp.OrgMembers) > 0 {
		var memberIDs []string
		var roleByEmail = make(map[string]string)
		for _, m := range omp.OrgMembers {
			roleByEmail[m.MemberID] = m.Role
			memberIDs = append(memberIDs, m.MemberID)
		}

		usrReq := protomfx.UsersByIDsReq{Ids: memberIDs}
		page, err := svc.users.GetUsersByIDs(ctx, &usrReq)
		if err != nil {
			return OrgMembersPage{}, err
		}

		for _, user := range page.Users {
			mbr := OrgMember{
				MemberID: user.Id,
				Email:    user.Email,
				Role:     roleByEmail[user.Id],
			}
			oms = append(oms, mbr)
		}
	}

	mpg := OrgMembersPage{
		OrgMembers: oms,
		PageMetadata: PageMetadata{
			Total:  omp.Total,
			Offset: omp.Offset,
			Limit:  omp.Limit,
		},
	}

	return mpg, nil
}

func (svc service) canAssignMembers(ctx context.Context, token, orgID string, memberIDs ...string) error {
	if err := svc.canAccessOrg(ctx, token, orgID, Admin); err != nil {
		return err
	}

	for _, memberID := range memberIDs {
		role, err := svc.members.RetrieveRole(ctx, memberID, orgID)
		if err != nil {
			return err
		}

		if role == Owner {
			return errors.ErrAuthorization
		}
	}

	return nil
}
