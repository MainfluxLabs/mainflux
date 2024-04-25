// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

var _ mainflux.AuthServiceClient = (*authServiceMock)(nil)

type authServiceMock struct {
	roles        map[string]string
	usersByEmail map[string]users.User
}

// NewAuthService creates mock of users service.
func NewAuthService(adminID string, userList []users.User) mainflux.AuthServiceClient {
	usersByEmail := make(map[string]users.User)
	roles := map[string]string{"root": adminID}

	for _, user := range userList {
		usersByEmail[user.Email] = user
	}

	return &authServiceMock{
		roles:        roles,
		usersByEmail: usersByEmail,
	}
}

func (svc authServiceMock) Identify(ctx context.Context, in *mainflux.Token, opts ...grpc.CallOption) (*mainflux.UserIdentity, error) {
	if u, ok := svc.usersByEmail[in.Value]; ok {
		return &mainflux.UserIdentity{Id: u.ID, Email: u.Email}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Issue(ctx context.Context, in *mainflux.IssueReq, opts ...grpc.CallOption) (*mainflux.Token, error) {
	if u, ok := svc.usersByEmail[in.GetEmail()]; ok {
		switch in.Type {
		default:
			return &mainflux.Token{Value: u.Email}, nil
		}
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Authorize(ctx context.Context, req *mainflux.AuthorizeReq, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	u, ok := svc.usersByEmail[req.Token]
	if !ok {
		return &empty.Empty{}, errors.ErrAuthentication
	}

	switch req.Subject {
	case "root":
		if svc.roles["root"] != u.ID {
			return &empty.Empty{}, errors.ErrAuthorization
		}
	default:
		return &empty.Empty{}, errors.ErrAuthorization
	}

	return &empty.Empty{}, nil
}

func (svc authServiceMock) AssignRole(ctx context.Context, in *mainflux.AssignRoleReq, opts ...grpc.CallOption) (r *empty.Empty, err error) {
	panic("not implemented")
}

func (svc authServiceMock) RetrieveRole(ctx context.Context, req *mainflux.RetrieveRoleReq, _ ...grpc.CallOption) (r *mainflux.RetrieveRoleRes, err error) {
	panic("not implemented")
}
