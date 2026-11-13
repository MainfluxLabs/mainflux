// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"strings"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	domain "github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	recoveryTokenPrefix     = "recovery-"
	impersonatedTokenPrefix = "impersonated-"
)

var _ domain.AuthClient = (*authServiceMock)(nil)

type authServiceMock struct {
	platformRoles map[string]string
	orgRoles      map[string]map[string]string
	usersByEmail  map[string]users.User
	orgs          map[string]auth.Org
	impersonators map[string]domain.Identity
}

// NewAuthService creates mock of auth service client.
// Org owners come from orgList, other org roles come from orgMemberships.
func NewAuthService(adminID string, userList []users.User, orgList []auth.Org, orgMemberships ...auth.OrgMembership) domain.AuthClient {
	usersByEmail := make(map[string]users.User)
	platformRoles := map[string]string{adminID: auth.RoleRootAdmin}
	orgRoles := make(map[string]map[string]string)
	orgs := make(map[string]auth.Org)

	for _, user := range userList {
		usersByEmail[user.Email] = user
		if user.Role != "" {
			platformRoles[user.ID] = user.Role
		}
	}

	for _, o := range orgList {
		orgs[o.ID] = o
		orgMemberships = append(orgMemberships, auth.OrgMembership{OrgID: o.ID, MemberID: o.OwnerID, Role: auth.Owner})
	}

	for _, om := range orgMemberships {
		if _, ok := orgRoles[om.OrgID]; !ok {
			orgRoles[om.OrgID] = make(map[string]string)
		}
		orgRoles[om.OrgID][om.MemberID] = om.Role
	}

	return &authServiceMock{
		platformRoles: platformRoles,
		orgRoles:      orgRoles,
		usersByEmail:  usersByEmail,
		orgs:          orgs,
		impersonators: make(map[string]domain.Identity),
	}
}

func (svc authServiceMock) Identify(_ context.Context, token string) (domain.Identity, error) {
	if u, ok := svc.usersByEmail[token]; ok {
		return domain.Identity{ID: u.ID, Email: u.Email}, nil
	}

	impersonator, ok := svc.impersonators[token]
	if !ok {
		return domain.Identity{}, errors.ErrAuthentication
	}

	email := strings.TrimPrefix(token, impersonatedTokenPrefix)
	u := svc.usersByEmail[email]
	identity := domain.Identity{
		ID:           u.ID,
		Email:        u.Email,
		Impersonator: &impersonator,
	}

	return identity, nil
}

func (svc authServiceMock) IdentifyRecovery(ctx context.Context, token string) (domain.Identity, error) {
	email, ok := strings.CutPrefix(token, recoveryTokenPrefix)
	if !ok {
		return domain.Identity{}, errors.ErrAuthentication
	}

	return svc.Identify(ctx, email)
}

func (svc authServiceMock) Issue(_ context.Context, id, email string, keyType uint32) (string, error) {
	if u, ok := svc.usersByEmail[email]; ok {
		switch keyType {
		case domain.RecoveryKey:
			return recoveryTokenPrefix + u.Email, nil
		default:
			return u.Email, nil
		}
	}
	return "", errors.ErrAuthentication
}

func (svc authServiceMock) IssueImpersonated(_ context.Context, id, email string, impersonator domain.Identity) (string, error) {
	if _, ok := svc.usersByEmail[email]; !ok {
		return "", errors.ErrAuthentication
	}

	token := impersonatedTokenPrefix + email
	svc.impersonators[token] = impersonator

	return token, nil
}

func (svc authServiceMock) RefreshToken(_ context.Context, token string) (string, error) {
	u, ok := svc.usersByEmail[token]
	if !ok {
		return "", errors.ErrAuthentication
	}

	return u.Email, nil
}

func (svc authServiceMock) RevokeSessions(_ context.Context, id string) error {
	for token, impersonator := range svc.impersonators {
		if impersonator.ID == id {
			delete(svc.impersonators, token)
		}
	}

	return nil
}

func (svc authServiceMock) Authorize(_ context.Context, ar domain.AuthzReq) error {
	u, ok := svc.usersByEmail[ar.Token]
	if !ok {
		return errors.ErrAuthentication
	}

	switch ar.Subject {
	case auth.RootSub:
		if !svc.isPlatformAdmin(u.ID) {
			return errors.ErrAuthorization
		}
	case auth.OrgSub:
		if err := svc.canAccessOrg(u.ID, ar.Object, ar.Action); err != nil {
			return err
		}
	default:
		return errors.ErrAuthorization
	}

	return nil
}

