// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

var _ mainflux.AuthServiceClient = (*authServiceMock)(nil)

type MockSubjectSet struct {
	Object   string
	Relation string
}

type authServiceMock struct {
	users    map[string]string
	policies map[string][]MockSubjectSet
}

// NewAuthService creates mock of users service.
func NewAuthService(users map[string]string, policies map[string][]MockSubjectSet) mainflux.AuthServiceClient {
	return &authServiceMock{users, policies}
}

func (svc authServiceMock) Identify(ctx context.Context, in *mainflux.Token, opts ...grpc.CallOption) (*mainflux.UserIdentity, error) {
	if id, ok := svc.users[in.Value]; ok {
		return &mainflux.UserIdentity{Id: id, Email: id}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Issue(ctx context.Context, in *mainflux.IssueReq, opts ...grpc.CallOption) (*mainflux.Token, error) {
	if id, ok := svc.users[in.GetEmail()]; ok {
		switch in.Type {
		default:
			return &mainflux.Token{Value: id}, nil
		}
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Authorize(ctx context.Context, req *mainflux.AuthorizeReq, _ ...grpc.CallOption) (r *mainflux.AuthorizeRes, err error) {
	for _, policy := range svc.policies[req.GetSub()] {
		if policy.Relation == req.GetAct() && policy.Object == req.GetObj() {
			return &mainflux.AuthorizeRes{Authorized: true}, nil
		}
	}
	return nil, errors.ErrAuthorization
}

func (svc authServiceMock) Members(ctx context.Context, req *mainflux.MembersReq, _ ...grpc.CallOption) (r *mainflux.MembersRes, err error) {
	panic("not implemented")
}

func (svc authServiceMock) Assign(ctx context.Context, req *mainflux.Assignment, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	panic("not implemented")
}

func (svc authServiceMock) CanAccessGroup(ctx context.Context, in *mainflux.AccessGroupReq, opts ...grpc.CallOption) (*empty.Empty, error) {
	panic("not implemented")
}
