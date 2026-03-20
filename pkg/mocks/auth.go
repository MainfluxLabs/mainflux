// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	domainauth "github.com/MainfluxLabs/mainflux/pkg/domain/auth"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
)

var _ domainauth.Client = (*authServiceMock)(nil)

type authServiceMock struct {
	roles        map[string][]string
	usersByEmail map[string]users.User
	orgs         map[string]domainauth.Org
}

// NewAuthService creates mock of users service.
func NewAuthService(adminID string, userList []users.User, orgList []domainauth.Org) domainauth.Client {
	usersByEmail := make(map[string]users.User)
	roles := map[string][]string{domainauth.RootSub: {adminID}}
	orgs := make(map[string]domainauth.Org)

	for _, user := range userList {
		usersByEmail[user.Email] = user
		roles[user.Role] = append(roles[user.Role], user.ID)
	}

	for _, o := range orgList {
		orgs[o.ID] = o
	}

	return &authServiceMock{
		roles:        roles,
		usersByEmail: usersByEmail,
		orgs:         orgs,
	}
}

func (svc authServiceMock) Identify(_ context.Context, token string) (domainauth.Identity, error) {
	if u, ok := svc.usersByEmail[token]; ok {
		return domainauth.Identity{ID: u.ID, Email: u.Email}, nil
	}
	return domainauth.Identity{}, errors.ErrAuthentication
}

func (svc authServiceMock) Issue(_ context.Context, id, email string, _ uint32) (string, error) {
	if u, ok := svc.usersByEmail[email]; ok {
		return u.Email, nil
	}
	return "", errors.ErrAuthentication
}

func (svc authServiceMock) Authorize(_ context.Context, req domainauth.AuthzReq) error {
	u, ok := svc.usersByEmail[req.Token]
	if !ok {
		return errors.ErrAuthentication
	}

	switch req.Subject {
	case domainauth.RootSub:
		if !contains(svc.roles[domainauth.RootSub], u.ID) {
			return errors.ErrAuthorization
		}
	case domainauth.OrgSub:
		if err := svc.canAccessOrg(u.ID, req.Action); err != nil {
			return err
		}
	default:
		return errors.ErrAuthorization
	}

	return nil
}

func contains(ids []string, id string) bool {
	for _, existingID := range ids {
		if existingID == id {
			return true
		}
	}
	return false
}

func (svc authServiceMock) canAccessOrg(userID, action string) error {
	isRoot := contains(svc.roles[domainauth.RootSub], userID)
	isOwner := isRoot || contains(svc.roles[domainauth.Owner], userID)
	isEditor := isOwner || contains(svc.roles[domainauth.Editor], userID)
	isViewer := isEditor || contains(svc.roles[domainauth.Viewer], userID)

	switch action {
	case domainauth.RootSub:
		if !isRoot {
			return errors.ErrAuthorization
		}
		return nil
	case domainauth.Owner:
		if !isOwner {
			return errors.ErrAuthorization
		}
		return nil
	case domainauth.Editor:
		if !isEditor {
			return errors.ErrAuthorization
		}
		return nil
	case domainauth.Viewer:
		if !isViewer {
			return errors.ErrAuthorization
		}
		return nil
	default:
		return errors.ErrAuthorization
	}
}

func (svc authServiceMock) GetOwnerIDByOrg(_ context.Context, orgID string) (string, error) {
	for id, org := range svc.orgs {
		if id == orgID {
			return org.OwnerID, nil
		}
	}
	return "", dbutil.ErrNotFound
}

func (svc authServiceMock) AssignRole(ctx context.Context, id, role string) error {
	panic("not implemented")
}

func (svc authServiceMock) RetrieveRole(ctx context.Context, id string) (string, error) {
	panic("not implemented")
}

func (svc authServiceMock) CreateDormantOrgInvite(ctx context.Context, token, orgID, inviteeRole string, groupInvites []domainauth.GroupInvite, platformInviteID string) error {
	panic("not implemented")
}

func (svc authServiceMock) ActivateOrgInvite(ctx context.Context, platformInviteID, userID, redirectPath string) error {
	panic("not implemented")
}

func (svc authServiceMock) GetDormantOrgInviteByPlatformInvite(ctx context.Context, platformInviteID string) (domainauth.OrgInvite, error) {
	return domainauth.OrgInvite{}, dbutil.ErrNotFound
}

func (svc authServiceMock) ViewOrg(_ context.Context, orgToken, orgID string) (domainauth.Org, error) {
	org, ok := svc.orgs[orgID]
	if !ok {
		return domainauth.Org{}, dbutil.ErrNotFound
	}

	return org, nil
}
