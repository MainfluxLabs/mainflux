// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	user "github.com/MainfluxLabs/mainflux/users"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

var _ mainflux.AuthServiceClient = (*authServiceMock)(nil)

type authServiceMock struct {
	authz map[string]string
}

// NewAuthService creates mock of users service.
func NewAuthService(adminID string, users map[string]user.User) mainflux.AuthServiceClient {
	authz := make(map[string]string)
	authz["root"] = adminID

	mockUsers = users
	if mockUsersByID == nil {
		mockUsersByID = make(map[string]user.User)
	}
	for _, u := range users {
		mockUsersByID[u.ID] = u
	}

	return &authServiceMock{
		authz: authz,
	}
}

func (svc authServiceMock) Identify(ctx context.Context, in *mainflux.Token, opts ...grpc.CallOption) (*mainflux.UserIdentity, error) {
	if u, ok := mockUsers[in.Value]; ok {
		return &mainflux.UserIdentity{Id: u.ID, Email: u.Email}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Issue(ctx context.Context, in *mainflux.IssueReq, opts ...grpc.CallOption) (*mainflux.Token, error) {
	if u, ok := mockUsers[in.GetEmail()]; ok {
		switch in.Type {
		default:
			return &mainflux.Token{Value: u.Email}, nil
		}
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Authorize(ctx context.Context, req *mainflux.AuthorizeReq, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	u, ok := mockUsers[req.Token]
	if !ok {
		return &empty.Empty{}, errors.ErrAuthentication
	}

	if svc.authz["root"] != u.ID {
		return &empty.Empty{}, errors.ErrAuthorization
	}

	return &empty.Empty{}, nil
}

func (svc authServiceMock) Members(ctx context.Context, req *mainflux.MembersReq, _ ...grpc.CallOption) (r *mainflux.MembersRes, err error) {
	panic("not implemented")
}

func (svc authServiceMock) Assign(ctx context.Context, req *mainflux.Assignment, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	panic("not implemented")
}

func (svc authServiceMock) CanAccessGroup(ctx context.Context, in *mainflux.AccessGroupReq, opts ...grpc.CallOption) (r *empty.Empty, err error) {
	panic("not implemented")
}

func (svc authServiceMock) AssignRole(ctx context.Context, in *mainflux.AssignRoleReq, opts ...grpc.CallOption) (r *empty.Empty, err error) {
	panic("not implemented")
}
