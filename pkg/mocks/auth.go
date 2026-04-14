// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	domain "github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ domain.AuthClient = (*authServiceMock)(nil)

type authServiceMock struct {
	roles        map[string][]string
	usersByEmail map[string]users.User
	orgs         map[string]auth.Org
}

// NewAuthService creates mock of auth service client.
func NewAuthService(adminID string, userList []users.User, orgList []auth.Org) domain.AuthClient {
	usersByEmail := make(map[string]users.User)
	roles := map[string][]string{auth.RootSub: {adminID}}
	orgs := make(map[string]auth.Org)

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

func (svc authServiceMock) Identify(_ context.Context, token string) (domain.Identity, error) {
	if u, ok := svc.usersByEmail[token]; ok {
		return domain.Identity{ID: u.ID, Email: u.Email}, nil
	}
	return domain.Identity{}, errors.ErrAuthentication
}

func (svc authServiceMock) Issue(_ context.Context, id, email string, keyType uint32) (string, error) {
	if u, ok := svc.usersByEmail[email]; ok {
		switch keyType {
		default:
			return u.Email, nil
		}
	}
	return "", errors.ErrAuthentication
}

func (svc authServiceMock) Authorize(_ context.Context, ar domain.AuthzReq) error {
	u, ok := svc.usersByEmail[ar.Token]
	if !ok {
		return errors.ErrAuthentication
	}

	switch ar.Subject {
	case auth.RootSub:
		if !contains(svc.roles[auth.RootSub], u.ID) {
			return errors.ErrAuthorization
		}
	case auth.OrgSub:
		if err := svc.canAccessOrg(u.ID, ar.Action); err != nil {
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
	isRoot := contains(svc.roles[auth.RootSub], userID)
	isOwner := isRoot || contains(svc.roles[auth.Owner], userID)
	isEditor := isOwner || contains(svc.roles[auth.Editor], userID)
	isViewer := isEditor || contains(svc.roles[auth.Viewer], userID)

	switch action {
	case auth.RootSub:
		if !isRoot {
			return errors.ErrAuthorization
		}
		return nil
	case auth.Owner:
		if !isOwner {
			return errors.ErrAuthorization
		}
		return nil
	case auth.Editor:
		if !isEditor {
			return errors.ErrAuthorization
		}
		return nil
	case auth.Viewer:
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

func (svc authServiceMock) AssignRole(_ context.Context, _, _ string) error {
	panic("not implemented")
}

func (svc authServiceMock) RetrieveRole(_ context.Context, _ string) (string, error) {
	panic("not implemented")
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