func (svc authServiceMock) isPlatformAdmin(userID string) bool {
	role := svc.platformRoles[userID]
	return role == auth.RoleRootAdmin || role == auth.RoleAdmin
}

func (svc authServiceMock) canAccessOrg(userID, orgID, action string) error {
	if svc.isPlatformAdmin(userID) {
		return nil
	}

	role, ok := svc.orgRoles[orgID][userID]
	if !ok {
		return status.Error(codes.NotFound, dbutil.ErrNotFound.Error())
	}

	switch role {
	case auth.Owner:
		return nil
	case auth.Admin:
		if action != auth.Owner {
			return nil
		}
	case auth.Editor:
		if action == auth.Viewer || action == auth.Editor {
			return nil
		}
	case auth.Viewer:
		if action == auth.Viewer {
			return nil
		}
	}

	return errors.ErrAuthorization
}

func (svc authServiceMock) GetOwnerIDByOrg(_ context.Context, orgID string) (string, error) {
	for id, org := range svc.orgs {
		if id == orgID {
			return org.OwnerID, nil
		}
	}
	return "", dbutil.ErrNotFound
}

func (svc authServiceMock) AssignRole(_ context.Context, id, role string) error {
	// The roles table allows a single role per user.
	if svc.platformRoles[id] != "" {
		return dbutil.ErrCreateEntity
	}

	svc.platformRoles[id] = role
	return nil
}

func (svc authServiceMock) RetrieveRole(_ context.Context, id string) (string, error) {
	return svc.platformRoles[id], nil
}

func (svc authServiceMock) RetrieveRoles(_ context.Context, ids []string) (map[string]string, error) {
	roles := make(map[string]string)
	for _, id := range ids {
		if role, ok := svc.platformRoles[id]; ok {
			roles[id] = role
		}
	}

	return roles, nil
}

func (svc authServiceMock) UnassignRole(_ context.Context, id string) error {
	if _, ok := svc.platformRoles[id]; !ok {
		return dbutil.ErrNotFound
	}

	delete(svc.platformRoles, id)
	return nil
}

func (svc authServiceMock) CreateDormantOrgInvite(_ context.Context, _, _, _, _ string, _ []domain.GroupInvite) error {
	panic("not implemented")
}

func (svc authServiceMock) ActivateOrgInvite(_ context.Context, _, _, _ string) error {
	panic("not implemented")
}

func (svc authServiceMock) GetDormantOrgInviteByPlatformInvite(_ context.Context, _ string) (domain.OrgInvite, error) {
	return domain.OrgInvite{}, status.Error(codes.NotFound, dbutil.ErrNotFound.Error())
}

func (svc authServiceMock) ViewOrg(_ context.Context, token, orgID string) (domain.Org, error) {
	org, ok := svc.orgs[orgID]
	if !ok {
		return domain.Org{}, dbutil.ErrNotFound
	}

	return domain.Org{
		ID:      org.ID,
		OwnerID: org.OwnerID,
		Name:    org.Name,
	}, nil
}

func (svc authServiceMock) GetAdminsByOrg(_ context.Context, orgID string) ([]domain.OrgMembership, error) {
	var oms []domain.OrgMembership
	for memberID, role := range svc.orgRoles[orgID] {
		if role == auth.Owner || role == auth.Admin {
			oms = append(oms, domain.OrgMembership{
				MemberID: memberID,
				Role:     role,
			})
		}
	}

	return oms, nil
}

func (svc authServiceMock) GetOrgIDsByAdmin(_ context.Context, memberID string) ([]string, error) {
	var orgIDs []string
	for orgID, roles := range svc.orgRoles {
		if role := roles[memberID]; role == auth.Owner || role == auth.Admin {
			orgIDs = append(orgIDs, orgID)
		}
	}

	return orgIDs, nil
}

func (svc authServiceMock) GetOrgMembership(_ context.Context, orgID, memberID string) (domain.OrgMembership, error) {
	role := svc.orgRoles[orgID][memberID]
	return domain.OrgMembership{OrgID: orgID, MemberID: memberID, Role: role}, nil
}
