// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

var _ protomfx.AuthServiceClient = (*authServiceMock)(nil)

type authServiceMock struct {
	roles        map[string]string
	usersByEmail map[string]users.User
}

// NewAuthService creates mock of users service.
func NewAuthService(adminID string, userList []users.User) protomfx.AuthServiceClient {
	usersByEmail := make(map[string]users.User)
	roles := map[string]string{auth.RootSubject: adminID}

	for _, user := range userList {
		usersByEmail[user.Email] = user
	}

	return &authServiceMock{
		roles:        roles,
		usersByEmail: usersByEmail,
	}
}

func (svc authServiceMock) Identify(ctx context.Context, in *protomfx.Token, opts ...grpc.CallOption) (*protomfx.UserIdentity, error) {
	if u, ok := svc.usersByEmail[in.Value]; ok {
		return &protomfx.UserIdentity{Id: u.ID, Email: u.Email}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Issue(ctx context.Context, in *protomfx.IssueReq, opts ...grpc.CallOption) (*protomfx.Token, error) {
	if u, ok := svc.usersByEmail[in.GetEmail()]; ok {
		switch in.Type {
		default:
			return &protomfx.Token{Value: u.Email}, nil
		}
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Authorize(ctx context.Context, req *protomfx.AuthorizeReq, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	u, ok := svc.usersByEmail[req.Token]
	if !ok {
		return &empty.Empty{}, errors.ErrAuthentication
	}

	switch req.Subject {
	case auth.RootSubject:
		if svc.roles[auth.RootSubject] != u.ID {
			return &empty.Empty{}, errors.ErrAuthorization
		}
	default:
		return &empty.Empty{}, errors.ErrAuthorization
	}

	return &empty.Empty{}, nil
}

func (svc authServiceMock) AssignRole(ctx context.Context, in *protomfx.AssignRoleReq, opts ...grpc.CallOption) (r *empty.Empty, err error) {
	panic("not implemented")
}

func (svc authServiceMock) RetrieveRole(ctx context.Context, req *protomfx.RetrieveRoleReq, _ ...grpc.CallOption) (r *protomfx.RetrieveRoleRes, err error) {
	panic("not implemented")
}
