package mocks

import (
	"context"

	domainauth "github.com/MainfluxLabs/mainflux/pkg/domain/auth"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var _ domainauth.Client = (*authServiceMock)(nil)

type SubjectSet struct {
	Object   string
	Relation string
}

type authServiceMock struct {
	users map[string]string
	authz map[string][]SubjectSet
}

// NewAuth creates mock of auth service client.
func NewAuth(users map[string]string, authz map[string][]SubjectSet) domainauth.Client {
	return &authServiceMock{users, authz}
}

func (svc authServiceMock) Identify(_ context.Context, token string) (domainauth.Identity, error) {
	if id, ok := svc.users[token]; ok {
		return domainauth.Identity{ID: id, Email: id}, nil
	}
	return domainauth.Identity{}, errors.ErrAuthentication
}

func (svc authServiceMock) Issue(_ context.Context, id, email string, keyType uint32) (string, error) {
	if id, ok := svc.users[email]; ok {
		switch keyType {
		default:
			return id, nil
		}
	}
	return "", errors.ErrAuthentication
}

func (svc authServiceMock) Authorize(_ context.Context, ar domainauth.AuthzReq) error {
	if ar.Token != "token" {
		return errors.ErrAuthorization
	}

	return nil
}

func (svc authServiceMock) GetOwnerIDByOrg(context.Context, string) (string, error) {
	panic("not implemented")
}

func (svc authServiceMock) AssignRole(context.Context, string, string) error {
	panic("not implemented")
}

func (svc authServiceMock) RetrieveRole(context.Context, string) (string, error) {
	panic("not implemented")
}

func (svc authServiceMock) CreateDormantOrgInvite(context.Context, string, string, string, string, []domainauth.GroupInvite) error {
	panic("not implemented")
}

func (svc authServiceMock) ActivateOrgInvite(context.Context, string, string, string) error {
	panic("not implemented")
}

func (svc authServiceMock) GetDormantOrgInviteByPlatformInvite(context.Context, string) (domainauth.OrgInvite, error) {
	panic("not implemented")
}

func (svc authServiceMock) ViewOrg(context.Context, string, string) (domainauth.Org, error) {
	panic("not implemented")
}
