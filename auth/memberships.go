package auth

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var (
	// ErrCreateOrgMembership indicates failure to create org membership.
	ErrCreateOrgMembership = errors.New("failed to create org membership")

	// ErrRemoveOrgMembership indicates failure to remove org membership.
	ErrRemoveOrgMembership = errors.New("failed to remove org membership")

	// ErrOrgMembershipExists indicates that membership already exists.
	ErrOrgMembershipExists = errors.New("org membership already exists")
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

type OrgMembershipsBackup struct {
	OrgMemberships []OrgMembership
}

type OrgMembershipsRepository interface {
	// Save saves memberships.
	Save(ctx context.Context, oms ...OrgMembership) error

	// Update updates memberships.
	Update(ctx context.Context, oms ...OrgMembership) error

	// Remove removes memberships.
	Remove(ctx context.Context, orgID string, memberIDs ...string) error

	// RetrieveRole retrieves role of membership specified by memberID and orgID.
	RetrieveRole(ctx context.Context, memberID, orgID string) (string, error)

	// RetrieveByOrg retrieves org memberships identified by orgID.
	RetrieveByOrg(ctx context.Context, orgID string, pm apiutil.PageMetadata) (OrgMembershipsPage, error)

	// BackupAll retrieves all memberships.
	BackupAll(ctx context.Context) ([]OrgMembership, error)

	// BackupByOrg retrieves all memberships by org ID.
	BackupByOrg(ctx context.Context, orgID string) ([]OrgMembership, error)
}

// OrgMemberships specify an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type OrgMemberships interface {
	// CreateOrgMemberships adds memberships with member emails into the org identified by orgID.
	CreateOrgMemberships(ctx context.Context, token, orgID string, oms ...OrgMembership) error

	// RemoveOrgMemberships removes memberships with member ids from org identified by orgID.
	RemoveOrgMemberships(ctx context.Context, token string, orgID string, memberIDs ...string) error

	// UpdateOrgMemberships updates membership roles in an org.
	UpdateOrgMemberships(ctx context.Context, token, orgID string, oms ...OrgMembership) error

	// ListOrgMemberships retrieves memberships created for an org identified by orgID.
	ListOrgMemberships(ctx context.Context, token, orgID string, pm apiutil.PageMetadata) (OrgMembershipsPage, error)

	// ViewOrgMembership retrieves membership identified by memberID and orgID.
	ViewOrgMembership(ctx context.Context, token, orgID, memberID string) (OrgMembership, error)

	// BackupOrgMemberships retrieves all org memberships for given org ID.
	BackupOrgMemberships(ctx context.Context, token string, orgID string) (OrgMembershipsBackup, error)

	// RestoreOrgMemberships adds all org memberships for given org ID from a backup.
	RestoreOrgMemberships(ctx context.Context, token string, orgID string, backup OrgMembershipsBackup) error
}

func (svc service) CreateOrgMemberships(ctx context.Context, token, orgID string, oms ...OrgMembership) error {
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

func (svc service) ViewOrgMembership(ctx context.Context, token, orgID, memberID string) (OrgMembership, error) {
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

func (svc service) ListOrgMemberships(ctx context.Context, token string, orgID string, pm apiutil.PageMetadata) (OrgMembershipsPage, error) {
	if err := svc.canAccessOrg(ctx, token, orgID, Viewer); err != nil {
		return OrgMembershipsPage{}, err
	}

	memberships, err := svc.memberships.BackupByOrg(ctx, orgID)
	if err != nil {
		return OrgMembershipsPage{}, errors.Wrap(ErrRetrieveMembershipsByOrg, err)
	}

	if len(memberships) == 0 {
		return OrgMembershipsPage{
			OrgMemberships: []OrgMembership{},
			PageMetadata: apiutil.PageMetadata{
				Total:  0,
				Offset: pm.Offset,
				Limit:  pm.Limit,
				Email:  pm.Email,
			},
		}, nil
	}

	memberIDs := make([]string, 0, len(memberships))
	membershipByMemberID := make(map[string]OrgMembership, len(memberships))
	for _, m := range memberships {
		memberIDs = append(memberIDs, m.MemberID)
		membershipByMemberID[m.MemberID] = m
	}

	userReq := &protomfx.UsersByIDsReq{
		Ids: memberIDs,
		PageMetadata: &protomfx.PageMetadata{
			Email:  pm.Email,
			Order:  pm.Order,
			Dir:    pm.Dir,
			Limit:  pm.Limit,
			Offset: pm.Offset,
		},
	}

	res, err := svc.users.GetUsersByIDs(ctx, userReq)
	if err != nil {
		return OrgMembershipsPage{}, err
	}

	var oms []OrgMembership
	for _, u := range res.Users {
		if m, ok := membershipByMemberID[u.Id]; ok {
			m.Email = u.Email
			oms = append(oms, m)
		}
	}

	return OrgMembershipsPage{
		OrgMemberships: oms,
		PageMetadata: apiutil.PageMetadata{
			Total:  res.PageMetadata.Total,
			Offset: res.PageMetadata.Offset,
			Limit:  res.PageMetadata.Limit,
			Email:  res.PageMetadata.Email,
			Order:  res.PageMetadata.Order,
			Dir:    res.PageMetadata.Dir,
		},
	}, nil
}

func (svc service) UpdateOrgMemberships(ctx context.Context, token, orgID string, members ...OrgMembership) error {
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

func (svc service) RemoveOrgMemberships(ctx context.Context, token string, orgID string, memberIDs ...string) error {
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

func (svc service) BackupOrgMemberships(ctx context.Context, token string, orgID string) (OrgMembershipsBackup, error) {
	if err := svc.canAccessOrg(ctx, token, orgID, Owner); err != nil {
		return OrgMembershipsBackup{}, err
	}

	memberships, err := svc.memberships.BackupByOrg(ctx, orgID)
	if err != nil {
		return OrgMembershipsBackup{}, err
	}

	var memberIDs []string
	for _, gm := range memberships {
		memberIDs = append(memberIDs, gm.MemberID)
	}

	usersResp, err := svc.users.GetUsersByIDs(ctx, &protomfx.UsersByIDsReq{Ids: memberIDs})
	if err != nil {
		return OrgMembershipsBackup{}, err
	}

	emailMap := make(map[string]string)
	for _, user := range usersResp.Users {
		emailMap[user.Id] = user.Email
	}

	for i := range memberships {
		memberships[i].Email = emailMap[memberships[i].MemberID]
	}

	return OrgMembershipsBackup{
		OrgMemberships: memberships,
	}, nil
}

func (svc service) RestoreOrgMemberships(ctx context.Context, token string, orgID string, backup OrgMembershipsBackup) error {
	if err := svc.canAccessOrg(ctx, token, orgID, Owner); err != nil {
		return err
	}

	if err := svc.memberships.Save(ctx, backup.OrgMemberships...); err != nil {
		return err
	}

	return nil
}
