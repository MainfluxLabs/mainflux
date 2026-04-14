// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var errUnsupported = errors.New("not supported in standalone mode")

var _ domain.AuthClient = (*singleUserRepo)(nil)

type singleUserRepo struct {
	email string
	token string
}

// NewAuthService creates single user repository for constrained environments.
func NewAuthService(email, token string) domain.AuthClient {
	return singleUserRepo{
		email: email,
		token: token,
	}
}

func (repo singleUserRepo) Issue(ctx context.Context, id, email string, keyType uint32) (string, error) {
	if repo.token != email {
		return "", errors.ErrAuthentication
	}

	return repo.token, nil
}

func (repo singleUserRepo) Identify(ctx context.Context, token string) (domain.Identity, error) {
	if repo.token != token {
		return domain.Identity{}, errors.ErrAuthentication
	}

	return domain.Identity{ID: repo.email, Email: repo.email}, nil
}

func (repo singleUserRepo) Authorize(ctx context.Context, ar domain.AuthzReq) error {
	return errUnsupported
}

func (repo singleUserRepo) GetOwnerIDByOrg(ctx context.Context, orgID string) (string, error) {
	return "", errUnsupported
}

func (repo singleUserRepo) AssignRole(ctx context.Context, id, role string) error {
	return errUnsupported
}

func (repo singleUserRepo) RetrieveRole(ctx context.Context, id string) (string, error) {
	return "", errUnsupported
}

func (repo singleUserRepo) CreateDormantOrgInvite(ctx context.Context, token, orgID, inviteeRole, platformInviteID string, groupInvites []domain.GroupInvite) error {
	panic("not implemented")
}

func (repo singleUserRepo) ActivateOrgInvite(ctx context.Context, platformInviteID, userID, redirectPath string) error {
	panic("not implemented")
}

func (repo singleUserRepo) GetDormantOrgInviteByPlatformInvite(ctx context.Context, platformInviteID string) (domain.OrgInvite, error) {
	panic("not implemented")
}

func (repo singleUserRepo) ViewOrg(ctx context.Context, token, orgID string) (domain.Org, error) {
	panic("not implemented")
}
